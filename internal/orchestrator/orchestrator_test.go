package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/adapter"
	"github.com/Yates-Labs/thunk/internal/cluster"
	githubmodel "github.com/Yates-Labs/thunk/internal/ingest/github"
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

// TestGitHubAdapterIntegration demonstrates how the orchestrator uses the adapter
func TestGitHubAdapterIntegration(t *testing.T) {
	// Create sample GitHub data
	issue := &githubmodel.Issue{
		ID:        12345,
		Number:    1,
		Title:     "Sample issue",
		Body:      "This is a test issue",
		State:     "open",
		Author:    "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Labels:    []string{"bug"},
		Comments: []githubmodel.Comment{
			{
				ID:        100,
				Author:    "reviewer",
				Body:      "Good catch!",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	pr := &githubmodel.PullRequest{
		ID:          67890,
		Number:      2,
		Title:       "Sample PR",
		Description: "This is a test PR",
		State:       "closed",
		Author:      "developer",
		BaseBranch:  "main",
		HeadBranch:  "feature/test",
		Merged:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create adapter
	ghAdapter := adapter.NewGitHubAdapter()

	// Convert artifacts
	issueArtifact, err := ghAdapter.ConvertIssue(issue)
	if err != nil {
		t.Fatalf("Failed to convert issue: %v", err)
	}

	prArtifact, err := ghAdapter.ConvertPullRequest(pr)
	if err != nil {
		t.Fatalf("Failed to convert PR: %v", err)
	}
	// Create repository activity
	activity := &cluster.RepositoryActivity{
		Platform:  ghAdapter.GetPlatform(),
		Artifacts: []cluster.Artifact{*issueArtifact, *prArtifact},
	}

	// Verify the activity
	if activity.Platform != cluster.PlatformGitHub {
		t.Errorf("Expected platform GitHub, got %s", activity.Platform)
	}

	if len(activity.Artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(activity.Artifacts))
	}

	// Verify issue artifact
	if activity.Artifacts[0].Type != cluster.ArtifactIssue {
		t.Errorf("Expected first artifact to be issue, got %s", activity.Artifacts[0].Type)
	}

	if activity.Artifacts[0].Number != 1 {
		t.Errorf("Expected issue number 1, got %d", activity.Artifacts[0].Number)
	}

	if len(activity.Artifacts[0].Discussions) != 1 {
		t.Errorf("Expected 1 discussion on issue, got %d", len(activity.Artifacts[0].Discussions))
	}

	// Verify PR artifact
	if activity.Artifacts[1].Type != cluster.ArtifactPullRequest {
		t.Errorf("Expected second artifact to be PR, got %s", activity.Artifacts[1].Type)
	}

	if activity.Artifacts[1].State != "merged" {
		t.Errorf("Expected PR state merged, got %s", activity.Artifacts[1].State)
	}

	t.Logf("Successfully converted %d artifacts from GitHub", len(activity.Artifacts))
}

// TestDetectPlatform tests platform detection from URLs
func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		url              string
		expectedPlatform cluster.SourcePlatform
		expectedOwner    string
		expectedRepo     string
	}{
		{
			url:              "https://github.com/Yates-Labs/thunk",
			expectedPlatform: cluster.PlatformGitHub,
			expectedOwner:    "Yates-Labs",
			expectedRepo:     "thunk",
		},
		{
			url:              "git@github.com:Yates-Labs/thunk.git",
			expectedPlatform: cluster.PlatformGitHub,
			expectedOwner:    "Yates-Labs",
			expectedRepo:     "thunk",
		},
		{
			url:              "/local/path/to/repo",
			expectedPlatform: cluster.PlatformGit,
			expectedOwner:    "",
			expectedRepo:     "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			platform, owner, repo := detectPlatform(tt.url)
			if platform != tt.expectedPlatform {
				t.Errorf("Expected platform %s, got %s", tt.expectedPlatform, platform)
			}
			if owner != tt.expectedOwner {
				t.Errorf("Expected owner %s, got %s", tt.expectedOwner, owner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("Expected repo %s, got %s", tt.expectedRepo, repo)
			}
		})
	}
}

// TestParseHostedGitURL tests the generic URL parser
func TestParseHostedGitURL(t *testing.T) {
	tests := []struct {
		url           string
		host          string
		expectedOwner string
		expectedRepo  string
	}{
		{
			url:           "https://github.com/owner/repo",
			host:          "github.com",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			url:           "git@github.com:owner/repo.git",
			host:          "github.com",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			url:           "https://github.com/owner/repo/",
			host:          "github.com",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			url:           "github.com/owner/repo.git",
			host:          "github.com",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			owner, repo := parseHostedGitURL(tt.url, tt.host)
			if owner != tt.expectedOwner {
				t.Errorf("Expected owner %s, got %s", tt.expectedOwner, owner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("Expected repo %s, got %s", tt.expectedRepo, repo)
			}
		})
	}
}

// TestAnalyzeGitHubRepositoryPlaceholder tests the GitHub analysis workflow structure
func TestAnalyzeGitHubRepositoryPlaceholder(t *testing.T) {
	// This test verifies the function works with GitHub URLs and tokens
	// Actual GitHub API calls would require authentication
	t.Skip("Requires GitHub API token and network access")

	ctx := context.Background()
	token := "" // Placeholder - would need actual token
	repoURL := "https://github.com/test-owner/test-repo"

	_, err := AnalyzeRepository(ctx, repoURL, token)
	if err != nil {
		t.Logf("Expected error without valid token: %v", err)
	}
}
