package rag

import (
	"context"
	"fmt"
)

// Retriever provides high-level semantic retrieval for episode embeddings.
type Retriever struct {
	embedder    Embedder
	vectorStore VectorStore
}

// NewRetriever creates a new Retriever instance.
func NewRetriever(embedder Embedder, vectorStore VectorStore) (*Retriever, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	if vectorStore == nil {
		return nil, fmt.Errorf("vector store cannot be nil")
	}

	return &Retriever{
		embedder:    embedder,
		vectorStore: vectorStore,
	}, nil
}

// RetrieveContextForEpisode retrieves topK similar episodes based on a given episode ID.
func (r *Retriever) RetrieveContextForEpisode(
	ctx context.Context,
	episodeID string,
	topK int,
	opts *SearchOptions,
) ([]ContextChunk, error) {
	if episodeID == "" {
		return nil, fmt.Errorf("episode ID cannot be empty")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK must be positive, got %d", topK)
	}

	episodeFilter := &SearchOptions{
		EpisodeIDs: []string{episodeID},
	}

	// Check if episode exists
	existenceMap, err := r.vectorStore.Query(ctx, []string{episodeID})
	if err != nil {
		return nil, fmt.Errorf("failed to check episode existence: %w", err)
	}
	if !existenceMap[episodeID] {
		return nil, fmt.Errorf("episode %s not found in vector store", episodeID)
	}

	searchOpts := &SearchOptions{}
	if opts != nil {
		searchOpts.Repository = opts.Repository
		searchOpts.Metadata = opts.Metadata
	}

	// Retrieve the episode to get its text
	episodeChunks, err := r.vectorStore.Search(ctx, nil, 1, episodeFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve episode: %w", err)
	}
	if len(episodeChunks) == 0 {
		return nil, fmt.Errorf("episode %s not found", episodeID)
	}

	// Embed the episode's text to use as the query vector
	episode := episodeChunks[0]
	embeddingRecords, err := r.embedder.Embed(ctx, []string{episode.Text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed episode text: %w", err)
	}
	if len(embeddingRecords) == 0 {
		return nil, fmt.Errorf("no embedding generated for episode")
	}

	queryVector := embeddingRecords[0].Embedding

	// Search for topK+1 to account for the episode itself in results
	chunks, err := r.vectorStore.Search(ctx, queryVector, topK+1, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar episodes: %w", err)
	}

	// Filter out the original episode from results if present
	filteredChunks := make([]ContextChunk, 0, topK)
	for _, chunk := range chunks {
		if chunk.EpisodeID != episodeID {
			filteredChunks = append(filteredChunks, chunk)
			if len(filteredChunks) >= topK {
				break
			}
		}
	}

	return filteredChunks, nil
}

// RetrieveContextForQuery performs semantic search using a free-text query.
func (r *Retriever) RetrieveContextForQuery(
	ctx context.Context,
	query string,
	topK int,
	opts *SearchOptions,
) ([]ContextChunk, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK must be positive, got %d", topK)
	}

	// Generate embedding for the query
	embeddingRecords, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(embeddingRecords) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}

	queryVector := embeddingRecords[0].Embedding

	// Perform vector similarity search
	chunks, err := r.vectorStore.Search(ctx, queryVector, topK, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for query: %w", err)
	}

	return chunks, nil
}

// RetrieveContextForQueryWithFilters is a convenience function for semantic search with explicit filter parameters.
func (r *Retriever) RetrieveContextForQueryWithFilters(
	ctx context.Context,
	query string,
	topK int,
	episodeIDs []string,
	repository string,
) ([]ContextChunk, error) {
	opts := &SearchOptions{}

	if len(episodeIDs) > 0 {
		opts.EpisodeIDs = episodeIDs
	}
	if repository != "" {
		opts.Repository = repository
	}

	return r.RetrieveContextForQuery(ctx, query, topK, opts)
}

// RetrieveMultipleEpisodes retrieves context for multiple episode IDs efficiently.
func (r *Retriever) RetrieveMultipleEpisodes(
	ctx context.Context,
	episodeIDs []string,
	topK int,
	opts *SearchOptions,
) (map[string][]ContextChunk, error) {
	if len(episodeIDs) == 0 {
		return map[string][]ContextChunk{}, nil
	}

	results := make(map[string][]ContextChunk, len(episodeIDs))

	for _, episodeID := range episodeIDs {
		chunks, err := r.RetrieveContextForEpisode(ctx, episodeID, topK, opts)
		if err != nil {
			results[episodeID] = []ContextChunk{}
			continue
		}
		results[episodeID] = chunks
	}

	return results, nil
}
