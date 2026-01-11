package retriever

import (
	"fmt"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
	"github.com/shashwatssp/deeprecall/internal/services/context"
)

type Retriever struct {
	store    *VectorStore
	embedder *context.Embedder
	cfg      *config.Config
}

func NewRetriever(cfg *config.Config) (*Retriever, error) {
	store, err := NewVectorStore(cfg.Retrieval.DBPath)
	if err != nil {
		return nil, err
	}

	return &Retriever{
		store:    store,
		embedder: context.NewEmbedder(cfg),
		cfg:      cfg,
	}, nil
}

// Retrieve finds relevant chunks for a query
func (r *Retriever) Retrieve(query string) ([]*models.RetrievalResult, error) {
	// Generate query embedding
	queryEmb, err := r.embedder.CreateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Search vector store
	results, err := r.store.Search(
		queryEmb,
		r.cfg.Retrieval.TopK,
		r.cfg.Retrieval.SimilarityThreshold,
	)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// IndexChunks adds chunks to the retriever
func (r *Retriever) IndexChunks(chunks []*models.Chunk) error {
	return r.store.AddChunks(chunks)
}

// GetStats returns retriever statistics
func (r *Retriever) GetStats() (totalDocs, totalChunks int, err error) {
	return r.store.GetStats()
}

// Close closes the retriever
func (r *Retriever) Close() error {
	return r.store.Close()
}
