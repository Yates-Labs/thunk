package github

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/v77/github"
)

// NewClient creates a GitHub API client with authentication
// token: GitHub personal access token
func NewClient(token string) *github.Client {
	return github.NewClient(nil).WithAuthToken(token)
}

// GetIssue fetches a GitHub issue with all comments and timeline
func GetIssue(ctx context.Context, client *github.Client, owner, repo string, number int) (*Issue, error) {
	ghIssue, _, err := client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	issue := ParseIssue(ghIssue)

	// Get comments
	comments, err := ParseIssueComments(ctx, client, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	issue.Comments = comments
	issue.CommentCount = len(comments)

	// Get timeline
	timeline, err := ParseTimeline(ctx, client, owner, repo, number)
	if err != nil {
		timeline = []TimelineEvent{} // Don't fail on timeline errors
	}
	issue.Timeline = timeline
	issue.CrossReferences = extractCrossReferences(timeline)

	return issue, nil
}

// GetPullRequest fetches a GitHub pull request with all discussions
func GetPullRequest(ctx context.Context, client *github.Client, owner, repo string, number int) (*PullRequest, error) {
	ghPR, _, err := client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	pr := ParsePullRequest(ghPR)

	// Get issue comments
	comments, err := ParseIssueComments(ctx, client, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	pr.Comments = comments

	// Get review comments
	reviewComments, err := ParseReviewComments(ctx, client, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get review comments: %w", err)
	}
	pr.ReviewComments = reviewComments

	// Get reviews
	reviews, err := ParseReviews(ctx, client, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}
	pr.Reviews = reviews

	// Get timeline
	timeline, err := ParseTimeline(ctx, client, owner, repo, number)
	if err != nil {
		timeline = []TimelineEvent{}
	}
	pr.Timeline = timeline
	pr.CrossReferences = extractCrossReferences(timeline)

	return pr, nil
}

// ParseIssue converts a go-github Issue to our Issue struct
func ParseIssue(ghIssue *github.Issue) *Issue {
	issue := &Issue{
		ID:        ghIssue.GetID(),
		Number:    ghIssue.GetNumber(),
		Title:     ghIssue.GetTitle(),
		Body:      ghIssue.GetBody(),
		State:     ghIssue.GetState(),
		CreatedAt: ghIssue.GetCreatedAt().Time,
		UpdatedAt: ghIssue.GetUpdatedAt().Time,
		URL:       ghIssue.GetURL(),
		HTMLURL:   ghIssue.GetHTMLURL(),
	}

	if user := ghIssue.GetUser(); user != nil {
		issue.Author = user.GetLogin()
	}

	for _, label := range ghIssue.Labels {
		if label != nil {
			issue.Labels = append(issue.Labels, label.GetName())
		}
	}

	for _, assignee := range ghIssue.Assignees {
		if assignee != nil {
			issue.Assignees = append(issue.Assignees, assignee.GetLogin())
		}
	}

	if ghIssue.Milestone != nil {
		issue.Milestone = ParseMilestone(ghIssue.Milestone)
	}

	if ghIssue.Reactions != nil {
		issue.Reactions = ParseReactions(ghIssue.Reactions)
	}

	if ghIssue.ClosedAt != nil {
		closedAt := ghIssue.GetClosedAt().Time
		issue.ClosedAt = &closedAt
	}

	return issue
}

// ParsePullRequest converts a go-github PullRequest to our PullRequest struct
func ParsePullRequest(ghPR *github.PullRequest) *PullRequest {
	pr := &PullRequest{
		ID:                  ghPR.GetID(),
		Number:              ghPR.GetNumber(),
		Title:               ghPR.GetTitle(),
		Description:         ghPR.GetBody(),
		State:               ghPR.GetState(),
		CreatedAt:           ghPR.GetCreatedAt().Time,
		UpdatedAt:           ghPR.GetUpdatedAt().Time,
		URL:                 ghPR.GetURL(),
		HTMLURL:             ghPR.GetHTMLURL(),
		Merged:              ghPR.GetMerged(),
		Mergeable:           ghPR.GetMergeable(),
		Draft:               ghPR.GetDraft(),
		Additions:           ghPR.GetAdditions(),
		Deletions:           ghPR.GetDeletions(),
		ChangedFiles:        ghPR.GetChangedFiles(),
		MergeCommitSHA:      ghPR.GetMergeCommitSHA(),
		MaintainerCanModify: ghPR.GetMaintainerCanModify(),
	}

	if user := ghPR.GetUser(); user != nil {
		pr.Author = user.GetLogin()
	}

	if base := ghPR.GetBase(); base != nil {
		pr.BaseBranch = base.GetRef()
	}
	if head := ghPR.GetHead(); head != nil {
		pr.HeadBranch = head.GetRef()
	}

	if ghPR.MergedAt != nil {
		mergedAt := ghPR.GetMergedAt().Time
		pr.MergedAt = &mergedAt
	}
	if ghPR.ClosedAt != nil {
		closedAt := ghPR.GetClosedAt().Time
		pr.ClosedAt = &closedAt
	}

	for _, label := range ghPR.Labels {
		if label != nil {
			pr.Labels = append(pr.Labels, label.GetName())
		}
	}

	for _, assignee := range ghPR.Assignees {
		if assignee != nil {
			pr.Assignees = append(pr.Assignees, assignee.GetLogin())
		}
	}

	for _, reviewer := range ghPR.RequestedReviewers {
		if reviewer != nil {
			pr.RequestedReviewers = append(pr.RequestedReviewers, reviewer.GetLogin())
		}
	}

	if ghPR.Milestone != nil {
		pr.Milestone = ParseMilestone(ghPR.Milestone)
	}

	return pr
}

// ParseIssueComments fetches all comments for an issue/PR with pagination
func ParseIssueComments(ctx context.Context, client *github.Client, owner, repo string, number int) ([]Comment, error) {
	var allComments []Comment

	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := client.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, handleAPIError(err, "failed to list comments")
		}

		for _, comment := range comments {
			if comment != nil {
				allComments = append(allComments, ParseComment(comment))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	sortCommentsByTime(allComments)
	return allComments, nil
}

// ParseReviewComments fetches all review comments for a PR with pagination
func ParseReviewComments(ctx context.Context, client *github.Client, owner, repo string, number int) ([]ReviewComment, error) {
	var allComments []ReviewComment

	opts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := client.PullRequests.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, handleAPIError(err, "failed to list review comments")
		}

		for _, comment := range comments {
			if comment != nil {
				allComments = append(allComments, ParseReviewComment(comment))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	sortReviewCommentsByTime(allComments)
	return allComments, nil
}

// ParseReviews fetches all reviews for a PR with pagination
func ParseReviews(ctx context.Context, client *github.Client, owner, repo string, number int) ([]Review, error) {
	var allReviews []Review

	opts := &github.ListOptions{PerPage: 100}

	for {
		reviews, resp, err := client.PullRequests.ListReviews(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, handleAPIError(err, "failed to list reviews")
		}

		for _, review := range reviews {
			if review != nil {
				allReviews = append(allReviews, ParseReview(review))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	sortReviewsByTime(allReviews)
	return allReviews, nil
}

// ParseTimeline fetches timeline events for cross-references
func ParseTimeline(ctx context.Context, client *github.Client, owner, repo string, number int) ([]TimelineEvent, error) {
	var events []TimelineEvent

	opts := &github.ListOptions{PerPage: 100}

	for {
		timelineEvents, resp, err := client.Issues.ListIssueTimeline(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, handleAPIError(err, "failed to list timeline")
		}

		for _, event := range timelineEvents {
			if event != nil {
				events = append(events, ParseTimelineEvent(event))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return events, nil
}

// ParseComment converts a go-github IssueComment to our Comment struct
func ParseComment(ghComment *github.IssueComment) Comment {
	comment := Comment{
		ID:        ghComment.GetID(),
		Body:      ghComment.GetBody(),
		CreatedAt: ghComment.GetCreatedAt().Time,
		UpdatedAt: ghComment.GetUpdatedAt().Time,
		URL:       ghComment.GetURL(),
		HTMLURL:   ghComment.GetHTMLURL(),
	}

	if user := ghComment.GetUser(); user != nil {
		comment.Author = user.GetLogin()
	}

	if ghComment.Reactions != nil {
		comment.Reactions = ParseReactions(ghComment.Reactions)
	}

	return comment
}

// ParseReviewComment converts a go-github PullRequestComment to our ReviewComment struct
func ParseReviewComment(ghComment *github.PullRequestComment) ReviewComment {
	rc := ReviewComment{
		ID:                ghComment.GetID(),
		Body:              ghComment.GetBody(),
		Path:              ghComment.GetPath(),
		Line:              ghComment.GetLine(),
		StartLine:         ghComment.GetStartLine(),
		Side:              ghComment.GetSide(),
		StartSide:         ghComment.GetStartSide(),
		OriginalLine:      ghComment.GetOriginalLine(),
		OriginalStartLine: ghComment.GetOriginalStartLine(),
		CommitID:          ghComment.GetCommitID(),
		OriginalCommitID:  ghComment.GetOriginalCommitID(),
		SubjectType:       ghComment.GetSubjectType(),
		CreatedAt:         ghComment.GetCreatedAt().Time,
		UpdatedAt:         ghComment.GetUpdatedAt().Time,
		URL:               ghComment.GetURL(),
		HTMLURL:           ghComment.GetHTMLURL(),
		DiffHunk:          ghComment.GetDiffHunk(),
	}

	if ghComment.InReplyTo != nil {
		rc.InReplyToID = ghComment.GetInReplyTo()
	}

	if user := ghComment.GetUser(); user != nil {
		rc.Author = user.GetLogin()
	}

	if ghComment.Reactions != nil {
		rc.Reactions = ParseReactions(ghComment.Reactions)
	}

	return rc
}

// ParseReview converts a go-github PullRequestReview to our Review struct
func ParseReview(ghReview *github.PullRequestReview) Review {
	review := Review{
		ID:          ghReview.GetID(),
		Body:        ghReview.GetBody(),
		State:       ghReview.GetState(),
		SubmittedAt: ghReview.GetSubmittedAt().Time,
		URL:         ghReview.GetHTMLURL(),
	}

	if user := ghReview.GetUser(); user != nil {
		review.Author = user.GetLogin()
	}

	return review
}

// ParseMilestone converts a go-github Milestone to our Milestone struct
func ParseMilestone(ghMilestone *github.Milestone) *Milestone {
	if ghMilestone == nil {
		return nil
	}

	m := &Milestone{
		ID:          ghMilestone.GetID(),
		Number:      ghMilestone.GetNumber(),
		Title:       ghMilestone.GetTitle(),
		Description: ghMilestone.GetDescription(),
		State:       ghMilestone.GetState(),
		CreatedAt:   ghMilestone.GetCreatedAt().Time,
	}

	if ghMilestone.DueOn != nil {
		dueOn := ghMilestone.GetDueOn().Time
		m.DueOn = &dueOn
	}

	return m
}

// ParseReactions converts go-github Reactions to our Reactions struct
func ParseReactions(ghReactions *github.Reactions) *Reactions {
	if ghReactions == nil {
		return nil
	}

	return &Reactions{
		TotalCount: ghReactions.GetTotalCount(),
		PlusOne:    ghReactions.GetPlusOne(),
		MinusOne:   ghReactions.GetMinusOne(),
		Laugh:      ghReactions.GetLaugh(),
		Confused:   ghReactions.GetConfused(),
		Heart:      ghReactions.GetHeart(),
		Hooray:     ghReactions.GetHooray(),
		Rocket:     ghReactions.GetRocket(),
		Eyes:       ghReactions.GetEyes(),
	}
}

// ParseTimelineEvent converts a go-github Timeline to our TimelineEvent struct
func ParseTimelineEvent(ghEvent *github.Timeline) TimelineEvent {
	event := TimelineEvent{
		Event:     ghEvent.GetEvent(),
		CreatedAt: ghEvent.GetCreatedAt().Time,
	}

	if actor := ghEvent.GetActor(); actor != nil {
		event.Actor = actor.GetLogin()
	}

	if ghEvent.GetEvent() == "cross-referenced" && ghEvent.Source != nil {
		source := &TimelineSource{}

		if issue := ghEvent.Source.Issue; issue != nil {
			if issue.IsPullRequest() {
				source.Type = "pull_request"
			} else {
				source.Type = "issue"
			}
			source.Number = issue.GetNumber()
			source.Title = issue.GetTitle()
			source.State = issue.GetState()
			source.URL = issue.GetHTMLURL()
		}

		event.Source = source
	}

	return event
}

// extractCrossReferences extracts cross-references from timeline events
func extractCrossReferences(timeline []TimelineEvent) []CrossRef {
	var refs []CrossRef

	for _, event := range timeline {
		if event.Event == "cross-referenced" && event.Source != nil {
			refs = append(refs, CrossRef{
				Type:      event.Source.Type,
				Number:    event.Source.Number,
				Title:     event.Source.Title,
				State:     event.Source.State,
				URL:       event.Source.URL,
				CreatedAt: event.CreatedAt,
			})
		}
	}

	return refs
}

// handleAPIError wraps API errors with context and detects rate limiting
func handleAPIError(err error, msg string) error {
	if err == nil {
		return nil
	}

	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return fmt.Errorf("%s: hit primary rate limit (used %d of %d, resets at %v): %w",
			msg, rateLimitErr.Rate.Used, rateLimitErr.Rate.Limit, rateLimitErr.Rate.Reset.Time, err)
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		retryAfter := abuseErr.GetRetryAfter()
		return fmt.Errorf("%s: hit secondary rate limit (retry after %v): %w",
			msg, retryAfter, err)
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// sortCommentsByTime sorts comments by creation time, then ID
func sortCommentsByTime(comments []Comment) {
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID < comments[j].ID
		}
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
}

// sortReviewCommentsByTime sorts review comments by creation time, then ID
func sortReviewCommentsByTime(comments []ReviewComment) {
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID < comments[j].ID
		}
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
}

// sortReviewsByTime sorts reviews by submission time, then ID
func sortReviewsByTime(reviews []Review) {
	sort.Slice(reviews, func(i, j int) bool {
		if reviews[i].SubmittedAt.Equal(reviews[j].SubmittedAt) {
			return reviews[i].ID < reviews[j].ID
		}
		return reviews[i].SubmittedAt.Before(reviews[j].SubmittedAt)
	})
}

// GetIssuesByLabel filters issues by label
func GetIssuesByLabel(issues []Issue, label string) []Issue {
	filtered := make([]Issue, 0)
	for _, issue := range issues {
		for _, l := range issue.Labels {
			if l == label {
				filtered = append(filtered, issue)
				break
			}
		}
	}
	return filtered
}

// GetIssuesByAuthor filters issues by author
func GetIssuesByAuthor(issues []Issue, author string) []Issue {
	filtered := make([]Issue, 0)
	for _, issue := range issues {
		if issue.Author == author {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// GetPRsByAuthor filters pull requests by author
func GetPRsByAuthor(prs []PullRequest, author string) []PullRequest {
	filtered := make([]PullRequest, 0)
	for _, pr := range prs {
		if pr.Author == author {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

// ParseBodyReferences extracts issue/PR references from body text
// Looks for patterns like "Fixes #123", "Closes #456", etc.
func ParseBodyReferences(body string) []int {
	var refs []int
	patterns := []string{"fixes", "closes", "resolves", "fix", "close", "resolve", "related to", "see"}

	words := strings.Fields(strings.ToLower(body))
	for i, word := range words {
		for _, pattern := range patterns {
			if word == pattern && i+1 < len(words) {
				nextWord := words[i+1]
				if strings.HasPrefix(nextWord, "#") {
					var num int
					_, err := fmt.Sscanf(nextWord, "#%d", &num)
					if err == nil {
						refs = append(refs, num)
					}
				}
			}
		}
	}

	return refs
}
