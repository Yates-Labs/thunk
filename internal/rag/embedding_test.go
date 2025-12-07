package rag

import (
	"context"
	"os"
	"testing"
)

func TestNewOpenAIEmbedder_MissingAPIKey(t *testing.T) {
	// Save original API key
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	// Unset API key
	os.Unsetenv("OPENAI_API_KEY")

	_, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != ErrMissingAPIKey {
		t.Errorf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestOpenAIEmbedder_EmptyTexts(t *testing.T) {
	// Skip if no API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	_, err = embedder.Embed(context.Background(), []string{})
	if err != ErrEmptyTexts {
		t.Errorf("expected ErrEmptyTexts, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed(t *testing.T) {
	// Skip if no API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	embedder, err := NewOpenAIEmbedder("text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	texts := []string{"hello world", "test embedding"}
	records, err := embedder.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(records) != len(texts) {
		t.Errorf("expected %d records, got %d", len(texts), len(records))
	}

	for i, record := range records {
		if record.Text != texts[i] {
			t.Errorf("record[%d].Text = %q, want %q", i, record.Text, texts[i])
		}
		if len(record.Embedding) != 1536 {
			t.Errorf("record[%d] embedding dimension = %d, want 1536", i, len(record.Embedding))
		}
		if record.Model != "text-embedding-3-small" {
			t.Errorf("record[%d].Model = %q, want %q", i, record.Model, "text-embedding-3-small")
		}
	}
}

func TestOpenAIEmbedder_GetModel(t *testing.T) {
	// Skip if no API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	embedder, err := NewOpenAIEmbedder("text-embedding-3-large", 3072)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	if embedder.GetModel() != "text-embedding-3-large" {
		t.Errorf("GetModel() = %q, want %q", embedder.GetModel(), "text-embedding-3-large")
	}

	if embedder.GetDimension() != 3072 {
		t.Errorf("GetDimension() = %d, want %d", embedder.GetDimension(), 3072)
	}
}
