package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/rag"
)

// TestMilvusStore_EmptyRecords tests error handling for empty records
func TestMilvusStore_EmptyRecords(t *testing.T) {
	ctx := context.Background()
	config := DefaultMilvusConfig()

	// Note: This will fail to connect if Milvus isn't running
	// For unit testing without Milvus, we'd need to mock the client
	store := &MilvusStore{
		config: config,
	}

	metadata := map[string]interface{}{
		"episode_id": "E1",
	}

	err := store.Insert(ctx, []rag.EmbeddingRecord{}, metadata)
	if err != ErrEmptyRecords {
		t.Errorf("Expected ErrEmptyRecords, got: %v", err)
	}
}

// TestMilvusStore_MissingMetadata tests error handling for missing metadata
func TestMilvusStore_MissingMetadata(t *testing.T) {
	ctx := context.Background()
	config := DefaultMilvusConfig()

	store := &MilvusStore{
		config: config,
	}

	records := []rag.EmbeddingRecord{
		{
			Text:      "Test",
			Embedding: make([]float32, 3072),
			Index:     0,
			Model:     "test-model",
		},
	}

	// Missing episode_id in metadata
	metadata := map[string]interface{}{
		"start_date": time.Now(),
	}

	err := store.Insert(ctx, records, metadata)
	if err == nil || !errors.Is(err, ErrMissingMetadata) {
		t.Errorf("Expected ErrMissingMetadata, got: %v", err)
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

	embedder, err := rag.NewOpenAIEmbedder("text-embedding-3-small", 1536)
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

	metadata1 := map[string]interface{}{
		"episode_id":   "episode-001",
		"start_date":   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		"end_date":     time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		"authors":      []string{"alice@example.com", "bob@example.com"},
		"commit_count": 5,
		"file_count":   3,
	}

	err = store.Insert(ctx, records1, metadata1)
	if err != nil {
		t.Fatalf("failed to insert episode-001: %v", err)
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

	metadata2 := map[string]interface{}{
		"episode_id":   "episode-002",
		"start_date":   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		"end_date":     time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
		"authors":      []string{"charlie@example.com"},
		"commit_count": 3,
		"file_count":   2,
	}

	err = store.Insert(ctx, records2, metadata2)
	if err != nil {
		t.Fatalf("failed to insert episode-002: %v", err)
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

	embedder, err := rag.NewOpenAIEmbedder("text-embedding-3-small", 1536)
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

	metadata := map[string]interface{}{
		"episode_id":   "sim-001",
		"start_date":   time.Now(),
		"end_date":     time.Now(),
		"authors":      []string{"test@example.com"},
		"commit_count": 10,
		"file_count":   5,
	}

	err = store.Insert(ctx, records, metadata)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
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

	embedder, err := rag.NewOpenAIEmbedder("text-embedding-3-small", 1536)
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

	// Insert multiple episodes using pre-embedded vectors
	recordIndex := 0
	for i, episodeID := range episodeIDs {
		records := allRecords[recordIndex : recordIndex+2]
		recordIndex += 2

		metadata := map[string]interface{}{
			"episode_id":   episodeID,
			"start_date":   time.Now().Add(time.Duration(i) * 24 * time.Hour),
			"end_date":     time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
			"authors":      []string{fmt.Sprintf("author%d@example.com", i+1)},
			"commit_count": (i + 1) * 5,
			"file_count":   (i + 1) * 2,
		}

		err = store.Insert(ctx, records, metadata)
		if err != nil {
			t.Fatalf("failed to insert %s: %v", episodeID, err)
		}
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

	if len(allResults) != 4 {
		t.Errorf("expected 4 results (2 per episode), got %d", len(allResults))
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

	embedder, err := rag.NewOpenAIEmbedder("text-embedding-3-small", 1536)
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

	metadata := map[string]interface{}{
		"episode_id":   "large-001",
		"start_date":   time.Now(),
		"end_date":     time.Now(),
		"authors":      []string{"author@example.com"},
		"commit_count": 1,
		"file_count":   50,
	}

	err = store.Insert(ctx, records, metadata)
	if err != nil {
		t.Fatalf("failed to insert large text: %v", err)
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
