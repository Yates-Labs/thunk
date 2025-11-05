package main

import (
	"fmt"

	"github.com/Yates-Labs/thunk/internal/git"
)

func main() {
	// Clone a repository
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
}
