package narrative

import (
	"strings"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/Yates-Labs/thunk/internal/rag"
)

func TestAssemblePrompt_MissingTarget(t *testing.T) {
	_, err := AssemblePrompt(nil, nil)
	if err != ErrMissingTargetEpisode {
		t.Fatalf("expected ErrMissingTargetEpisode, got %v", err)
	}
}

func TestAssemblePrompt_Smoke(t *testing.T) {
	episode := &cluster.Episode{
		ID: "E1",
		Commits: []git.Commit{
			{
				Hash:        "abc123def456",
				Message:     "Add authentication middleware",
				Author:      git.Author{Name: "Alice", Email: "alice@example.com"},
				CommittedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			{
				Hash:        "def456ghi789",
				Message:     "Implement JWT token validation",
				Author:      git.Author{Name: "Bob", Email: "bob@example.com"},
				CommittedAt: time.Date(2024, 1, 16, 14, 30, 0, 0, time.UTC),
			},
		},
		Artifacts: []cluster.Artifact{
			{
				ID:          "123",
				Number:      42,
				Type:        cluster.ArtifactPullRequest,
				Title:       "Add JWT authentication",
				Description: "Implements JWT-based authentication.",
				State:       "merged",
			},
		},
	}

	contextChunks := []rag.ContextChunk{
		{EpisodeID: "E2", Text: "Migrated user sessions to Redis", Score: 0.85},
		{EpisodeID: "E3", Text: "Added OAuth2 integration", Score: 0.72},
	}

	prompt, err := AssemblePrompt(episode, contextChunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Minimal key checks (avoid brittle formatting tests)
	if !strings.Contains(prompt, "**Episode ID:** E1") {
		t.Fatal("missing episode id")
	}
	if !strings.Contains(prompt, "Add authentication middleware") {
		t.Fatal("missing commit message")
	}
	if !strings.Contains(prompt, "Alice") || !strings.Contains(prompt, "Bob") {
		t.Fatal("missing author names")
	}
	if !strings.Contains(prompt, "pull_request #42") {
		t.Fatal("missing artifact info")
	}
	if !strings.Contains(prompt, "# Related Development Context") {
		t.Fatal("missing context section")
	}
	// Ensure higher score context appears before lower score context
	if strings.Index(prompt, "Episode E2") > strings.Index(prompt, "Episode E3") {
		t.Fatal("context not ordered by score descending")
	}
	if !strings.Contains(prompt, "Generate a narrative summary") {
		t.Fatal("missing task instructions")
	}
}

func TestAssemblePrompt_NoContext(t *testing.T) {
	episode := &cluster.Episode{
		ID: "E1",
		Commits: []git.Commit{
			{
				Hash:        "abc123",
				Message:     "Initial commit",
				Author:      git.Author{Name: "Alice"},
				CommittedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	prompt, err := AssemblePrompt(episode, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "**Episode ID:** E1") {
		t.Fatal("missing episode id")
	}
	if strings.Contains(prompt, "# Related Development Context") {
		t.Fatal("should not include context section when no chunks provided")
	}
}

func TestAssemblePrompt_ContextIncludesAllProvided(t *testing.T) {
	episode := &cluster.Episode{
		ID: "E1",
		Commits: []git.Commit{
			{
				Hash:        "abc123",
				Message:     "Test",
				Author:      git.Author{Name: "Alice"},
				CommittedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	// 10 chunks => should include all provided context chunks
	contextChunks := make([]rag.ContextChunk, 10)
	for i := 0; i < 10; i++ {
		contextChunks[i] = rag.ContextChunk{
			EpisodeID: string(rune('A' + i)),
			Text:      "Context",
			Score:     float32(10 - i),
		}
	}

	prompt, err := AssemblePrompt(episode, contextChunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parts := strings.Split(prompt, "# Related Development Context")
	if len(parts) < 2 {
		t.Fatal("missing context section")
	}
	count := strings.Count(parts[1], "**Episode")
	if count != 10 {
		t.Fatalf("expected all 10 context episodes, got %d", count)
	}
}

func TestGetTimeRange(t *testing.T) {
	commits := []git.Commit{
		{CommittedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{CommittedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{CommittedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
	}

	start, end := getTimeRange(commits)
	if !start.Equal(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start: %v", start)
	}
	if !end.Equal(time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected end: %v", end)
	}
}

func TestGetUniqueAuthors(t *testing.T) {
	commits := []git.Commit{
		{Author: git.Author{Name: "Alice"}},
		{Author: git.Author{Name: "Bob"}},
		{Author: git.Author{Name: "Alice"}},
		{Author: git.Author{Name: "Charlie"}},
	}

	authors := getUniqueAuthors(commits)
	if len(authors) != 3 {
		t.Fatalf("expected 3 unique authors, got %d (%v)", len(authors), authors)
	}

	seen := map[string]bool{}
	for _, a := range authors {
		seen[a] = true
	}
	if !seen["Alice"] || !seen["Bob"] || !seen["Charlie"] {
		t.Fatal("missing expected authors")
	}
}
