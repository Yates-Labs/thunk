package narrative

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// OpenAILLM implements the LLM interface using OpenAI's API.
type OpenAILLM struct {
	client openai.Client
	config LLMConfig
}

// NewOpenAILLM creates an OpenAI-backed LLM implementation.
// Returns an error if the API key is missing or invalid.
func NewOpenAILLM(config LLMConfig) (*OpenAILLM, error) {
	// Use config API key or fall back to environment variable
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: missing API key (set OPENAI_API_KEY or provide in config)", ErrInvalidConfig)
	}
	if config.Model == "" {
		return nil, fmt.Errorf("%w: missing model name", ErrInvalidConfig)
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAILLM{
		client: client,
		config: config,
	}, nil
}

// Generate sends the prompt to OpenAI and returns the generated text.
func (o *OpenAILLM) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("%w: prompt cannot be empty", ErrInvalidConfig)
	}

	// Build the chat completion parameters
	params := openai.ChatCompletionNewParams{
		Model: shared.ChatModel(o.config.Model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	// Set optional parameters if configured
	if o.config.Temperature > 0 {
		params.Temperature = openai.Float(float64(o.config.Temperature))
	}
	if o.config.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(o.config.MaxTokens))
	}

	// Call the OpenAI API
	completion, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrLLMFailed, err)
	}

	// Validate the response
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("%w: no response generated", ErrLLMFailed)
	}

	return completion.Choices[0].Message.Content, nil
}
