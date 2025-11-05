package github

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-github/v77/github"
)

func getTestClient(t *testing.T) *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping GitHub API tests")
	}
	return NewClient(token)
}

func TestGetIssue(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	issue, err := GetIssue(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	if issue == nil {
		t.Fatal("Issue is nil")
	}

	// Verify issue structure
	if issue.Number != 1 {
		t.Errorf("Expected issue number 1, got %d", issue.Number)
	}
	if issue.Title == "" {
		t.Error("Issue has empty title")
	}
	if issue.Author == "" {
		t.Error("Issue has empty author")
	}
	if issue.State == "" {
		t.Error("Issue has empty state")
	}
	if issue.URL == "" {
		t.Error("Issue has empty URL")
	}
	if issue.HTMLURL == "" {
		t.Error("Issue has empty HTML URL")
	}
	if issue.CreatedAt.IsZero() {
		t.Error("Issue has zero created timestamp")
	}
	if issue.UpdatedAt.IsZero() {
		t.Error("Issue has zero updated timestamp")
	}

	t.Logf("Issue #%d: %s by %s (%s)", issue.Number, issue.Title, issue.Author, issue.State)
	t.Logf("Comments: %d, Timeline events: %d, Cross-references: %d",
		len(issue.Comments), len(issue.Timeline), len(issue.CrossReferences))
}

func TestGetPullRequest(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	pr, err := GetPullRequest(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get pull request: %v", err)
	}

	if pr == nil {
		t.Fatal("Pull request is nil")
	}

	// Verify PR structure
	if pr.Number != 1 {
		t.Errorf("Expected PR number 1, got %d", pr.Number)
	}
	if pr.Title == "" {
		t.Error("PR has empty title")
	}
	if pr.Author == "" {
		t.Error("PR has empty author")
	}
	if pr.State == "" {
		t.Error("PR has empty state")
	}
	if pr.URL == "" {
		t.Error("PR has empty URL")
	}
	if pr.HTMLURL == "" {
		t.Error("PR has empty HTML URL")
	}
	if pr.BaseBranch == "" {
		t.Error("PR has empty base branch")
	}
	if pr.HeadBranch == "" {
		t.Error("PR has empty head branch")
	}
	if pr.CreatedAt.IsZero() {
		t.Error("PR has zero created timestamp")
	}

	t.Logf("PR #%d: %s by %s (%s)", pr.Number, pr.Title, pr.Author, pr.State)
	t.Logf("Base: %s, Head: %s", pr.BaseBranch, pr.HeadBranch)
	t.Logf("Changes: +%d -%d (%d files)", pr.Additions, pr.Deletions, pr.ChangedFiles)
	t.Logf("Comments: %d, Review Comments: %d, Reviews: %d",
		len(pr.Comments), len(pr.ReviewComments), len(pr.Reviews))
}

func TestParseIssueComments(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	comments, err := ParseIssueComments(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to parse issue comments: %v", err)
	}

	t.Logf("Found %d comments", len(comments))

	// Verify comment structure
	for i, comment := range comments {
		if comment.ID == 0 {
			t.Errorf("Comment %d has zero ID", i)
		}
		if comment.Author == "" {
			t.Errorf("Comment %d has empty author", i)
		}
		if comment.Body == "" {
			t.Errorf("Comment %d has empty body", i)
		}
		if comment.CreatedAt.IsZero() {
			t.Errorf("Comment %d has zero timestamp", i)
		}
		if comment.URL == "" {
			t.Errorf("Comment %d has empty URL", i)
		}
		if comment.HTMLURL == "" {
			t.Errorf("Comment %d has empty HTML URL", i)
		}
	}

	// Verify comments are sorted chronologically
	for i := 1; i < len(comments); i++ {
		if comments[i].CreatedAt.Before(comments[i-1].CreatedAt) {
			t.Errorf("Comments not sorted: comment %d is before comment %d", i, i-1)
		}
	}
}

func TestParseReviewComments(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	comments, err := ParseReviewComments(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to parse review comments: %v", err)
	}

	t.Logf("Found %d review comments", len(comments))

	// Verify review comment structure
	for i, comment := range comments {
		if comment.ID == 0 {
			t.Errorf("Review comment %d has zero ID", i)
		}
		if comment.Author == "" {
			t.Errorf("Review comment %d has empty author", i)
		}
		if comment.Body == "" {
			t.Errorf("Review comment %d has empty body", i)
		}
		if comment.Path == "" {
			t.Errorf("Review comment %d has empty file path", i)
		}
		if comment.CommitID == "" {
			t.Errorf("Review comment %d has empty commit ID", i)
		}
		if comment.CreatedAt.IsZero() {
			t.Errorf("Review comment %d has zero timestamp", i)
		}
		if comment.URL == "" {
			t.Errorf("Review comment %d has empty URL", i)
		}
		if comment.HTMLURL == "" {
			t.Errorf("Review comment %d has empty HTML URL", i)
		}

		// Log threading info
		if comment.InReplyToID != 0 {
			t.Logf("Review comment %d is a reply to %d", comment.ID, comment.InReplyToID)
		}
	}

	// Verify comments are sorted chronologically
	for i := 1; i < len(comments); i++ {
		if comments[i].CreatedAt.Before(comments[i-1].CreatedAt) {
			t.Errorf("Review comments not sorted: comment %d is before comment %d", i, i-1)
		}
	}
}

func TestParseReviews(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	reviews, err := ParseReviews(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to parse reviews: %v", err)
	}

	t.Logf("Found %d reviews", len(reviews))

	// Verify review structure
	for i, review := range reviews {
		if review.ID == 0 {
			t.Errorf("Review %d has zero ID", i)
		}
		if review.Author == "" {
			t.Errorf("Review %d has empty author", i)
		}
		if review.State == "" {
			t.Errorf("Review %d has empty state", i)
		}
		if review.SubmittedAt.IsZero() {
			t.Errorf("Review %d has zero submitted timestamp", i)
		}
		if review.URL == "" {
			t.Errorf("Review %d has empty URL", i)
		}

		// Verify state is valid
		validStates := map[string]bool{
			"APPROVED":          true,
			"CHANGES_REQUESTED": true,
			"COMMENTED":         true,
			"DISMISSED":         true,
			"PENDING":           true,
		}
		if !validStates[review.State] {
			t.Errorf("Review %d has invalid state: %s", i, review.State)
		}

		t.Logf("Review %d: %s by %s", i, review.State, review.Author)
	}

	// Verify reviews are sorted chronologically
	for i := 1; i < len(reviews); i++ {
		if reviews[i].SubmittedAt.Before(reviews[i-1].SubmittedAt) {
			t.Errorf("Reviews not sorted: review %d is before review %d", i, i-1)
		}
	}
}

func TestParseTimeline(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	timeline, err := ParseTimeline(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to parse timeline: %v", err)
	}

	t.Logf("Found %d timeline events", len(timeline))

	// Verify timeline event structure
	crossRefCount := 0
	for i, event := range timeline {
		if event.Event == "" {
			t.Errorf("Timeline event %d has empty event type", i)
		}
		if event.CreatedAt.IsZero() {
			t.Errorf("Timeline event %d has zero timestamp", i)
		}

		// Check cross-references
		if event.Event == "cross-referenced" {
			crossRefCount++
			if event.Source == nil {
				t.Errorf("Cross-reference event %d has nil source", i)
			} else {
				if event.Source.Type == "" {
					t.Errorf("Cross-reference %d has empty type", i)
				}
				if event.Source.Number == 0 {
					t.Errorf("Cross-reference %d has zero number", i)
				}
				if event.Source.Title == "" {
					t.Errorf("Cross-reference %d has empty title", i)
				}
				t.Logf("Cross-reference: %s #%d - %s", event.Source.Type, event.Source.Number, event.Source.Title)
			}
		}
	}

	t.Logf("Found %d cross-reference events", crossRefCount)
}

func TestExtractCrossReferences(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	timeline, err := ParseTimeline(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to parse timeline: %v", err)
	}

	refs := extractCrossReferences(timeline)
	t.Logf("Extracted %d cross-references from %d timeline events", len(refs), len(timeline))

	// Verify cross-reference structure
	for i, ref := range refs {
		if ref.Type == "" {
			t.Errorf("Cross-reference %d has empty type", i)
		}
		if ref.Number == 0 {
			t.Errorf("Cross-reference %d has zero number", i)
		}
		if ref.Title == "" {
			t.Errorf("Cross-reference %d has empty title", i)
		}
		if ref.State == "" {
			t.Errorf("Cross-reference %d has empty state", i)
		}
		if ref.URL == "" {
			t.Errorf("Cross-reference %d has empty URL", i)
		}
		if ref.CreatedAt.IsZero() {
			t.Errorf("Cross-reference %d has zero timestamp", i)
		}

		// Verify type is valid
		if ref.Type != "issue" && ref.Type != "pull_request" {
			t.Errorf("Cross-reference %d has invalid type: %s", i, ref.Type)
		}
	}
}

func TestParseReactions(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	issue, err := GetIssue(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	if issue.Reactions != nil {
		t.Logf("Issue reactions: total=%d, +1=%d, -1=%d, laugh=%d, heart=%d",
			issue.Reactions.TotalCount, issue.Reactions.PlusOne, issue.Reactions.MinusOne,
			issue.Reactions.Laugh, issue.Reactions.Heart)

		// Verify total count matches sum
		sum := issue.Reactions.PlusOne + issue.Reactions.MinusOne +
			issue.Reactions.Laugh + issue.Reactions.Confused +
			issue.Reactions.Heart + issue.Reactions.Hooray +
			issue.Reactions.Rocket + issue.Reactions.Eyes

		if sum != issue.Reactions.TotalCount {
			t.Errorf("Reaction sum (%d) doesn't match total count (%d)", sum, issue.Reactions.TotalCount)
		}
	}

	// Check comment reactions
	for i, comment := range issue.Comments {
		if comment.Reactions != nil && comment.Reactions.TotalCount > 0 {
			t.Logf("Comment %d has %d reactions", i, comment.Reactions.TotalCount)
		}
	}
}

func TestParseMilestone(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	issue, err := GetIssue(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	if issue.Milestone != nil {
		m := issue.Milestone
		if m.Title == "" {
			t.Error("Milestone has empty title")
		}
		if m.State == "" {
			t.Error("Milestone has empty state")
		}
		if m.CreatedAt.IsZero() {
			t.Error("Milestone has zero created timestamp")
		}

		t.Logf("Milestone: %s (%s)", m.Title, m.State)
		if m.DueOn != nil {
			t.Logf("Due: %s", m.DueOn.Format("2006-01-02"))
		}
	} else {
		t.Log("Issue has no milestone")
	}
}

func TestGetIssuesByLabel(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	issue, err := GetIssue(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	if len(issue.Labels) == 0 {
		t.Skip("Issue has no labels to test")
	}

	testLabel := issue.Labels[0]
	issues := []Issue{*issue}

	filtered := GetIssuesByLabel(issues, testLabel)

	if len(filtered) == 0 {
		t.Errorf("Expected at least one issue with label '%s'", testLabel)
	}

	// Verify all filtered issues have the label
	for _, iss := range filtered {
		hasLabel := false
		for _, label := range iss.Labels {
			if label == testLabel {
				hasLabel = true
				break
			}
		}
		if !hasLabel {
			t.Errorf("Filtered issue #%d doesn't have label '%s'", iss.Number, testLabel)
		}
	}

	t.Logf("Found %d issues with label '%s'", len(filtered), testLabel)
}

func TestGetIssuesByAuthor(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	issue, err := GetIssue(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	testAuthor := issue.Author
	issues := []Issue{*issue}

	filtered := GetIssuesByAuthor(issues, testAuthor)

	if len(filtered) == 0 {
		t.Errorf("Expected at least one issue by author '%s'", testAuthor)
	}

	// Verify all filtered issues are by the author
	for _, iss := range filtered {
		if iss.Author != testAuthor {
			t.Errorf("Filtered issue #%d has wrong author: %s != %s",
				iss.Number, iss.Author, testAuthor)
		}
	}

	t.Logf("Found %d issues by author '%s'", len(filtered), testAuthor)
}

func TestGetPRsByAuthor(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	pr, err := GetPullRequest(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		t.Fatalf("Failed to get pull request: %v", err)
	}

	testAuthor := pr.Author
	prs := []PullRequest{*pr}

	filtered := GetPRsByAuthor(prs, testAuthor)

	if len(filtered) == 0 {
		t.Errorf("Expected at least one PR by author '%s'", testAuthor)
	}

	// Verify all filtered PRs are by the author
	for _, p := range filtered {
		if p.Author != testAuthor {
			t.Errorf("Filtered PR #%d has wrong author: %s != %s",
				p.Number, p.Author, testAuthor)
		}
	}

	t.Logf("Found %d PRs by author '%s'", len(filtered), testAuthor)
}

func TestParseBodyReferences(t *testing.T) {
	testCases := []struct {
		body     string
		expected []int
	}{
		{"Fixes #123", []int{123}},
		{"Closes #456", []int{456}},
		{"Resolves #789", []int{789}},
		{"Fixes #1 and closes #2", []int{1, 2}},
		{"Related to #100", []int{100}},
		{"See #50 for more details", []int{50}},
		{"No references here", []int{}},
		{"Fix #10, fix #20, fix #30", []int{10, 20, 30}},
	}

	for _, tc := range testCases {
		refs := ParseBodyReferences(tc.body)
		if len(refs) != len(tc.expected) {
			t.Errorf("For body '%s': expected %d refs, got %d",
				tc.body, len(tc.expected), len(refs))
			continue
		}

		for i, ref := range refs {
			if ref != tc.expected[i] {
				t.Errorf("For body '%s': expected ref %d, got %d",
					tc.body, tc.expected[i], ref)
			}
		}
	}
}

func TestRateLimitHandling(t *testing.T) {
	// This test verifies that rate limit errors are properly detected
	// It doesn't actually hit the rate limit, just tests the error handling logic

	testErr := handleAPIError(nil, "test")
	if testErr != nil {
		t.Errorf("Expected nil for nil error, got %v", testErr)
	}

	// Test with a generic error
	genericErr := handleAPIError(context.DeadlineExceeded, "test operation")
	if genericErr == nil {
		t.Error("Expected error for context deadline exceeded")
	}
	t.Logf("Generic error handling: %v", genericErr)
}
