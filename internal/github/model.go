package github

import "time"

// Issue represents GitHub issue data with discussions
// Optimized for narrative generation with full discussion context
type Issue struct {
	ID              int64           `json:"id"`
	Number          int             `json:"number"`
	Title           string          `json:"title"`
	Body            string          `json:"body"`
	State           string          `json:"state"`
	Author          string          `json:"author"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ClosedAt        *time.Time      `json:"closed_at,omitempty"`
	Labels          []string        `json:"labels"`
	Assignees       []string        `json:"assignees"`
	Milestone       *Milestone      `json:"milestone,omitempty"`
	Comments        []Comment       `json:"comments"`
	Timeline        []TimelineEvent `json:"timeline"`
	Reactions       *Reactions      `json:"reactions,omitempty"`
	URL             string          `json:"url"`
	HTMLURL         string          `json:"html_url"`
	CommentCount    int             `json:"comment_count"`
	CrossReferences []CrossRef      `json:"cross_references"`
}

// PullRequest represents GitHub pull request data with discussions
// Designed to capture complete code review context
type PullRequest struct {
	ID                  int64           `json:"id"`
	Number              int             `json:"number"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	State               string          `json:"state"`
	Author              string          `json:"author"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	MergedAt            *time.Time      `json:"merged_at,omitempty"`
	ClosedAt            *time.Time      `json:"closed_at,omitempty"`
	Labels              []string        `json:"labels"`
	Assignees           []string        `json:"assignees"`
	RequestedReviewers  []string        `json:"requested_reviewers"`
	Milestone           *Milestone      `json:"milestone,omitempty"`
	Comments            []Comment       `json:"comments"`
	ReviewComments      []ReviewComment `json:"review_comments"`
	Reviews             []Review        `json:"reviews"`
	Timeline            []TimelineEvent `json:"timeline"`
	BaseBranch          string          `json:"base_branch"`
	HeadBranch          string          `json:"head_branch"`
	Merged              bool            `json:"merged"`
	Mergeable           bool            `json:"mergeable"`
	Draft               bool            `json:"draft"`
	Additions           int             `json:"additions"`
	Deletions           int             `json:"deletions"`
	ChangedFiles        int             `json:"changed_files"`
	MergeCommitSHA      string          `json:"merge_commit_sha,omitempty"`
	MaintainerCanModify bool            `json:"maintainer_can_modify"`
	URL                 string          `json:"url"`
	HTMLURL             string          `json:"html_url"`
	CrossReferences     []CrossRef      `json:"cross_references"`
}

// Comment represents a comment on an issue or PR
type Comment struct {
	ID        int64      `json:"id"`
	Author    string     `json:"author"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Reactions *Reactions `json:"reactions,omitempty"`
	URL       string     `json:"url"`
	HTMLURL   string     `json:"html_url"`
}

// ReviewComment represents a code review comment on a PR
// Includes threading and location metadata for context reconstruction
type ReviewComment struct {
	ID                int64      `json:"id"`
	Author            string     `json:"author"`
	Body              string     `json:"body"`
	Path              string     `json:"path"`
	Line              int        `json:"line,omitempty"`
	StartLine         int        `json:"start_line,omitempty"`
	Side              string     `json:"side,omitempty"`
	StartSide         string     `json:"start_side,omitempty"`
	OriginalLine      int        `json:"original_line,omitempty"`
	OriginalStartLine int        `json:"original_start_line,omitempty"`
	InReplyToID       int64      `json:"in_reply_to_id,omitempty"`
	CommitID          string     `json:"commit_id"`
	OriginalCommitID  string     `json:"original_commit_id"`
	SubjectType       string     `json:"subject_type,omitempty"`
	DiffHunk          string     `json:"diff_hunk,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Reactions         *Reactions `json:"reactions,omitempty"`
	URL               string     `json:"url"`
	HTMLURL           string     `json:"html_url"`
}

// Review represents a PR review
type Review struct {
	ID          int64     `json:"id"`
	Author      string    `json:"author"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
	URL         string    `json:"url"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	DueOn       *time.Time `json:"due_on,omitempty"`
}

// TimelineEvent represents an event in the issue/PR timeline
type TimelineEvent struct {
	Event     string          `json:"event"`
	Actor     string          `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
	Source    *TimelineSource `json:"source,omitempty"`
}

// TimelineSource represents the source of a cross-reference
type TimelineSource struct {
	Type   string `json:"type"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// CrossRef represents a cross-reference between issues/PRs
// Extracted from timeline for easier access
type CrossRef struct {
	Type      string    `json:"type"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// Reactions represents GitHub reactions to content
type Reactions struct {
	TotalCount int `json:"total_count"`
	PlusOne    int `json:"plus_one"`
	MinusOne   int `json:"minus_one"`
	Laugh      int `json:"laugh"`
	Confused   int `json:"confused"`
	Heart      int `json:"heart"`
	Hooray     int `json:"hooray"`
	Rocket     int `json:"rocket"`
	Eyes       int `json:"eyes"`
}
