package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// ExportFormat represents supported export formats
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
)

// EpisodeExport represents an episode with enrichment counts for export
type EpisodeExport struct {
	ID           string    `json:"id"`
	CommitCount  int       `json:"commit_count"`
	AuthorCount  int       `json:"author_count"`
	PRCount      int       `json:"pr_count"`
	IssueCount   int       `json:"issue_count"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Duration     string    `json:"duration"`
	Authors      []string  `json:"authors"`
	CommitHashes []string  `json:"commit_hashes"`
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
	authors := ep.GetCommitAuthors()
	authorNames := make([]string, len(authors))
	for i, author := range authors {
		authorNames[i] = author.Name
	}

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

	var startDate, endDate time.Time
	if len(ep.Commits) > 0 {
		startDate = ep.Commits[0].CommittedAt
		endDate = ep.Commits[len(ep.Commits)-1].CommittedAt
	}

	return EpisodeExport{
		ID:           ep.ID,
		CommitCount:  len(ep.Commits),
		AuthorCount:  len(authors),
		PRCount:      prCount,
		IssueCount:   issueCount,
		StartDate:    startDate,
		EndDate:      endDate,
		Duration:     ep.GetDuration().String(),
		Authors:      authorNames,
		CommitHashes: commitHashes,
	}
}

// exportJSON writes episodes as JSON
func exportJSON(exports []EpisodeExport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exports)
}
