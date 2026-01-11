package context

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	openai "github.com/sashabaranov/go-openai"
	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
)

type Embedder struct {
	client *openai.Client
	cfg    *config.Config
}

func NewEmbedder(cfg *config.Config) *Embedder {
	client := openai.NewClient(cfg.LLM.APIKey)
	return &Embedder{
		client: client,
		cfg:    cfg,
	}
}

// CreateEmbeddings generates embeddings for chunks
func (e *Embedder) CreateEmbeddings(chunks []*models.Chunk) error {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	// Process in batches
	batchSize := e.cfg.Context.Embeddings.BatchSize
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := e.getEmbeddings(batch)
		if err != nil {
			return fmt.Errorf("failed to get embeddings: %w", err)
		}

		for j, emb := range embeddings {
			chunks[i+j].Embedding = emb
		}
	}

	return nil
}

// CreateEmbedding creates a single embedding
func (e *Embedder) CreateEmbedding(text string) ([]float32, error) {
	embeddings, err := e.getEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

func (e *Embedder) getEmbeddings(texts []string) ([][]float32, error) {
	resp, err := e.client.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequestStrings{
			Input: texts,
			Model: openai.EmbeddingModel(e.cfg.Context.Embeddings.Model),
		},
	)
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// ChunkDocument splits a document into chunks
func (e *Embedder) ChunkDocument(doc *models.Document) []*models.Chunk {
	cfg := e.cfg.Context.Chunking
	text := doc.Content

	var chunks []*models.Chunk

	switch cfg.Method {
	case "fixed":
		chunks = e.chunkFixed(doc, text, cfg.ChunkSize, cfg.ChunkOverlap)
	case "recursive":
		chunks = e.chunkRecursive(doc, text, cfg.ChunkSize, cfg.ChunkOverlap)
	default:
		chunks = e.chunkFixed(doc, text, cfg.ChunkSize, cfg.ChunkOverlap)
	}

	// Filter out chunks that are too small
	filtered := make([]*models.Chunk, 0)
	for _, chunk := range chunks {
		if len(strings.TrimSpace(chunk.Content)) >= cfg.MinChunkSize {
			filtered = append(filtered, chunk)
		}
	}

	return filtered
}

func (e *Embedder) chunkFixed(doc *models.Document, text string, size, overlap int) []*models.Chunk {
	var chunks []*models.Chunk
	runes := []rune(text)

	for i := 0; i < len(runes); i += (size - overlap) {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}

		content := string(runes[i:end])
		chunk := &models.Chunk{
			DocumentID: doc.ID,
			Content:    strings.TrimSpace(content),
			Index:      len(chunks),
			Metadata: map[string]interface{}{
				"source":    doc.FilePath,
				"doc_id":    doc.ID,
				"chunk_idx": len(chunks),
			},
		}
		chunks = append(chunks, chunk)

		if end >= len(runes) {
			break
		}
	}

	return chunks
}

func (e *Embedder) chunkRecursive(doc *models.Document, text string, size, overlap int) []*models.Chunk {
	// Split by paragraphs first
	paragraphs := strings.Split(text, "\n\n")

	var chunks []*models.Chunk
	var currentChunk strings.Builder
	chunkIdx := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If paragraph itself is too large, split it
		if len(para) > size {
			sentences := splitSentences(para)
			for _, sent := range sentences {
				if currentChunk.Len()+len(sent) > size {
					if currentChunk.Len() > 0 {
						chunk := &models.Chunk{
							DocumentID: doc.ID,
							Content:    strings.TrimSpace(currentChunk.String()),
							Index:      chunkIdx,
							Metadata: map[string]interface{}{
								"source":    doc.FilePath,
								"doc_id":    doc.ID,
								"chunk_idx": chunkIdx,
							},
						}
						chunks = append(chunks, chunk)
						chunkIdx++

						// Keep overlap
						words := strings.Fields(currentChunk.String())
						overlapWords := overlap / 10
						if overlapWords > len(words) {
							overlapWords = len(words)
						}
						currentChunk.Reset()
						if overlapWords > 0 {
							currentChunk.WriteString(strings.Join(words[len(words)-overlapWords:], " "))
							currentChunk.WriteString(" ")
						}
					}
				}
				currentChunk.WriteString(sent)
				currentChunk.WriteString(" ")
			}
		} else {
			if currentChunk.Len()+len(para) > size {
				if currentChunk.Len() > 0 {
					chunk := &models.Chunk{
						DocumentID: doc.ID,
						Content:    strings.TrimSpace(currentChunk.String()),
						Index:      chunkIdx,
						Metadata: map[string]interface{}{
							"source":    doc.FilePath,
							"doc_id":    doc.ID,
							"chunk_idx": chunkIdx,
						},
					}
					chunks = append(chunks, chunk)
					chunkIdx++
					currentChunk.Reset()
				}
			}
			currentChunk.WriteString(para)
			currentChunk.WriteString("\n\n")
		}
	}

	// Add remaining chunk
	if currentChunk.Len() > 0 {
		chunk := &models.Chunk{
			DocumentID: doc.ID,
			Content:    strings.TrimSpace(currentChunk.String()),
			Index:      chunkIdx,
			Metadata: map[string]interface{}{
				"source":    doc.FilePath,
				"doc_id":    doc.ID,
				"chunk_idx": chunkIdx,
			},
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			if i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}
