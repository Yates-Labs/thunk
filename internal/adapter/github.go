package adapter

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	githubmodel "github.com/Yates-Labs/thunk/internal/ingest/github"
)

// Common errors for adapter operations
var (
	ErrInvalidIssueType = errors.New("invalid issue type: expected *github.Issue")
	ErrInvalidPRType    = errors.New("invalid pull request type: expected *github.PullRequest")
)

// GitHubAdapter implements the Adapter interface for GitHub
type GitHubAdapter struct{}

// NewGitHubAdapter creates a new GitHub adapter instance
func NewGitHubAdapter() *GitHubAdapter {
	return &GitHubAdapter{}
}

// GetPlatform returns the GitHub platform identifier
func (a *GitHubAdapter) GetPlatform() cluster.SourcePlatform {
	return cluster.PlatformGitHub
}

// ConvertIssue converts a GitHub issue to a cluster.Artifact
func (a *GitHubAdapter) ConvertIssue(issue interface{}) (*cluster.Artifact, error) {
	ghIssue, ok := issue.(*githubmodel.Issue)
	if !ok {
		return nil, ErrInvalidIssueType
	}
	return convertGitHubIssue(ghIssue), nil
}

// ConvertPullRequest converts a GitHub pull request to a cluster.Artifact
func (a *GitHubAdapter) ConvertPullRequest(pr interface{}) (*cluster.Artifact, error) {
	ghPR, ok := pr.(*githubmodel.PullRequest)
	if !ok {
		return nil, ErrInvalidPRType
	}
	return convertGitHubPullRequest(ghPR), nil
}

// FetchArtifacts fetches all artifacts (issues and PRs) from GitHub
func (a *GitHubAdapter) FetchArtifacts(ctx context.Context, token, owner, repo string) ([]cluster.Artifact, error) {
	// Create GitHub client
	client := githubmodel.NewClient(token)

	var artifacts []cluster.Artifact

	fmt.Printf("Fetching issues from GitHub...\n")

	// Fetch all issues (this includes both issues and PRs in GitHub's API)
	ghIssues, err := githubmodel.ListAllIssues(ctx, client, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	fmt.Printf("Found %d issues/PRs, converting...\n", len(ghIssues))

	// Convert issues directly from the lightweight API objects
	for _, ghIssue := range ghIssues {
		if ghIssue.IsPullRequest() {
			// Skip PRs here - we'll get them from the PR endpoint
			continue
		}

		// Convert lightweight issue to our model
		issue := githubmodel.ParseIssue(ghIssue)

		// Convert to artifact using adapter
		artifact, err := a.ConvertIssue(issue)
		if err != nil {
			fmt.Printf("Warning: failed to convert issue #%d: %v\n", ghIssue.GetNumber(), err)
			continue
		}

		artifacts = append(artifacts, *artifact)
	}

	fmt.Printf("Fetching pull requests from GitHub...\n")

	// Fetch all pull requests with lightweight details
	ghPRs, err := githubmodel.ListAllPullRequests(ctx, client, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	fmt.Printf("Found %d PRs, converting...\n", len(ghPRs))

	for _, ghPR := range ghPRs {
		// Convert lightweight PR to our model
		pr := githubmodel.ParsePullRequest(ghPR)

		// Convert to artifact using adapter
		artifact, err := a.ConvertPullRequest(pr)
		if err != nil {
			fmt.Printf("Warning: failed to convert PR #%d: %v\n", ghPR.GetNumber(), err)
			continue
		}

		artifacts = append(artifacts, *artifact)
	}

	fmt.Printf("Successfully converted %d artifacts\n", len(artifacts))

	return artifacts, nil
}

// convertGitHubIssue converts a GitHub issue to a standardized cluster.Artifact
// convertGitHubIssue converts a GitHub issue to a standardized cluster.Artifact
func convertGitHubIssue(issue *githubmodel.Issue) *cluster.Artifact {
	artifact := &cluster.Artifact{
		ID:          fmt.Sprintf("issue-%d", issue.ID),
		Number:      issue.Number,
		Type:        cluster.ArtifactIssue,
		Title:       issue.Title,
		Description: issue.Body,
		State:       issue.State,
		Author: git.Author{
			Name:  issue.Author,
			Email: "", // GitHub API doesn't provide email in issue context
		},
		Assignees: issue.Assignees,
		Labels:    issue.Labels,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
		URL:       issue.HTMLURL,
	}

	// Convert discussions (comments)
	artifact.Discussions = make([]cluster.Discussion, 0, len(issue.Comments))
	for _, comment := range issue.Comments {
		artifact.Discussions = append(artifact.Discussions, convertGitHubComment(comment))
	}

	// Set metadata
	artifact.Metadata = cluster.ArtifactMetadata{
		RelatedArtifacts: extractRelatedArtifacts(issue.CrossReferences),
	}

	if issue.Milestone != nil {
		artifact.Metadata.Milestone = issue.Milestone.Title
		artifact.Metadata.DueDate = issue.Milestone.DueOn
	}

	return artifact
}

// convertGitHubPullRequest converts a GitHub pull request to a standardized cluster.Artifact
func convertGitHubPullRequest(pr *githubmodel.PullRequest) *cluster.Artifact {
	artifact := &cluster.Artifact{
		ID:          fmt.Sprintf("pr-%d", pr.ID),
		Number:      pr.Number,
		Type:        cluster.ArtifactPullRequest,
		Title:       pr.Title,
		Description: pr.Description,
		State:       normalizeState(pr.State, pr.Merged),
		Author: git.Author{
			Name:  pr.Author,
			Email: "", // GitHub API doesn't provide email in PR context
		},
		Assignees: pr.Assignees,
		Labels:    pr.Labels,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
		ClosedAt:  pr.ClosedAt,
		MergedAt:  pr.MergedAt,
		URL:       pr.HTMLURL,
	}

	// Convert all discussions (comments, review comments, and reviews)
	artifact.Discussions = make([]cluster.Discussion, 0,
		len(pr.Comments)+len(pr.ReviewComments)+len(pr.Reviews))

	// Add issue comments
	for _, comment := range pr.Comments {
		artifact.Discussions = append(artifact.Discussions, convertGitHubComment(comment))
	}

	// Add review comments
	for _, reviewComment := range pr.ReviewComments {
		artifact.Discussions = append(artifact.Discussions, convertGitHubReviewComment(reviewComment))
	}

	// Add reviews
	for _, review := range pr.Reviews {
		artifact.Discussions = append(artifact.Discussions, convertGitHubReview(review))
	}

	// Sort discussions by creation time
	sortDiscussions(artifact.Discussions)

	// Set PR-specific metadata
	artifact.Metadata = cluster.ArtifactMetadata{
		BaseBranch:       pr.BaseBranch,
		HeadBranch:       pr.HeadBranch,
		Additions:        pr.Additions,
		Deletions:        pr.Deletions,
		ChangedFiles:     pr.ChangedFiles,
		ReviewState:      determineReviewState(pr.Reviews),
		IsDraft:          pr.Draft,
		RelatedArtifacts: extractRelatedArtifacts(pr.CrossReferences),
	}

	if pr.Milestone != nil {
		artifact.Metadata.Milestone = pr.Milestone.Title
		artifact.Metadata.DueDate = pr.Milestone.DueOn
	}

	return artifact
}

// convertGitHubComment converts a GitHub issue comment to a cluster.Discussion
func convertGitHubComment(comment githubmodel.Comment) cluster.Discussion {
	return cluster.Discussion{
		ID:   fmt.Sprintf("comment-%d", comment.ID),
		Type: cluster.DiscussionComment,
		Author: git.Author{
			Name:  comment.Author,
			Email: "",
		},
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
		Reactions: convertGitHubReactions(comment.Reactions),
	}
}

// convertGitHubReviewComment converts a GitHub review comment to a cluster.Discussion
func convertGitHubReviewComment(comment githubmodel.ReviewComment) cluster.Discussion {
	discussion := cluster.Discussion{
		ID:   fmt.Sprintf("review-comment-%d", comment.ID),
		Type: cluster.DiscussionReviewThread,
		Author: git.Author{
			Name:  comment.Author,
			Email: "",
		},
		Body:       comment.Body,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
		FilePath:   comment.Path,
		LineNumber: comment.Line,
		CommitHash: comment.CommitID,
		Reactions:  convertGitHubReactions(comment.Reactions),
	}

	// Set thread context if this is a reply
	if comment.InReplyToID != 0 {
		discussion.ParentID = fmt.Sprintf("review-comment-%d", comment.InReplyToID)
		discussion.ThreadID = discussion.ParentID // Use parent as thread ID
	} else {
		discussion.ThreadID = discussion.ID // This starts a new thread
	}

	return discussion
}

// convertGitHubReview converts a GitHub review to a cluster.Discussion
func convertGitHubReview(review githubmodel.Review) cluster.Discussion {
	return cluster.Discussion{
		ID:   fmt.Sprintf("review-%d", review.ID),
		Type: cluster.DiscussionReview,
		Author: git.Author{
			Name:  review.Author,
			Email: "",
		},
		Body:        review.Body,
		CreatedAt:   review.SubmittedAt,
		UpdatedAt:   review.SubmittedAt,
		ReviewState: normalizeReviewState(review.State),
	}
}

// convertGitHubReactions converts GitHub reactions to cluster.Reactions
func convertGitHubReactions(reactions *githubmodel.Reactions) cluster.Reactions {
	if reactions == nil {
		return cluster.Reactions{}
	}

	return cluster.Reactions{
		ThumbsUp:   reactions.PlusOne,
		ThumbsDown: reactions.MinusOne,
		Laugh:      reactions.Laugh,
		Hooray:     reactions.Hooray,
		Confused:   reactions.Confused,
		Heart:      reactions.Heart,
		Rocket:     reactions.Rocket,
		Eyes:       reactions.Eyes,
		TotalCount: reactions.TotalCount,
	}
}

// extractRelatedArtifacts extracts related artifact IDs from cross-references
func extractRelatedArtifacts(crossRefs []githubmodel.CrossRef) []string {
	if len(crossRefs) == 0 {
		return nil
	}

	related := make([]string, 0, len(crossRefs))
	for _, ref := range crossRefs {
		var prefix string
		if ref.Type == "pull_request" {
			prefix = "pr"
		} else {
			prefix = "issue"
		}
		related = append(related, fmt.Sprintf("%s-%d", prefix, ref.Number))
	}
	return related
}

// normalizeState converts GitHub state to a normalized state
// For PRs, also considers merged status
func normalizeState(state string, merged bool) string {
	if merged {
		return "merged"
	}
	return state
}

// normalizeReviewState converts GitHub review states to normalized states
func normalizeReviewState(state string) string {
	switch state {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes_requested"
	case "COMMENTED":
		return "commented"
	case "DISMISSED":
		return "dismissed"
	default:
		return state
	}
}

// determineReviewState determines the overall review state from all reviews
// Priority: changes_requested > approved > commented
func determineReviewState(reviews []githubmodel.Review) string {
	if len(reviews) == 0 {
		return ""
	}

	hasApproval := false
	hasChangesRequested := false

	// Use the most recent review from each unique reviewer
	reviewerStates := make(map[string]string)
	for _, review := range reviews {
		// Later reviews override earlier ones
		reviewerStates[review.Author] = normalizeReviewState(review.State)
	}

	// Check the final states
	for _, state := range reviewerStates {
		switch state {
		case "changes_requested":
			hasChangesRequested = true
		case "approved":
			hasApproval = true
		}
	}

	if hasChangesRequested {
		return "changes_requested"
	}
	if hasApproval {
		return "approved"
	}
	return "commented"
}

// sortDiscussions sorts discussions by creation time
func sortDiscussions(discussions []cluster.Discussion) {
	// Simple bubble sort - small arrays so performance is fine
	n := len(discussions)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if discussions[j].CreatedAt.After(discussions[j+1].CreatedAt) {
				discussions[j], discussions[j+1] = discussions[j+1], discussions[j]
			}
		}
	}
}

// ConvertIssues is a convenience function to convert multiple issues
func ConvertIssues(issues []*githubmodel.Issue) []*cluster.Artifact {
	artifacts := make([]*cluster.Artifact, 0, len(issues))
	for _, issue := range issues {
		artifacts = append(artifacts, convertGitHubIssue(issue))
	}
	return artifacts
}

// ConvertPullRequests is a convenience function to convert multiple PRs
func ConvertPullRequests(prs []*githubmodel.PullRequest) []*cluster.Artifact {
	artifacts := make([]*cluster.Artifact, 0, len(prs))
	for _, pr := range prs {
		artifacts = append(artifacts, convertGitHubPullRequest(pr))
	}
	return artifacts
}

// ParseArtifactID parses an artifact ID and returns the type and number
// Format: "issue-123" or "pr-456"
func ParseArtifactID(id string) (artifactType string, number int, err error) {
	var numStr string
	if len(id) > 6 && id[:6] == "issue-" {
		artifactType = "issue"
		numStr = id[6:]
	} else if len(id) > 3 && id[:3] == "pr-" {
		artifactType = "pr"
		numStr = id[3:]
	} else {
		return "", 0, fmt.Errorf("invalid artifact ID format: %s", id)
	}

	number, err = strconv.Atoi(numStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid artifact number in ID %s: %w", id, err)
	}

	return artifactType, number, nil
}
