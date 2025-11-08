package orchestrator

import (
	"strings"

	"github.com/Yates-Labs/thunk/internal/cluster"
)

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

// detectPlatform detects the source platform from a repository URL
// Returns platform, owner, and repo name
func detectPlatform(repoURL string) (cluster.SourcePlatform, string, string) {
	// Check for GitHub
	if strings.Contains(repoURL, "github.com") {
		owner, repo := parseHostedGitURL(repoURL, "github.com")
		return cluster.PlatformGitHub, owner, repo
	}

	// ? We would add support for other platforms here.

	// Default to Git for local paths or unknown URLs
	return cluster.PlatformGit, "", extractRepoName(repoURL)
}

// parseHostedGitURL is a generic parser for hosted git services
func parseHostedGitURL(url, host string) (owner, repo string) {
	// Remove protocol if present
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")

	// Replace colon with slash for SSH URLs
	url = strings.Replace(url, ":", "/", 1)

	// Remove host prefix
	url = strings.TrimPrefix(url, host+"/")

	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Split into parts
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "", url
}
