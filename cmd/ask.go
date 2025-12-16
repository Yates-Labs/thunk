package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Yates-Labs/thunk/internal/narrative"
	"github.com/Yates-Labs/thunk/internal/orchestrator"
	"github.com/Yates-Labs/thunk/internal/rag"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	topK           int
	maxContextSize int
	reindex        bool
	verbose        bool
)

var askCmd = &cobra.Command{
	Use:   "ask [repository] [question]",
	Short: "Ask a question about a repository using RAG",
	Long: `Ask a natural language question about a repository using RAG (Retrieval-Augmented Generation).
	
This command:
1. Analyzes the repository and extracts episodes
2. Indexes episodes into a vector store (Milvus)
3. Retrieves relevant context for your question
4. Generates a narrative answer using an LLM (OpenAI)

Required environment variables:
  OPENAI_API_KEY     - OpenAI API key for embeddings and LLM
  MILVUS_ADDRESS     - Milvus server address (default: localhost:19530)

Examples:
  thunk ask /path/to/repo "What were the main features added last month?"
  thunk ask https://github.com/user/repo "Who worked on authentication?" --topk 5
  thunk ask . "Summarize the recent bug fixes" --verbose`,
	Args: cobra.ExactArgs(2),
	RunE: runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)
	askCmd.Flags().IntVar(&topK, "topk", 3, "Number of similar episodes to retrieve for context")
	askCmd.Flags().IntVar(&maxContextSize, "max-context", 5000, "Maximum context size in tokens")
	askCmd.Flags().BoolVar(&reindex, "reindex", false, "Force reindexing of episodes")
	askCmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed progress and context")
}

func runAsk(cmd *cobra.Command, args []string) error {
	repo := args[0]
	question := args[1]
	ctx := context.Background()

	// Load .env file if it exists
	loadEnvFile(".env")

	// Check required environment variables
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	milvusAddr := os.Getenv("MILVUS_ADDRESS")
	if milvusAddr == "" {
		milvusAddr = "localhost:19530"
	}

	// Styling
	var (
		headerColor   = lipgloss.Color("#F780FF") // Bright pink
		questionColor = lipgloss.Color("#8BE9FD") // Cyan
		answerColor   = lipgloss.Color("#E9E9F4") // Light purple/white
		contextColor  = lipgloss.Color("#6272A4") // Muted purple
		errorColor    = lipgloss.Color("#FF5555") // Red
		successColor  = lipgloss.Color("#50FA7B") // Green
	)

	headerStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true)

	questionStyle := lipgloss.NewStyle().
		Foreground(questionColor).
		Italic(true)

	answerStyle := lipgloss.NewStyle().
		Foreground(answerColor)

	contextStyle := lipgloss.NewStyle().
		Foreground(contextColor).
		Italic(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	successStyle := lipgloss.NewStyle().
		Foreground(successColor)

	// Print question
	fmt.Println()
	fmt.Println(headerStyle.Render("Question:"))
	fmt.Println(questionStyle.Render(question))
	fmt.Println()

	// Step 1: Analyze repository
	if verbose {
		fmt.Println(contextStyle.Render("→ Analyzing repository..."))
	}
	episodes, err := orchestrator.AnalyzeRepository(ctx, repo)
	if err != nil {
		return fmt.Errorf("%s %w", errorStyle.Render("Error:"), err)
	}

	if len(episodes) == 0 {
		return fmt.Errorf("%s No episodes found in repository", errorStyle.Render("Error:"))
	}

	if verbose {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Found %d episodes", len(episodes))))
	}

	// Step 2: Create RAG pipeline
	if verbose {
		fmt.Println(contextStyle.Render("→ Initializing RAG pipeline..."))
	}

	config := orchestrator.RAGConfig{
		TopK:              topK,
		MaxContextSize:    maxContextSize,
		ReindexOnDemand:   reindex,
		EmbedderModel:     "text-embedding-3-large",
		EmbedderDimension: 3072,
		MilvusConfig: rag.MilvusConfig{
			Address:        milvusAddr,
			CollectionName: "thunk_episodes",
			Dimension:      3072,
			MetricType:     "COSINE",
			IndexType:      "HNSW",
			M:              16,
			EfConstruction: 256,
		},
		LLMConfig: narrative.LLMConfig{
			Model:       "gpt-4o",
			Temperature: 0.7,
			MaxTokens:   2000,
			APIKey:      apiKey,
		},
	}

	pipeline, err := orchestrator.NewRAGPipeline(ctx, config)
	if err != nil {
		return fmt.Errorf("%s Failed to create RAG pipeline: %w", errorStyle.Render("Error:"), err)
	}
	defer pipeline.Close()

	if verbose {
		fmt.Println(successStyle.Render("✓ RAG pipeline initialized"))
	}

	// Step 3: Index episodes (if needed)
	if verbose || reindex {
		fmt.Println(contextStyle.Render("→ Indexing episodes..."))
	}
	if err := pipeline.IndexEpisodes(ctx, episodes); err != nil {
		return fmt.Errorf("%s Failed to index episodes: %w", errorStyle.Render("Error:"), err)
	}

	if verbose || reindex {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Indexed %d episodes", len(episodes))))
	}

	// Step 4: Generate answer using RAG
	if verbose {
		fmt.Println(contextStyle.Render("→ Retrieving relevant context and generating answer..."))
	}

	narr, err := pipeline.GenerateProjectNarrativeRAG(ctx, question, episodes)
	if err != nil {
		return fmt.Errorf("%s Failed to generate answer: %w", errorStyle.Render("Error:"), err)
	}

	// Print answer
	fmt.Println(headerStyle.Render("Answer:"))
	fmt.Println()

	// Format the narrative text with proper wrapping
	answerText := strings.TrimSpace(narr.Text)
	fmt.Println(answerStyle.Render(answerText))
	fmt.Println()

	return nil
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// Silently ignore if .env doesn't exist
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"`)
		value = strings.Trim(value, `'`)

		// Set environment variable (don't override existing ones)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
