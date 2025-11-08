package adapter

import (
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	githubmodel "github.com/Yates-Labs/thunk/internal/ingest/github"
)

// Sample GitHub API responses for testing

func createSampleIssue() *githubmodel.Issue {
	createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
	closedAt := time.Date(2024, 1, 17, 14, 0, 0, 0, time.UTC)

	return &githubmodel.Issue{
		ID:        12345,
		Number:    42,
		Title:     "Fix authentication bug",
		Body:      "Users are unable to log in with OAuth",
		State:     "closed",
		Author:    "alice",
		Labels:    []string{"bug", "security"},
		Assignees: []string{"bob", "charlie"},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ClosedAt:  &closedAt,
		HTMLURL:   "https://github.com/owner/repo/issues/42",
		Comments: []githubmodel.Comment{
			{
				ID:        1001,
				Author:    "bob",
				Body:      "I'll investigate this",
				CreatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
				Reactions: &githubmodel.Reactions{
					TotalCount: 2,
					PlusOne:    2,
				},
			},
			{
				ID:        1002,
				Author:    "charlie",
				Body:      "Fixed in PR #43",
				CreatedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
			},
		},
		Milestone: &githubmodel.Milestone{
			ID:     500,
			Number: 5,
			Title:  "v1.2.0",
			DueOn:  &closedAt,
		},
		CrossReferences: []githubmodel.CrossRef{
			{
				Type:   "pull_request",
				Number: 43,
				Title:  "Fix OAuth authentication",
				State:  "merged",
				URL:    "https://github.com/owner/repo/pull/43",
			},
		},
	}
}

func createSamplePullRequest() *githubmodel.PullRequest {
	createdAt := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 2, 3, 15, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2024, 2, 3, 15, 0, 0, 0, time.UTC)

	return &githubmodel.PullRequest{
		ID:                 67890,
		Number:             43,
		Title:              "Fix OAuth authentication",
		Description:        "This PR fixes the OAuth authentication bug reported in #42",
		State:              "closed",
		Author:             "bob",
		Labels:             []string{"bug", "security"},
		Assignees:          []string{"bob"},
		RequestedReviewers: []string{"alice"},
		BaseBranch:         "main",
		HeadBranch:         "fix/oauth-auth",
		Merged:             true,
		Draft:              false,
		Additions:          145,
		Deletions:          32,
		ChangedFiles:       5,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		MergedAt:           &mergedAt,
		HTMLURL:            "https://github.com/owner/repo/pull/43",
		Comments: []githubmodel.Comment{
			{
				ID:        2001,
				Author:    "alice",
				Body:      "Looks good overall, just a few minor comments",
				CreatedAt: time.Date(2024, 2, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 2, 2, 10, 0, 0, 0, time.UTC),
			},
		},
		ReviewComments: []githubmodel.ReviewComment{
			{
				ID:        3001,
				Author:    "alice",
				Body:      "Consider adding error handling here",
				Path:      "src/auth/oauth.go",
				Line:      42,
				CommitID:  "abc123",
				CreatedAt: time.Date(2024, 2, 2, 10, 30, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 2, 2, 10, 30, 0, 0, time.UTC),
				DiffHunk:  "@@ -40,6 +40,8 @@\n func authenticate() {\n+  // new code\n }",
			},
			{
				ID:          3002,
				Author:      "bob",
				Body:        "Good catch, I'll add that",
				Path:        "src/auth/oauth.go",
				Line:        42,
				CommitID:    "abc123",
				InReplyToID: 3001,
				CreatedAt:   time.Date(2024, 2, 2, 11, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2024, 2, 2, 11, 0, 0, 0, time.UTC),
			},
		},
		Reviews: []githubmodel.Review{
			{
				ID:          4001,
				Author:      "alice",
				Body:        "Approved after updates",
				State:       "APPROVED",
				SubmittedAt: time.Date(2024, 2, 3, 14, 0, 0, 0, time.UTC),
			},
		},
		Milestone: &githubmodel.Milestone{
			ID:     500,
			Number: 5,
			Title:  "v1.2.0",
		},
		CrossReferences: []githubmodel.CrossRef{
			{
				Type:   "issue",
				Number: 42,
				Title:  "Fix authentication bug",
				State:  "closed",
				URL:    "https://github.com/owner/repo/issues/42",
			},
		},
	}
}

// Test converting a GitHub issue to a cluster artifact
func TestConvertGitHubIssue(t *testing.T) {
	issue := createSampleIssue()
	artifact := convertGitHubIssue(issue)

	// Verify basic fields
	if artifact.ID != "issue-12345" {
		t.Errorf("Expected ID 'issue-12345', got '%s'", artifact.ID)
	}
	if artifact.Number != 42 {
		t.Errorf("Expected Number 42, got %d", artifact.Number)
	}
	if artifact.Type != cluster.ArtifactIssue {
		t.Errorf("Expected Type 'issue', got '%s'", artifact.Type)
	}
	if artifact.Title != "Fix authentication bug" {
		t.Errorf("Expected Title 'Fix authentication bug', got '%s'", artifact.Title)
	}
	if artifact.State != "closed" {
		t.Errorf("Expected State 'closed', got '%s'", artifact.State)
	}
	if artifact.Author.Name != "alice" {
		t.Errorf("Expected Author 'alice', got '%s'", artifact.Author.Name)
	}

	// Verify labels
	if len(artifact.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(artifact.Labels))
	}

	// Verify assignees
	if len(artifact.Assignees) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(artifact.Assignees))
	}

	// Verify discussions (comments)
	if len(artifact.Discussions) != 2 {
		t.Errorf("Expected 2 discussions, got %d", len(artifact.Discussions))
	}
	if artifact.Discussions[0].Type != cluster.DiscussionComment {
		t.Errorf("Expected discussion type 'comment', got '%s'", artifact.Discussions[0].Type)
	}
	if artifact.Discussions[0].Author.Name != "bob" {
		t.Errorf("Expected first comment author 'bob', got '%s'", artifact.Discussions[0].Author.Name)
	}
	if artifact.Discussions[0].Reactions.TotalCount != 2 {
		t.Errorf("Expected 2 reactions, got %d", artifact.Discussions[0].Reactions.TotalCount)
	}

	// Verify metadata
	if artifact.Metadata.Milestone != "v1.2.0" {
		t.Errorf("Expected Milestone 'v1.2.0', got '%s'", artifact.Metadata.Milestone)
	}
	if len(artifact.Metadata.RelatedArtifacts) != 1 {
		t.Errorf("Expected 1 related artifact, got %d", len(artifact.Metadata.RelatedArtifacts))
	}
	if artifact.Metadata.RelatedArtifacts[0] != "pr-43" {
		t.Errorf("Expected related artifact 'pr-43', got '%s'", artifact.Metadata.RelatedArtifacts[0])
	}

	// Verify timestamps
	if artifact.ClosedAt == nil {
		t.Error("Expected ClosedAt to be set")
	}
}

// Test converting a GitHub pull request to a cluster artifact
func TestConvertGitHubPullRequest(t *testing.T) {
	pr := createSamplePullRequest()
	artifact := convertGitHubPullRequest(pr)

	// Verify basic fields
	if artifact.ID != "pr-67890" {
		t.Errorf("Expected ID 'pr-67890', got '%s'", artifact.ID)
	}
	if artifact.Number != 43 {
		t.Errorf("Expected Number 43, got %d", artifact.Number)
	}
	if artifact.Type != cluster.ArtifactPullRequest {
		t.Errorf("Expected Type 'pull_request', got '%s'", artifact.Type)
	}
	if artifact.Title != "Fix OAuth authentication" {
		t.Errorf("Expected Title 'Fix OAuth authentication', got '%s'", artifact.Title)
	}
	if artifact.State != "merged" {
		t.Errorf("Expected State 'merged', got '%s'", artifact.State)
	}
	if artifact.Author.Name != "bob" {
		t.Errorf("Expected Author 'bob', got '%s'", artifact.Author.Name)
	}

	// Verify PR-specific metadata
	if artifact.Metadata.BaseBranch != "main" {
		t.Errorf("Expected BaseBranch 'main', got '%s'", artifact.Metadata.BaseBranch)
	}
	if artifact.Metadata.HeadBranch != "fix/oauth-auth" {
		t.Errorf("Expected HeadBranch 'fix/oauth-auth', got '%s'", artifact.Metadata.HeadBranch)
	}
	if artifact.Metadata.Additions != 145 {
		t.Errorf("Expected Additions 145, got %d", artifact.Metadata.Additions)
	}
	if artifact.Metadata.Deletions != 32 {
		t.Errorf("Expected Deletions 32, got %d", artifact.Metadata.Deletions)
	}
	if artifact.Metadata.ChangedFiles != 5 {
		t.Errorf("Expected ChangedFiles 5, got %d", artifact.Metadata.ChangedFiles)
	}
	if artifact.Metadata.IsDraft {
		t.Error("Expected IsDraft to be false")
	}

	// Verify discussions include all types (comments, review comments, reviews)
	expectedDiscussions := 4 // 1 comment + 2 review comments + 1 review
	if len(artifact.Discussions) != expectedDiscussions {
		t.Errorf("Expected %d discussions, got %d", expectedDiscussions, len(artifact.Discussions))
	}

	// Find and verify the review comment thread
	var foundReviewComment bool
	var foundReviewReply bool
	for _, d := range artifact.Discussions {
		if d.Type == cluster.DiscussionReviewThread {
			if d.FilePath == "src/auth/oauth.go" && d.LineNumber == 42 {
				foundReviewComment = true
				if d.ParentID == "" {
					// This is the parent comment
					if d.ThreadID != d.ID {
						t.Errorf("Expected ThreadID to equal ID for parent comment")
					}
				} else {
					// This is a reply
					foundReviewReply = true
					if d.ParentID != "review-comment-3001" {
						t.Errorf("Expected ParentID 'review-comment-3001', got '%s'", d.ParentID)
					}
				}
			}
		}
	}
	if !foundReviewComment {
		t.Error("Expected to find review comment on src/auth/oauth.go:42")
	}
	if !foundReviewReply {
		t.Error("Expected to find review comment reply")
	}

	// Verify review state
	if artifact.Metadata.ReviewState != "approved" {
		t.Errorf("Expected ReviewState 'approved', got '%s'", artifact.Metadata.ReviewState)
	}

	// Verify related artifacts
	if len(artifact.Metadata.RelatedArtifacts) != 1 {
		t.Errorf("Expected 1 related artifact, got %d", len(artifact.Metadata.RelatedArtifacts))
	}
	if artifact.Metadata.RelatedArtifacts[0] != "issue-42" {
		t.Errorf("Expected related artifact 'issue-42', got '%s'", artifact.Metadata.RelatedArtifacts[0])
	}

	// Verify merged timestamp
	if artifact.MergedAt == nil {
		t.Error("Expected MergedAt to be set")
	}
}

// Test the adapter interface implementation
func TestGitHubAdapter_ConvertIssue(t *testing.T) {
	adapter := NewGitHubAdapter()
	issue := createSampleIssue()

	artifact, err := adapter.ConvertIssue(issue)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if artifact.Number != 42 {
		t.Errorf("Expected Number 42, got %d", artifact.Number)
	}
}

func TestGitHubAdapter_ConvertPullRequest(t *testing.T) {
	adapter := NewGitHubAdapter()
	pr := createSamplePullRequest()

	artifact, err := adapter.ConvertPullRequest(pr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if artifact.Number != 43 {
		t.Errorf("Expected Number 43, got %d", artifact.Number)
	}
}

func TestGitHubAdapter_GetPlatform(t *testing.T) {
	adapter := NewGitHubAdapter()
	if adapter.GetPlatform() != cluster.PlatformGitHub {
		t.Errorf("Expected platform 'github', got '%s'", adapter.GetPlatform())
	}
}

func TestGitHubAdapter_InvalidIssueType(t *testing.T) {
	adapter := NewGitHubAdapter()
	_, err := adapter.ConvertIssue("invalid")
	if err != ErrInvalidIssueType {
		t.Errorf("Expected ErrInvalidIssueType, got %v", err)
	}
}

func TestGitHubAdapter_InvalidPRType(t *testing.T) {
	adapter := NewGitHubAdapter()
	_, err := adapter.ConvertPullRequest("invalid")
	if err != ErrInvalidPRType {
		t.Errorf("Expected ErrInvalidPRType, got %v", err)
	}
}

// Test review state determination
func TestDetermineReviewState(t *testing.T) {
	tests := []struct {
		name     string
		reviews  []githubmodel.Review
		expected string
	}{
		{
			name:     "No reviews",
			reviews:  []githubmodel.Review{},
			expected: "",
		},
		{
			name: "Single approval",
			reviews: []githubmodel.Review{
				{ID: 1, Author: "alice", State: "APPROVED", SubmittedAt: time.Now()},
			},
			expected: "approved",
		},
		{
			name: "Single changes requested",
			reviews: []githubmodel.Review{
				{ID: 1, Author: "alice", State: "CHANGES_REQUESTED", SubmittedAt: time.Now()},
			},
			expected: "changes_requested",
		},
		{
			name: "Changes requested overrides approval",
			reviews: []githubmodel.Review{
				{ID: 1, Author: "alice", State: "APPROVED", SubmittedAt: time.Now()},
				{ID: 2, Author: "bob", State: "CHANGES_REQUESTED", SubmittedAt: time.Now()},
			},
			expected: "changes_requested",
		},
		{
			name: "Latest review from same reviewer wins",
			reviews: []githubmodel.Review{
				{ID: 1, Author: "alice", State: "CHANGES_REQUESTED", SubmittedAt: time.Now()},
				{ID: 2, Author: "alice", State: "APPROVED", SubmittedAt: time.Now().Add(time.Hour)},
			},
			expected: "approved",
		},
		{
			name: "Only comments",
			reviews: []githubmodel.Review{
				{ID: 1, Author: "alice", State: "COMMENTED", SubmittedAt: time.Now()},
			},
			expected: "commented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineReviewState(tt.reviews)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test state normalization
func TestNormalizeState(t *testing.T) {
	tests := []struct {
		state    string
		merged   bool
		expected string
	}{
		{"open", false, "open"},
		{"closed", false, "closed"},
		{"closed", true, "merged"},
		{"open", true, "merged"}, // Shouldn't happen but merged takes precedence
	}

	for _, tt := range tests {
		result := normalizeState(tt.state, tt.merged)
		if result != tt.expected {
			t.Errorf("normalizeState(%s, %v) = %s, expected %s",
				tt.state, tt.merged, result, tt.expected)
		}
	}
}

// Test review state normalization
func TestNormalizeReviewState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "changes_requested"},
		{"COMMENTED", "commented"},
		{"DISMISSED", "dismissed"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := normalizeReviewState(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeReviewState(%s) = %s, expected %s",
				tt.input, result, tt.expected)
		}
	}
}

// Test artifact ID parsing
func TestParseArtifactID(t *testing.T) {
	tests := []struct {
		id          string
		expectType  string
		expectNum   int
		expectError bool
	}{
		{"issue-42", "issue", 42, false},
		{"pr-123", "pr", 123, false},
		{"issue-1", "issue", 1, false},
		{"invalid", "", 0, true},
		{"issue-abc", "", 0, true},
		{"pr-", "", 0, true},
	}

	for _, tt := range tests {
		artType, num, err := ParseArtifactID(tt.id)
		if tt.expectError {
			if err == nil {
				t.Errorf("Expected error for ID '%s', got nil", tt.id)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for ID '%s': %v", tt.id, err)
			}
			if artType != tt.expectType {
				t.Errorf("Expected type '%s', got '%s'", tt.expectType, artType)
			}
			if num != tt.expectNum {
				t.Errorf("Expected number %d, got %d", tt.expectNum, num)
			}
		}
	}
}

// Test batch conversion functions
func TestConvertIssues(t *testing.T) {
	issues := []*githubmodel.Issue{
		createSampleIssue(),
		{
			ID:        99999,
			Number:    100,
			Title:     "Another issue",
			State:     "open",
			Author:    "dave",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	artifacts := ConvertIssues(issues)
	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Number != 42 {
		t.Errorf("Expected first artifact number 42, got %d", artifacts[0].Number)
	}
	if artifacts[1].Number != 100 {
		t.Errorf("Expected second artifact number 100, got %d", artifacts[1].Number)
	}
}

func TestConvertPullRequests(t *testing.T) {
	prs := []*githubmodel.PullRequest{
		createSamplePullRequest(),
		{
			ID:         11111,
			Number:     200,
			Title:      "Another PR",
			State:      "open",
			Author:     "eve",
			BaseBranch: "main",
			HeadBranch: "feature/new",
			Merged:     false,
			Draft:      true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	artifacts := ConvertPullRequests(prs)
	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Number != 43 {
		t.Errorf("Expected first artifact number 43, got %d", artifacts[0].Number)
	}
	if artifacts[1].Number != 200 {
		t.Errorf("Expected second artifact number 200, got %d", artifacts[1].Number)
	}
	if !artifacts[1].Metadata.IsDraft {
		t.Error("Expected second artifact to be draft")
	}
}

// Test discussion sorting
func TestSortDiscussions(t *testing.T) {
	now := time.Now()
	discussions := []cluster.Discussion{
		{ID: "3", CreatedAt: now.Add(2 * time.Hour)},
		{ID: "1", CreatedAt: now},
		{ID: "2", CreatedAt: now.Add(1 * time.Hour)},
	}

	sortDiscussions(discussions)

	if discussions[0].ID != "1" {
		t.Errorf("Expected first discussion ID '1', got '%s'", discussions[0].ID)
	}
	if discussions[1].ID != "2" {
		t.Errorf("Expected second discussion ID '2', got '%s'", discussions[1].ID)
	}
	if discussions[2].ID != "3" {
		t.Errorf("Expected third discussion ID '3', got '%s'", discussions[2].ID)
	}
}

// Test extracting related artifacts
func TestExtractRelatedArtifacts(t *testing.T) {
	crossRefs := []githubmodel.CrossRef{
		{Type: "issue", Number: 10, Title: "Issue 10"},
		{Type: "pull_request", Number: 20, Title: "PR 20"},
		{Type: "issue", Number: 30, Title: "Issue 30"},
	}

	related := extractRelatedArtifacts(crossRefs)
	if len(related) != 3 {
		t.Errorf("Expected 3 related artifacts, got %d", len(related))
	}
	if related[0] != "issue-10" {
		t.Errorf("Expected 'issue-10', got '%s'", related[0])
	}
	if related[1] != "pr-20" {
		t.Errorf("Expected 'pr-20', got '%s'", related[1])
	}
	if related[2] != "issue-30" {
		t.Errorf("Expected 'issue-30', got '%s'", related[2])
	}
}

// Test empty cross-references
func TestExtractRelatedArtifacts_Empty(t *testing.T) {
	related := extractRelatedArtifacts([]githubmodel.CrossRef{})
	if related != nil {
		t.Errorf("Expected nil for empty cross-references, got %v", related)
	}
}
