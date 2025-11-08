package adapter

import (
	"context"

	"github.com/Yates-Labs/thunk/internal/cluster"
)

// Adapter defines the interface for converting platform-specific artifacts
// into the standardized cluster.Artifact model
type Adapter interface {
	// ConvertIssue converts a platform-specific issue to a cluster.Artifact
	ConvertIssue(issue interface{}) (*cluster.Artifact, error)

	// ConvertPullRequest converts a platform-specific pull request to a cluster.Artifact
	ConvertPullRequest(pr interface{}) (*cluster.Artifact, error)

	// GetPlatform returns the source platform identifier
	GetPlatform() cluster.SourcePlatform

	// FetchArtifacts fetches all artifacts (issues, PRs, etc.) as a standardized type from the platform
	FetchArtifacts(ctx context.Context, token, owner, repo string) ([]cluster.Artifact, error)
}
