package cluster

import "time"

// SourcePlatform represents the origin platform of repository data
type SourcePlatform string

const (
	PlatformGit       SourcePlatform = "git"
	PlatformGitHub    SourcePlatform = "github"
	PlatformGitLab    SourcePlatform = "gitlab"
	PlatformBitbucket SourcePlatform = "bitbucket"
	PlatformLocal     SourcePlatform = "local"
)

// ArtifactType represents the type of development artifact
type ArtifactType string

const (
	ArtifactIssue        ArtifactType = "issue"
	ArtifactPullRequest  ArtifactType = "pull_request"
	ArtifactMergeRequest ArtifactType = "merge_request" // GitLab terminology
	ArtifactTicket       ArtifactType = "ticket"
)

// RepositoryActivity represents unified repository data across all platforms
// Designed to aggregate commits, artifacts, and discussions for narrative generation
type RepositoryActivity struct {
	Platform       SourcePlatform `json:"platform"`
	RepositoryURL  string         `json:"repository_url"`
	RepositoryName string         `json:"repository_name"`
	Owner          string         `json:"owner"`
	DefaultBranch  string         `json:"default_branch"`
	Commits        []Commit       `json:"commits"`
	Artifacts      []Artifact     `json:"artifacts"`
	FetchedAt      time.Time      `json:"fetched_at"`
}

// Commit represents a unified code commit across platforms
// Normalizes commit data from git, GitHub, GitLab, etc.
type Commit struct {
	Hash           string       `json:"hash"`
	ShortHash      string       `json:"short_hash"` // First 8 chars for display
	Author         Author       `json:"author"`
	Committer      Author       `json:"committer"`
	Message        string       `json:"message"`
	MessageSubject string       `json:"message_subject"` // First line of message
	MessageBody    string       `json:"message_body"`    // Rest of message
	CommittedAt    time.Time    `json:"committed_at"`
	ParentHashes   []string     `json:"parent_hashes"`
	IsMerge        bool         `json:"is_merge"`
	Branch         string       `json:"branch,omitempty"`
	FilesChanged   []FileChange `json:"files_changed"`
	Stats          CommitStats  `json:"stats"`
}

// Author represents commit or artifact author information
type Author struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Username  string    `json:"username,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// FileChange represents a file modification in a commit
type FileChange struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"` // For renames
	Status    string `json:"status"`             // added, modified, deleted, renamed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	IsBinary  bool   `json:"is_binary"`
	Patch     string `json:"patch,omitempty"` // Actual diff content (optional for large repos)
}

// CommitStats represents aggregate statistics for a commit
type CommitStats struct {
	FilesChanged int `json:"files_changed"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	NetChange    int `json:"net_change"` // Additions - Deletions
}

// Artifact represents unified development artifacts (issues, PRs, tickets)
// Normalizes issue/PR data across GitHub, GitLab, Bitbucket, etc.
type Artifact struct {
	ID          string           `json:"id"`
	Number      int              `json:"number"`
	Type        ArtifactType     `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	State       string           `json:"state"` // open, closed, merged
	Author      Author           `json:"author"`
	Assignees   []string         `json:"assignees"`
	Labels      []string         `json:"labels"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	ClosedAt    *time.Time       `json:"closed_at,omitempty"`
	MergedAt    *time.Time       `json:"merged_at,omitempty"`
	Discussions []Discussion     `json:"discussions"`
	Metadata    ArtifactMetadata `json:"metadata"`
	URL         string           `json:"url"`
}

// ArtifactMetadata contains type-specific metadata for artifacts
type ArtifactMetadata struct {
	// Pull Request / Merge Request specific
	BaseBranch   string `json:"base_branch,omitempty"`
	HeadBranch   string `json:"head_branch,omitempty"`
	Additions    int    `json:"additions,omitempty"`
	Deletions    int    `json:"deletions,omitempty"`
	ChangedFiles int    `json:"changed_files,omitempty"`
	ReviewState  string `json:"review_state,omitempty"`
	IsDraft      bool   `json:"is_draft,omitempty"`

	// Issue / Ticket specific
	Priority  string     `json:"priority,omitempty"`
	Milestone string     `json:"milestone,omitempty"`
	DueDate   *time.Time `json:"due_date,omitempty"`

	// Cross-references
	RelatedArtifacts []string `json:"related_artifacts,omitempty"`
}

// Discussion represents unified conversation threads
// Normalizes comments, reviews, and discussion threads across platforms
type Discussion struct {
	ID        string         `json:"id"`
	Type      DiscussionType `json:"type"`
	Author    Author         `json:"author"`
	Body      string         `json:"body"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Thread context
	ParentID string `json:"parent_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`

	// Code review specific
	FilePath    string `json:"file_path,omitempty"`
	LineNumber  int    `json:"line_number,omitempty"`
	CommitHash  string `json:"commit_hash,omitempty"`
	ReviewState string `json:"review_state,omitempty"` // approved, changes_requested, commented

	// Engagement
	Reactions Reactions `json:"reactions,omitempty"`
}

// DiscussionType represents the type of discussion entry
type DiscussionType string

const (
	DiscussionComment      DiscussionType = "comment"
	DiscussionReview       DiscussionType = "review"
	DiscussionReviewThread DiscussionType = "review_thread"
	DiscussionNote         DiscussionType = "note" // GitLab terminology
)

// Reactions represents engagement reactions on discussions
type Reactions struct {
	ThumbsUp   int `json:"thumbs_up"`
	ThumbsDown int `json:"thumbs_down"`
	Laugh      int `json:"laugh"`
	Hooray     int `json:"hooray"`
	Confused   int `json:"confused"`
	Heart      int `json:"heart"`
	Rocket     int `json:"rocket"`
	Eyes       int `json:"eyes"`
	TotalCount int `json:"total_count"`
}
