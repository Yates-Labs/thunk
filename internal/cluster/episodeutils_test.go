package cluster

import (
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// Helper function to create test authors
func createAuthor(name, email string) git.Author {
	return git.Author{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

// Helper function to create test commits
func createCommit(hash string, author git.Author) git.Commit {
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:8]
	}
	return git.Commit{
		Hash:        hash,
		ShortHash:   shortHash,
		Author:      author,
		Committer:   author,
		Message:     "Test commit",
		CommittedAt: time.Now(),
		Stats: git.CommitStats{
			FilesChanged: 1,
			Additions:    10,
			Deletions:    5,
			NetChange:    5,
		},
	}
}

func TestEpisode_GetCommitAuthors(t *testing.T) {
	author1 := createAuthor("Alice", "alice@example.com")
	author2 := createAuthor("Bob", "bob@example.com")
	author3 := createAuthor("Alice", "alice@example.com") // Same as author1

	tests := []struct {
		name           string
		commits        []git.Commit
		expectedCount  int
		expectedEmails map[string]bool
		description    string
	}{
		{
			name:           "empty commits",
			commits:        []git.Commit{},
			expectedCount:  0,
			expectedEmails: map[string]bool{},
			description:    "should return empty slice for no commits",
		},
		{
			name: "single commit single author",
			commits: []git.Commit{
				createCommit("abc123", author1),
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should return one author for single commit",
		},
		{
			name: "multiple commits same author",
			commits: []git.Commit{
				createCommit("abc123", author1),
				createCommit("def456", author3), // Same email as author1
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should deduplicate authors with same email",
		},
		{
			name: "multiple commits different authors",
			commits: []git.Commit{
				createCommit("abc123", author1),
				createCommit("def456", author2),
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should return all unique authors",
		},
		{
			name: "multiple commits mixed authors",
			commits: []git.Commit{
				createCommit("abc123", author1),
				createCommit("def456", author2),
				createCommit("ghi789", author1),
				createCommit("jkl012", author2),
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should deduplicate multiple commits from same authors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			episode := &Episode{
				ID:      "test-episode",
				Commits: tt.commits,
			}

			authors := episode.GetCommitAuthors()

			if len(authors) != tt.expectedCount {
				t.Errorf("GetCommitAuthors() returned %d authors, expected %d. %s",
					len(authors), tt.expectedCount, tt.description)
			}

			// Verify all expected emails are present
			foundEmails := make(map[string]bool)
			for _, author := range authors {
				foundEmails[author.Email] = true
			}

			for email := range tt.expectedEmails {
				if !foundEmails[email] {
					t.Errorf("Expected author with email %s not found", email)
				}
			}

			for email := range foundEmails {
				if !tt.expectedEmails[email] {
					t.Errorf("Unexpected author with email %s found", email)
				}
			}
		})
	}
}

func TestEpisode_GetDiscussionAuthors(t *testing.T) {
	author1 := createAuthor("Alice", "alice@example.com")
	author2 := createAuthor("Bob", "bob@example.com")
	author3 := createAuthor("Alice", "alice@example.com") // Same as author1

	tests := []struct {
		name           string
		discussions    []Discussion
		expectedCount  int
		expectedEmails map[string]bool
		description    string
	}{
		{
			name:           "empty discussions",
			discussions:    []Discussion{},
			expectedCount:  0,
			expectedEmails: map[string]bool{},
			description:    "should return empty slice for no discussions",
		},
		{
			name: "single discussion single author",
			discussions: []Discussion{
				{
					ID:        "d1",
					Type:      DiscussionComment,
					Author:    author1,
					Body:      "Test comment",
					CreatedAt: time.Now(),
				},
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should return one author for single discussion",
		},
		{
			name: "multiple discussions same author",
			discussions: []Discussion{
				{
					ID:        "d1",
					Type:      DiscussionComment,
					Author:    author1,
					Body:      "Test comment 1",
					CreatedAt: time.Now(),
				},
				{
					ID:        "d2",
					Type:      DiscussionReview,
					Author:    author3, // Same email as author1
					Body:      "Test review",
					CreatedAt: time.Now(),
				},
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should deduplicate authors with same email",
		},
		{
			name: "multiple discussions different authors",
			discussions: []Discussion{
				{
					ID:        "d1",
					Type:      DiscussionComment,
					Author:    author1,
					Body:      "Test comment",
					CreatedAt: time.Now(),
				},
				{
					ID:        "d2",
					Type:      DiscussionReview,
					Author:    author2,
					Body:      "Test review",
					CreatedAt: time.Now(),
				},
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should return all unique authors",
		},
		{
			name: "multiple discussions mixed authors",
			discussions: []Discussion{
				{
					ID:        "d1",
					Type:      DiscussionComment,
					Author:    author1,
					Body:      "Comment 1",
					CreatedAt: time.Now(),
				},
				{
					ID:        "d2",
					Type:      DiscussionReview,
					Author:    author2,
					Body:      "Review 1",
					CreatedAt: time.Now(),
				},
				{
					ID:        "d3",
					Type:      DiscussionComment,
					Author:    author1,
					Body:      "Comment 2",
					CreatedAt: time.Now(),
				},
				{
					ID:        "d4",
					Type:      DiscussionReviewThread,
					Author:    author2,
					Body:      "Thread comment",
					CreatedAt: time.Now(),
				},
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should deduplicate multiple discussions from same authors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			episode := &Episode{
				ID:          "test-episode",
				Discussions: tt.discussions,
			}

			authors := episode.GetDiscussionAuthors()

			if len(authors) != tt.expectedCount {
				t.Errorf("GetDiscussionAuthors() returned %d authors, expected %d. %s",
					len(authors), tt.expectedCount, tt.description)
			}

			// Verify all expected emails are present
			foundEmails := make(map[string]bool)
			for _, author := range authors {
				foundEmails[author.Email] = true
			}

			for email := range tt.expectedEmails {
				if !foundEmails[email] {
					t.Errorf("Expected author with email %s not found", email)
				}
			}

			for email := range foundEmails {
				if !tt.expectedEmails[email] {
					t.Errorf("Unexpected author with email %s found", email)
				}
			}
		})
	}
}

func TestEpisode_GetArtifactAuthors(t *testing.T) {
	author1 := createAuthor("Alice", "alice@example.com")
	author2 := createAuthor("Bob", "bob@example.com")
	author3 := createAuthor("Alice", "alice@example.com") // Same as author1

	tests := []struct {
		name           string
		artifacts      []Artifact
		expectedCount  int
		expectedEmails map[string]bool
		description    string
	}{
		{
			name:           "empty artifacts",
			artifacts:      []Artifact{},
			expectedCount:  0,
			expectedEmails: map[string]bool{},
			description:    "should return empty slice for no artifacts",
		},
		{
			name: "single artifact single author",
			artifacts: []Artifact{
				{
					ID:          "a1",
					Number:      1,
					Type:        ArtifactIssue,
					Title:       "Test issue",
					Description: "Test description",
					Author:      author1,
					State:       "open",
					CreatedAt:   time.Now(),
				},
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should return one author for single artifact",
		},
		{
			name: "multiple artifacts same author",
			artifacts: []Artifact{
				{
					ID:          "a1",
					Number:      1,
					Type:        ArtifactIssue,
					Title:       "Issue 1",
					Description: "Description 1",
					Author:      author1,
					State:       "open",
					CreatedAt:   time.Now(),
				},
				{
					ID:          "a2",
					Number:      2,
					Type:        ArtifactPullRequest,
					Title:       "PR 1",
					Description: "Description 2",
					Author:      author3, // Same email as author1
					State:       "open",
					CreatedAt:   time.Now(),
				},
			},
			expectedCount: 1,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
			},
			description: "should deduplicate authors with same email",
		},
		{
			name: "multiple artifacts different authors",
			artifacts: []Artifact{
				{
					ID:          "a1",
					Number:      1,
					Type:        ArtifactIssue,
					Title:       "Issue 1",
					Description: "Description 1",
					Author:      author1,
					State:       "open",
					CreatedAt:   time.Now(),
				},
				{
					ID:          "a2",
					Number:      2,
					Type:        ArtifactPullRequest,
					Title:       "PR 1",
					Description: "Description 2",
					Author:      author2,
					State:       "open",
					CreatedAt:   time.Now(),
				},
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should return all unique authors",
		},
		{
			name: "multiple artifacts mixed authors",
			artifacts: []Artifact{
				{
					ID:          "a1",
					Number:      1,
					Type:        ArtifactIssue,
					Title:       "Issue 1",
					Description: "Description 1",
					Author:      author1,
					State:       "open",
					CreatedAt:   time.Now(),
				},
				{
					ID:          "a2",
					Number:      2,
					Type:        ArtifactPullRequest,
					Title:       "PR 1",
					Description: "Description 2",
					Author:      author2,
					State:       "open",
					CreatedAt:   time.Now(),
				},
				{
					ID:          "a3",
					Number:      3,
					Type:        ArtifactIssue,
					Title:       "Issue 2",
					Description: "Description 3",
					Author:      author1,
					State:       "closed",
					CreatedAt:   time.Now(),
				},
				{
					ID:          "a4",
					Number:      4,
					Type:        ArtifactMergeRequest,
					Title:       "MR 1",
					Description: "Description 4",
					Author:      author2,
					State:       "merged",
					CreatedAt:   time.Now(),
				},
			},
			expectedCount: 2,
			expectedEmails: map[string]bool{
				"alice@example.com": true,
				"bob@example.com":   true,
			},
			description: "should deduplicate multiple artifacts from same authors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			episode := &Episode{
				ID:        "test-episode",
				Artifacts: tt.artifacts,
			}

			authors := episode.GetArtifactAuthors()

			if len(authors) != tt.expectedCount {
				t.Errorf("GetArtifactAuthors() returned %d authors, expected %d. %s",
					len(authors), tt.expectedCount, tt.description)
			}

			// Verify all expected emails are present
			foundEmails := make(map[string]bool)
			for _, author := range authors {
				foundEmails[author.Email] = true
			}

			for email := range tt.expectedEmails {
				if !foundEmails[email] {
					t.Errorf("Expected author with email %s not found", email)
				}
			}

			for email := range foundEmails {
				if !tt.expectedEmails[email] {
					t.Errorf("Unexpected author with email %s found", email)
				}
			}
		})
	}
}

// Test all three methods together in a comprehensive episode
func TestEpisode_AllAuthorMethods(t *testing.T) {
	author1 := createAuthor("Alice", "alice@example.com")
	author2 := createAuthor("Bob", "bob@example.com")
	author3 := createAuthor("Charlie", "charlie@example.com")

	episode := &Episode{
		ID: "comprehensive-episode",
		Commits: []git.Commit{
			createCommit("abc123", author1),
			createCommit("def456", author2),
		},
		Artifacts: []Artifact{
			{
				ID:        "a1",
				Number:    1,
				Type:      ArtifactIssue,
				Title:     "Issue 1",
				Author:    author2, // Same as commit author
				State:     "open",
				CreatedAt: time.Now(),
			},
			{
				ID:        "a2",
				Number:    2,
				Type:      ArtifactPullRequest,
				Title:     "PR 1",
				Author:    author3, // New author
				State:     "open",
				CreatedAt: time.Now(),
			},
		},
		Discussions: []Discussion{
			{
				ID:        "d1",
				Type:      DiscussionComment,
				Author:    author1, // Same as commit author
				Body:      "Comment",
				CreatedAt: time.Now(),
			},
			{
				ID:        "d2",
				Type:      DiscussionReview,
				Author:    author3, // Same as artifact author
				Body:      "Review",
				CreatedAt: time.Now(),
			},
		},
	}

	// Test commit authors
	commitAuthors := episode.GetCommitAuthors()
	if len(commitAuthors) != 2 {
		t.Errorf("GetCommitAuthors() returned %d authors, expected 2", len(commitAuthors))
	}

	// Test artifact authors
	artifactAuthors := episode.GetArtifactAuthors()
	if len(artifactAuthors) != 2 {
		t.Errorf("GetArtifactAuthors() returned %d authors, expected 2", len(artifactAuthors))
	}

	// Test discussion authors
	discussionAuthors := episode.GetDiscussionAuthors()
	if len(discussionAuthors) != 2 {
		t.Errorf("GetDiscussionAuthors() returned %d authors, expected 2", len(discussionAuthors))
	}

	// Verify that each method returns the correct unique authors
	commitEmails := make(map[string]bool)
	for _, author := range commitAuthors {
		commitEmails[author.Email] = true
	}
	if !commitEmails["alice@example.com"] || !commitEmails["bob@example.com"] {
		t.Error("GetCommitAuthors() missing expected authors")
	}

	artifactEmails := make(map[string]bool)
	for _, author := range artifactAuthors {
		artifactEmails[author.Email] = true
	}
	if !artifactEmails["bob@example.com"] || !artifactEmails["charlie@example.com"] {
		t.Error("GetArtifactAuthors() missing expected authors")
	}

	discussionEmails := make(map[string]bool)
	for _, author := range discussionAuthors {
		discussionEmails[author.Email] = true
	}
	if !discussionEmails["alice@example.com"] || !discussionEmails["charlie@example.com"] {
		t.Error("GetDiscussionAuthors() missing expected authors")
	}
}

func TestEpisode_GetDuration(t *testing.T) {
	author := createAuthor("Alice", "alice@example.com")

	// Create specific times for testing
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	oneHourLater := baseTime.Add(1 * time.Hour)
	oneDayLater := baseTime.Add(24 * time.Hour)
	oneWeekLater := baseTime.Add(7 * 24 * time.Hour)

	tests := []struct {
		name             string
		commits          []git.Commit
		expectedDuration time.Duration
		description      string
	}{
		{
			name:             "empty commits",
			commits:          []git.Commit{},
			expectedDuration: 0,
			description:      "should return zero duration for no commits",
		},
		{
			name: "single commit",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "Test",
					CommittedAt: baseTime,
				},
			},
			expectedDuration: 0,
			description:      "should return zero duration for single commit",
		},
		{
			name: "two commits one hour apart",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "First commit",
					CommittedAt: baseTime,
				},
				{
					Hash:        "def456",
					ShortHash:   "def456",
					Author:      author,
					Committer:   author,
					Message:     "Second commit",
					CommittedAt: oneHourLater,
				},
			},
			expectedDuration: 1 * time.Hour,
			description:      "should return one hour duration",
		},
		{
			name: "two commits one day apart",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "First commit",
					CommittedAt: baseTime,
				},
				{
					Hash:        "def456",
					ShortHash:   "def456",
					Author:      author,
					Committer:   author,
					Message:     "Second commit",
					CommittedAt: oneDayLater,
				},
			},
			expectedDuration: 24 * time.Hour,
			description:      "should return one day duration",
		},
		{
			name: "commits in reverse chronological order",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "Newer commit",
					CommittedAt: oneDayLater,
				},
				{
					Hash:        "def456",
					ShortHash:   "def456",
					Author:      author,
					Committer:   author,
					Message:     "Older commit",
					CommittedAt: baseTime,
				},
			},
			expectedDuration: 24 * time.Hour,
			description:      "should handle reverse chronological order",
		},
		{
			name: "multiple commits spanning a week",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "First commit",
					CommittedAt: baseTime,
				},
				{
					Hash:        "def456",
					ShortHash:   "def456",
					Author:      author,
					Committer:   author,
					Message:     "Middle commit",
					CommittedAt: baseTime.Add(3 * 24 * time.Hour),
				},
				{
					Hash:        "ghi789",
					ShortHash:   "ghi789",
					Author:      author,
					Committer:   author,
					Message:     "Last commit",
					CommittedAt: oneWeekLater,
				},
			},
			expectedDuration: 7 * 24 * time.Hour,
			description:      "should return one week duration",
		},
		{
			name: "commits with random order",
			commits: []git.Commit{
				{
					Hash:        "abc123",
					ShortHash:   "abc123",
					Author:      author,
					Committer:   author,
					Message:     "Middle commit",
					CommittedAt: baseTime.Add(12 * time.Hour),
				},
				{
					Hash:        "def456",
					ShortHash:   "def456",
					Author:      author,
					Committer:   author,
					Message:     "Last commit",
					CommittedAt: oneDayLater,
				},
				{
					Hash:        "ghi789",
					ShortHash:   "ghi789",
					Author:      author,
					Committer:   author,
					Message:     "First commit",
					CommittedAt: baseTime,
				},
			},
			expectedDuration: 24 * time.Hour,
			description:      "should find oldest and newest regardless of order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			episode := &Episode{
				ID:      "test-episode",
				Commits: tt.commits,
			}

			duration := episode.GetDuration()

			if duration != tt.expectedDuration {
				t.Errorf("GetDuration() returned %v, expected %v. %s",
					duration, tt.expectedDuration, tt.description)
			}
		})
	}
}
