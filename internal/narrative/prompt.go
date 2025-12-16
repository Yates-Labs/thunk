package narrative

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/Yates-Labs/thunk/internal/rag"
)

var (
	ErrMissingTargetEpisode = errors.New("target episode required for episode-level narrative")
)

func AssemblePrompt(targetEpisode *cluster.Episode, contextChunks []rag.ContextChunk) (string, error) {
	if targetEpisode == nil {
		return "", ErrMissingTargetEpisode
	}

	// Sort context chunks by relevance score (highest first), even if already sorted.
	sorted := make([]rag.ContextChunk, len(contextChunks))
	copy(sorted, contextChunks)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })

	return assembleEpisodePrompt(targetEpisode, sorted), nil
}

func assembleEpisodePrompt(ep *cluster.Episode, contextChunks []rag.ContextChunk) string {
	var b strings.Builder

	b.WriteString("You are a technical writer specializing in software development narratives. ")
	b.WriteString("Your task is to generate a coherent, human-readable narrative that explains ")
	b.WriteString("what happened during this development episode and why it matters.\n\n")

	b.WriteString("# Episode to Summarize\n\n")
	b.WriteString(fmt.Sprintf("**Episode ID:** %s\n\n", ep.ID))

	start, end := getTimeRange(ep.Commits)
	authors := getUniqueAuthors(ep.Commits)

	b.WriteString(fmt.Sprintf("**Commits:** %d commits\n\n", len(ep.Commits)))
	b.WriteString(fmt.Sprintf("**Time Range:** %s to %s\n\n", formatDateOrNA(start), formatDateOrNA(end)))

	if len(authors) == 0 {
		b.WriteString("**Authors:** N/A\n\n")
	} else {
		b.WriteString(fmt.Sprintf("**Authors:** %s\n\n", strings.Join(authors, ", ")))
	}

	b.WriteString("**Commit Messages:**\n")
	if len(ep.Commits) == 0 {
		b.WriteString("- (none)\n\n")
	} else {
		for _, c := range ep.Commits {
			hash := c.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			b.WriteString(fmt.Sprintf("- %s %s (by %s)\n", hash, c.Message, c.Author.Name))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("**Related Artifacts:** %d items\n\n", len(ep.Artifacts)))
	if len(ep.Artifacts) == 0 {
		b.WriteString("- (none)\n\n")
	} else {
		for _, a := range ep.Artifacts {
			b.WriteString(fmt.Sprintf("- **%s #%d:** %s\n", a.Type, a.Number, a.Title))
			if a.Description != "" {
				desc := a.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				b.WriteString(fmt.Sprintf("  %s\n", desc))
			}
		}
		b.WriteString("\n")
	}

	if len(contextChunks) > 0 {
		b.WriteString("# Related Development Context\n\n")
		b.WriteString("The following are similar episodes from the repository history that may provide useful context:\n\n")

		for _, ch := range contextChunks {
			b.WriteString(fmt.Sprintf("**Episode %s** (relevance: %.2f)\n", ch.EpisodeID, ch.Score))
			b.WriteString(ch.Text + "\n\n")
		}
	}

	b.WriteString("# Task\n\n")
	b.WriteString("Generate a narrative summary (2-4 paragraphs) that:\n")
	b.WriteString("1. Explains what was accomplished in this episode\n")
	b.WriteString("2. Describes the technical approach and key decisions\n")
	b.WriteString("3. Connects this work to related development efforts\n")
	b.WriteString("4. Highlights the impact and significance of the changes\n\n")
	b.WriteString("Write in past tense, use clear technical language, and focus on the 'why' behind the changes, not just the 'what'. ")
	b.WriteString("Do not invent details or motivations; base all statements strictly on the episode data and provided context. ")
	b.WriteString("Use related episodes only for background and connections, not as actions performed in this episode. ")
	b.WriteString("Explain technical decisions and tradeoffs rather than restating commit messages verbatim.\n")

	return b.String()
}

func getTimeRange(commits []git.Commit) (time.Time, time.Time) {
	if len(commits) == 0 {
		return time.Time{}, time.Time{}
	}
	start := commits[0].CommittedAt
	end := commits[0].CommittedAt
	for _, c := range commits[1:] {
		if c.CommittedAt.Before(start) {
			start = c.CommittedAt
		}
		if c.CommittedAt.After(end) {
			end = c.CommittedAt
		}
	}
	return start, end
}

func getUniqueAuthors(commits []git.Commit) []string {
	seen := make(map[string]struct{})
	var authors []string

	for _, c := range commits {
		name := strings.TrimSpace(c.Author.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		authors = append(authors, name)
	}

	sort.Strings(authors)
	return authors
}

func formatDateOrNA(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("2006-01-02")
}
