package rag

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMilvusStore_EmptyRecords tests that empty records are handled gracefully (no-op)
func TestMilvusStore_EmptyRecords(t *testing.T) {
	ctx := context.Background()
	config := DefaultMilvusConfig()

	// Note: This will fail to connect if Milvus isn't running
	// For unit testing without Milvus, we'd need to mock the client
	store := &MilvusStore{
		config: config,
	}

	// Empty records should be a no-op
	err := store.Insert(ctx, []EpisodeRecord{})
	if err != nil {
		t.Errorf("Expected nil for empty records, got: %v", err)
	}
}

// TestDefaultMilvusConfig tests default configuration
func TestDefaultMilvusConfig(t *testing.T) {
	config := DefaultMilvusConfig()

	if config.Address == "" {
		t.Error("Expected non-empty address")
	}

	if config.CollectionName == "" {
		t.Error("Expected non-empty collection name")
	}

	if config.Dimension != 3072 {
		t.Errorf("Expected dimension 3072, got %d", config.Dimension)
	}

	if config.IndexType != "HNSW" {
		t.Errorf("Expected index type HNSW, got %s", config.IndexType)
	}

	if config.MetricType != "COSINE" {
		t.Errorf("Expected metric type COSINE, got %s", config.MetricType)
	}
}

// Integration test: Insert, Search, Delete full workflow
func TestMilvusStore_Integration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	config := DefaultMilvusConfig()
	config.Dimension = 1536
	config.CollectionName = "thunk_test_integration"

	store, err := NewMilvusStore(ctx, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Clean up any existing data
	_ = store.Delete(ctx, []string{"episode-001", "episode-002"})

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Test 1: Insert first episode
	texts1 := []string{
		"This is a commit about implementing authentication",
		"Added user login functionality with JWT tokens",
	}
	records1, err := embedder.Embed(ctx, texts1)
	if err != nil {
		t.Fatalf("failed to embed texts1: %v", err)
	}

	episode1 := EpisodeRecord{
		EpisodeID:   "episode-001",
		Text:        "This is a commit about implementing authentication\nAdded user login functionality with JWT tokens",
		Embedding:   records1[0].Embedding,
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Authors:     []string{"alice@example.com", "bob@example.com"},
		CommitCount: 5,
		FileCount:   3,
	}

	err = store.Insert(ctx, []EpisodeRecord{episode1})
	if err != nil {
		t.Fatalf("failed to insert episode-001: %v", err)
	}
	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to flush episode-001: %v", err)
	}
	t.Log("✓ Inserted episode-001")

	// Test 2: Insert second episode
	texts2 := []string{
		"Fixed database connection pooling issues",
		"Optimized query performance with indexes",
	}
	records2, err := embedder.Embed(ctx, texts2)
	if err != nil {
		t.Fatalf("failed to embed texts2: %v", err)
	}

	episode2 := EpisodeRecord{
		EpisodeID:   "episode-002",
		Text:        "Fixed database connection pooling issues\nOptimized query performance with indexes",
		Embedding:   records2[0].Embedding,
		StartDate:   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
		Authors:     []string{"charlie@example.com"},
		CommitCount: 3,
		FileCount:   2,
	}

	err = store.Insert(ctx, []EpisodeRecord{episode2})
	if err != nil {
		t.Fatalf("failed to insert episode-002: %v", err)
	}
	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to flush episode-002: %v", err)
	}
	t.Log("✓ Inserted episode-002")

	// Test 3: Search for authentication-related content
	queryTexts := []string{"user authentication login"}
	queryRecords, err := embedder.Embed(ctx, queryTexts)
	if err != nil {
		t.Fatalf("failed to embed query: %v", err)
	}

	results, err := store.Search(ctx, queryRecords[0].Embedding, 5, nil)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected search results, got none")
	}
	t.Logf("✓ Found %d results for authentication query", len(results))

	// Verify first result is from episode-001 (authentication related)
	if results[0].EpisodeID != "episode-001" {
		t.Errorf("expected first result from episode-001, got %s", results[0].EpisodeID)
	}

	// Test 4: Search with episode filter
	filterOpts := &SearchOptions{
		EpisodeIDs: []string{"episode-002"},
	}
	filteredResults, err := store.Search(ctx, queryRecords[0].Embedding, 5, filterOpts)
	if err != nil {
		t.Fatalf("failed to search with filter: %v", err)
	}

	for _, result := range filteredResults {
		if result.EpisodeID != "episode-002" {
			t.Errorf("filtered search returned wrong episode: %s", result.EpisodeID)
		}
	}
	t.Logf("✓ Filtered search returned %d results from episode-002", len(filteredResults))

	// Test 5: Verify result fields
	if len(results) > 0 {
		chunk := results[0]
		if chunk.Text == "" {
			t.Error("result text is empty")
		}
		if chunk.Score == 0 {
			t.Error("result score is zero")
		}
		if chunk.CommitCount == 0 {
			t.Error("commit count is zero")
		}
		if chunk.FileCount == 0 {
			t.Error("file count is zero")
		}
		if len(chunk.Authors) == 0 {
			t.Error("authors list is empty")
		}
		t.Log("✓ Result fields validated")
	}

	// Test 6: Get stats
	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	t.Logf("✓ Collection stats: %v", stats)

	// Test 7: Delete one episode
	err = store.Delete(ctx, []string{"episode-001"})
	if err != nil {
		t.Fatalf("failed to delete episode-001: %v", err)
	}
	t.Log("✓ Deleted episode-001")

	// Wait a moment for deletion to propagate
	time.Sleep(1 * time.Second)

	// Verify deletion - search should now primarily return episode-002
	resultsAfterDelete, err := store.Search(ctx, queryRecords[0].Embedding, 10, nil)
	if err != nil {
		t.Fatalf("failed to search after delete: %v", err)
	}

	for _, result := range resultsAfterDelete {
		if result.EpisodeID == "episode-001" {
			t.Error("deleted episode still appears in search results")
		}
	}
	t.Log("✓ Verified episode-001 deleted")

	// Test 8: Clean up remaining data
	err = store.Delete(ctx, []string{"episode-002"})
	if err != nil {
		t.Fatalf("failed to delete episode-002: %v", err)
	}
	t.Log("✓ Cleaned up all test data")
}

// Integration test: Search similarity rankings
func TestMilvusStore_Integration_SearchSimilarity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	config := DefaultMilvusConfig()
	config.Dimension = 1536
	config.CollectionName = "thunk_test_similarity"

	store, err := NewMilvusStore(ctx, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Clean up
	_ = store.Delete(ctx, []string{"sim-001"})

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Insert documents with varying topics
	texts := []string{
		"Python machine learning with scikit-learn and tensorflow",
		"JavaScript React frontend development and hooks",
		"Deep learning neural networks with PyTorch",
		"Golang concurrent programming with goroutines",
		"Python data analysis with pandas and numpy",
	}

	records, err := embedder.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("failed to embed texts: %v", err)
	}

	episodeRecords := make([]EpisodeRecord, len(records))
	for i, record := range records {
		episodeRecords[i] = EpisodeRecord{
			EpisodeID:   "sim-001",
			Text:        texts[i],
			Embedding:   record.Embedding,
			StartDate:   time.Now(),
			EndDate:     time.Now(),
			Authors:     []string{"test@example.com"},
			CommitCount: 10,
			FileCount:   5,
		}
	}

	err = store.Insert(ctx, episodeRecords)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}
	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	t.Log("✓ Inserted test documents")

	// Search for Python-related content
	queryTexts := []string{"Python programming and data science"}
	queryRecords, err := embedder.Embed(ctx, queryTexts)
	if err != nil {
		t.Fatalf("failed to embed query: %v", err)
	}

	results, err := store.Search(ctx, queryRecords[0].Embedding, 3, nil)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) < 3 {
		t.Fatalf("expected at least 3 results, got %d", len(results))
	}

	// Top results should be Python-related (scores should be descending)
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Errorf("scores not in descending order: %.4f < %.4f", results[i].Score, results[i+1].Score)
		}
	}

	t.Logf("✓ Search results ranked by similarity:")
	for i, result := range results {
		t.Logf("  %d. [%.4f] %s", i+1, result.Score, result.Text)
	}

	// Clean up
	_ = store.Delete(ctx, []string{"sim-001"})
}

// Integration test: Multiple episodes and batch operations
func TestMilvusStore_Integration_BatchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	config := DefaultMilvusConfig()
	config.Dimension = 1536
	config.CollectionName = "thunk_test_batch"

	store, err := NewMilvusStore(ctx, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	episodeIDs := []string{"batch-001", "batch-002"}

	// Clean up any existing data and wait for it to propagate
	_ = store.Delete(ctx, episodeIDs)

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Batch embed all texts at once to reduce API calls
	allTexts := []string{
		"Episode 1 first commit message",
		"Episode 1 second commit message",
		"Episode 2 first commit message",
		"Episode 2 second commit message",
		"commit message", // query text
	}

	allRecords, err := embedder.Embed(ctx, allTexts)
	if err != nil {
		t.Fatalf("failed to embed texts: %v", err)
	}

	// Insert multiple episodes using batch insert for better performance
	episodeRecords := make([]EpisodeRecord, len(episodeIDs))
	for i, episodeID := range episodeIDs {
		episodeRecords[i] = EpisodeRecord{
			EpisodeID:   episodeID,
			Text:        allTexts[i],
			Embedding:   allRecords[i].Embedding,
			StartDate:   time.Now().Add(time.Duration(i) * 24 * time.Hour),
			EndDate:     time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
			Authors:     []string{fmt.Sprintf("author%d@example.com", i+1)},
			CommitCount: (i + 1) * 5,
			FileCount:   (i + 1) * 2,
		}
	}

	err = store.Insert(ctx, episodeRecords)
	if err != nil {
		t.Fatalf("failed to batch insert episodes: %v", err)
	}

	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	t.Logf("✓ Inserted %d episodes", len(episodeIDs))

	// Use pre-embedded query vector
	queryVector := allRecords[4].Embedding

	// Search with filter to only get our test episodes
	filterOpts := &SearchOptions{
		EpisodeIDs: episodeIDs,
	}
	allResults, err := store.Search(ctx, queryVector, 10, filterOpts)
	if err != nil {
		t.Fatalf("failed to search all: %v", err)
	}

	if len(allResults) != 2 {
		t.Errorf("expected 2 results (1 per episode), got %d", len(allResults))
	}
	t.Logf("✓ Search returned %d results across all episodes", len(allResults))

	// Batch delete
	err = store.Delete(ctx, episodeIDs)
	if err != nil {
		t.Fatalf("failed to batch delete: %v", err)
	}
	t.Log("✓ Batch deleted all episodes")

	// Verify all deleted
	time.Sleep(1 * time.Second)
	resultsAfterDelete, err := store.Search(ctx, queryVector, 10, nil)
	if err != nil {
		t.Fatalf("failed to search after delete: %v", err)
	}

	for _, result := range resultsAfterDelete {
		for _, episodeID := range episodeIDs {
			if result.EpisodeID == episodeID {
				t.Errorf("deleted episode %s still in results", episodeID)
			}
		}
	}
	t.Log("✓ Verified all episodes deleted")
}

// Integration test: Large text handling
func TestMilvusStore_Integration_LargeText(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	config := DefaultMilvusConfig()
	config.Dimension = 1536
	config.CollectionName = "thunk_test_large"

	store, err := NewMilvusStore(ctx, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_ = store.Delete(ctx, []string{"large-001"})

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Create a large text chunk (simulate a large commit message or diff)
	largeText := "This is a comprehensive commit that includes multiple changes: "
	for i := 0; i < 100; i++ {
		largeText += fmt.Sprintf("Added feature %d, fixed bug %d, updated documentation %d. ", i, i, i)
	}

	texts := []string{largeText}
	records, err := embedder.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("failed to embed large text: %v", err)
	}

	episode := EpisodeRecord{
		EpisodeID:   "large-001",
		Text:        largeText,
		Embedding:   records[0].Embedding,
		StartDate:   time.Now(),
		EndDate:     time.Now(),
		Authors:     []string{"author@example.com"},
		CommitCount: 1,
		FileCount:   50,
	}

	err = store.Insert(ctx, []EpisodeRecord{episode})
	if err != nil {
		t.Fatalf("failed to insert large text: %v", err)
	}
	err = store.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	t.Logf("✓ Inserted large text (%d characters)", len(largeText))

	// Search and verify
	queryTexts := []string{"feature and bug fix"}
	queryRecords, err := embedder.Embed(ctx, queryTexts)
	if err != nil {
		t.Fatalf("failed to embed query: %v", err)
	}

	results, err := store.Search(ctx, queryRecords[0].Embedding, 1, nil)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results for large text")
	}

	if len(results[0].Text) < 1000 {
		t.Errorf("retrieved text seems truncated: %d chars", len(results[0].Text))
	}
	t.Logf("✓ Retrieved large text (%d characters)", len(results[0].Text))

	// Clean up
	_ = store.Delete(ctx, []string{"large-001"})
}
