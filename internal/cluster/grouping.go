package cluster

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// GroupingConfig defines parameters for episode grouping heuristics
type GroupingConfig struct {
	// Maximum time gap between commits in the same episode
	MaxTimeGap time.Duration

	// Minimum number of commits to form an episode
	MinCommits int

	// Weight factors for similarity scoring (should sum to 1.0)
	TimeWeight     float64
	AuthorWeight   float64
	FileWeight     float64
	MessageWeight  float64
	ArtifactWeight float64

	// Similarity thresholds
	MinSimilarityScore float64 // Minimum score to group commits together
}

// DefaultGroupingConfig returns sensible default grouping parameters
func DefaultGroupingConfig() GroupingConfig {
	return GroupingConfig{
		MaxTimeGap:         24 * time.Hour, // 24 hours
		MinCommits:         1,
		TimeWeight:         0.3,
		AuthorWeight:       0.25,
		FileWeight:         0.25,
		MessageWeight:      0.1,
		ArtifactWeight:     0.1,
		MinSimilarityScore: 0.5,
	}
}

// GroupIntoEpisodes groups commits and artifacts into logical episodes
// using heuristics based on time, author, file paths, and artifact references
func (ra *RepositoryActivity) GroupIntoEpisodes(config GroupingConfig) []Episode {
	if len(ra.Commits) == 0 {
		return []Episode{}
	}

	// Sort commits by time (oldest first)
	commits := make([]git.Commit, len(ra.Commits))
	copy(commits, ra.Commits)
	sortCommitsByTime(commits)

	// Build artifact reference map for quick lookup
	artifactRefMap := buildArtifactReferenceMap(ra.Artifacts)

	var episodes []Episode
	var currentEpisode *Episode

	for i, commit := range commits {
		if currentEpisode == nil {
			// Start a new episode
			currentEpisode = &Episode{
				Commits: []git.Commit{commit},
			}
			// Check for artifact references
			addReferencedArtifacts(currentEpisode, commit, artifactRefMap, ra.Artifacts)
		} else {
			// Calculate similarity with current episode
			similarity := calculateEpisodeSimilarity(currentEpisode, commit, config)

			if similarity >= config.MinSimilarityScore {
				// Add to current episode
				currentEpisode.Commits = append(currentEpisode.Commits, commit)
				addReferencedArtifacts(currentEpisode, commit, artifactRefMap, ra.Artifacts)
			} else {
				// Finalize current episode if it meets minimum criteria
				if len(currentEpisode.Commits) >= config.MinCommits {
					currentEpisode.ID = fmt.Sprintf("E%d", len(episodes)+1)
					episodes = append(episodes, *currentEpisode)
				}

				// Start new episode
				currentEpisode = &Episode{
					Commits: []git.Commit{commit},
				}
				addReferencedArtifacts(currentEpisode, commit, artifactRefMap, ra.Artifacts)
			}
		}

		// Check if this is the last commit
		if i == len(commits)-1 && currentEpisode != nil {
			if len(currentEpisode.Commits) >= config.MinCommits {
				currentEpisode.ID = fmt.Sprintf("E%d", len(episodes)+1)
				episodes = append(episodes, *currentEpisode)
			}
		}
	}

	return episodes
}

// sortCommitsByTime sorts commits in chronological order (oldest first)
func sortCommitsByTime(commits []git.Commit) {
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].CommittedAt.Before(commits[j].CommittedAt)
	})
}

// calculateEpisodeSimilarity calculates how similar a commit is to an episode
func calculateEpisodeSimilarity(episode *Episode, commit git.Commit, config GroupingConfig) float64 {
	if len(episode.Commits) == 0 {
		return 0
	}

	lastCommit := episode.Commits[len(episode.Commits)-1]

	// Time similarity (inverse of time gap, normalized)
	timeScore := calculateTimeScore(lastCommit, commit, config.MaxTimeGap)

	// Author similarity
	authorScore := calculateAuthorScore(episode, commit)

	// File path overlap
	fileScore := calculateFileScore(episode, commit)

	// Commit message similarity
	messageScore := calculateMessageScore(episode, commit)

	// Artifact reference similarity
	artifactScore := calculateArtifactScore(episode, commit)

	// Weighted average
	totalScore := (timeScore * config.TimeWeight) +
		(authorScore * config.AuthorWeight) +
		(fileScore * config.FileWeight) +
		(messageScore * config.MessageWeight) +
		(artifactScore * config.ArtifactWeight)

	return totalScore
}

// calculateTimeScore returns 1.0 if within max gap, decays to 0 beyond that
func calculateTimeScore(lastCommit, commit git.Commit, maxGap time.Duration) float64 {
	timeDiff := commit.CommittedAt.Sub(lastCommit.CommittedAt)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	if timeDiff > maxGap {
		return 0
	}

	// Linear decay: 1.0 at 0 time diff, 0.0 at maxGap
	return 1.0 - (float64(timeDiff) / float64(maxGap))
}

// calculateAuthorScore returns 1.0 if author matches any in episode
func calculateAuthorScore(episode *Episode, commit git.Commit) float64 {
	for _, episodeCommit := range episode.Commits {
		if episodeCommit.Author.Email == commit.Author.Email {
			return 1.0
		}
	}
	return 0.0
}

// calculateFileScore calculates file path overlap using Jaccard similarity
func calculateFileScore(episode *Episode, commit git.Commit) float64 {
	// Collect all file paths from episode
	episodeFiles := make(map[string]bool)
	for _, episodeCommit := range episode.Commits {
		for _, diff := range episodeCommit.Diffs {
			episodeFiles[diff.FilePath] = true
			if diff.OldPath != "" {
				episodeFiles[diff.OldPath] = true
			}
		}
	}

	// Collect file paths from new commit
	commitFiles := make(map[string]bool)
	for _, diff := range commit.Diffs {
		commitFiles[diff.FilePath] = true
		if diff.OldPath != "" {
			commitFiles[diff.OldPath] = true
		}
	}

	if len(episodeFiles) == 0 || len(commitFiles) == 0 {
		return 0.0
	}

	// Calculate Jaccard similarity: intersection / union
	intersection := 0
	union := len(episodeFiles)

	for file := range commitFiles {
		if episodeFiles[file] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// calculateMessageScore looks for common keywords and patterns in commit messages
func calculateMessageScore(episode *Episode, commit git.Commit) float64 {
	// Extract keywords from new commit message
	commitKeywords := extractKeywords(commit.MessageSubject)
	if len(commitKeywords) == 0 {
		return 0.0
	}

	// Check for overlap with episode commit messages
	maxOverlap := 0.0
	for _, episodeCommit := range episode.Commits {
		episodeKeywords := extractKeywords(episodeCommit.MessageSubject)
		if len(episodeKeywords) == 0 {
			continue
		}

		// Calculate overlap
		overlap := 0
		for keyword := range commitKeywords {
			if episodeKeywords[keyword] {
				overlap++
			}
		}

		overlapScore := float64(overlap) / float64(len(commitKeywords))
		if overlapScore > maxOverlap {
			maxOverlap = overlapScore
		}
	}

	return maxOverlap
}

// extractKeywords extracts meaningful words from commit message
func extractKeywords(message string) map[string]bool {
	// Common stop words to ignore
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "in": true, "is": true,
		"it": true, "of": true, "on": true, "or": true, "that": true, "the": true,
		"to": true, "was": true, "will": true, "with": true,
	}

	message = strings.ToLower(message)
	words := regexp.MustCompile(`\w+`).FindAllString(message, -1)

	keywords := make(map[string]bool)
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords[word] = true
		}
	}

	return keywords
}

// calculateArtifactScore checks if commit references same artifacts as episode
func calculateArtifactScore(episode *Episode, commit git.Commit) float64 {
	if len(episode.Artifacts) == 0 {
		return 0.0
	}

	// Extract artifact references from commit message (e.g., #123, PR-456)
	commitRefs := extractArtifactReferences(commit.Message)
	if len(commitRefs) == 0 {
		return 0.0
	}

	// Check overlap with episode artifacts
	episodeRefs := make(map[string]bool)
	for _, artifact := range episode.Artifacts {
		episodeRefs[fmt.Sprintf("#%d", artifact.Number)] = true
		episodeRefs[artifact.ID] = true
	}

	overlap := 0
	for ref := range commitRefs {
		if episodeRefs[ref] {
			overlap++
		}
	}

	if len(commitRefs) == 0 {
		return 0.0
	}

	return float64(overlap) / float64(len(commitRefs))
}

// extractArtifactReferences finds artifact references in text (e.g., #12, PR-456)
func extractArtifactReferences(text string) map[string]bool {
	refs := make(map[string]bool)

	// Match patterns like #12, PR-456, issue-789, etc.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`#(\d+)`),
		regexp.MustCompile(`(?i)PR-?(\d+)`),
		regexp.MustCompile(`(?i)issue-?(\d+)`),
		regexp.MustCompile(`(?i)MR-?(\d+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 0 {
				refs[match[0]] = true
			}
		}
	}

	return refs
}

// buildArtifactReferenceMap creates a map of artifact references to artifacts
func buildArtifactReferenceMap(artifacts []Artifact) map[string]*Artifact {
	refMap := make(map[string]*Artifact)

	for i := range artifacts {
		artifact := &artifacts[i]
		refMap[fmt.Sprintf("#%d", artifact.Number)] = artifact
		refMap[artifact.ID] = artifact

		// Add common reference patterns
		refMap[fmt.Sprintf("PR-%d", artifact.Number)] = artifact
		refMap[fmt.Sprintf("issue-%d", artifact.Number)] = artifact
		refMap[fmt.Sprintf("MR-%d", artifact.Number)] = artifact
	}

	return refMap
}

// addReferencedArtifacts finds and adds artifacts referenced in commit message
func addReferencedArtifacts(episode *Episode, commit git.Commit, refMap map[string]*Artifact, allArtifacts []Artifact) {
	refs := extractArtifactReferences(commit.Message)

	// Track artifacts already in episode to avoid duplicates
	existingArtifacts := make(map[string]bool)
	for _, artifact := range episode.Artifacts {
		existingArtifacts[artifact.ID] = true
	}

	for ref := range refs {
		if artifact, exists := refMap[ref]; exists {
			if !existingArtifacts[artifact.ID] {
				episode.Artifacts = append(episode.Artifacts, *artifact)
				existingArtifacts[artifact.ID] = true
			}
		}
	}

	// Also check if commit hash is referenced in artifact discussions
	for i := range allArtifacts {
		artifact := &allArtifacts[i]
		if existingArtifacts[artifact.ID] {
			continue
		}

		for _, discussion := range artifact.Discussions {
			if discussion.CommitHash == commit.Hash {
				episode.Artifacts = append(episode.Artifacts, *artifact)
				existingArtifacts[artifact.ID] = true
				break
			}
		}
	}
}
