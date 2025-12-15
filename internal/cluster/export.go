package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// ExportFormat represents supported export formats
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
)

// EpisodeExport represents an episode with enrichment counts for export
type EpisodeExport struct {
	ID           string       `json:"id"`
	CommitCount  int          `json:"commit_count"`
	AuthorCount  int          `json:"author_count"`
	PRCount      int          `json:"pr_count"`
	IssueCount   int          `json:"issue_count"`
	StartDate    time.Time    `json:"start_date"`
	EndDate      time.Time    `json:"end_date"`
	Duration     string       `json:"duration"`
	Authors      []string     `json:"authors"`
	CommitHashes []string     `json:"commit_hashes"`
	Commits      []git.Commit `json:"commits"`
	Artifacts    []Artifact   `json:"artifacts"`
}

// ExportEpisodes exports episodes in JSON format
func ExportEpisodes(episodes []Episode, format string, writer io.Writer) error {
	exportFormat := ExportFormat(strings.ToLower(format))

	// Convert episodes to export format with enrichment
	exports := make([]EpisodeExport, len(episodes))
	for i, ep := range episodes {
		exports[i] = enrichEpisode(ep)
	}

	if exportFormat != FormatJSON {
		return fmt.Errorf("unsupported export format: %s (supported: json)", format)
	}

	return exportJSON(exports, writer)
}

// enrichEpisode converts an Episode to EpisodeExport with calculated enrichments
func enrichEpisode(ep Episode) EpisodeExport {
	authorNames := ep.GetAuthorNames()

	commitHashes := make([]string, len(ep.Commits))
	for i, commit := range ep.Commits {
		commitHashes[i] = commit.Hash
	}

	// Count artifacts by type
	prCount := 0
	issueCount := 0
	for _, artifact := range ep.Artifacts {
		switch artifact.Type {
		case ArtifactPullRequest, ArtifactMergeRequest:
			prCount++
		case ArtifactIssue, ArtifactTicket:
			issueCount++
		}
	}

	startDate, endDate := ep.GetDateRange()

	return EpisodeExport{
		ID:           ep.ID,
		CommitCount:  len(ep.Commits),
		AuthorCount:  len(authorNames),
		PRCount:      prCount,
		IssueCount:   issueCount,
		StartDate:    startDate,
		EndDate:      endDate,
		Duration:     ep.GetDuration().String(),
		Authors:      authorNames,
		CommitHashes: commitHashes,
		Commits:      ep.Commits,
		Artifacts:    ep.Artifacts,
	}
}

// exportJSON writes episodes as JSON
func exportJSON(exports []EpisodeExport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exports)
}
