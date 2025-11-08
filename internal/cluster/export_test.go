package cluster

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

func TestExportEpisodes_JSON(t *testing.T) {
	episodes := createTestEpisodes()
	var buf bytes.Buffer

	err := ExportEpisodes(episodes, "json", &buf)
	if err != nil {
		t.Fatalf("ExportEpisodes failed: %v", err)
	}

	// Parse the JSON output
	var exports []EpisodeExport
	if err := json.Unmarshal(buf.Bytes(), &exports); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify structure
	if len(exports) != 2 {
		t.Errorf("Expected 2 episodes, got %d", len(exports))
	}

	// Verify first episode
	if exports[0].ID != "E1" {
		t.Errorf("Expected ID 'E1', got '%s'", exports[0].ID)
	}
	if exports[0].CommitCount != 2 {
		t.Errorf("Expected 2 commits, got %d", exports[0].CommitCount)
	}
	if exports[0].AuthorCount != 1 {
		t.Errorf("Expected 1 author, got %d", exports[0].AuthorCount)
	}
	if exports[0].PRCount != 1 {
		t.Errorf("Expected 1 PR, got %d", exports[0].PRCount)
	}
}

func TestExportEpisodes_UnsupportedFormat(t *testing.T) {
	episodes := createTestEpisodes()
	var buf bytes.Buffer

	err := ExportEpisodes(episodes, "xml", &buf)
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported export format") {
		t.Errorf("Expected 'unsupported export format' error, got: %v", err)
	}
}

func TestExportEpisodes_CaseInsensitive(t *testing.T) {
	episodes := createTestEpisodes()
	var buf bytes.Buffer

	// Test uppercase format
	err := ExportEpisodes(episodes, "JSON", &buf)
	if err != nil {
		t.Fatalf("ExportEpisodes failed with uppercase format: %v", err)
	}
}

func TestEnrichEpisode(t *testing.T) {
	now := time.Now()
	episode := Episode{
		ID: "test-episode",
		Commits: []git.Commit{
			{
				Hash:        "abc123",
				Message:     "First commit",
				Author:      git.Author{Name: "Alice", Email: "alice@example.com"},
				CommittedAt: now,
			},
			{
				Hash:        "def456",
				Message:     "Second commit",
				Author:      git.Author{Name: "Bob", Email: "bob@example.com"},
				CommittedAt: now.Add(1 * time.Hour),
			},
		},
		Artifacts: []Artifact{
			{
				ID:     "pr-1",
				Type:   ArtifactPullRequest,
				Title:  "Test PR",
				Number: 1,
			},
			{
				ID:     "issue-1",
				Type:   ArtifactIssue,
				Title:  "Test Issue",
				Number: 2,
			},
		},
	}

	export := enrichEpisode(episode)

	if export.ID != "test-episode" {
		t.Errorf("Expected ID 'test-episode', got '%s'", export.ID)
	}
	if export.CommitCount != 2 {
		t.Errorf("Expected 2 commits, got %d", export.CommitCount)
	}
	if export.AuthorCount != 2 {
		t.Errorf("Expected 2 authors, got %d", export.AuthorCount)
	}
	if export.PRCount != 1 {
		t.Errorf("Expected 1 PR, got %d", export.PRCount)
	}
	if export.IssueCount != 1 {
		t.Errorf("Expected 1 issue, got %d", export.IssueCount)
	}
	if len(export.Authors) != 2 {
		t.Errorf("Expected 2 author names, got %d", len(export.Authors))
	}
	if len(export.CommitHashes) != 2 {
		t.Errorf("Expected 2 commit hashes, got %d", len(export.CommitHashes))
	}
}

func TestExportEpisodes_EmptyEpisodes(t *testing.T) {
	var episodes []Episode
	var buf bytes.Buffer

	err := ExportEpisodes(episodes, "json", &buf)
	if err != nil {
		t.Fatalf("ExportEpisodes failed with empty episodes: %v", err)
	}

	// Should produce valid empty JSON array
	if buf.String() != "[]\n" {
		t.Errorf("Expected empty JSON array, got: %s", buf.String())
	}
}

// Helper function to create test episodes
func createTestEpisodes() []Episode {
	now := time.Now()
	return []Episode{
		{
			ID: "E1",
			Commits: []git.Commit{
				{
					Hash:        "abc123",
					Message:     "Initial commit",
					Author:      git.Author{Name: "Alice", Email: "alice@example.com"},
					CommittedAt: now,
				},
				{
					Hash:        "def456",
					Message:     "Add feature",
					Author:      git.Author{Name: "Alice", Email: "alice@example.com"},
					CommittedAt: now.Add(30 * time.Minute),
				},
			},
			Artifacts: []Artifact{
				{
					ID:     "pr-1",
					Type:   ArtifactPullRequest,
					Number: 1,
					Title:  "Add new feature",
				},
			},
		},
		{
			ID: "E2",
			Commits: []git.Commit{
				{
					Hash:        "ghi789",
					Message:     "Fix bug",
					Author:      git.Author{Name: "Bob", Email: "bob@example.com"},
					CommittedAt: now.Add(2 * time.Hour),
				},
			},
			Artifacts: []Artifact{
				{
					ID:     "issue-1",
					Type:   ArtifactIssue,
					Number: 2,
					Title:  "Bug report",
				},
				{
					ID:     "pr-2",
					Type:   ArtifactPullRequest,
					Number: 3,
					Title:  "Bug fix",
				},
			},
		},
	}
}
