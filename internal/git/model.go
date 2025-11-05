package git

import "time"

// Author represents Git author/committer information
// Optimized for narrative generation with full contact and temporal data
type Author struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	When  time.Time `json:"when"`
}

// Diff represents a file change in a commit
// Includes detailed metadata for understanding code evolution
type Diff struct {
	FilePath  string `json:"file_path"`
	OldPath   string `json:"old_path,omitempty"` // For renames
	Status    string `json:"status"`             // "added", "modified", "deleted", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"` // Actual diff content (optional for large repos)
	IsBinary  bool   `json:"is_binary"`
	FileType  string `json:"file_type"` // Extension/language for context
}

// Commit represents a Git commit with full metadata
// Designed to capture the complete context of each change
type Commit struct {
	Hash           string      `json:"hash"`
	ShortHash      string      `json:"short_hash"` // First 8 chars for display
	Author         Author      `json:"author"`
	Committer      Author      `json:"committer"`
	Message        string      `json:"message"`
	MessageSubject string      `json:"message_subject"` // First line of message
	MessageBody    string      `json:"message_body"`    // Rest of message
	ParentHashes   []string    `json:"parent_hashes"`
	TreeHash       string      `json:"tree_hash"`
	Diffs          []Diff      `json:"diffs"`
	Stats          CommitStats `json:"stats"`
	IsMerge        bool        `json:"is_merge"`
}

// CommitStats represents aggregate statistics for a commit
type CommitStats struct {
	FilesChanged int `json:"files_changed"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	NetChange    int `json:"net_change"` // Additions - Deletions
}

// Branch represents a Git branch with metadata
type Branch struct {
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	IsRemote bool   `json:"is_remote"`
	IsHead   bool   `json:"is_head"`
}

// Repository represents a Git repository with parsed metadata
// Central data structure for narrative generation
type Repository struct {
	URL          string   `json:"url"`
	LocalPath    string   `json:"local_path,omitempty"`
	Branches     []Branch `json:"branches"`
	Commits      []Commit `json:"commits"`
	HeadHash     string   `json:"head_hash"`
	HeadBranch   string   `json:"head_branch"`
	TotalCommits int      `json:"total_commits"`
}
