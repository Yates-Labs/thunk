package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
)

// AnalyzeRepository analyzes a Git repository and returns grouped episodes
// The repo parameter can be either a local path or a remote URL
// Uses default grouping configuration
func AnalyzeRepository(ctx context.Context, repo string) ([]cluster.Episode, error) {
	config := cluster.DefaultGroupingConfig()
	return AnalyzeRepositoryWithConfig(ctx, repo, config)
}

// AnalyzeRepositoryWithConfig analyzes a repository with custom grouping configuration
func AnalyzeRepositoryWithConfig(ctx context.Context, repo string, config cluster.GroupingConfig) ([]cluster.Episode, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before analysis: %w", err)
	}

	// Step 1: Ingest repository data
	activity, err := ingestRepository(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to ingest repository: %w", err)
	}

	// Check for context cancellation after ingestion
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled after ingestion: %w", err)
	}

	// Step 2: Group commits into episodes
	episodes := activity.GroupIntoEpisodes(config)

	return episodes, nil
}

// ingestRepository handles the ingestion of repository data
// Supports both local paths and remote URLs
func ingestRepository(ctx context.Context, repo string) (*cluster.RepositoryActivity, error) {
	// Try to open as local repository first
	gitRepo, err := git.OpenRepository(repo)
	if err != nil {
		// If local open fails, try cloning from remote URL
		gitRepo, err = git.CloneRepository(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to open or clone repository '%s': %w", repo, err)
		}
	}

	// Parse repository with reasonable defaults
	// maxCommits: 0 = unlimited, includePatch: false for performance
	repoData, err := git.ParseRepository(gitRepo, repo, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	// Convert to RepositoryActivity
	activity := &cluster.RepositoryActivity{
		Platform:       cluster.PlatformGit,
		RepositoryURL:  repo,
		RepositoryName: extractRepoName(repo),
		Owner:          "", // Git doesn't have owner concept
		DefaultBranch:  repoData.HeadBranch,
		Commits:        repoData.Commits,
		Artifacts:      []cluster.Artifact{}, // Git doesn't have artifacts
		FetchedAt:      time.Now(),
	}

	return activity, nil
}

// extractRepoName extracts the repository name from a path or URL
func extractRepoName(repo string) string {
	// Remove trailing slash if present
	if len(repo) > 0 && repo[len(repo)-1] == '/' {
		repo = repo[:len(repo)-1]
	}

	// Find the last slash
	lastSlash := -1
	for i := len(repo) - 1; i >= 0; i-- {
		if repo[i] == '/' {
			lastSlash = i
			break
		}
	}

	// Extract name after last slash
	name := repo
	if lastSlash >= 0 && lastSlash < len(repo)-1 {
		name = repo[lastSlash+1:]
	}

	// Remove .git suffix if present
	if len(name) > 4 && name[len(name)-4:] == ".git" {
		name = name[:len(name)-4]
	}

	return name
}
