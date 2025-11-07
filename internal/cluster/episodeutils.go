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

// GetDiscussionAuthors extracts unique authors from the episode's discussions
func (e *Episode) GetDiscussionAuthors() []git.Author {
	// Discover unique authors from discussions
	authorMap := make(map[string]git.Author)
	for _, discussion := range e.Discussions {
		key := discussion.Author.Email
		if _, exists := authorMap[key]; !exists {
			authorMap[key] = discussion.Author
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
