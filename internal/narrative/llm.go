// Package narrative provides LLM-powered narrative generation for development episodes.
// It defines a provider-agnostic LLM interface with concrete implementations for OpenAI
// and deterministic mocks for testing. The generator consumes pre-assembled prompts and
// returns structured narrative objects.
package narrative

import (
	"context"
	"errors"
)

var (
	ErrLLMFailed     = errors.New("LLM request failed")
	ErrInvalidConfig = errors.New("invalid LLM configuration")
)

// LLM defines the interface for interacting with language models.
// Implementations must be stateless and thread-safe.
type LLM interface {
	// Generate produces text from a prompt using the configured model.
	// Returns the generated text or an error if generation fails.
	Generate(ctx context.Context, prompt string) (string, error)
}

// LLMConfig holds common configuration options for LLM providers.
type LLMConfig struct {
	// Model specifies the model identifier (e.g., "gpt-4", "gpt-3.5-turbo")
	Model string

	// Temperature controls randomness (0.0 = deterministic, 2.0 = very random)
	Temperature float32

	// MaxTokens limits the response length (0 = use provider default)
	MaxTokens int

	// APIKey is the authentication key for the provider
	APIKey string
}

// DefaultLLMConfig returns sensible defaults for narrative generation.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Model:       "gpt-4o",
		Temperature: 0, // model default
		MaxTokens:   2000,
	}
}
