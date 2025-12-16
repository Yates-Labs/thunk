package narrative

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrGenerationFailed = errors.New("narrative generation failed")
)

// Narrative represents a generated human-readable explanation of an episode.
type Narrative struct {
	// EpisodeID identifies the episode this narrative describes
	EpisodeID string `json:"episode_id"`

	// Text is the generated narrative content
	Text string `json:"text"`

	// GeneratedAt is when this narrative was created
	GeneratedAt time.Time `json:"generated_at"`

	// Model is the LLM model used to generate this narrative
	Model string `json:"model"`
}

// Generator produces narratives from episodes using an LLM.
// It invokes an LLM on an already-assembled prompt.
type Generator struct {
	llm    LLM
	config LLMConfig
}

// NewGenerator creates a narrative generator with the given LLM implementation.
func NewGenerator(llm LLM, config LLMConfig) *Generator {
	return &Generator{
		llm:    llm,
		config: config,
	}
}

// Generate creates a narrative by invoking the LLM with an already-assembled prompt.
// It must not perform retrieval or prompt construction.
func (g *Generator) Generate(ctx context.Context, episodeID string, prompt string) (*Narrative, error) {
	if g.llm == nil {
		return nil, fmt.Errorf("%w: LLM is required", ErrGenerationFailed)
	}
	if episodeID == "" {
		return nil, fmt.Errorf("%w: episode ID is required", ErrGenerationFailed)
	}
	if prompt == "" {
		return nil, fmt.Errorf("%w: prompt is required", ErrGenerationFailed)
	}

	text, err := g.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: LLM invocation failed: %w", ErrGenerationFailed, err)
	}

	return &Narrative{
		EpisodeID:   episodeID,
		Text:        text,
		GeneratedAt: time.Now(),
		Model:       g.config.Model,
	}, nil
}
