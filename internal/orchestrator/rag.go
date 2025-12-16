package orchestrator

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/ingest/git"
	"github.com/Yates-Labs/thunk/internal/narrative"
	"github.com/Yates-Labs/thunk/internal/rag"
)

// RAGConfig holds configuration for the RAG-based narrative generation pipeline.
type RAGConfig struct {
	// TopK is the number of similar episodes to retrieve as context
	TopK int

	// MaxContextSize is the maximum number of context chunks to include in the prompt
	MaxContextSize int

	// ReindexOnDemand forces re-indexing of episodes before retrieval
	ReindexOnDemand bool

	// EmbedderModel is the model to use for embeddings (e.g., "text-embedding-3-large")
	EmbedderModel string

	// EmbedderDimension is the vector dimension for embeddings
	EmbedderDimension int

	// LLMConfig holds the LLM configuration for narrative generation
	LLMConfig narrative.LLMConfig

	// MilvusConfig holds the Milvus vector store configuration
	MilvusConfig rag.MilvusConfig
}

// DefaultRAGConfig returns sensible defaults for the RAG pipeline.
func DefaultRAGConfig() RAGConfig {
	return RAGConfig{
		TopK:              5,
		MaxContextSize:    10,
		ReindexOnDemand:   false,
		EmbedderModel:     "text-embedding-3-large",
		EmbedderDimension: 3072,
		LLMConfig:         narrative.DefaultLLMConfig(),
		MilvusConfig:      rag.DefaultMilvusConfig(),
	}
}

// RAGPipeline orchestrates end-to-end RAG-based narrative generation.
type RAGPipeline struct {
	config      RAGConfig
	embedder    rag.Embedder
	vectorStore rag.VectorStore
	retriever   *rag.Retriever
	generator   *narrative.Generator
}

// NewRAGPipeline creates a new RAG pipeline with the given configuration.
func NewRAGPipeline(ctx context.Context, config RAGConfig) (*RAGPipeline, error) {
	// Initialize embedder
	embedder, err := rag.NewOpenAIEmbedder(config.EmbedderModel, config.EmbedderDimension)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize vector store
	vectorStore, err := rag.NewMilvusStore(ctx, config.MilvusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	// Initialize retriever
	retriever, err := rag.NewRetriever(embedder, vectorStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	// Initialize LLM
	llm, err := narrative.NewOpenAILLM(config.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// Initialize generator
	generator := narrative.NewGenerator(llm, config.LLMConfig)

	return &RAGPipeline{
		config:      config,
		embedder:    embedder,
		vectorStore: vectorStore,
		retriever:   retriever,
		generator:   generator,
	}, nil
}

// Close releases resources held by the RAG pipeline.
func (p *RAGPipeline) Close() error {
	if p.vectorStore != nil {
		return p.vectorStore.Close()
	}
	return nil
}

// IndexEpisodes indexes episode summaries into the vector store.
// This should be called before generating narratives to ensure episodes are searchable.
func (p *RAGPipeline) IndexEpisodes(ctx context.Context, episodes []cluster.Episode) error {
	log.Printf("[RAG Pipeline] Indexing %d episodes", len(episodes))

	// Convert episodes to summaries
	summaries := make([]rag.EpisodeSummary, len(episodes))
	for i, ep := range episodes {
		startDate, endDate := ep.GetDateRange()

		summaryText := generateEpisodeSummaryText(&ep)

		summaries[i] = rag.EpisodeSummary{
			EpisodeID:   ep.ID,
			Title:       generateEpisodeTitle(&ep),
			Summary:     summaryText,
			StartDate:   startDate,
			EndDate:     endDate,
			Authors:     ep.GetAuthorNames(),
			CommitCount: len(ep.Commits),
			FileCount:   ep.GetFileCount(),
		}
	}

	// Set up indexing options
	opts := rag.IndexOptions{
		BatchSize:    10,
		ForceReindex: p.config.ReindexOnDemand,
		SkipExisting: !p.config.ReindexOnDemand,
	}

	// Index episodes
	if err := rag.IndexEpisodes(ctx, summaries, p.embedder, p.vectorStore, opts); err != nil {
		return fmt.Errorf("failed to index episodes: %w", err)
	}

	log.Printf("[RAG Pipeline] Successfully indexed %d episodes", len(episodes))
	return nil
}

// GenerateEpisodeNarrativeRAG generates a narrative for a specific episode using RAG.
// The pipeline: retrieval -> prompt assembly -> LLM generation -> Narrative
func (p *RAGPipeline) GenerateEpisodeNarrativeRAG(
	ctx context.Context,
	episode *cluster.Episode,
) (*narrative.Narrative, error) {
	if episode == nil {
		return nil, fmt.Errorf("episode cannot be nil")
	}

	log.Printf("[RAG Pipeline] Generating narrative for episode %s", episode.ID)

	// Stage 1: Retrieval - Get similar episodes as context
	log.Printf("[RAG Pipeline] Stage 1: Retrieving top-%d similar episodes", p.config.TopK)
	contextChunks, err := p.retriever.RetrieveContextForEpisode(
		ctx,
		episode.ID,
		p.config.TopK,
		nil, // No filtering
	)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}
	log.Printf("[RAG Pipeline] Retrieved %d context chunks", len(contextChunks))

	// Apply max context size limit
	if len(contextChunks) > p.config.MaxContextSize {
		contextChunks = contextChunks[:p.config.MaxContextSize]
		log.Printf("[RAG Pipeline] Trimmed context to %d chunks (max size)", p.config.MaxContextSize)
	}

	// Stage 2: Prompt Assembly - Build prompt with episode and context
	log.Printf("[RAG Pipeline] Stage 2: Assembling prompt with %d context chunks", len(contextChunks))
	prompt, err := narrative.AssemblePrompt(episode, contextChunks)
	if err != nil {
		return nil, fmt.Errorf("prompt assembly failed: %w", err)
	}
	log.Printf("[RAG Pipeline] Assembled prompt (%d characters)", len(prompt))

	// Stage 3: LLM Generation - Generate narrative
	log.Printf("[RAG Pipeline] Stage 3: Generating narrative with LLM")
	narr, err := p.generator.Generate(ctx, episode.ID, prompt)
	if err != nil {
		return nil, fmt.Errorf("narrative generation failed: %w", err)
	}
	log.Printf("[RAG Pipeline] Successfully generated narrative (%d characters)", len(narr.Text))

	return narr, nil
}

// GenerateProjectNarrativeRAG generates a project-level narrative using RAG.
// This retrieves relevant episodes across the entire repository to create a high-level summary.
func (p *RAGPipeline) GenerateProjectNarrativeRAG(
	ctx context.Context,
	query string,
	episodes []cluster.Episode,
) (*narrative.Narrative, error) {
	log.Printf("[RAG Pipeline] Generating project narrative for query: %s", query)

	// Stage 1: Retrieval - Get most relevant episodes for the query
	log.Printf("[RAG Pipeline] Stage 1: Retrieving top-%d relevant episodes", p.config.TopK)
	contextChunks, err := p.retriever.RetrieveContextForQuery(
		ctx,
		query,
		p.config.TopK,
		nil, // No filtering
	)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}
	log.Printf("[RAG Pipeline] Retrieved %d context chunks", len(contextChunks))

	// Hybrid Search: Check for specific PR/Issue references in the query
	// If found, manually find the episode containing that artifact and add it to context
	prRegex := regexp.MustCompile(`(?i)(?:pr|pull request|issue)\s*#?(\d+)`)
	matches := prRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) > 1 {
			number, err := strconv.Atoi(match[1])
			if err == nil {
				// Find episode containing this artifact
				for _, ep := range episodes {
					found := false
					for _, art := range ep.Artifacts {
						if art.Number == number {
							found = true
							break
						}
					}

					if found {
						// Check if already in context
						alreadyInContext := false
						for _, chunk := range contextChunks {
							if chunk.EpisodeID == ep.ID {
								alreadyInContext = true
								break
							}
						}

						if !alreadyInContext {
							// Create a synthetic chunk
							startDate, endDate := ep.GetDateRange()
							chunk := rag.ContextChunk{
								EpisodeID:   ep.ID,
								Text:        generateEpisodeSummaryText(&ep),
								Score:       1.0, // Max score for exact match
								StartDate:   startDate,
								EndDate:     endDate,
								Authors:     ep.GetAuthorNames(),
								CommitCount: len(ep.Commits),
								FileCount:   ep.GetFileCount(),
							}

							// Prepend to context chunks
							contextChunks = append([]rag.ContextChunk{chunk}, contextChunks...)
						}
						break // Found the episode, move to next match
					}
				}
			}
		}
	}

	// Apply max context size limit
	if len(contextChunks) > p.config.MaxContextSize {
		contextChunks = contextChunks[:p.config.MaxContextSize]
		log.Printf("[RAG Pipeline] Trimmed context to %d chunks (max size)", p.config.MaxContextSize)
	}

	// Stage 2: Assemble prompt with query and retrieved context
	log.Printf("[RAG Pipeline] Stage 2: Assembling project-level prompt with %d context chunks", len(contextChunks))
	prompt := assembleProjectQueryPrompt(query, episodes, contextChunks)
	log.Printf("[RAG Pipeline] Assembled prompt (%d characters)", len(prompt))
	narr, err := p.generator.Generate(ctx, "project", prompt)
	if err != nil {
		return nil, fmt.Errorf("narrative generation failed: %w", err)
	}
	log.Printf("[RAG Pipeline] Successfully generated project narrative (%d characters)", len(narr.Text))

	return narr, nil
}

// GenerateMultipleNarrativesRAG generates narratives for multiple episodes efficiently.
func (p *RAGPipeline) GenerateMultipleNarrativesRAG(
	ctx context.Context,
	episodes []cluster.Episode,
) ([]*narrative.Narrative, error) {
	log.Printf("[RAG Pipeline] Generating narratives for %d episodes", len(episodes))

	narratives := make([]*narrative.Narrative, 0, len(episodes))

	for i, episode := range episodes {
		log.Printf("[RAG Pipeline] Processing episode %d/%d: %s", i+1, len(episodes), episode.ID)

		narr, err := p.GenerateEpisodeNarrativeRAG(ctx, &episode)
		if err != nil {
			log.Printf("[RAG Pipeline] Warning: Failed to generate narrative for episode %s: %v", episode.ID, err)
			// Continue with remaining episodes
			continue
		}

		narratives = append(narratives, narr)
	}

	log.Printf("[RAG Pipeline] Successfully generated %d/%d narratives", len(narratives), len(episodes))
	return narratives, nil
}

// Helper functions

func generateEpisodeTitle(ep *cluster.Episode) string {
	if len(ep.Commits) > 0 {
		// Use first commit message as title
		return ep.Commits[0].Message
	}
	if len(ep.Artifacts) > 0 {
		// Use first artifact title
		return ep.Artifacts[0].Title
	}
	return fmt.Sprintf("Episode %s", ep.ID)
}

func generateEpisodeSummaryText(ep *cluster.Episode) string {
	var summary string

	// Add commit information
	if len(ep.Commits) > 0 {
		summary += fmt.Sprintf("Commits (%d):\n", len(ep.Commits))
		for i, commit := range ep.Commits {
			if i >= 5 { // Limit to first 5 commits
				summary += fmt.Sprintf("... and %d more commits\n", len(ep.Commits)-5)
				break
			}
			summary += fmt.Sprintf("- %s (by %s)\n", commit.Message, commit.Author.Name)
		}
	}

	// Add artifact information with descriptions for better searchability
	if len(ep.Artifacts) > 0 {
		summary += fmt.Sprintf("\nArtifacts (%d):\n", len(ep.Artifacts))
		for i, artifact := range ep.Artifacts {
			if i >= 20 { // Increased limit to 20 artifacts
				summary += fmt.Sprintf("... and %d more artifacts\n", len(ep.Artifacts)-20)
				break
			}
			// Include both PR/Issue notation for better matching
			artifactType := artifact.Type
			if artifactType == cluster.ArtifactPullRequest {
				summary += fmt.Sprintf("- PR #%d: %s\n", artifact.Number, artifact.Title)
			} else if artifactType == cluster.ArtifactIssue {
				summary += fmt.Sprintf("- Issue #%d: %s\n", artifact.Number, artifact.Title)
			} else {
				summary += fmt.Sprintf("- %s #%d: %s\n", artifactType, artifact.Number, artifact.Title)
			}

			// Include description if available (truncated for embedding efficiency)
			if artifact.Description != "" {
				desc := artifact.Description
				if len(desc) > 500 {
					desc = desc[:500] + "..."
				}
				summary += fmt.Sprintf("  Description: %s\n", desc)
			}
		}
	}

	// Add metadata
	authors := ep.GetCommitAuthors()
	if len(authors) > 0 {
		summary += fmt.Sprintf("\nAuthors: %s\n", authors[0])
		for i := 1; i < len(authors) && i < 5; i++ {
			summary += fmt.Sprintf(", %s", authors[i])
		}
	}

	return summary
}

func createProjectMetaEpisode(episodes []cluster.Episode) *cluster.Episode {
	// Aggregate all commits and artifacts from episodes
	var allCommits []git.Commit
	var allArtifacts []cluster.Artifact

	for _, ep := range episodes {
		allCommits = append(allCommits, ep.Commits...)
		allArtifacts = append(allArtifacts, ep.Artifacts...)
	}

	// Create a synthetic meta-episode
	// Note: Episode struct doesn't have StartDate/EndDate fields
	// These are computed dynamically from commits
	return &cluster.Episode{
		ID:        "project-level",
		Commits:   allCommits,
		Artifacts: allArtifacts,
	}
}

func getEarliestDate(episodes []cluster.Episode) time.Time {
	if len(episodes) == 0 {
		return time.Time{}
	}

	var earliest time.Time
	for _, ep := range episodes {
		for _, commit := range ep.Commits {
			if earliest.IsZero() || commit.CommittedAt.Before(earliest) {
				earliest = commit.CommittedAt
			}
		}
	}
	return earliest
}

func getLatestDate(episodes []cluster.Episode) time.Time {
	if len(episodes) == 0 {
		return time.Time{}
	}

	var latest time.Time
	for _, ep := range episodes {
		for _, commit := range ep.Commits {
			if latest.IsZero() || commit.CommittedAt.After(latest) {
				latest = commit.CommittedAt
			}
		}
	}
	return latest
}

// assembleProjectQueryPrompt creates a prompt for answering a specific query about the project
func assembleProjectQueryPrompt(query string, episodes []cluster.Episode, contextChunks []rag.ContextChunk) string {
	var b strings.Builder

	b.WriteString("You are a technical writer specializing in software development narratives. ")
	b.WriteString("Your task is to answer the following question about a software project ")
	b.WriteString("based on the development history and relevant context provided.\n\n")

	b.WriteString("# Question\n\n")
	b.WriteString(fmt.Sprintf("%s\n\n", query))

	// Project overview
	totalCommits := 0
	allAuthors := make(map[string]bool)
	var earliest, latest time.Time

	for _, ep := range episodes {
		totalCommits += len(ep.Commits)
		for _, commit := range ep.Commits {
			allAuthors[commit.Author.Name] = true
			if earliest.IsZero() || commit.CommittedAt.Before(earliest) {
				earliest = commit.CommittedAt
			}
			if latest.IsZero() || commit.CommittedAt.After(latest) {
				latest = commit.CommittedAt
			}
		}
	}

	b.WriteString("# Project Overview\n\n")
	b.WriteString(fmt.Sprintf("**Episodes:** %d development episodes\n\n", len(episodes)))
	b.WriteString(fmt.Sprintf("**Total Commits:** %d commits\n\n", totalCommits))
	b.WriteString(fmt.Sprintf("**Contributors:** %d unique authors\n\n", len(allAuthors)))
	if !earliest.IsZero() && !latest.IsZero() {
		b.WriteString(fmt.Sprintf("**Time Range:** %s to %s\n\n", earliest.Format("2006-01-02"), latest.Format("2006-01-02")))
	}

	// Relevant context from RAG retrieval
	if len(contextChunks) > 0 {
		b.WriteString("# Relevant Development History\n\n")
		b.WriteString("The following episodes are most relevant to your question:\n\n")

		for i, ch := range contextChunks {
			b.WriteString(fmt.Sprintf("## Episode %d: %s (relevance: %.2f)\n\n", i+1, ch.EpisodeID, ch.Score))
			b.WriteString(ch.Text + "\n\n")
		}
	}

	b.WriteString("# Task\n\n")
	b.WriteString("Based on the relevant development history above, answer the question clearly and concisely.\n\n")
	b.WriteString("Guidelines:\n")
	b.WriteString("- Focus your answer specifically on what was asked\n")
	b.WriteString("- Use 2-4 paragraphs unless the question requires more detail\n")
	b.WriteString("- Base all statements strictly on the provided episode data\n")
	b.WriteString("- Do not invent details or motivations not present in the history\n")
	b.WriteString("- Use clear technical language and explain key concepts\n")
	b.WriteString("- If the question cannot be fully answered from the available data, state what is known and what is uncertain\n\n")

	return b.String()
}
