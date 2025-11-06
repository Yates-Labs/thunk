package git

import (
	"testing"
	"time"
)

func TestCloneRepository(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	if repo == nil {
		t.Fatal("Repository is nil")
	}
}

func TestParseBranches(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	branches, err := ParseBranches(repo)
	if err != nil {
		t.Fatalf("Failed to parse branches: %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("No branches found")
	}

	// Check that at least one branch has HEAD
	hasHead := false
	for _, branch := range branches {
		if branch.IsHead {
			hasHead = true
		}
		// Verify branch structure
		if branch.Name == "" {
			t.Error("Branch has empty name")
		}
		if branch.Hash == "" {
			t.Error("Branch has empty hash")
		}
	}

	if !hasHead {
		t.Error("No branch marked as HEAD")
	}
}

func TestParseCommits(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Test without patches for speed
	commits, err := ParseCommits(repo, 5, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("No commits found")
	}

	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}

	// Verify commit structure
	for i, commit := range commits {
		if commit.Hash == "" {
			t.Errorf("Commit %d has empty hash", i)
		}
		if commit.ShortHash == "" || len(commit.ShortHash) != 8 {
			t.Errorf("Commit %d has invalid short hash: %s", i, commit.ShortHash)
		}
		if commit.Author.Name == "" {
			t.Errorf("Commit %d has empty author name", i)
		}
		if commit.Author.Email == "" {
			t.Errorf("Commit %d has empty author email", i)
		}
		if commit.Author.When.IsZero() {
			t.Errorf("Commit %d has zero timestamp", i)
		}
		if commit.CommittedAt.IsZero() {
			t.Errorf("Commit %d has zero committed at timestamp", i)
		}
		if commit.Message == "" {
			t.Errorf("Commit %d has empty message", i)
		}
		if commit.MessageSubject == "" {
			t.Errorf("Commit %d has empty message subject", i)
		}
		// Stats validation
		if commit.Stats.FilesChanged != len(commit.Diffs) {
			t.Errorf("Commit %d: stats files changed (%d) doesn't match diffs length (%d)",
				i, commit.Stats.FilesChanged, len(commit.Diffs))
		}
		if commit.Stats.NetChange != (commit.Stats.Additions - commit.Stats.Deletions) {
			t.Errorf("Commit %d: net change calculation incorrect", i)
		}
	}
}

func TestParseCommitWithPatch(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Parse commits with patches included
	commits, err := ParseCommits(repo, 2, true)
	if err != nil {
		t.Fatalf("Failed to parse commits with patches: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("No commits found")
	}

	// Check that patches are included for non-binary files
	foundPatch := false
	for _, commit := range commits {
		for _, diff := range commit.Diffs {
			if !diff.IsBinary && diff.Patch != "" {
				foundPatch = true
				break
			}
		}
		if foundPatch {
			break
		}
	}

	// Note: It's possible all files are binary or no changes, so we just log this
	t.Logf("Found patch content: %v", foundPatch)
}

func TestParseCommitDiffs(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 5, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("No commits to test")
	}

	// Check that diffs are properly parsed
	for i, commit := range commits {
		t.Logf("Commit %d (%s) has %d diffs", i, commit.ShortHash, len(commit.Diffs))

		for j, diff := range commit.Diffs {
			if diff.FilePath == "" {
				t.Errorf("Commit %d, Diff %d has empty file path", i, j)
			}
			if diff.Status == "" {
				t.Errorf("Commit %d, Diff %d has empty status", i, j)
			}

			// Verify status is valid
			validStatuses := map[string]bool{
				"added":    true,
				"modified": true,
				"deleted":  true,
				"renamed":  true,
			}
			if !validStatuses[diff.Status] {
				t.Errorf("Commit %d, Diff %d has invalid status: %s", i, j, diff.Status)
			}

			// If renamed, old path should be set
			if diff.Status == "renamed" && diff.OldPath == "" {
				t.Errorf("Commit %d, Diff %d is renamed but has no old path", i, j)
			}

			// Verify file type is extracted
			if diff.FileType == "" && diff.FilePath != "" {
				t.Logf("Commit %d, Diff %d: file %s has no extension (may be intentional)", i, j, diff.FilePath)
			}
		}
	}
}

func TestParseRepository(t *testing.T) {
	url := "https://github.com/Yates-Labs/thunk"
	repo, err := CloneRepository(url)
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	repoData, err := ParseRepository(repo, url, 10, false)
	if err != nil {
		t.Fatalf("Failed to parse repository: %v", err)
	}

	if repoData.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, repoData.URL)
	}

	if len(repoData.Branches) == 0 {
		t.Error("No branches parsed")
	}

	if len(repoData.Commits) == 0 {
		t.Error("No commits parsed")
	}

	if len(repoData.Commits) > 10 {
		t.Errorf("Expected at most 10 commits, got %d", len(repoData.Commits))
	}

	if repoData.HeadHash == "" {
		t.Error("HEAD hash is empty")
	}

	if repoData.HeadBranch == "" {
		t.Error("HEAD branch is empty")
	}

	if repoData.TotalCommits != len(repoData.Commits) {
		t.Errorf("Total commits (%d) doesn't match commits length (%d)",
			repoData.TotalCommits, len(repoData.Commits))
	}

	// Verify that commits have branch pointers assigned
	branchAssignments := 0
	for _, commit := range repoData.Commits {
		if commit.Branch != nil {
			branchAssignments++
			t.Logf("Commit %s is on branch %s", commit.ShortHash, commit.Branch.Name)
		}
	}
	t.Logf("Branch assignments: %d out of %d commits", branchAssignments, len(repoData.Commits))
}

func TestGetCommitsByAuthor(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 10, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) == 0 {
		t.Skip("No commits to filter")
	}

	// Use the first commit's author email
	testEmail := commits[0].Author.Email
	filtered := GetCommitsByAuthor(commits, testEmail)

	if len(filtered) == 0 {
		t.Errorf("Expected at least one commit by %s", testEmail)
	}

	// Verify all filtered commits are by the author
	for _, commit := range filtered {
		if commit.Author.Email != testEmail {
			t.Errorf("Filtered commit has wrong author: %s != %s",
				commit.Author.Email, testEmail)
		}
	}
}

func TestGetCommitsByDateRange(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 10, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) < 2 {
		t.Skip("Need at least 2 commits to test date range")
	}

	// Get date range from first and last commit
	start := commits[len(commits)-1].Author.When.Add(-24 * time.Hour)
	end := commits[0].Author.When.Add(24 * time.Hour)

	filtered := GetCommitsByDateRange(commits, start, end)

	// All commits should be within range
	if len(filtered) != len(commits) {
		t.Logf("Expected %d commits in range, got %d", len(commits), len(filtered))
	}

	// Verify all are within range
	for _, commit := range filtered {
		if commit.Author.When.Before(start) || commit.Author.When.After(end) {
			t.Errorf("Commit %s is outside date range", commit.ShortHash)
		}
	}
}

func TestGetFileHistory(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 10, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) == 0 {
		t.Skip("No commits to test")
	}

	// Find a file that was changed
	var testFile string
	for _, commit := range commits {
		if len(commit.Diffs) > 0 {
			testFile = commit.Diffs[0].FilePath
			break
		}
	}

	if testFile == "" {
		t.Skip("No file changes found")
	}

	history := GetFileHistory(commits, testFile)

	if len(history) == 0 {
		t.Errorf("Expected at least one commit for file %s", testFile)
	}

	// Verify all commits in history touched the file
	for _, commit := range history {
		found := false
		for _, diff := range commit.Diffs {
			if diff.FilePath == testFile || diff.OldPath == testFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Commit %s in history doesn't contain file %s",
				commit.ShortHash, testFile)
		}
	}
}

func TestGetContributorStats(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 10, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	if len(commits) == 0 {
		t.Skip("No commits to analyze")
	}

	stats := GetContributorStats(commits)

	if len(stats) == 0 {
		t.Error("No contributor stats generated")
	}

	// Verify stats integrity
	totalCommits := 0
	for author, stat := range stats {
		if stat.CommitCount <= 0 {
			t.Errorf("Author %s has invalid commit count: %d", author, stat.CommitCount)
		}
		totalCommits += stat.CommitCount

		t.Logf("Author: %s - Commits: %d, +%d -%d",
			author, stat.CommitCount, stat.Additions, stat.Deletions)
	}

	if totalCommits != len(commits) {
		t.Errorf("Total commits in stats (%d) doesn't match parsed commits (%d)",
			totalCommits, len(commits))
	}
}

func TestCommitMergeDetection(t *testing.T) {
	repo, err := CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	commits, err := ParseCommits(repo, 20, false)
	if err != nil {
		t.Fatalf("Failed to parse commits: %v", err)
	}

	// Log merge commits
	mergeCount := 0
	for _, commit := range commits {
		if commit.IsMerge {
			mergeCount++
			t.Logf("Merge commit found: %s - %s", commit.ShortHash, commit.MessageSubject)
			if len(commit.ParentHashes) <= 1 {
				t.Errorf("Merge commit %s has only %d parent(s)", commit.ShortHash, len(commit.ParentHashes))
			}
		}
	}

	t.Logf("Found %d merge commits out of %d total", mergeCount, len(commits))
}
