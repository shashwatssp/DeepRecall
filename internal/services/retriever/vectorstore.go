package retriever

import (
	"encoding/gob"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/shashwatssp/deeprecall/internal/models"
	bolt "go.etcd.io/bbolt"
)

var (
	chunksBucket = []byte("chunks")
	metaBucket   = []byte("metadata")
)

// VectorStore provides efficient vector storage and similarity search
type VectorStore struct {
	db     *bolt.DB
	mu     sync.RWMutex
	memory map[string]*models.Chunk // In-memory cache for faster access
}

// NewVectorStore creates a new vector store
func NewVectorStore(dbPath string) (*VectorStore, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(chunksBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metaBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	vs := &VectorStore{
		db:     db,
		memory: make(map[string]*models.Chunk),
	}

	// Load chunks into memory for faster retrieval
	if err := vs.loadIntoMemory(); err != nil {
		return nil, err
	}

	return vs, nil
}

// Close closes the database
func (vs *VectorStore) Close() error {
	return vs.db.Close()
}

// AddChunks adds multiple chunks to the store
func (vs *VectorStore) AddChunks(chunks []*models.Chunk) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chunksBucket)

		for _, chunk := range chunks {
			// Generate ID if not set
			if chunk.ID == "" {
				chunk.ID = fmt.Sprintf("%s_%d", chunk.DocumentID, chunk.Index)
			}

			// Encode chunk
			var buf []byte
			enc := gob.NewEncoder(&bufferWrapper{&buf})
			if err := enc.Encode(chunk); err != nil {
				return fmt.Errorf("failed to encode chunk: %w", err)
			}

			// Store in database
			if err := bucket.Put([]byte(chunk.ID), buf); err != nil {
				return err
			}

			// Store in memory cache
			vs.memory[chunk.ID] = chunk
		}

		return nil
	})
}

// Search performs similarity search
func (vs *VectorStore) Search(queryEmbedding []float32, topK int, threshold float64) ([]*models.RetrievalResult, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	var results []*models.RetrievalResult

	// Calculate similarity scores for all chunks in memory
	for _, chunk := range vs.memory {
		if len(chunk.Embedding) == 0 {
			continue
		}

		score := cosineSimilarity(queryEmbedding, chunk.Embedding)
		if score >= threshold {
			results = append(results, &models.RetrievalResult{
				Chunk:      chunk,
				Score:      score,
				DocumentID: chunk.DocumentID,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// DeleteByDocumentID removes all chunks for a document
func (vs *VectorStore) DeleteByDocumentID(documentID string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chunksBucket)
		cursor := bucket.Cursor()

		var toDelete [][]byte
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var chunk models.Chunk
			dec := gob.NewDecoder(&bufferWrapper{&v})
			if err := dec.Decode(&chunk); err != nil {
				continue
			}

			if chunk.DocumentID == documentID {
				toDelete = append(toDelete, k)
				delete(vs.memory, string(k))
			}
		}

		for _, key := range toDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetStats returns statistics about the vector store
func (vs *VectorStore) GetStats() (int, int, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	totalChunks := len(vs.memory)

	// Count unique documents
	docs := make(map[string]bool)
	for _, chunk := range vs.memory {
		docs[chunk.DocumentID] = true
	}

	return len(docs), totalChunks, nil
}

// loadIntoMemory loads all chunks from disk into memory
func (vs *VectorStore) loadIntoMemory() error {
	return vs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chunksBucket)
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var chunk models.Chunk
			dec := gob.NewDecoder(&bufferWrapper{&v})
			if err := dec.Decode(&chunk); err != nil {
				return err
			}

			vs.memory[string(k)] = &chunk
		}

		return nil
	})
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// bufferWrapper wraps a byte slice to implement io.Writer and io.Reader
type bufferWrapper struct {
	buf *[]byte
}

func (bw *bufferWrapper) Write(p []byte) (n int, err error) {
	*bw.buf = append(*bw.buf, p...)
	return len(p), nil
}

func (bw *bufferWrapper) Read(p []byte) (n int, err error) {
	n = copy(p, *bw.buf)
	*bw.buf = (*bw.buf)[n:]
	return n, nil
}
