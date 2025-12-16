package narrative

import (
	"context"
	"fmt"
	"strings"
)

// MockLLM is a deterministic LLM implementation for testing.
// It returns predictable responses based on prompt content.
type MockLLM struct {
	// Response is the fixed text returned by Generate.
	// If empty, a default response is generated from the prompt.
	Response string

	// Error, if set, is returned by Generate instead of a response.
	Error error

	// LastPrompt stores the most recent prompt passed to Generate.
	LastPrompt string
}

// NewMockLLM creates a mock LLM with the given fixed response.
func NewMockLLM(response string) *MockLLM {
	return &MockLLM{Response: response}
}

// NewMockLLMWithError creates a mock LLM that always returns an error.
func NewMockLLMWithError(err error) *MockLLM {
	return &MockLLM{Error: err}
}

// Generate returns the configured response or generates a deterministic one.
func (m *MockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	m.LastPrompt = prompt

	if m.Error != nil {
		return "", m.Error
	}

	if m.Response != "" {
		return m.Response, nil
	}

	// Generate a deterministic response based on prompt content
	return generateMockResponse(prompt), nil
}

// generateMockResponse creates a predictable narrative from the prompt.
func generateMockResponse(prompt string) string {
	var b strings.Builder

	// Extract episode ID if present
	episodeID := "unknown"
	if strings.Contains(prompt, "**Episode ID:**") {
		parts := strings.Split(prompt, "**Episode ID:**")
		if len(parts) > 1 {
			lines := strings.Split(parts[1], "\n")
			if len(lines) > 0 {
				episodeID = strings.TrimSpace(lines[0])
			}
		}
	}

	commitCount := countCommitBullets(prompt)

	// Generate narrative
	b.WriteString(fmt.Sprintf("This narrative describes episode %s, ", episodeID))
	b.WriteString(fmt.Sprintf("which consisted of %d commits. ", commitCount))
	b.WriteString("The development work focused on implementing key features and fixing bugs. ")
	b.WriteString("Technical decisions were made to improve code quality and maintainability. ")

	return b.String()
}

func countCommitBullets(prompt string) int {
	commitHeader := "**Commit Messages:**"
	idx := strings.Index(prompt, commitHeader)
	if idx >= 0 {
		remainder := prompt[idx+len(commitHeader):]
		// Take content until the next blank line.
		if split := strings.SplitN(remainder, "\n\n", 2); len(split) > 0 {
			remainder = split[0]
		}
		count := 0
		for _, line := range strings.Split(remainder, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				count++
			}
		}
		if count > 0 {
			return count
		}
	}

	// Fallback: count bullet lines in the entire prompt.
	count := 0
	for _, line := range strings.Split(prompt, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			count++
		}
	}
	return count
}
