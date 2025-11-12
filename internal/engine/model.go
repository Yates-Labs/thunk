package engine

import (
	"time"
)

// SchemaVersion is the current version of the narrative/RAG data model.
const SchemaVersion = "v1"

// NarrativeScope identifies the level at which a narrative was generated.
type NarrativeScope string

const (
	ScopeCommit  NarrativeScope = "commit"
	ScopeEpisode NarrativeScope = "episode"
	// ScopeRepository NarrativeScope = "repository" // reserved for future use
)

// Section is a logical subdivision of a narrative (e.g. Context, Changes, Impact).
type Section struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Citation links a span of narrative text back to a concrete source (commit, artifact, file, etc.).
type Citation struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	URL        string `json:"url,omitempty"`
	SpanStart  int    `json:"span_start"`
	SpanEnd    int    `json:"span_end"`
	Evidence   string `json:"evidence,omitempty"`
}

// GenerationMetadata captures model configuration and token usage for a narrative.
type GenerationMetadata struct {
	Provider          string  `json:"provider"`
	Model             string  `json:"model"`
	ResponseID        string  `json:"response_id,omitempty"`
	SystemFingerprint string  `json:"system_fingerprint,omitempty"`
	PromptID          string  `json:"prompt_id,omitempty"`
	Temperature       float32 `json:"temperature,omitempty"`
	MaxTokens         int     `json:"max_tokens,omitempty"`
	InputTokens       int     `json:"input_tokens,omitempty"`
	OutputTokens      int     `json:"output_tokens,omitempty"`
	ReasoningTokens   int     `json:"reasoning_tokens,omitempty"`
	TotalTokens       int     `json:"total_tokens,omitempty"`
	LatencyMS         int     `json:"latency_ms,omitempty"`
}

// EmbeddingMetadata references an externally stored embedding vector.
type EmbeddingMetadata struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	EmbeddingID string  `json:"embedding_id"`
	Dimension   int     `json:"dimension"`
	Similarity  float32 `json:"similarity,omitempty"`
}

// Narrative contains generated explanatory text plus provenance and evaluation metadata.
type Narrative struct {
	ID          string         `json:"id"`
	Scope       NarrativeScope `json:"scope"`
	Subject     string         `json:"subject"`
	Content     string         `json:"content"`
	Sections    []Section      `json:"sections,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	Version     string         `json:"version"`
	ArtifactIDs []string       `json:"artifact_ids,omitempty"`
	CommitHash  string         `json:"commit_hash,omitempty"`
	EpisodeID   string         `json:"episode_id,omitempty"`

	Generation GenerationMetadata `json:"generation"`          // links to model usage; used by future generation audit tooling
	Embedding  *EmbeddingMetadata `json:"embedding,omitempty"` // populated by rag/embedder; references stored vector
	Citations  []Citation         `json:"citations,omitempty"` // grounding sources for RAG answers

	Tags []string `json:"tags,omitempty"`
}

// ChangeSummary is a semantic highlight for a single file change in a commit.
type ChangeSummary struct {
	FilePath  string `json:"file_path"`
	Summary   string `json:"summary,omitempty"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"`
}

// CommitSummary binds a commit hash to its narrative and aggregated statistics.
type CommitSummary struct {
	CommitHash   string          `json:"commit_hash"`
	ShortHash    string          `json:"short_hash"`
	Title        string          `json:"title"`
	Summary      string          `json:"summary"`
	KeyChanges   []ChangeSummary `json:"key_changes,omitempty"`
	FilesChanged int             `json:"files_changed,omitempty"`
	Additions    int             `json:"additions,omitempty"`
	Deletions    int             `json:"deletions,omitempty"`
	Authors      []string        `json:"authors,omitempty"`
	ArtifactRefs []string        `json:"artifact_refs,omitempty"`
	CommittedAt  time.Time       `json:"committed_at"`

	Narrative Narrative `json:"narrative"`
	Keywords  []string  `json:"keywords,omitempty"`

	IsMerge  bool `json:"is_merge,omitempty"`
	IsRevert bool `json:"is_revert,omitempty"`
	IsChore  bool `json:"is_chore,omitempty"`
}

// EpisodeSummary aggregates metrics and narrative for a cluster episode.
type EpisodeSummary struct {
	EpisodeID string        `json:"episode_id"`
	Title     string        `json:"title,omitempty"`
	Summary   string        `json:"summary"`
	Start     time.Time     `json:"start"`
	End       time.Time     `json:"end"`
	Duration  time.Duration `json:"duration"`

	CommitHashes []string `json:"commit_hashes"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`

	CommitCount int `json:"commit_count"`
	AuthorCount int `json:"author_count"`
	IssueCount  int `json:"issue_count,omitempty"`
	PRCount     int `json:"pr_count,omitempty"`

	TotalAdditions int `json:"total_additions,omitempty"`
	TotalDeletions int `json:"total_deletions,omitempty"`

	ReviewsApproved         int `json:"reviews_approved,omitempty"`
	ReviewsChangesRequested int `json:"reviews_changes_requested,omitempty"`
	ReviewsCommented        int `json:"reviews_commented,omitempty"`

	Narrative Narrative `json:"narrative"`
	Keywords  []string  `json:"keywords,omitempty"`
}
