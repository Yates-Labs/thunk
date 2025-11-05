package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/Yates-Labs/thunk/internal/ingest/github"
)

func main() {
	// === Git Example ===
	fmt.Println("=== Git Repository Analysis ===")
	repo, err := git.CloneRepository("https://github.com/Yates-Labs/thunk")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Parse repository data (get 5 commits, no patches)
	repoData, err := git.ParseRepository(repo, "https://github.com/Yates-Labs/thunk", 5, false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display basic info
	fmt.Printf("Repository: %s\n", repoData.URL)
	fmt.Printf("Branch: %s\n", repoData.HeadBranch)
	fmt.Printf("Commits: %d\n\n", len(repoData.Commits))

	// Show recent commits
	for _, commit := range repoData.Commits {
		fmt.Printf("%s - %s by %s\n",
			commit.ShortHash,
			commit.MessageSubject,
			commit.Author.Name)
	}

	// === GitHub Example ===
	fmt.Println("\n=== GitHub Issue/PR Analysis ===")
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Note: Set GITHUB_TOKEN environment variable to fetch GitHub data")
		return
	}

	client := github.NewClient(token)
	ctx := context.Background()

	// Fetch multiple issues
	fmt.Println("\nFetching issues...")
	for i := 1; i <= 3; i++ {
		issue, err := github.GetIssue(ctx, client, "Yates-Labs", "thunk", i)
		if err != nil {
			fmt.Printf("Issue #%d: Not found or error: %v\n", i, err)
			continue
		}

		fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("  Author: %s\n", issue.Author)
		fmt.Printf("  State: %s\n", issue.State)
		fmt.Printf("  Comments: %d\n", len(issue.Comments))
		if len(issue.Labels) > 0 {
			fmt.Printf("  Labels: %v\n", issue.Labels)
		}
	}

	// Fetch a pull request with details
	fmt.Println("\nFetching PR #1 details...")
	pr, err := github.GetPullRequest(ctx, client, "Yates-Labs", "thunk", 1)
	if err != nil {
		fmt.Printf("Error fetching PR: %v\n", err)
		return
	}

	fmt.Printf("\nPR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("  Author: %s\n", pr.Author)
	fmt.Printf("  State: %s\n", pr.State)
	fmt.Printf("  Base: %s → Head: %s\n", pr.BaseBranch, pr.HeadBranch)
	fmt.Printf("  Changes: +%d -%d (%d files)\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
	fmt.Printf("  Reviews: %d\n", len(pr.Reviews))
}
