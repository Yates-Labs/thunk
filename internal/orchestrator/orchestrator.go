package orchestrator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Yates-Labs/thunk/internal/adapter"
	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env
	_ = godotenv.Load("../../.env")
}

// AnalyzeRepository analyzes a Git repository and returns grouped episodes
// The repo parameter can be either a local path or a remote URL
// Uses default grouping configuration
// Token is automatically loaded from GITHUB_TOKEN environment variable if not provided
func AnalyzeRepository(ctx context.Context, repo string, token ...string) ([]cluster.Episode, error) {
	config := cluster.DefaultGroupingConfig()
	return AnalyzeRepositoryWithConfig(ctx, repo, config, token...)
}

// AnalyzeRepositoryWithConfig analyzes a repository with custom grouping configuration
// Token is automatically loaded from GITHUB_TOKEN environment variable if not provided
func AnalyzeRepositoryWithConfig(ctx context.Context, repo string, config cluster.GroupingConfig, token ...string) ([]cluster.Episode, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before analysis: %w", err)
	}

	// Extract token: use provided token, otherwise fall back to env var
	var apiToken string
	if len(token) > 0 && token[0] != "" {
		apiToken = token[0]
	} else {
		apiToken = os.Getenv("GITHUB_TOKEN")
	}

	// Step 1: Ingest repository data
	activity, err := ingestRepository(ctx, repo, apiToken)
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
// Detects platform from URL and fetches additional artifacts if token is provided
func ingestRepository(ctx context.Context, repo, token string) (*cluster.RepositoryActivity, error) {
	// Detect platform from URL
	platform, owner, repoName := detectPlatform(repo)

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
		Platform:       platform,
		RepositoryURL:  repo,
		RepositoryName: repoName,
		Owner:          owner,
		DefaultBranch:  repoData.HeadBranch,
		Commits:        repoData.Commits,
		Artifacts:      []cluster.Artifact{},
		FetchedAt:      time.Now(),
	}

	// Enrich with platform-specific artifacts if token provided
	if token != "" && owner != "" && repoName != "" {
		if err := enrichWithArtifacts(ctx, activity, token, owner, repoName); err != nil {
			// Log error but don't fail - continue with just git data
			fmt.Printf("Warning: failed to fetch artifacts from %s: %v\n", platform, err)
		}
	}

	return activity, nil
}

// enrichWithArtifacts dispatches to platform-specific enrichment based on the activity's platform
func enrichWithArtifacts(ctx context.Context, activity *cluster.RepositoryActivity, token, owner, repo string) error {
	var platformAdapter adapter.Adapter

	switch activity.Platform {
	case cluster.PlatformGitHub:
		platformAdapter = adapter.NewGitHubAdapter()
	// ? This is where we would implement other platforms
	default:
		return nil
	}

	// Use the adapter to fetch artifacts
	artifacts, err := platformAdapter.FetchArtifacts(ctx, token, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch artifacts: %w", err)
	}

	// Add artifacts to activity
	activity.Artifacts = append(activity.Artifacts, artifacts...)

	return nil
}
