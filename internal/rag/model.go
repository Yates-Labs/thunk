package rag

import (
	"context"
	"time"
)

// EpisodeSummary aggregates metrics and narrative for a cluster episode.
type EpisodeSummary struct {
	EpisodeID   string    `json:"episode_id"`
	Title       string    `json:"title,omitempty"`
	Summary     string    `json:"summary"`
	StartDate   time.Time `json:"start_date,omitempty"`
	EndDate     time.Time `json:"end_date,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
	CommitCount int       `json:"commit_count"`
	FileCount   int       `json:"file_count"`
}

// VectorStore defines the interface for vector storage and similarity search
// Implementations should support episode embeddings for RAG pipelines
type VectorStore interface {
	// Insert efficiently inserts multiple episodes in a single operation
	Insert(ctx context.Context, episodes []EpisodeRecord) error

	// Flush ensures all pending data is persisted
	Flush(ctx context.Context) error

	// Search performs top-K similarity search with optional filtering
	Search(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error)

	// Query checks which episode IDs exist in the store
	// Returns a map where keys are episode IDs and values indicate existence
	Query(ctx context.Context, episodeIDs []string) (map[string]bool, error)

	// Delete removes records by episode IDs
	Delete(ctx context.Context, episodeIDs []string) error

	// GetStats returns collection statistics (record count, index status, etc.)
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// Close releases resources and closes connections
	Close() error
}

// IndexOptions provides configuration for episode indexing
type IndexOptions struct {
	// BatchSize determines how many episodes to embed at once
	BatchSize int

	// ForceReindex will delete and re-insert episodes even if they exist
	ForceReindex bool

	// SkipExisting will check if episode already exists and skip if present
	SkipExisting bool
}
