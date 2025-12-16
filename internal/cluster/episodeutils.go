package cluster

import (
	"time"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// GetCommitAuthors extracts unique authors from the episode's commits
func (e *Episode) GetCommitAuthors() []git.Author {
	// Discover unique authors from commits
	authorMap := make(map[string]git.Author)
	for _, commit := range e.Commits {
		key := commit.Author.Email
		if _, exists := authorMap[key]; !exists {
			authorMap[key] = commit.Author
		}
	}

	return authorMapToSlice(authorMap)
}

// GetDiscussionAuthors extracts unique authors from discussions within the episode's artifacts
func (e *Episode) GetDiscussionAuthors() []git.Author {
	// Discover unique authors from discussions nested in artifacts
	authorMap := make(map[string]git.Author)
	for _, artifact := range e.Artifacts {
		for _, discussion := range artifact.Discussions {
			key := discussion.Author.Email
			if _, exists := authorMap[key]; !exists {
				authorMap[key] = discussion.Author
			}
		}
	}

	return authorMapToSlice(authorMap)
}

// GetArtifactAuthors extracts unique authors from the episode's artifacts
func (e *Episode) GetArtifactAuthors() []git.Author {
	// Discover unique authors from artifacts
	authorMap := make(map[string]git.Author)
	for _, artifact := range e.Artifacts {
		key := artifact.Author.Email
		if _, exists := authorMap[key]; !exists {
			authorMap[key] = artifact.Author
		}
	}

	return authorMapToSlice(authorMap)
}

// Helper function to convert author map to slice
func authorMapToSlice(authorMap map[string]git.Author) []git.Author {
	authors := make([]git.Author, 0, len(authorMap))
	for _, author := range authorMap {
		authors = append(authors, author)
	}
	return authors
}

// GetDuration returns the time span from oldest to newest commit in the episode
// Returns zero duration if there are no commits or only one commit
func (e *Episode) GetDuration() time.Duration {
	if len(e.Commits) <= 1 {
		return 0
	}

	var oldest, newest time.Time

	for i, commit := range e.Commits {
		if i == 0 {
			oldest = commit.CommittedAt
			newest = commit.CommittedAt
		} else {
			if commit.CommittedAt.Before(oldest) {
				oldest = commit.CommittedAt
			}
			if commit.CommittedAt.After(newest) {
				newest = commit.CommittedAt
			}
		}
	}

	return newest.Sub(oldest)
}

// GetAuthorNames extracts unique author names from commits and artifacts
// Returns a sorted list of author names
func (e *Episode) GetAuthorNames() []string {
	authorMap := make(map[string]bool)

	// Collect from commits
	for _, commit := range e.Commits {
		if commit.Author.Name != "" {
			authorMap[commit.Author.Name] = true
		}
	}

	// Collect from artifacts
	for _, artifact := range e.Artifacts {
		if artifact.Author.Name != "" {
			authorMap[artifact.Author.Name] = true
		}
	}

	// Convert to slice
	authors := make([]string, 0, len(authorMap))
	for author := range authorMap {
		authors = append(authors, author)
	}

	return authors
}

// GetDateRange returns the start and end times for the episode
// Examines both commits and artifacts to find the earliest and latest timestamps
func (e *Episode) GetDateRange() (time.Time, time.Time) {
	var earliest, latest time.Time

	// Check commits
	for _, commit := range e.Commits {
		commitTime := commit.CommittedAt
		if commitTime.IsZero() {
			continue
		}
		if earliest.IsZero() || commitTime.Before(earliest) {
			earliest = commitTime
		}
		if latest.IsZero() || commitTime.After(latest) {
			latest = commitTime
		}
	}

	// Check artifacts
	for _, artifact := range e.Artifacts {
		createdAt := artifact.CreatedAt
		if !createdAt.IsZero() && (earliest.IsZero() || createdAt.Before(earliest)) {
			earliest = createdAt
		}

		updatedAt := artifact.UpdatedAt
		if !updatedAt.IsZero() && (latest.IsZero() || updatedAt.After(latest)) {
			latest = updatedAt
		}

		if artifact.ClosedAt != nil {
			closedAt := *artifact.ClosedAt
			if !closedAt.IsZero() && (latest.IsZero() || closedAt.After(latest)) {
				latest = closedAt
			}
		}

		if artifact.MergedAt != nil {
			mergedAt := *artifact.MergedAt
			if !mergedAt.IsZero() && (latest.IsZero() || mergedAt.After(latest)) {
				latest = mergedAt
			}
		}
	}

	return earliest, latest
}

// GetFileCount returns the total number of unique files changed across all commits
func (e *Episode) GetFileCount() int {
	fileSet := make(map[string]bool)

	for _, commit := range e.Commits {
		for _, diff := range commit.Diffs {
			// Use FilePath as the primary identifier
			if diff.FilePath != "" {
				fileSet[diff.FilePath] = true
			}
			// Also track old path for renames
			if diff.OldPath != "" && diff.OldPath != diff.FilePath {
				fileSet[diff.OldPath] = true
			}
		}
	}

	// Also count files from PR metadata if available
	for _, artifact := range e.Artifacts {
		if artifact.Type == ArtifactPullRequest || artifact.Type == ArtifactMergeRequest {
			// Use the ChangedFiles count from metadata as additional context
			// but don't double-count with the commit diffs
			if artifact.Metadata.ChangedFiles > len(fileSet) {
				// This handles cases where we have PR metadata but not full commit diffs
				return artifact.Metadata.ChangedFiles
			}
		}
	}

	return len(fileSet)
}
