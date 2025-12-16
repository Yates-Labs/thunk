package rag

import (
	"context"
	"fmt"
	"time"
)

// SearchOptions provides filtering options for vector search
type SearchOptions struct {
	EpisodeIDs []string               `json:"episode_ids,omitempty"` // Filter by specific episode IDs
	Repository string                 `json:"repository,omitempty"`  // Filter by repository name
	Metadata   map[string]interface{} `json:"metadata,omitempty"`    // Additional metadata filters
}

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

// DefaultIndexOptions returns sensible defaults for indexing
func DefaultIndexOptions() IndexOptions {
	return IndexOptions{
		BatchSize:    10, // Batch size for embedding API calls
		ForceReindex: false,
		SkipExisting: true,
	}
}

// IndexEpisodes processes episode summaries and stores their embeddings in the vector store
// This function:
// 1. Converts each episode summary to text
// 2. Generates embeddings in batches
// 3. Stores embeddings with metadata in Milvus
// 4. Supports re-indexing options (skip existing, force reindex)
func IndexEpisodes(
	ctx context.Context,
	episodes []EpisodeSummary,
	embedder Embedder,
	vectorStore VectorStore,
	opts IndexOptions,
) error {
	if len(episodes) == 0 {
		return nil
	}

	if embedder == nil {
		return fmt.Errorf("embedder cannot be nil")
	}

	if vectorStore == nil {
		return fmt.Errorf("vector store cannot be nil")
	}

	// Handle re-indexing: delete existing episodes if force reindex is enabled
	if opts.ForceReindex {
		episodeIDs := make([]string, len(episodes))
		for i, ep := range episodes {
			episodeIDs[i] = ep.EpisodeID
		}

		if err := vectorStore.Delete(ctx, episodeIDs); err != nil {
			return fmt.Errorf("failed to delete existing episodes: %w", err)
		}
	}

	// Filter episodes if skip existing is enabled
	episodesToIndex := episodes
	if opts.SkipExisting && !opts.ForceReindex {
		// For simplicity, we'll process all and let duplicates be handled by deletion
		// A more sophisticated implementation could query existing episode IDs first
		episodesToIndex = filterNewEpisodes(ctx, episodes, vectorStore)
	}

	// Process episodes in batches
	for batchStart := 0; batchStart < len(episodesToIndex); batchStart += opts.BatchSize {
		batchEnd := batchStart + opts.BatchSize
		if batchEnd > len(episodesToIndex) {
			batchEnd = len(episodesToIndex)
		}

		batch := episodesToIndex[batchStart:batchEnd]

		// Convert episodes to text
		texts := make([]string, len(batch))
		for i, episode := range batch {
			texts[i] = episode.Summary
		}

		// Generate embeddings for the batch
		embeddingRecords, err := embedder.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings for batch starting at %d: %w", batchStart, err)
		}

		// Use batch insert for efficient storage
		episodeRecords := make([]EpisodeRecord, len(batch))
		for i, episode := range batch {
			episodeRecords[i] = EpisodeRecord{
				EpisodeID:   episode.EpisodeID,
				Text:        embeddingRecords[i].Text,
				Embedding:   embeddingRecords[i].Embedding,
				StartDate:   episode.StartDate,
				EndDate:     episode.EndDate,
				Authors:     episode.Authors,
				CommitCount: episode.CommitCount,
				FileCount:   episode.FileCount,
			}
		}

		if err := vectorStore.Insert(ctx, episodeRecords); err != nil {
			return fmt.Errorf("failed to insert batch starting at %d: %w", batchStart, err)
		}

		// Flush after each batch
		if err := vectorStore.Flush(ctx); err != nil {
			return fmt.Errorf("failed to flush batch starting at %d: %w", batchStart, err)
		}
	}

	return nil
}

// filterNewEpisodes removes episodes that already exist in the vector store
func filterNewEpisodes(
	ctx context.Context,
	episodes []EpisodeSummary,
	vectorStore VectorStore,
) []EpisodeSummary {
	if len(episodes) == 0 {
		return episodes
	}

	// Extract episode IDs
	episodeIDs := make([]string, len(episodes))
	for i, ep := range episodes {
		episodeIDs[i] = ep.EpisodeID
	}

	// Query which episodes exist
	existingMap, err := vectorStore.Query(ctx, episodeIDs)
	if err != nil {
		// If query fails, return all episodes to be safe
		// The caller will handle any errors during insertion
		return episodes
	}

	// Filter out existing episodes
	newEpisodes := make([]EpisodeSummary, 0, len(episodes))
	for _, ep := range episodes {
		if !existingMap[ep.EpisodeID] {
			newEpisodes = append(newEpisodes, ep)
		}
	}

	return newEpisodes
}
