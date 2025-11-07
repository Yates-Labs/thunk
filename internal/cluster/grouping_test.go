package cluster

import (
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// Helper to create test commits
func createTestCommit(hash, message string, author git.Author, committedAt time.Time, files []string) git.Commit {
	diffs := make([]git.Diff, len(files))
	for i, file := range files {
		diffs[i] = git.Diff{
			FilePath:  file,
			Status:    "modified",
			Additions: 10,
			Deletions: 5,
		}
	}

	return git.Commit{
		Hash:           hash,
		ShortHash:      hash[:7],
		Author:         author,
		Committer:      author,
		Message:        message,
		MessageSubject: message,
		CommittedAt:    committedAt,
		Diffs:          diffs,
		Stats: git.CommitStats{
			FilesChanged: len(files),
			Additions:    10 * len(files),
			Deletions:    5 * len(files),
		},
	}
}

func TestDefaultGroupingConfig(t *testing.T) {
	config := DefaultGroupingConfig()

	if config.MaxTimeGap != 24*time.Hour {
		t.Errorf("Expected MaxTimeGap to be 24 hours, got %v", config.MaxTimeGap)
	}

	if config.MinCommits != 1 {
		t.Errorf("Expected MinCommits to be 1, got %d", config.MinCommits)
	}

	if config.MinSimilarityScore != 0.5 {
		t.Errorf("Expected MinSimilarityScore to be 0.5, got %f", config.MinSimilarityScore)
	}

	// Check weights sum close to 1.0
	totalWeight := config.TimeWeight + config.AuthorWeight + config.FileWeight +
		config.MessageWeight + config.ArtifactWeight
	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("Expected weights to sum to ~1.0, got %f", totalWeight)
	}
}

func TestGroupIntoEpisodes_EmptyCommits(t *testing.T) {
	ra := &RepositoryActivity{
		Commits: []git.Commit{},
	}

	episodes := ra.GroupIntoEpisodes(DefaultGroupingConfig())

	if len(episodes) != 0 {
		t.Errorf("Expected 0 episodes for empty commits, got %d", len(episodes))
	}
}

func TestGroupIntoEpisodes_SingleCommit(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Initial commit", author, baseTime, []string{"main.go"}),
		},
	}

	episodes := ra.GroupIntoEpisodes(DefaultGroupingConfig())

	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode, got %d", len(episodes))
	}

	if len(episodes[0].Commits) != 1 {
		t.Errorf("Expected 1 commit in episode, got %d", len(episodes[0].Commits))
	}
}

func TestGroupIntoEpisodes_SameAuthorAndFiles(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Add feature", author, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Fix bug", author, baseTime.Add(1*time.Hour), []string{"main.go"}),
			createTestCommit("ghi9012", "Update tests", author, baseTime.Add(2*time.Hour), []string{"main.go"}),
		},
	}

	episodes := ra.GroupIntoEpisodes(DefaultGroupingConfig())

	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode for related commits, got %d", len(episodes))
	}

	if len(episodes[0].Commits) != 3 {
		t.Errorf("Expected 3 commits in episode, got %d", len(episodes[0].Commits))
	}
}

func TestGroupIntoEpisodes_DifferentAuthors(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	alice := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}
	bob := git.Author{Name: "Bob", Email: "bob@example.com", When: baseTime}

	config := DefaultGroupingConfig()
	config.AuthorWeight = 1.0 // Make author the only factor
	config.TimeWeight = 0
	config.FileWeight = 0
	config.MessageWeight = 0
	config.ArtifactWeight = 0

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Alice commit 1", alice, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Alice commit 2", alice, baseTime.Add(1*time.Hour), []string{"main.go"}),
			createTestCommit("ghi9012", "Bob commit", bob, baseTime.Add(2*time.Hour), []string{"main.go"}),
		},
	}

	episodes := ra.GroupIntoEpisodes(config)

	if len(episodes) != 2 {
		t.Fatalf("Expected 2 episodes (one per author), got %d", len(episodes))
	}

	// First episode should have Alice's commits
	if len(episodes[0].Commits) != 2 {
		t.Errorf("Expected 2 commits in first episode, got %d", len(episodes[0].Commits))
	}

	// Second episode should have Bob's commit
	if len(episodes[1].Commits) != 1 {
		t.Errorf("Expected 1 commit in second episode, got %d", len(episodes[1].Commits))
	}
}

func TestGroupIntoEpisodes_TimeGap(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	config := DefaultGroupingConfig()
	config.MaxTimeGap = 2 * time.Hour
	config.TimeWeight = 0.8 // Make time more important
	config.AuthorWeight = 0.1
	config.FileWeight = 0.1
	config.MessageWeight = 0
	config.ArtifactWeight = 0

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Commit 1", author, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Commit 2", author, baseTime.Add(1*time.Hour), []string{"main.go"}),
			createTestCommit("ghi9012", "Commit 3", author, baseTime.Add(5*time.Hour), []string{"main.go"}), // Large gap
		},
	}

	episodes := ra.GroupIntoEpisodes(config)

	if len(episodes) != 2 {
		t.Fatalf("Expected 2 episodes due to time gap, got %d", len(episodes))
	}

	if len(episodes[0].Commits) != 2 {
		t.Errorf("Expected 2 commits in first episode, got %d", len(episodes[0].Commits))
	}

	if len(episodes[1].Commits) != 1 {
		t.Errorf("Expected 1 commit in second episode, got %d", len(episodes[1].Commits))
	}
}

func TestGroupIntoEpisodes_DifferentFiles(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	config := DefaultGroupingConfig()
	config.FileWeight = 1.0 // Make file overlap the only factor
	config.TimeWeight = 0
	config.AuthorWeight = 0
	config.MessageWeight = 0
	config.ArtifactWeight = 0

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Update main", author, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Update main again", author, baseTime.Add(1*time.Hour), []string{"main.go"}),
			createTestCommit("ghi9012", "Update utils", author, baseTime.Add(2*time.Hour), []string{"utils.go"}), // Different file
		},
	}

	episodes := ra.GroupIntoEpisodes(config)

	if len(episodes) != 2 {
		t.Fatalf("Expected 2 episodes due to different files, got %d", len(episodes))
	}
}

func TestGroupIntoEpisodes_WithArtifactReferences(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	artifact := Artifact{
		ID:        "issue-123",
		Number:    123,
		Type:      ArtifactIssue,
		Title:     "Test Issue",
		Author:    author,
		State:     "open",
		CreatedAt: baseTime,
	}

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Fix bug #123", author, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Address #123 review", author, baseTime.Add(1*time.Hour), []string{"main.go"}),
		},
		Artifacts: []Artifact{artifact},
	}

	episodes := ra.GroupIntoEpisodes(DefaultGroupingConfig())

	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode, got %d", len(episodes))
	}

	if len(episodes[0].Artifacts) == 0 {
		t.Errorf("Expected artifacts to be linked, but found none")
	}

	// Check that the artifact was linked
	found := false
	for _, a := range episodes[0].Artifacts {
		if a.ID == "issue-123" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected artifact #123 to be linked to episode")
	}
}

func TestGroupIntoEpisodes_MinCommits(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	alice := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}
	bob := git.Author{Name: "Bob", Email: "bob@example.com", When: baseTime}

	config := DefaultGroupingConfig()
	config.MinCommits = 2 // Require at least 2 commits
	config.AuthorWeight = 1.0
	config.TimeWeight = 0
	config.FileWeight = 0
	config.MessageWeight = 0
	config.ArtifactWeight = 0

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Alice 1", alice, baseTime, []string{"main.go"}),
			createTestCommit("def5678", "Alice 2", alice, baseTime.Add(1*time.Hour), []string{"main.go"}),
			createTestCommit("ghi9012", "Bob 1", bob, baseTime.Add(2*time.Hour), []string{"main.go"}), // Only one commit from Bob
		},
	}

	episodes := ra.GroupIntoEpisodes(config)

	// Should only have 1 episode (Alice's), Bob's single commit is filtered out
	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode (Bob's filtered out), got %d", len(episodes))
	}

	if len(episodes[0].Commits) != 2 {
		t.Errorf("Expected 2 commits in episode, got %d", len(episodes[0].Commits))
	}
}

func TestSortCommitsByTime(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	commits := []git.Commit{
		createTestCommit("ghi9012", "Third", author, baseTime.Add(2*time.Hour), []string{"main.go"}),
		createTestCommit("abc1234", "First", author, baseTime, []string{"main.go"}),
		createTestCommit("def5678", "Second", author, baseTime.Add(1*time.Hour), []string{"main.go"}),
	}

	sortCommitsByTime(commits)

	if commits[0].Hash != "abc1234" {
		t.Errorf("Expected first commit to be abc1234, got %s", commits[0].Hash)
	}

	if commits[1].Hash != "def5678" {
		t.Errorf("Expected second commit to be def5678, got %s", commits[1].Hash)
	}

	if commits[2].Hash != "ghi9012" {
		t.Errorf("Expected third commit to be ghi9012, got %s", commits[2].Hash)
	}
}

func TestCalculateTimeScore(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	commit1 := createTestCommit("abc1234", "First", author, baseTime, []string{"main.go"})
	commit2 := createTestCommit("def5678", "Second", author, baseTime.Add(12*time.Hour), []string{"main.go"})
	commit3 := createTestCommit("ghi9012", "Third", author, baseTime.Add(25*time.Hour), []string{"main.go"})

	maxGap := 24 * time.Hour

	// Test within gap (50% of maxGap)
	score := calculateTimeScore(commit1, commit2, maxGap)
	if score < 0.4 || score > 0.6 {
		t.Errorf("Expected score around 0.5 for 12h gap in 24h window, got %f", score)
	}

	// Test beyond gap
	score = calculateTimeScore(commit1, commit3, maxGap)
	if score != 0.0 {
		t.Errorf("Expected score 0.0 for gap beyond maxGap, got %f", score)
	}

	// Test zero gap
	score = calculateTimeScore(commit1, commit1, maxGap)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for same time, got %f", score)
	}
}

func TestCalculateAuthorScore(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	alice := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}
	bob := git.Author{Name: "Bob", Email: "bob@example.com", When: baseTime}

	episode := &Episode{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Alice commit", alice, baseTime, []string{"main.go"}),
		},
	}

	aliceCommit := createTestCommit("def5678", "Another Alice commit", alice, baseTime, []string{"main.go"})
	bobCommit := createTestCommit("ghi9012", "Bob commit", bob, baseTime, []string{"main.go"})

	// Same author
	score := calculateAuthorScore(episode, aliceCommit)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for same author, got %f", score)
	}

	// Different author
	score = calculateAuthorScore(episode, bobCommit)
	if score != 0.0 {
		t.Errorf("Expected score 0.0 for different author, got %f", score)
	}
}

func TestCalculateFileScore(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	episode := &Episode{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Commit", author, baseTime, []string{"main.go", "utils.go"}),
		},
	}

	// Same files
	sameFilesCommit := createTestCommit("def5678", "Commit", author, baseTime, []string{"main.go", "utils.go"})
	score := calculateFileScore(episode, sameFilesCommit)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for identical files, got %f", score)
	}

	// Partial overlap
	partialCommit := createTestCommit("ghi9012", "Commit", author, baseTime, []string{"main.go", "other.go"})
	score = calculateFileScore(episode, partialCommit)
	if score <= 0.0 || score >= 1.0 {
		t.Errorf("Expected score between 0 and 1 for partial overlap, got %f", score)
	}

	// No overlap
	noOverlapCommit := createTestCommit("jkl3456", "Commit", author, baseTime, []string{"different.go"})
	score = calculateFileScore(episode, noOverlapCommit)
	if score != 0.0 {
		t.Errorf("Expected score 0.0 for no overlap, got %f", score)
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		message          string
		expectedKeywords []string
		notExpected      []string
	}{
		{
			message:          "Fix bug in authentication module",
			expectedKeywords: []string{"fix", "bug", "authentication", "module"},
			notExpected:      []string{"in"},
		},
		{
			message:          "Add new feature",
			expectedKeywords: []string{"add", "new", "feature"},
			notExpected:      []string{"a", "the"},
		},
		{
			message:          "Update README with installation instructions",
			expectedKeywords: []string{"update", "readme", "installation", "instructions"},
			notExpected:      []string{"with"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			keywords := extractKeywords(tt.message)

			for _, expected := range tt.expectedKeywords {
				if !keywords[expected] {
					t.Errorf("Expected keyword '%s' not found in: %v", expected, keywords)
				}
			}

			for _, unexpected := range tt.notExpected {
				if keywords[unexpected] {
					t.Errorf("Unexpected keyword '%s' found in: %v", unexpected, keywords)
				}
			}
		})
	}
}

func TestExtractArtifactReferences(t *testing.T) {
	tests := []struct {
		text     string
		expected []string
	}{
		{
			text:     "Fix bug #123",
			expected: []string{"#123"},
		},
		{
			text:     "Addresses PR-456 and issue-789",
			expected: []string{"PR-456", "issue-789"},
		},
		{
			text:     "Fixes #123, #456, and #789",
			expected: []string{"#123", "#456", "#789"},
		},
		{
			text:     "Merge MR-999 into main",
			expected: []string{"MR-999"},
		},
		{
			text:     "No references here",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			refs := extractArtifactReferences(tt.text)

			if len(refs) != len(tt.expected) {
				t.Errorf("Expected %d references, got %d", len(tt.expected), len(refs))
			}

			for _, expected := range tt.expected {
				if !refs[expected] {
					t.Errorf("Expected reference '%s' not found in: %v", expected, refs)
				}
			}
		})
	}
}

func TestBuildArtifactReferenceMap(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	artifacts := []Artifact{
		{
			ID:        "issue-123",
			Number:    123,
			Type:      ArtifactIssue,
			Title:     "Test Issue",
			Author:    author,
			CreatedAt: baseTime,
		},
		{
			ID:        "pr-456",
			Number:    456,
			Type:      ArtifactPullRequest,
			Title:     "Test PR",
			Author:    author,
			CreatedAt: baseTime,
		},
	}

	refMap := buildArtifactReferenceMap(artifacts)

	// Check various reference formats
	if refMap["#123"] == nil {
		t.Error("Expected #123 reference to be mapped")
	}

	if refMap["issue-123"] == nil {
		t.Error("Expected issue-123 reference to be mapped")
	}

	if refMap["PR-456"] == nil {
		t.Error("Expected PR-456 reference to be mapped")
	}

	if refMap["#456"] == nil {
		t.Error("Expected #456 reference to be mapped")
	}

	// Verify the mapped artifacts are correct
	if refMap["#123"].Number != 123 {
		t.Error("Mapped artifact has wrong number")
	}
}

func TestCalculateMessageScore(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	author := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}

	episode := &Episode{
		Commits: []git.Commit{
			createTestCommit("abc1234", "Fix authentication bug", author, baseTime, []string{"main.go"}),
		},
	}

	// Similar message
	similarCommit := createTestCommit("def5678", "Fix authentication issue", author, baseTime, []string{"main.go"})
	score := calculateMessageScore(episode, similarCommit)
	if score <= 0.0 {
		t.Errorf("Expected positive score for similar messages, got %f", score)
	}

	// Completely different message
	differentCommit := createTestCommit("ghi9012", "Update documentation", author, baseTime, []string{"main.go"})
	score = calculateMessageScore(episode, differentCommit)
	if score != 0.0 {
		t.Errorf("Expected score 0.0 for completely different message, got %f", score)
	}
}

func TestGroupIntoEpisodes_ComplexScenario(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	alice := git.Author{Name: "Alice", Email: "alice@example.com", When: baseTime}
	bob := git.Author{Name: "Bob", Email: "bob@example.com", When: baseTime}

	artifact := Artifact{
		ID:        "issue-100",
		Number:    100,
		Type:      ArtifactIssue,
		Title:     "Feature Request",
		Author:    alice,
		CreatedAt: baseTime,
	}

	ra := &RepositoryActivity{
		Commits: []git.Commit{
			// Episode 1: Alice working on feature #100
			createTestCommit("abc1234", "Start feature #100", alice, baseTime, []string{"feature.go"}),
			createTestCommit("def5678", "Continue feature #100", alice, baseTime.Add(1*time.Hour), []string{"feature.go"}),

			// Episode 2: Bob working on different files (large time gap)
			createTestCommit("ghi9012", "Fix unrelated bug", bob, baseTime.Add(30*time.Hour), []string{"utils.go"}),

			// Episode 3: Alice finishing feature #100
			createTestCommit("jkl3456", "Complete feature #100", alice, baseTime.Add(48*time.Hour), []string{"feature.go"}),
		},
		Artifacts: []Artifact{artifact},
	}

	config := DefaultGroupingConfig()
	config.MaxTimeGap = 24 * time.Hour

	episodes := ra.GroupIntoEpisodes(config)

	// Should have 3 episodes due to time gaps and different contexts
	if len(episodes) < 2 {
		t.Errorf("Expected at least 2 episodes in complex scenario, got %d", len(episodes))
	}

	// First episode should have Alice's initial commits and link to artifact
	if len(episodes[0].Commits) < 1 {
		t.Error("Expected commits in first episode")
	}

	if len(episodes[0].Artifacts) == 0 {
		t.Error("Expected artifact to be linked in first episode")
	}
}
