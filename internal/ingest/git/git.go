package git

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"
)

// OpenRepository opens a Git repository from a local path
func OpenRepository(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

// CloneRepository clones a Git repository to memory
func CloneRepository(url string) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: url,
	})
}

// ParseBranches extracts all branches from a repository
func ParseBranches(repo *git.Repository) ([]Branch, error) {
	var branches []Branch

	// Get HEAD reference
	head, err := repo.Head()
	var headHash string
	if err == nil {
		headHash = head.Hash().String()
	}

	// Get all references
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branches = append(branches, Branch{
				Name:     ref.Name().Short(),
				Hash:     ref.Hash().String(),
				IsRemote: false,
				IsHead:   ref.Hash().String() == headHash,
			})
		} else if ref.Name().IsRemote() {
			branches = append(branches, Branch{
				Name:     ref.Name().Short(),
				Hash:     ref.Hash().String(),
				IsRemote: true,
				IsHead:   false,
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return branches, nil
}

// ParseAuthor converts go-git Signature to Author
func ParseAuthor(sig object.Signature) Author {
	return Author{
		Name:  sig.Name,
		Email: sig.Email,
		When:  sig.When,
	}
}

// getFileType extracts file extension for context
func getFileType(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// ParseCommitDiffs extracts diffs for a commit with detailed metadata
func ParseCommitDiffs(commit *object.Commit, includePatch bool) ([]Diff, error) {
	var diffs []Diff

	// Get parent commit for diff comparison
	parent, err := commit.Parents().Next()
	if err != nil {
		// First commit has no parent, get tree changes
		tree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get tree: %w", err)
		}

		// All files are added in first commit
		err = tree.Files().ForEach(func(file *object.File) error {
			isBinary, _ := file.IsBinary()
			content := ""
			if !isBinary && includePatch {
				content, _ = file.Contents()
			}

			lines := 0
			if !isBinary {
				c, _ := file.Contents()
				lines = strings.Count(c, "\n") + 1
			}

			diffs = append(diffs, Diff{
				FilePath:  file.Name,
				Status:    "added",
				Additions: lines,
				Deletions: 0,
				IsBinary:  isBinary,
				FileType:  getFileType(file.Name),
				Patch:     content,
			})
			return nil
		})

		return diffs, err
	}

	// Get diff with parent
	patch, err := parent.Patch(commit)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch: %w", err)
	}

	// Parse file patches
	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()

		diff := Diff{}

		if from == nil && to != nil {
			// File added
			diff.FilePath = to.Path()
			diff.Status = "added"
			diff.FileType = getFileType(to.Path())
		} else if from != nil && to == nil {
			// File deleted
			diff.FilePath = from.Path()
			diff.Status = "deleted"
			diff.FileType = getFileType(from.Path())
		} else if from != nil && to != nil {
			// File modified or renamed
			diff.FilePath = to.Path()
			diff.OldPath = from.Path()
			diff.FileType = getFileType(to.Path())
			if from.Path() != to.Path() {
				diff.Status = "renamed"
			} else {
				diff.Status = "modified"
			}
		}

		// Check if binary
		isBinary := filePatch.IsBinary()
		diff.IsBinary = isBinary

		// Count additions and deletions from chunks
		additions := 0
		deletions := 0
		patchText := ""

		for _, chunk := range filePatch.Chunks() {
			content := chunk.Content()
			if includePatch {
				patchText += content
			}

			// Count lines based on chunk type
			switch chunk.Type() {
			case 1: // Added
				additions += strings.Count(content, "\n")
			case 2: // Deleted
				deletions += strings.Count(content, "\n")
			}
		}

		diff.Additions = additions
		diff.Deletions = deletions
		if includePatch && !isBinary {
			diff.Patch = patchText
		}

		diffs = append(diffs, diff)
	}

	return diffs, nil
}

// parseCommitMessage splits commit message into subject and body
func parseCommitMessage(message string) (subject, body string) {
	lines := strings.SplitN(message, "\n", 2)
	subject = strings.TrimSpace(lines[0])
	if len(lines) > 1 {
		body = strings.TrimSpace(lines[1])
	}
	return
}

// ParseCommit converts a go-git Commit to our Commit struct with full metadata
func ParseCommit(commit *object.Commit, includePatch bool) (*Commit, error) {
	// Parse parent hashes
	parentHashes := make([]string, 0, commit.NumParents())
	err := commit.Parents().ForEach(func(parent *object.Commit) error {
		parentHashes = append(parentHashes, parent.Hash.String())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse parents: %w", err)
	}

	// Parse diffs
	diffs, err := ParseCommitDiffs(commit, includePatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diffs: %w", err)
	}

	// Calculate statistics
	stats := CommitStats{
		FilesChanged: len(diffs),
	}
	for _, diff := range diffs {
		stats.Additions += diff.Additions
		stats.Deletions += diff.Deletions
	}
	stats.NetChange = stats.Additions - stats.Deletions

	// Parse commit message
	subject, body := parseCommitMessage(commit.Message)

	return &Commit{
		Hash:           commit.Hash.String(),
		ShortHash:      commit.Hash.String()[:8],
		Author:         ParseAuthor(commit.Author),
		Committer:      ParseAuthor(commit.Committer),
		Message:        commit.Message,
		MessageSubject: subject,
		MessageBody:    body,
		CommittedAt:    commit.Committer.When,
		ParentHashes:   parentHashes,
		TreeHash:       commit.TreeHash.String(),
		Diffs:          diffs,
		Stats:          stats,
		IsMerge:        commit.NumParents() > 1,
		Branch:         nil, // Will be set by caller if needed
	}, nil
}

// ParseCommits extracts commits from a repository
// maxCommits: 0 for unlimited, >0 to limit
// includePatch: whether to include full diff patches (can be large)
func ParseCommits(repo *git.Repository, maxCommits int, includePatch bool) ([]Commit, error) {
	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit iterator
	commitIter, err := repo.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	commits := make([]Commit, 0)
	count := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		if maxCommits > 0 && count >= maxCommits {
			return fmt.Errorf("max commits reached")
		}

		commit, err := ParseCommit(c, includePatch)
		if err != nil {
			return fmt.Errorf("failed to parse commit %s: %w", c.Hash, err)
		}

		commits = append(commits, *commit)
		count++
		return nil
	})

	// "max commits reached" is not a real error
	if err != nil && err.Error() != "max commits reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

// ParseRepository extracts all metadata from a repository
// Optimized for narrative generation with configurable depth
func ParseRepository(repo *git.Repository, url string, maxCommits int, includePatch bool) (*Repository, error) {
	// Parse branches
	branches, err := ParseBranches(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse branches: %w", err)
	}

	// Parse commits
	commits, err := ParseCommits(repo, maxCommits, includePatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commits: %w", err)
	}

	// Get HEAD info
	head, err := repo.Head()
	var headHash, headBranch string
	if err == nil {
		headHash = head.Hash().String()
		headBranch = head.Name().Short()
	}

	// Associate commits with branches
	// Sort branches to prioritize main/master so shared history is attributed to them
	sort.Slice(branches, func(i, j int) bool {
		nameI := branches[i].Name
		nameJ := branches[j].Name

		// Prioritize main and master
		if nameI == "main" || nameI == "master" {
			return true
		}
		if nameJ == "main" || nameJ == "master" {
			return false
		}

		return nameI < nameJ
	})

	// Create a map of commit hash to branch for quick lookup
	commitToBranch := make(map[string]*Branch)
	for i := range branches {
		branch := &branches[i]

		// Get commit iterator for this branch
		commitIter, err := repo.Log(&git.LogOptions{
			From: plumbing.NewHash(branch.Hash),
		})
		if err != nil {
			continue
		}

		// Mark all commits in this branch
		commitIter.ForEach(func(c *object.Commit) error {
			commitHash := c.Hash.String()
			if _, exists := commitToBranch[commitHash]; !exists {
				commitToBranch[commitHash] = branch
			}
			return nil
		})
	}

	// Assign branch pointers to commits
	for i := range commits {
		if branch, exists := commitToBranch[commits[i].Hash]; exists {
			commits[i].Branch = branch
		}
	}

	return &Repository{
		URL:          url,
		Branches:     branches,
		Commits:      commits,
		HeadHash:     headHash,
		HeadBranch:   headBranch,
		TotalCommits: len(commits),
	}, nil
}

// GetCommitsByAuthor filters commits by author email
func GetCommitsByAuthor(commits []Commit, authorEmail string) []Commit {
	filtered := make([]Commit, 0)
	for _, commit := range commits {
		if commit.Author.Email == authorEmail {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

// GetCommitsByDateRange filters commits within a date range
func GetCommitsByDateRange(commits []Commit, start, end time.Time) []Commit {
	filtered := make([]Commit, 0)
	for _, commit := range commits {
		if commit.Author.When.After(start) && commit.Author.When.Before(end) {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

// GetFileHistory tracks all changes to a specific file
func GetFileHistory(commits []Commit, filePath string) []Commit {
	history := make([]Commit, 0)
	for _, commit := range commits {
		for _, diff := range commit.Diffs {
			if diff.FilePath == filePath || diff.OldPath == filePath {
				history = append(history, commit)
				break
			}
		}
	}
	return history
}

// GetContributorStats aggregates statistics by author
func GetContributorStats(commits []Commit) map[string]struct {
	CommitCount int
	Additions   int
	Deletions   int
} {
	stats := make(map[string]struct {
		CommitCount int
		Additions   int
		Deletions   int
	})

	for _, commit := range commits {
		key := fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email)
		entry := stats[key]
		entry.CommitCount++
		entry.Additions += commit.Stats.Additions
		entry.Deletions += commit.Stats.Deletions
		stats[key] = entry
	}

	return stats
}

// GetRemoteURL returns the URL for a given remote name (e.g., "origin")
// Returns empty string if remote doesn't exist
func GetRemoteURL(repo *git.Repository, remoteName string) string {
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return ""
	}

	config := remote.Config()
	if len(config.URLs) == 0 {
		return ""
	}

	return config.URLs[0]
}
