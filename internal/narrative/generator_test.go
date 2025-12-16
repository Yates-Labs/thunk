package narrative

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/Yates-Labs/thunk/internal/rag"
)

func TestGenerator_Generate_Success(t *testing.T) {
	mockLLM := NewMockLLM("This is a test narrative about the authentication work.")
	config := DefaultLLMConfig()
	config.Model = "test-model"

	gen := NewGenerator(mockLLM, config)

	episode := &cluster.Episode{
		ID: "E123",
		Commits: []git.Commit{
			{
				Hash:        "abc123",
				Message:     "Add JWT authentication",
				Author:      git.Author{Name: "Alice"},
				CommittedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	contextChunks := []rag.ContextChunk{
		{EpisodeID: "E100", Text: "Previous auth work", Score: 0.85},
	}

	prompt, err := AssemblePrompt(episode, contextChunks)
	if err != nil {
		t.Fatalf("unexpected prompt assembly error: %v", err)
	}

	ctx := context.Background()
	narrative, err := gen.Generate(ctx, episode.ID, prompt)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if narrative == nil {
		t.Fatal("narrative is nil")
	}

	if narrative.EpisodeID != "E123" {
		t.Errorf("expected episode ID E123, got %s", narrative.EpisodeID)
	}

	if narrative.Text != "This is a test narrative about the authentication work." {
		t.Errorf("unexpected narrative text: %s", narrative.Text)
	}

	if narrative.Model != "test-model" {
		t.Errorf("expected model test-model, got %s", narrative.Model)
	}

	if narrative.GeneratedAt.IsZero() {
		t.Error("generated timestamp is zero")
	}

	// Verify mock received the prompt
	if mockLLM.LastPrompt == "" {
		t.Error("mock LLM did not receive a prompt")
	}
	if !strings.Contains(mockLLM.LastPrompt, "E123") {
		t.Error("prompt does not contain episode ID")
	}
}

func TestGenerator_Generate_NilEpisode(t *testing.T) {
	mockLLM := NewMockLLM("test")
	gen := NewGenerator(mockLLM, DefaultLLMConfig())

	ctx := context.Background()
	_, err := gen.Generate(ctx, "", "")

	if err == nil {
		t.Fatal("expected error for nil episode")
	}

	if !errors.Is(err, ErrGenerationFailed) {
		t.Errorf("expected ErrGenerationFailed, got %v", err)
	}
}

func TestGenerator_Generate_LLMError(t *testing.T) {
	llmErr := errors.New("API rate limit exceeded")
	mockLLM := NewMockLLMWithError(llmErr)
	gen := NewGenerator(mockLLM, DefaultLLMConfig())

	episode := &cluster.Episode{
		ID: "E456",
	}

	ctx := context.Background()
	_, err := gen.Generate(ctx, episode.ID, "some prompt")

	if err == nil {
		t.Fatal("expected error from LLM")
	}

	if !errors.Is(err, ErrGenerationFailed) {
		t.Errorf("expected ErrGenerationFailed, got %v", err)
	}
}

func TestGenerator_Generate_WithoutContext(t *testing.T) {
	mockLLM := NewMockLLM("Narrative without context.")
	gen := NewGenerator(mockLLM, DefaultLLMConfig())

	episode := &cluster.Episode{
		ID: "E789",
		Commits: []git.Commit{
			{Hash: "ghi789", Message: "Refactor code", Author: git.Author{Name: "Charlie"}},
		},
	}

	prompt, err := AssemblePrompt(episode, nil)
	if err != nil {
		t.Fatalf("unexpected prompt assembly error: %v", err)
	}

	ctx := context.Background()
	narrative, err := gen.Generate(ctx, episode.ID, prompt)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if narrative.Text != "Narrative without context." {
		t.Errorf("unexpected text: %s", narrative.Text)
	}

	// Prompt should not include context section
	if strings.Contains(mockLLM.LastPrompt, "# Related Development Context") {
		t.Error("prompt should not include context section when no chunks provided")
	}
}

func TestGenerator_Generate_DeterministicMock(t *testing.T) {
	// Test using mock's auto-generated response
	mockLLM := &MockLLM{} // No fixed response
	gen := NewGenerator(mockLLM, DefaultLLMConfig())

	episode := &cluster.Episode{
		ID: "E999",
		Commits: []git.Commit{
			{Hash: "aaa", Message: "Commit 1", Author: git.Author{Name: "Dev1"}},
			{Hash: "bbb", Message: "Commit 2", Author: git.Author{Name: "Dev2"}},
		},
	}

	prompt, err := AssemblePrompt(episode, nil)
	if err != nil {
		t.Fatalf("unexpected prompt assembly error: %v", err)
	}

	ctx := context.Background()
	narrative, err := gen.Generate(ctx, episode.ID, prompt)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that auto-generated response mentions the episode
	if !strings.Contains(narrative.Text, "E999") {
		t.Errorf("expected narrative to mention E999, got: %s", narrative.Text)
	}

	// Check that it mentions the commit count
	if !strings.Contains(narrative.Text, "2 commits") {
		t.Errorf("expected narrative to mention 2 commits, got: %s", narrative.Text)
	}
}

func TestMockLLM_Generate(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockLLM
		prompt   string
		wantErr  bool
		wantText string
	}{
		{
			name:     "fixed response",
			mock:     NewMockLLM("Fixed narrative text"),
			prompt:   "Any prompt",
			wantErr:  false,
			wantText: "Fixed narrative text",
		},
		{
			name:    "error response",
			mock:    NewMockLLMWithError(errors.New("mock error")),
			prompt:  "Any prompt",
			wantErr: true,
		},
		{
			name:     "auto-generated response",
			mock:     &MockLLM{},
			prompt:   "**Episode ID:** E456\n- commit 1\n- commit 2",
			wantErr:  false,
			wantText: "E456", // Should contain episode ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			text, err := tt.mock.Generate(ctx, tt.prompt)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantText != "" && !strings.Contains(text, tt.wantText) {
				t.Errorf("expected text to contain %q, got %q", tt.wantText, text)
			}

			// Verify LastPrompt is stored
			if tt.mock.LastPrompt != tt.prompt {
				t.Errorf("expected LastPrompt to be %q, got %q", tt.prompt, tt.mock.LastPrompt)
			}
		})
	}
}

func TestNewGenerator(t *testing.T) {
	mockLLM := NewMockLLM("test")
	config := LLMConfig{
		Model:       "gpt-4",
		Temperature: 0.5,
		MaxTokens:   1000,
	}

	gen := NewGenerator(mockLLM, config)

	if gen == nil {
		t.Fatal("generator is nil")
	}

	if gen.llm != mockLLM {
		t.Error("LLM not set correctly")
	}

	if gen.config.Model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", gen.config.Model)
	}
}
