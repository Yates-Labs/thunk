package narrative

// ContextChunk represents a piece of related context for prompt assembly.
// It mirrors the structure from the RAG store but is defined here to avoid
// circular dependencies and keep the narrative package self-contained.
type ContextChunk struct {
	EpisodeID string
	Text      string
	Score     float32
}
