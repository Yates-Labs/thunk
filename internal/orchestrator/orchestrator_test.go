package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
)

func TestAnalyzeRepository_RealRepo(t *testing.T) {
	ctx := context.Background()

	// Test with this repository (should work if we're in the repo)
	episodes, err := AnalyzeRepository(ctx, "https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to analyze repository: %v", err)
	}

	if len(episodes) == 0 {
		t.Error("Expected at least one episode from repository")
	}

	// Verify episode structure
	for i, episode := range episodes {
		if episode.ID == "" {
			t.Errorf("Episode %d has empty ID", i)
		}

		if len(episode.Commits) == 0 {
			t.Errorf("Episode %d has no commits", i)
		}

		// Verify commits have proper structure
		for j, commit := range episode.Commits {
			if commit.Hash == "" {
				t.Errorf("Episode %d, Commit %d has empty hash", i, j)
			}
			if commit.Author.Email == "" {
				t.Errorf("Episode %d, Commit %d has empty author email", i, j)
			}
		}
	}
}

func TestAnalyzeRepository_ContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := AnalyzeRepository(ctx, "https://github.com/Yates-Labs/thunk")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestAnalyzeRepository_InvalidRepo(t *testing.T) {
	ctx := context.Background()

	// Test with non-existent local path
	_, err := AnalyzeRepository(ctx, "/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent repository")
	}

	// Test with invalid URL
	_, err = AnalyzeRepository(ctx, "https://github.com/invalid/repo/that/does/not/exist/123456789")
	if err == nil {
		t.Error("Expected error for invalid repository URL")
	}
}

func TestAnalyzeRepositoryWithConfig(t *testing.T) {
	// Create custom config with tighter time window
	config := cluster.GroupingConfig{
		MaxTimeGap:         6 * time.Hour, // Tighter time window
		MinCommits:         2,
		TimeWeight:         0.5,
		AuthorWeight:       0.3,
		FileWeight:         0.1,
		MessageWeight:      0.05,
		ArtifactWeight:     0.05,
		MinSimilarityScore: 0.3,
	}

	ctx := context.Background()

	episodes, err := AnalyzeRepositoryWithConfig(ctx, "https://github.com/Yates-Labs/thunk", config)
	if err != nil {
		t.Fatalf("Failed to analyze repository: %v", err)
	}

	// With stricter config, we might get different number of episodes
	// Just verify we get valid results
	if len(episodes) == 0 {
		t.Error("Expected at least one episode")
	}
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/path/to/myrepo",
			expected: "myrepo",
		},
		{
			input:    "https://github.com/user/myrepo",
			expected: "myrepo",
		},
		{
			input:    "https://github.com/user/myrepo.git",
			expected: "myrepo",
		},
		{
			input:    "myrepo",
			expected: "myrepo",
		},
		{
			input:    "/path/to/myrepo/",
			expected: "myrepo",
		},
		{
			input:    "https://github.com/user/myrepo/",
			expected: "myrepo",
		},
		{
			input:    "https://github.com/Yates-Labs/thunk",
			expected: "thunk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractRepoName(tt.input)
			if result != tt.expected {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeRepository_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Full end-to-end test
	episodes, err := AnalyzeRepository(ctx, "https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("End-to-end analysis failed: %v", err)
	}

	if len(episodes) == 0 {
		t.Fatal("Expected at least one episode")
	}

	t.Logf("Generated %d episodes", len(episodes))

	// Verify data quality
	totalCommits := 0
	for i, episode := range episodes {
		t.Logf("Episode %d (%s): %d commits", i+1, episode.ID, len(episode.Commits))
		totalCommits += len(episode.Commits)

		// Each episode should have at least one commit
		if len(episode.Commits) == 0 {
			t.Errorf("Episode %d is empty", i)
		}

		// Verify commits are properly ordered (oldest to newest within episode)
		for j := 1; j < len(episode.Commits); j++ {
			if episode.Commits[j].CommittedAt.Before(episode.Commits[j-1].CommittedAt) {
				t.Errorf("Episode %d has commits out of chronological order", i)
			}
		}
	}

	if totalCommits == 0 {
		t.Error("No commits found across all episodes")
	}

	t.Logf("Total commits across all episodes: %d", totalCommits)
}
