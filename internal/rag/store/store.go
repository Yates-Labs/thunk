package store

import (
	"context"
	"time"

	"github.com/Yates-Labs/thunk/internal/rag"
)

// ContextChunk represents a retrieved context with similarity score
// Used for RAG to provide relevant episode context to LLMs
type ContextChunk struct {
	EpisodeID   string                 `json:"episode_id"`
	Text        string                 `json:"text"`
	Score       float32                `json:"score"` // Similarity score (cosine distance)
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	Authors     []string               `json:"authors"`
	CommitCount int                    `json:"commit_count"`
	FileCount   int                    `json:"file_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SearchOptions provides filtering options for vector search
type SearchOptions struct {
	EpisodeIDs []string `json:"episode_ids,omitempty"` // Filter by specific episode IDs
}

// VectorStore defines the interface for vector storage and similarity search
// Implementations should support episode embeddings for RAG pipelines
type VectorStore interface {
	// Insert adds embedding records to the vector store
	Insert(ctx context.Context, records []rag.EmbeddingRecord, metadata map[string]interface{}) error

	// Search performs top-K similarity search with optional filtering
	Search(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error)

	// Delete removes records by episode IDs
	Delete(ctx context.Context, episodeIDs []string) error

	// GetStats returns collection statistics (record count, index status, etc.)
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// Close releases resources and closes connections
	Close() error
}
