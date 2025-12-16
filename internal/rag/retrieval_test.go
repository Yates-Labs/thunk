package rag

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockEmbedder implements Embedder interface for testing
type mockEmbedder struct {
	embedFunc func(ctx context.Context, texts []string) ([]EmbeddingRecord, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([]EmbeddingRecord, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, texts)
	}
	// Default: return simple embeddings
	records := make([]EmbeddingRecord, len(texts))
	for i, text := range texts {
		// Create a simple embedding based on text length
		embedding := make([]float32, 3)
		embedding[0] = float32(len(text))
		embedding[1] = float32(i)
		embedding[2] = 1.0
		records[i] = EmbeddingRecord{
			Text:      text,
			Embedding: embedding,
			Index:     i,
			Model:     "mock",
		}
	}
	return records, nil
}

// mockVectorStore implements VectorStore interface for testing
type mockVectorStore struct {
	episodes     map[string]EpisodeRecord
	searchFunc   func(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error)
	queryFunc    func(ctx context.Context, episodeIDs []string) (map[string]bool, error)
	insertFunc   func(ctx context.Context, episodes []EpisodeRecord) error
	flushFunc    func(ctx context.Context) error
	deleteFunc   func(ctx context.Context, episodeIDs []string) error
	getStatsFunc func(ctx context.Context) (map[string]interface{}, error)
	closeFunc    func() error
}

func (m *mockVectorStore) Insert(ctx context.Context, episodes []EpisodeRecord) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, episodes)
	}
	if m.episodes == nil {
		m.episodes = make(map[string]EpisodeRecord)
	}
	for _, ep := range episodes {
		m.episodes[ep.EpisodeID] = ep
	}
	return nil
}

func (m *mockVectorStore) Flush(ctx context.Context) error {
	if m.flushFunc != nil {
		return m.flushFunc(ctx)
	}
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, queryVector, topK, opts)
	}

	// Default: return all episodes if no filter, otherwise filter by episode ID
	chunks := []ContextChunk{}

	if opts != nil && len(opts.EpisodeIDs) > 0 {
		// Filter by episode IDs
		for _, id := range opts.EpisodeIDs {
			if ep, exists := m.episodes[id]; exists {
				chunks = append(chunks, ContextChunk{
					EpisodeID:   ep.EpisodeID,
					Text:        ep.Text,
					Score:       1.0,
					StartDate:   ep.StartDate,
					EndDate:     ep.EndDate,
					Authors:     ep.Authors,
					CommitCount: ep.CommitCount,
					FileCount:   ep.FileCount,
				})
			}
		}
	} else {
		// Return all episodes
		for _, ep := range m.episodes {
			chunks = append(chunks, ContextChunk{
				EpisodeID:   ep.EpisodeID,
				Text:        ep.Text,
				Score:       0.9,
				StartDate:   ep.StartDate,
				EndDate:     ep.EndDate,
				Authors:     ep.Authors,
				CommitCount: ep.CommitCount,
				FileCount:   ep.FileCount,
			})
			if len(chunks) >= topK {
				break
			}
		}
	}

	return chunks, nil
}

func (m *mockVectorStore) Query(ctx context.Context, episodeIDs []string) (map[string]bool, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, episodeIDs)
	}
	result := make(map[string]bool)
	for _, id := range episodeIDs {
		_, exists := m.episodes[id]
		result[id] = exists
	}
	return result, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, episodeIDs []string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, episodeIDs)
	}
	for _, id := range episodeIDs {
		delete(m.episodes, id)
	}
	return nil
}

func (m *mockVectorStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx)
	}
	return map[string]interface{}{"row_count": len(m.episodes)}, nil
}

func (m *mockVectorStore) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewRetriever(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{episodes: make(map[string]EpisodeRecord)}

	t.Run("Valid parameters", func(t *testing.T) {
		retriever, err := NewRetriever(embedder, store)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if retriever == nil {
			t.Fatal("Expected retriever to be non-nil")
		}
	})

	t.Run("Nil embedder", func(t *testing.T) {
		_, err := NewRetriever(nil, store)
		if err == nil {
			t.Fatal("Expected error for nil embedder")
		}
	})

	t.Run("Nil vector store", func(t *testing.T) {
		_, err := NewRetriever(embedder, nil)
		if err == nil {
			t.Fatal("Expected error for nil vector store")
		}
	})
}

func TestRetrieveContextForQuery(t *testing.T) {
	ctx := context.Background()

	// Setup mock data
	episodes := map[string]EpisodeRecord{
		"ep1": {
			EpisodeID:   "ep1",
			Text:        "Implemented authentication system",
			Embedding:   []float32{1.0, 0.0, 0.0},
			StartDate:   time.Now().Add(-48 * time.Hour),
			EndDate:     time.Now().Add(-24 * time.Hour),
			Authors:     []string{"alice"},
			CommitCount: 5,
			FileCount:   3,
		},
		"ep2": {
			EpisodeID:   "ep2",
			Text:        "Added database migrations",
			Embedding:   []float32{0.0, 1.0, 0.0},
			StartDate:   time.Now().Add(-24 * time.Hour),
			EndDate:     time.Now(),
			Authors:     []string{"bob"},
			CommitCount: 3,
			FileCount:   2,
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{episodes: episodes}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	t.Run("Successful query", func(t *testing.T) {
		chunks, err := retriever.RetrieveContextForQuery(ctx, "authentication", 2, nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(chunks) == 0 {
			t.Fatal("Expected non-empty results")
		}
	})

	t.Run("Empty query", func(t *testing.T) {
		_, err := retriever.RetrieveContextForQuery(ctx, "", 2, nil)
		if err == nil {
			t.Fatal("Expected error for empty query")
		}
	})

	t.Run("Invalid topK", func(t *testing.T) {
		_, err := retriever.RetrieveContextForQuery(ctx, "test", 0, nil)
		if err == nil {
			t.Fatal("Expected error for topK <= 0")
		}
	})

	t.Run("With filters", func(t *testing.T) {
		opts := &SearchOptions{
			EpisodeIDs: []string{"ep1"},
		}
		chunks, err := retriever.RetrieveContextForQuery(ctx, "test", 1, opts)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].EpisodeID != "ep1" {
			t.Errorf("Expected episode ep1, got %s", chunks[0].EpisodeID)
		}
	})
}

func TestRetrieveContextForEpisode(t *testing.T) {
	ctx := context.Background()

	// Setup mock data
	episodes := map[string]EpisodeRecord{
		"ep1": {
			EpisodeID:   "ep1",
			Text:        "Implemented authentication system",
			Embedding:   []float32{1.0, 0.0, 0.0},
			StartDate:   time.Now().Add(-48 * time.Hour),
			EndDate:     time.Now().Add(-24 * time.Hour),
			Authors:     []string{"alice"},
			CommitCount: 5,
			FileCount:   3,
		},
		"ep2": {
			EpisodeID:   "ep2",
			Text:        "Added database migrations",
			Embedding:   []float32{0.0, 1.0, 0.0},
			StartDate:   time.Now().Add(-24 * time.Hour),
			EndDate:     time.Now(),
			Authors:     []string{"bob"},
			CommitCount: 3,
			FileCount:   2,
		},
		"ep3": {
			EpisodeID:   "ep3",
			Text:        "Updated authentication tests",
			Embedding:   []float32{0.9, 0.1, 0.0},
			StartDate:   time.Now().Add(-12 * time.Hour),
			EndDate:     time.Now(),
			Authors:     []string{"alice"},
			CommitCount: 2,
			FileCount:   1,
		},
	}

	embedder := &mockEmbedder{}

	// Custom search function that returns similar episodes
	store := &mockVectorStore{
		episodes: episodes,
		searchFunc: func(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error) {
			// If filtering by episode ID, return that episode
			if opts != nil && len(opts.EpisodeIDs) > 0 {
				chunks := []ContextChunk{}
				for _, id := range opts.EpisodeIDs {
					if ep, exists := episodes[id]; exists {
						chunks = append(chunks, ContextChunk{
							EpisodeID:   ep.EpisodeID,
							Text:        ep.Text,
							Score:       1.0,
							StartDate:   ep.StartDate,
							EndDate:     ep.EndDate,
							Authors:     ep.Authors,
							CommitCount: ep.CommitCount,
							FileCount:   ep.FileCount,
						})
					}
				}
				return chunks, nil
			}

			// Return similar episodes (mock similarity)
			chunks := []ContextChunk{
				{
					EpisodeID:   "ep1",
					Text:        episodes["ep1"].Text,
					Score:       1.0,
					StartDate:   episodes["ep1"].StartDate,
					EndDate:     episodes["ep1"].EndDate,
					Authors:     episodes["ep1"].Authors,
					CommitCount: episodes["ep1"].CommitCount,
					FileCount:   episodes["ep1"].FileCount,
				},
				{
					EpisodeID:   "ep3",
					Text:        episodes["ep3"].Text,
					Score:       0.95,
					StartDate:   episodes["ep3"].StartDate,
					EndDate:     episodes["ep3"].EndDate,
					Authors:     episodes["ep3"].Authors,
					CommitCount: episodes["ep3"].CommitCount,
					FileCount:   episodes["ep3"].FileCount,
				},
			}

			if len(chunks) > topK {
				chunks = chunks[:topK]
			}
			return chunks, nil
		},
	}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	t.Run("Successful episode retrieval", func(t *testing.T) {
		chunks, err := retriever.RetrieveContextForEpisode(ctx, "ep1", 2, nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should return similar episodes but not ep1 itself
		for _, chunk := range chunks {
			if chunk.EpisodeID == "ep1" {
				t.Error("Original episode should be filtered out from results")
			}
		}
	})

	t.Run("Empty episode ID", func(t *testing.T) {
		_, err := retriever.RetrieveContextForEpisode(ctx, "", 2, nil)
		if err == nil {
			t.Fatal("Expected error for empty episode ID")
		}
	})

	t.Run("Invalid topK", func(t *testing.T) {
		_, err := retriever.RetrieveContextForEpisode(ctx, "ep1", -1, nil)
		if err == nil {
			t.Fatal("Expected error for negative topK")
		}
	})

	t.Run("Non-existent episode", func(t *testing.T) {
		_, err := retriever.RetrieveContextForEpisode(ctx, "non-existent", 2, nil)
		if err == nil {
			t.Fatal("Expected error for non-existent episode")
		}
	})
}

func TestRetrieveContextForQueryWithFilters(t *testing.T) {
	ctx := context.Background()

	episodes := map[string]EpisodeRecord{
		"ep1": {
			EpisodeID:   "ep1",
			Text:        "Feature A",
			Embedding:   []float32{1.0, 0.0, 0.0},
			StartDate:   time.Now(),
			EndDate:     time.Now(),
			Authors:     []string{"alice"},
			CommitCount: 1,
			FileCount:   1,
		},
		"ep2": {
			EpisodeID:   "ep2",
			Text:        "Feature B",
			Embedding:   []float32{0.0, 1.0, 0.0},
			StartDate:   time.Now(),
			EndDate:     time.Now(),
			Authors:     []string{"bob"},
			CommitCount: 1,
			FileCount:   1,
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{episodes: episodes}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	t.Run("With episode IDs filter", func(t *testing.T) {
		chunks, err := retriever.RetrieveContextForQueryWithFilters(
			ctx, "test", 5, []string{"ep1"}, "",
		)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].EpisodeID != "ep1" {
			t.Errorf("Expected ep1, got %s", chunks[0].EpisodeID)
		}
	})

	t.Run("With repository filter", func(t *testing.T) {
		chunks, err := retriever.RetrieveContextForQueryWithFilters(
			ctx, "test", 5, nil, "myrepo",
		)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		// Mock doesn't filter by repo, but should not error
		if chunks == nil {
			t.Fatal("Expected non-nil chunks")
		}
	})
}

func TestRetrieveMultipleEpisodes(t *testing.T) {
	ctx := context.Background()

	episodes := map[string]EpisodeRecord{
		"ep1": {
			EpisodeID: "ep1",
			Text:      "Episode 1",
			Embedding: []float32{1.0, 0.0, 0.0},
		},
		"ep2": {
			EpisodeID: "ep2",
			Text:      "Episode 2",
			Embedding: []float32{0.0, 1.0, 0.0},
		},
		"ep3": {
			EpisodeID: "ep3",
			Text:      "Episode 3",
			Embedding: []float32{0.0, 0.0, 1.0},
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		episodes: episodes,
		searchFunc: func(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error) {
			// Return the requested episode for filtering
			if opts != nil && len(opts.EpisodeIDs) > 0 {
				chunks := []ContextChunk{}
				for _, id := range opts.EpisodeIDs {
					if ep, exists := episodes[id]; exists {
						chunks = append(chunks, ContextChunk{
							EpisodeID: ep.EpisodeID,
							Text:      ep.Text,
							Score:     1.0,
						})
					}
				}
				return chunks, nil
			}

			// Return other episodes as similar ones
			chunks := []ContextChunk{}
			for id, ep := range episodes {
				// Skip the queried episode (simulated)
				if opts == nil || !contains(opts.EpisodeIDs, id) {
					chunks = append(chunks, ContextChunk{
						EpisodeID: ep.EpisodeID,
						Text:      ep.Text,
						Score:     0.9,
					})
					if len(chunks) >= topK {
						break
					}
				}
			}
			return chunks, nil
		},
	}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	t.Run("Multiple episodes", func(t *testing.T) {
		results, err := retriever.RetrieveMultipleEpisodes(
			ctx, []string{"ep1", "ep2"}, 2, nil,
		)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		if _, exists := results["ep1"]; !exists {
			t.Error("Expected ep1 in results")
		}
		if _, exists := results["ep2"]; !exists {
			t.Error("Expected ep2 in results")
		}
	})

	t.Run("Empty episode list", func(t *testing.T) {
		results, err := retriever.RetrieveMultipleEpisodes(ctx, []string{}, 2, nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("Expected empty results, got %d", len(results))
		}
	})

	t.Run("Non-existent episode continues", func(t *testing.T) {
		results, err := retriever.RetrieveMultipleEpisodes(
			ctx, []string{"ep1", "non-existent", "ep2"}, 2, nil,
		)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		// Should have results for ep1 and ep2, empty for non-existent
		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		if len(results["non-existent"]) != 0 {
			t.Error("Expected empty results for non-existent episode")
		}
	})
}

func TestEmbeddingError(t *testing.T) {
	ctx := context.Background()

	embedder := &mockEmbedder{
		embedFunc: func(ctx context.Context, texts []string) ([]EmbeddingRecord, error) {
			return nil, fmt.Errorf("embedding service unavailable")
		},
	}

	store := &mockVectorStore{episodes: make(map[string]EpisodeRecord)}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	_, err = retriever.RetrieveContextForQuery(ctx, "test query", 5, nil)
	if err == nil {
		t.Fatal("Expected error from embedding service")
	}
}

func TestSearchError(t *testing.T) {
	ctx := context.Background()

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		episodes: make(map[string]EpisodeRecord),
		searchFunc: func(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error) {
			return nil, fmt.Errorf("search service unavailable")
		},
	}

	retriever, err := NewRetriever(embedder, store)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	_, err = retriever.RetrieveContextForQuery(ctx, "test query", 5, nil)
	if err == nil {
		t.Fatal("Expected error from search service")
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
