package context

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

type Indexer struct {
	parser   *Parser
	embedder *Embedder
	cfg      *config.Config
	cacheMu  sync.RWMutex
	docCache map[string]*models.Document // filePath -> document
}

func NewIndexer(cfg *config.Config) *Indexer {
	return &Indexer{
		parser:   NewParser(),
		embedder: NewEmbedder(cfg),
		cfg:      cfg,
		docCache: make(map[string]*models.Document),
	}
}

// IndexFile processes a single file
func (idx *Indexer) IndexFile(filePath string, forceReindex bool) ([]*models.Chunk, error) {
	logger := utils.GetLogger()

	// Check if file needs reindexing
	if !forceReindex {
		cached, needsUpdate := idx.needsReindex(filePath)
		if !needsUpdate && cached != nil {
			logger.Debugf("Using cached document: %s", filePath)
			return idx.loadCachedChunks(cached.ID)
		}
	}

	logger.Infof("Indexing file: %s", filePath)

	// Parse document
	doc, err := idx.parser.ParseDocument(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	// Chunk document
	chunks := idx.embedder.ChunkDocument(doc)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks created from document")
	}

	logger.Infof("Created %d chunks from %s", len(chunks), filePath)

	// Generate embeddings
	if err := idx.embedder.CreateEmbeddings(chunks); err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Cache document and chunks
	if err := idx.cacheDocument(doc, chunks); err != nil {
		logger.Warnf("Failed to cache document: %v", err)
	}

	idx.cacheMu.Lock()
	idx.docCache[filePath] = doc
	idx.cacheMu.Unlock()

	return chunks, nil
}

// needsReindex checks if a file needs to be reindexed
func (idx *Indexer) needsReindex(filePath string) (*models.Document, bool) {
	idx.cacheMu.RLock()
	cached, exists := idx.docCache[filePath]
	idx.cacheMu.RUnlock()

	if !exists {
		// Check disk cache
		cached = idx.loadCachedDocument(filePath)
		if cached == nil {
			return nil, true
		}
	}

	// Check if file has been modified
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, true
	}

	// Calculate current hash
	currentHash, err := utils.FileHash(filePath)
	if err != nil {
		return nil, true
	}

	// Compare hashes
	if currentHash != cached.Hash {
		return cached, true
	}

	// Check modification time as backup
	if info.ModTime().After(cached.FileModTime) {
		return cached, true
	}

	return cached, false
}

// cacheDocument saves document and chunks to disk
func (idx *Indexer) cacheDocument(doc *models.Document, chunks []*models.Chunk) error {
	cacheDir := idx.cfg.Context.Embeddings.CacheDir
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// Save document metadata
	docPath := filepath.Join(cacheDir, doc.ID+".doc.gob")
	docFile, err := os.Create(docPath)
	if err != nil {
		return err
	}
	defer docFile.Close()

	enc := gob.NewEncoder(docFile)
	if err := enc.Encode(doc); err != nil {
		return err
	}

	// Save chunks with embeddings
	chunksPath := filepath.Join(cacheDir, doc.ID+".chunks.gob")
	chunksFile, err := os.Create(chunksPath)
	if err != nil {
		return err
	}
	defer chunksFile.Close()

	enc = gob.NewEncoder(chunksFile)
	if err := enc.Encode(chunks); err != nil {
		return err
	}

	return nil
}

// loadCachedDocument loads document metadata from disk
func (idx *Indexer) loadCachedDocument(filePath string) *models.Document {
	hash, err := utils.FileHash(filePath)
	if err != nil {
		return nil
	}

	cacheDir := idx.cfg.Context.Embeddings.CacheDir
	docPath := filepath.Join(cacheDir, hash+".doc.gob")

	file, err := os.Open(docPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var doc models.Document
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&doc); err != nil {
		return nil
	}

	return &doc
}

// loadCachedChunks loads chunks from disk
func (idx *Indexer) loadCachedChunks(docID string) ([]*models.Chunk, error) {
	cacheDir := idx.cfg.Context.Embeddings.CacheDir
	chunksPath := filepath.Join(cacheDir, docID+".chunks.gob")

	file, err := os.Open(chunksPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks []*models.Chunk
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&chunks); err != nil {
		return nil, err
	}

	return chunks, nil
}

// IndexDirectory indexes all supported files in a directory
func (idx *Indexer) IndexDirectory(dirPath string) (map[string][]*models.Chunk, error) {
	logger := utils.GetLogger()
	results := make(map[string][]*models.Chunk)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		supported := false
		for _, supportedExt := range idx.cfg.Context.SupportedExtensions {
			if ext == supportedExt {
				supported = true
				break
			}
		}

		if !supported {
			return nil
		}

		chunks, err := idx.IndexFile(path, false)
		if err != nil {
			logger.Errorf("Failed to index %s: %v", path, err)
			return nil // Continue with other files
		}

		results[path] = chunks
		return nil
	})

	return results, err
}
