package rag

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Common errors for embedding operations
var (
	ErrEmptyTexts      = errors.New("no texts provided for embedding")
	ErrMissingAPIKey   = errors.New("OPENAI_API_KEY environment variable not set")
	ErrEmbeddingFailed = errors.New("embedding generation failed")
)

// EmbeddingRecord represents a single text embedding with metadata
type EmbeddingRecord struct {
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
	Model     string    `json:"model"`
}

// Embedder defines the interface for generating text embeddings
type Embedder interface {
	// Embed generates embeddings for the provided texts
	Embed(ctx context.Context, texts []string) ([]EmbeddingRecord, error)

	// GetModel returns the embedding model identifier
	GetModel() string

	// GetDimension returns the embedding vector dimension
	GetDimension() int
}

// OpenAIEmbedder implements the Embedder interface using OpenAI's API
type OpenAIEmbedder struct {
	client    openai.Client
	model     string
	dimension int
}

// NewOpenAIEmbedder creates a new OpenAI embedder instance
func NewOpenAIEmbedder(model string, dimension int) (*OpenAIEmbedder, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &OpenAIEmbedder{
		client:    client,
		model:     model,
		dimension: dimension,
	}, nil
}

// GetModel returns the embedding model identifier
func (e *OpenAIEmbedder) GetModel() string {
	return e.model
}

// GetDimension returns the embedding vector dimension
func (e *OpenAIEmbedder) GetDimension() int {
	return e.dimension
}

// Embed generates embeddings for the provided texts using OpenAI's API
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([]EmbeddingRecord, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyTexts
	}

	resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model:          e.model,
		Dimensions:     openai.Int(int64(e.dimension)),
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	records := make([]EmbeddingRecord, len(resp.Data))
	for i, data := range resp.Data {
		// Convert []float64 to []float32
		embedding := make([]float32, len(data.Embedding))
		for j, val := range data.Embedding {
			embedding[j] = float32(val)
		}

		records[i] = EmbeddingRecord{
			Text:      texts[int(data.Index)],
			Embedding: embedding,
			Index:     int(data.Index),
			Model:     e.model,
		}
	}

	return records, nil
}
