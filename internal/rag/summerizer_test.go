package rag

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

func TestBuildEpisodeSummary_NilEpisode(t *testing.T) {
	result := BuildEpisodeSummary(nil)

	if result.EpisodeID != "" {
		t.Errorf("Expected empty EpisodeID, got %s", result.EpisodeID)
	}
	if result.Title != "" {
		t.Errorf("Expected empty Title, got %s", result.Title)
	}
	if result.Summary != "" {
		t.Errorf("Expected empty Summary, got %s", result.Summary)
	}
}

func TestBuildEpisodeSummary_OnlyCommits(t *testing.T) {
	episode := &cluster.Episode{
		ID: "episode-1",
		Commits: []git.Commit{
			{
				Hash:           "abc123",
				MessageSubject: "Add login feature",
				Message:        "Add login feature\n\nImplemented JWT authentication",
				Author: git.Author{
					Name: "alice",
					When: time.Date(2023, 4, 1, 10, 0, 0, 0, time.UTC),
				},
				CommittedAt: time.Date(2023, 4, 1, 10, 0, 0, 0, time.UTC),
			},
			{
				Hash:           "def456",
				MessageSubject: "Fix login bug",
				Message:        "Fix login bug",
				Author: git.Author{
					Name: "bob",
					When: time.Date(2023, 4, 2, 11, 0, 0, 0, time.UTC),
				},
				CommittedAt: time.Date(2023, 4, 2, 11, 0, 0, 0, time.UTC),
			},
		},
	}

	result := BuildEpisodeSummary(episode)

	if result.EpisodeID != "episode-1" {
		t.Errorf("Expected EpisodeID 'episode-1', got %s", result.EpisodeID)
	}

	if result.Title != "Add login feature" {
		t.Errorf("Expected title 'Add login feature', got %s", result.Title)
	}

	if !strings.Contains(result.Summary, "Commits:") {
		t.Error("Expected summary to contain 'Commits:' section")
	}

	if !strings.Contains(result.Summary, "- Add login feature") {
		t.Error("Expected summary to contain commit message")
	}

	if !strings.Contains(result.Summary, "Authors: alice, bob") {
		t.Errorf("Expected 'Authors: alice, bob', got summary: %s", result.Summary)
	}

	if !strings.Contains(result.Summary, "Date range: 2023-04-01 → 2023-04-02") {
		t.Errorf("Expected date range, got summary: %s", result.Summary)
	}
}

func TestBuildEpisodeSummary_OnlyArtifacts(t *testing.T) {
	episode := &cluster.Episode{
		ID: "episode-2",
		Artifacts: []cluster.Artifact{
			{
				ID:    "pr-1",
				Title: "Add user authentication",
				Type:  cluster.ArtifactPullRequest,
				Author: git.Author{
					Name: "charlie",
				},
				CreatedAt: time.Date(2023, 5, 1, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 5, 3, 14, 0, 0, 0, time.UTC),
			},
			{
				ID:    "issue-1",
				Title: "Implement OAuth support",
				Type:  cluster.ArtifactIssue,
				Author: git.Author{
					Name: "diana",
				},
				CreatedAt: time.Date(2023, 5, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 5, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	result := BuildEpisodeSummary(episode)

	if result.EpisodeID != "episode-2" {
		t.Errorf("Expected EpisodeID 'episode-2', got %s", result.EpisodeID)
	}

	if result.Title != "Add user authentication" {
		t.Errorf("Expected title from first artifact, got %s", result.Title)
	}

	if !strings.Contains(result.Summary, "PRs:") {
		t.Error("Expected summary to contain 'PRs:' section")
	}

	if !strings.Contains(result.Summary, "- Add user authentication") {
		t.Error("Expected summary to contain PR title")
	}

	if !strings.Contains(result.Summary, "Issues:") {
		t.Error("Expected summary to contain 'Issues:' section")
	}

	if !strings.Contains(result.Summary, "- Implement OAuth support") {
		t.Error("Expected summary to contain issue title")
	}

	if !strings.Contains(result.Summary, "Authors: charlie, diana") {
		t.Errorf("Expected authors, got summary: %s", result.Summary)
	}
}

func TestBuildEpisodeSummary_MixedContent(t *testing.T) {
	closedTime := time.Date(2023, 6, 5, 16, 0, 0, 0, time.UTC)
	mergedTime := time.Date(2023, 6, 5, 17, 0, 0, 0, time.UTC)

	episode := &cluster.Episode{
		ID: "episode-3",
		Commits: []git.Commit{
			{
				Hash:           "xyz789",
				MessageSubject: "Refactor auth module",
				Author: git.Author{
					Name: "eve",
				},
				CommittedAt: time.Date(2023, 6, 3, 12, 0, 0, 0, time.UTC),
			},
		},
		Artifacts: []cluster.Artifact{
			{
				ID:    "pr-2",
				Title: "Refactor authentication system",
				Type:  cluster.ArtifactPullRequest,
				Author: git.Author{
					Name: "eve",
				},
				CreatedAt: time.Date(2023, 6, 1, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 6, 4, 14, 0, 0, 0, time.UTC),
				MergedAt:  &mergedTime,
			},
			{
				ID:    "issue-2",
				Title: "Update auth dependencies",
				Type:  cluster.ArtifactIssue,
				Author: git.Author{
					Name: "frank",
				},
				CreatedAt: time.Date(2023, 6, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 6, 4, 11, 0, 0, 0, time.UTC),
				ClosedAt:  &closedTime,
			},
		},
	}

	result := BuildEpisodeSummary(episode)

	if result.EpisodeID != "episode-3" {
		t.Errorf("Expected EpisodeID 'episode-3', got %s", result.EpisodeID)
	}

	if !strings.Contains(result.Summary, "Commits:") {
		t.Error("Expected summary to contain 'Commits:' section")
	}

	if !strings.Contains(result.Summary, "PRs:") {
		t.Error("Expected summary to contain 'PRs:' section")
	}

	if !strings.Contains(result.Summary, "Issues:") {
		t.Error("Expected summary to contain 'Issues:' section")
	}

	if !strings.Contains(result.Summary, "Authors: eve, frank") {
		t.Errorf("Expected authors 'eve, frank', got summary: %s", result.Summary)
	}

	// Date range should include the merged time
	if !strings.Contains(result.Summary, "Date range: 2023-06-01 → 2023-06-05") {
		t.Errorf("Expected date range including merged time, got summary: %s", result.Summary)
	}
}

func TestBuildEpisodeSummary_AllArtifactTypes(t *testing.T) {
	episode := &cluster.Episode{
		ID: "episode-4",
		Artifacts: []cluster.Artifact{
			{
				ID:    "pr-1",
				Title: "Pull Request Title",
				Type:  cluster.ArtifactPullRequest,
				Author: git.Author{
					Name: "user1",
				},
				CreatedAt: time.Date(2023, 7, 1, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 7, 1, 9, 0, 0, 0, time.UTC),
			},
			{
				ID:    "mr-1",
				Title: "Merge Request Title",
				Type:  cluster.ArtifactMergeRequest,
				Author: git.Author{
					Name: "user2",
				},
				CreatedAt: time.Date(2023, 7, 2, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 7, 2, 9, 0, 0, 0, time.UTC),
			},
			{
				ID:    "issue-1",
				Title: "Issue Title",
				Type:  cluster.ArtifactIssue,
				Author: git.Author{
					Name: "user3",
				},
				CreatedAt: time.Date(2023, 7, 3, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 7, 3, 9, 0, 0, 0, time.UTC),
			},
			{
				ID:    "ticket-1",
				Title: "Ticket Title",
				Type:  cluster.ArtifactTicket,
				Author: git.Author{
					Name: "user4",
				},
				CreatedAt: time.Date(2023, 7, 4, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 7, 4, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	result := BuildEpisodeSummary(episode)
	fmt.Println(result.Summary)

	if !strings.Contains(result.Summary, "PRs:") {
		t.Error("Expected summary to contain 'PRs:' section")
	}

	if !strings.Contains(result.Summary, "MRs:") {
		t.Error("Expected summary to contain 'MRs:' section")
	}

	if !strings.Contains(result.Summary, "Issues:") {
		t.Error("Expected summary to contain 'Issues:' section")
	}

	if !strings.Contains(result.Summary, "Tickets:") {
		t.Error("Expected summary to contain 'Tickets:' section")
	}

	if !strings.Contains(result.Summary, "- Pull Request Title") {
		t.Error("Expected PR title in summary")
	}

	if !strings.Contains(result.Summary, "- Merge Request Title") {
		t.Error("Expected MR title in summary")
	}

	if !strings.Contains(result.Summary, "- Issue Title") {
		t.Error("Expected issue title in summary")
	}

	if !strings.Contains(result.Summary, "- Ticket Title") {
		t.Error("Expected ticket title in summary")
	}
}

func TestGenerateTitle_FromCommit(t *testing.T) {
	episode := &cluster.Episode{
		ID: "test",
		Commits: []git.Commit{
			{
				MessageSubject: "Implement new feature",
				Message:        "Implement new feature\n\nDetailed description",
			},
		},
	}

	title := generateTitle(episode)

	if title != "Implement new feature" {
		t.Errorf("Expected title 'Implement new feature', got %s", title)
	}
}

func TestGenerateTitle_FromArtifact(t *testing.T) {
	episode := &cluster.Episode{
		ID: "test",
		Artifacts: []cluster.Artifact{
			{
				Title: "Fix critical bug",
			},
		},
	}

	title := generateTitle(episode)

	if title != "Fix critical bug" {
		t.Errorf("Expected title 'Fix critical bug', got %s", title)
	}
}

func TestGenerateTitle_Fallback(t *testing.T) {
	episode := &cluster.Episode{
		ID: "episode-123",
	}

	title := generateTitle(episode)

	if title != "Episode episode-123" {
		t.Errorf("Expected fallback title, got %s", title)
	}
}

func TestExtractAuthors_UniqueAuthors(t *testing.T) {
	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				Author: git.Author{Name: "alice"},
			},
			{
				Author: git.Author{Name: "bob"},
			},
			{
				Author: git.Author{Name: "alice"}, // Duplicate
			},
		},
		Artifacts: []cluster.Artifact{
			{
				Author: git.Author{Name: "charlie"},
			},
			{
				Author: git.Author{Name: "bob"}, // Duplicate
			},
		},
	}

	authors := episode.GetAuthorNames()

	// Check we have exactly 3 unique authors
	if len(authors) != 3 {
		t.Errorf("Expected 3 unique authors, got %d: %v", len(authors), authors)
	}

	// Check all expected authors are present
	authorMap := make(map[string]bool)
	for _, author := range authors {
		authorMap[author] = true
	}

	expectedAuthors := []string{"alice", "bob", "charlie"}
	for _, expected := range expectedAuthors {
		if !authorMap[expected] {
			t.Errorf("Expected author %s not found", expected)
		}
	}
}

func TestExtractAuthors_EmptyNames(t *testing.T) {
	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				Author: git.Author{Name: ""}, // Empty name
			},
			{
				Author: git.Author{Name: "alice"},
			},
		},
	}

	authors := episode.GetAuthorNames()

	// Should only include non-empty names
	if len(authors) != 1 {
		t.Errorf("Expected 1 author, got %d: %v", len(authors), authors)
	}

	if len(authors) > 0 && authors[0] != "alice" {
		t.Errorf("Expected author 'alice', got %s", authors[0])
	}
}

func TestExtractDateRange_OnlyCommits(t *testing.T) {
	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				CommittedAt: time.Date(2023, 3, 10, 10, 0, 0, 0, time.UTC),
			},
			{
				CommittedAt: time.Date(2023, 3, 15, 11, 0, 0, 0, time.UTC),
			},
			{
				CommittedAt: time.Date(2023, 3, 12, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	start, end := episode.GetDateRange()
	dateRange := formatDateRange(start, end)

	expected := "2023-03-10 → 2023-03-15"
	if dateRange != expected {
		t.Errorf("Expected date range '%s', got '%s'", expected, dateRange)
	}
}

func TestExtractDateRange_WithArtifacts(t *testing.T) {
	closedTime := time.Date(2023, 3, 20, 16, 0, 0, 0, time.UTC)

	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				CommittedAt: time.Date(2023, 3, 10, 10, 0, 0, 0, time.UTC),
			},
		},
		Artifacts: []cluster.Artifact{
			{
				CreatedAt: time.Date(2023, 3, 8, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 3, 15, 14, 0, 0, 0, time.UTC),
				ClosedAt:  &closedTime,
			},
		},
	}

	start, end := episode.GetDateRange()
	dateRange := formatDateRange(start, end)

	expected := "2023-03-08 → 2023-03-20"
	if dateRange != expected {
		t.Errorf("Expected date range '%s', got '%s'", expected, dateRange)
	}
}

func TestExtractDateRange_SingleDate(t *testing.T) {
	singleDate := time.Date(2023, 4, 1, 10, 0, 0, 0, time.UTC)

	episode := &cluster.Episode{
		Commits: []git.Commit{
			{
				CommittedAt: singleDate,
			},
		},
	}

	start, end := episode.GetDateRange()
	dateRange := formatDateRange(start, end)

	expected := "2023-04-01"
	if dateRange != expected {
		t.Errorf("Expected single date '%s', got '%s'", expected, dateRange)
	}
}

func TestExtractDateRange_Empty(t *testing.T) {
	episode := &cluster.Episode{
		ID: "test",
	}

	start, end := episode.GetDateRange()
	dateRange := formatDateRange(start, end)

	if dateRange != "" {
		t.Errorf("Expected empty date range, got '%s'", dateRange)
	}
}

func TestBuildSummaryText_CommitWithoutSubject(t *testing.T) {
	episode := &cluster.Episode{
		ID: "test",
		Commits: []git.Commit{
			{
				MessageSubject: "",
				Message:        "Full commit message without subject",
				Author: git.Author{
					Name: "tester",
				},
				CommittedAt: time.Date(2023, 8, 1, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	summary := buildSummaryText(episode)

	if !strings.Contains(summary, "- Full commit message without subject") {
		t.Errorf("Expected full message when subject is empty, got: %s", summary)
	}
}
