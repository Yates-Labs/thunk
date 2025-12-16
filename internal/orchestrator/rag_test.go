package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

func TestDefaultRAGConfig(t *testing.T) {
	config := DefaultRAGConfig()

	if config.TopK != 5 {
		t.Errorf("Expected TopK=5, got %d", config.TopK)
	}
	if config.MaxContextSize != 10 {
		t.Errorf("Expected MaxContextSize=10, got %d", config.MaxContextSize)
	}
	if config.ReindexOnDemand {
		t.Error("Expected ReindexOnDemand=false")
	}
	if config.EmbedderModel != "text-embedding-3-large" {
		t.Errorf("Expected EmbedderModel=text-embedding-3-large, got %s", config.EmbedderModel)
	}
	if config.EmbedderDimension != 3072 {
		t.Errorf("Expected EmbedderDimension=3072, got %d", config.EmbedderDimension)
	}
}

func TestGenerateEpisodeTitle(t *testing.T) {
	tests := []struct {
		name     string
		episode  *cluster.Episode
		expected string
	}{
		{
			name: "with commits",
			episode: &cluster.Episode{
				ID: "E1",
				Commits: []git.Commit{
					{Message: "Add authentication"},
					{Message: "Fix bug"},
				},
			},
			expected: "Add authentication",
		},
		{
			name: "with artifacts, no commits",
			episode: &cluster.Episode{
				ID:      "E2",
				Commits: []git.Commit{},
				Artifacts: []cluster.Artifact{
					{Title: "Implement feature X", Number: 42},
				},
			},
			expected: "Implement feature X",
		},
		{
			name: "no commits or artifacts",
			episode: &cluster.Episode{
				ID: "E3",
			},
			expected: "Episode E3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateEpisodeTitle(tt.episode)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateEpisodeSummaryText(t *testing.T) {
	episode := &cluster.Episode{
		ID: "E1",
		Commits: []git.Commit{
			{
				Message: "Add feature",
				Author:  git.Author{Name: "Alice"},
			},
			{
				Message: "Fix bug",
				Author:  git.Author{Name: "Bob"},
			},
		},
		Artifacts: []cluster.Artifact{
			{
				Type:   cluster.ArtifactPullRequest,
				Number: 42,
				Title:  "PR Title",
			},
		},
	}

	summary := generateEpisodeSummaryText(episode)

	// Check that summary contains key information
	if summary == "" {
		t.Fatal("Summary is empty")
	}

	expectedStrings := []string{
		"Commits (2)",
		"Add feature",
		"Alice",
		"Fix bug",
		"Bob",
		"Artifacts (1)",
		"PR #42",
		"PR Title",
	}

	for _, expected := range expectedStrings {
		if !contains(summary, expected) {
			t.Errorf("Summary missing expected string: %q", expected)
		}
	}
}

func TestGenerateEpisodeSummaryText_LimitCommits(t *testing.T) {
	// Create episode with more than 5 commits
	commits := make([]git.Commit, 10)
	for i := 0; i < 10; i++ {
		commits[i] = git.Commit{
			Message: "Commit message",
			Author:  git.Author{Name: "Dev"},
		}
	}

	episode := &cluster.Episode{
		ID:      "E1",
		Commits: commits,
	}

	summary := generateEpisodeSummaryText(episode)

	// Should limit to first 5 commits + "and X more" message
	if !contains(summary, "... and 5 more commits") {
		t.Error("Summary should indicate truncated commits")
	}
}

func TestEpisodeGetFileCount(t *testing.T) {
	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				Diffs: []git.Diff{
					{FilePath: "file1.go"},
					{FilePath: "file2.go"},
				},
			},
			{
				Diffs: []git.Diff{
					{FilePath: "file2.go"}, // Duplicate
					{FilePath: "file3.go"},
				},
			},
		},
	}

	fileCount := episode.GetFileCount()

	if fileCount != 3 {
		t.Fatalf("Expected 3 unique files, got %d", fileCount)
	}
}

func TestCreateProjectMetaEpisode(t *testing.T) {
	now := time.Now()
	episodes := []cluster.Episode{
		{
			ID: "E1",
			Commits: []git.Commit{
				{Message: "Commit 1", CommittedAt: now.Add(-48 * time.Hour)},
			},
			Artifacts: []cluster.Artifact{
				{Title: "Artifact 1"},
			},
		},
		{
			ID: "E2",
			Commits: []git.Commit{
				{Message: "Commit 2", CommittedAt: now.Add(-24 * time.Hour)},
				{Message: "Commit 3", CommittedAt: now},
			},
			Artifacts: []cluster.Artifact{
				{Title: "Artifact 2"},
			},
		},
	}

	metaEpisode := createProjectMetaEpisode(episodes)

	if metaEpisode.ID != "project-level" {
		t.Errorf("Expected ID=project-level, got %s", metaEpisode.ID)
	}

	// Should aggregate all commits
	if len(metaEpisode.Commits) != 3 {
		t.Errorf("Expected 3 commits, got %d", len(metaEpisode.Commits))
	}

	// Should aggregate all artifacts
	if len(metaEpisode.Artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(metaEpisode.Artifacts))
	}

	// Verify dates are computed from commits
	startDate, endDate := metaEpisode.GetDateRange()
	expectedStart := now.Add(-48 * time.Hour)
	expectedEnd := now

	if startDate.Sub(expectedStart).Abs() > time.Second {
		t.Errorf("Start date should be earliest commit time, got %v, want %v", startDate, expectedStart)
	}
	if endDate.Sub(expectedEnd).Abs() > time.Second {
		t.Errorf("End date should be latest commit time, got %v, want %v", endDate, expectedEnd)
	}
}

func TestGetEarliestDate(t *testing.T) {
	now := time.Now()
	episodes := []cluster.Episode{
		{Commits: []git.Commit{{CommittedAt: now.Add(-24 * time.Hour)}}},
		{Commits: []git.Commit{{CommittedAt: now.Add(-48 * time.Hour)}}}, // Earliest
		{Commits: []git.Commit{{CommittedAt: now}}},
	}

	earliest := getEarliestDate(episodes)

	if !earliest.Equal(now.Add(-48 * time.Hour)) {
		t.Error("Should return earliest date")
	}
}

func TestGetEarliestDate_Empty(t *testing.T) {
	earliest := getEarliestDate([]cluster.Episode{})

	if !earliest.IsZero() {
		t.Error("Should return zero time for empty episodes")
	}
}

func TestGetLatestDate(t *testing.T) {
	now := time.Now()
	episodes := []cluster.Episode{
		{Commits: []git.Commit{{CommittedAt: now.Add(-48 * time.Hour)}}},
		{Commits: []git.Commit{{CommittedAt: now.Add(-24 * time.Hour)}}},
		{Commits: []git.Commit{{CommittedAt: now}}}, // Latest
	}

	latest := getLatestDate(episodes)

	if !latest.Equal(now) {
		t.Error("Should return latest date")
	}
}

func TestGetLatestDate_Empty(t *testing.T) {
	latest := getLatestDate([]cluster.Episode{})

	if !latest.IsZero() {
		t.Error("Should return zero time for empty episodes")
	}
}

// Integration test - requires live Milvus and OpenAI API
func TestNewRAGPipeline_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	config := DefaultRAGConfig()

	pipeline, err := NewRAGPipeline(ctx, config)
	if err != nil {
		t.Skipf("Failed to create pipeline (may need env vars): %v", err)
	}
	defer pipeline.Close()

	if pipeline.embedder == nil {
		t.Error("Embedder should be initialized")
	}
	if pipeline.vectorStore == nil {
		t.Error("Vector store should be initialized")
	}
	if pipeline.retriever == nil {
		t.Error("Retriever should be initialized")
	}
	if pipeline.generator == nil {
		t.Error("Generator should be initialized")
	}
}

// Integration test for episode indexing
func TestRAGPipeline_IndexEpisodes_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	config := DefaultRAGConfig()
	config.ReindexOnDemand = true // Force reindex for test

	pipeline, err := NewRAGPipeline(ctx, config)
	if err != nil {
		t.Skipf("Failed to create pipeline (may need env vars): %v", err)
	}
	defer pipeline.Close()

	// Create test episodes
	now := time.Now()
	episodes := []cluster.Episode{
		{
			ID: "test-ep-1",
			Commits: []git.Commit{
				{
					Message:     "Add authentication system",
					Author:      git.Author{Name: "Alice"},
					CommittedAt: now.Add(-24 * time.Hour),
				},
			},
		},
		{
			ID: "test-ep-2",
			Commits: []git.Commit{
				{
					Message:     "Implement user permissions",
					Author:      git.Author{Name: "Bob"},
					CommittedAt: now.Add(-12 * time.Hour),
				},
			},
		},
	}

	err = pipeline.IndexEpisodes(ctx, episodes)
	if err != nil {
		t.Fatalf("Failed to index episodes: %v", err)
	}

	// Clean up - delete test episodes
	defer func() {
		episodeIDs := []string{"test-ep-1", "test-ep-2"}
		pipeline.vectorStore.Delete(ctx, episodeIDs)
	}()

	// Verify episodes were indexed
	existingMap, err := pipeline.vectorStore.Query(ctx, []string{"test-ep-1", "test-ep-2"})
	if err != nil {
		t.Fatalf("Failed to query episodes: %v", err)
	}

	if !existingMap["test-ep-1"] {
		t.Error("test-ep-1 should be indexed")
	}
	if !existingMap["test-ep-2"] {
		t.Error("test-ep-2 should be indexed")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
