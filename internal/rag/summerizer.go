package rag

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
)

/*
Episode 12 — Authentication Refactor
Commits:
- Refactor login flow to use JWT
- Remove legacy session auth middleware
- Add unit tests for token verification

PRs:
- Replace session-based auth with JWT

Issues:
- Migrate authentication system

Authors: anthony, gavin
Date range: 2023-04-02 → 2023-04-07
*/

// Transform episodes into EpisodeSummary objects
func BuildEpisodeSummary(episode *cluster.Episode) EpisodeSummary {
	if episode == nil {
		return EpisodeSummary{}
	}

	// Generate title from commits or artifacts
	title := generateTitle(episode)

	// Build summary text
	summary := buildSummaryText(episode)

	// Extract metadata using Episode methods
	authors := episode.GetAuthorNames()
	startDate, endDate := episode.GetDateRange()
	commitCount := len(episode.Commits)
	fileCount := episode.GetFileCount()

	return EpisodeSummary{
		EpisodeID:   episode.ID,
		Title:       title,
		Summary:     summary,
		StartDate:   startDate,
		EndDate:     endDate,
		Authors:     authors,
		CommitCount: commitCount,
		FileCount:   fileCount,
	}
}

// generateTitle creates a concise title for the episode
func generateTitle(episode *cluster.Episode) string {
	// Try to derive from commit messages
	if len(episode.Commits) > 0 {
		// Use the first commit's subject as the basis
		firstSubject := episode.Commits[0].MessageSubject
		if firstSubject != "" {
			return firstSubject
		}
	}

	// Try to derive from artifacts
	if len(episode.Artifacts) > 0 {
		firstArtifact := episode.Artifacts[0]
		if firstArtifact.Title != "" {
			return firstArtifact.Title
		}
	}

	// Fallback
	return fmt.Sprintf("Episode %s", episode.ID)
}

// buildSummaryText constructs a formatted summary of the episode
func buildSummaryText(episode *cluster.Episode) string {
	var parts []string

	// Add commits section
	if len(episode.Commits) > 0 {
		commitLines := []string{"Commits:"}
		for _, commit := range episode.Commits {
			subject := commit.MessageSubject
			if subject == "" {
				subject = commit.Message
			}
			commitLines = append(commitLines, fmt.Sprintf("- %s", subject))
		}
		parts = append(parts, strings.Join(commitLines, "\n"))
	}

	// Add artifacts by type
	prs := []cluster.Artifact{}
	mrs := []cluster.Artifact{}
	issues := []cluster.Artifact{}
	tickets := []cluster.Artifact{}

	for _, artifact := range episode.Artifacts {
		switch artifact.Type {
		case cluster.ArtifactPullRequest:
			prs = append(prs, artifact)
		case cluster.ArtifactMergeRequest:
			mrs = append(mrs, artifact)
		case cluster.ArtifactIssue:
			issues = append(issues, artifact)
		case cluster.ArtifactTicket:
			tickets = append(tickets, artifact)
		}
	}

	// Add PRs section
	if len(prs) > 0 {
		prLines := []string{"\nPRs:"}
		for _, pr := range prs {
			prLines = append(prLines, fmt.Sprintf("- %s", pr.Title))
		}
		parts = append(parts, strings.Join(prLines, "\n"))
	}

	// Add MRs section
	if len(mrs) > 0 {
		mrLines := []string{"\nMRs:"}
		for _, mr := range mrs {
			mrLines = append(mrLines, fmt.Sprintf("- %s", mr.Title))
		}
		parts = append(parts, strings.Join(mrLines, "\n"))
	}

	// Add Issues section
	if len(issues) > 0 {
		issueLines := []string{"\nIssues:"}
		for _, issue := range issues {
			issueLines = append(issueLines, fmt.Sprintf("- %s", issue.Title))
		}
		parts = append(parts, strings.Join(issueLines, "\n"))
	}

	// Add Tickets section
	if len(tickets) > 0 {
		ticketLines := []string{"\nTickets:"}
		for _, ticket := range tickets {
			ticketLines = append(ticketLines, fmt.Sprintf("- %s", ticket.Title))
		}
		parts = append(parts, strings.Join(ticketLines, "\n"))
	}

	// Add authors section
	authors := episode.GetAuthorNames()
	if len(authors) > 0 {
		sort.Strings(authors)
		parts = append(parts, fmt.Sprintf("\nAuthors: %s", strings.Join(authors, ", ")))
	}

	// Add date range section
	start, end := episode.GetDateRange()
	dateRange := formatDateRange(start, end)
	if dateRange != "" {
		parts = append(parts, fmt.Sprintf("Date range: %s", dateRange))
	}

	return strings.Join(parts, "\n")
}

// formatDateRange formats start and end times as a date range string
func formatDateRange(earliest, latest time.Time) string {
	// Format date range
	if earliest.IsZero() && latest.IsZero() {
		return ""
	}

	if earliest.IsZero() {
		return latest.Format("2006-01-02")
	}

	if latest.IsZero() || earliest.Equal(latest) {
		return earliest.Format("2006-01-02")
	}

	return fmt.Sprintf("%s → %s", earliest.Format("2006-01-02"), latest.Format("2006-01-02"))
}
