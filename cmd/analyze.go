package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Yates-Labs/thunk/internal/cluster"
	"github.com/Yates-Labs/thunk/internal/orchestrator"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [repository]",
	Short: "Analyze a repository and display episodes",
	Long: `Analyze a Git repository (local path or remote URL) and display grouped episodes.
	
Each episode shows:
- Episode ID
- Number of unique authors
- Number of commits  
- Date range (start → end)

Examples:
  thunk analyze /path/to/local/repo
  thunk analyze https://github.com/user/repo
  thunk analyze https://github.com/user/repo --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	repo := args[0]
	ctx := context.Background()

	// Run the analysis
	episodes, err := orchestrator.AnalyzeRepository(ctx, repo)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	if len(episodes) == 0 {
		fmt.Println("No episodes found in repository")
		return nil
	}

	// Output based on format
	if jsonOutput {
		return outputJSON(episodes)
	}

	return outputTable(episodes)
}

func outputJSON(episodes []cluster.Episode) error {
	// Create structured output
	type EpisodeSummary struct {
		ID          string    `json:"id"`
		AuthorCount int       `json:"author_count"`
		CommitCount int       `json:"commit_count"`
		StartDate   time.Time `json:"start_date"`
		EndDate     time.Time `json:"end_date"`
		Duration    string    `json:"duration"`
	}

	summaries := make([]EpisodeSummary, len(episodes))
	for i, ep := range episodes {
		authors := ep.GetCommitAuthors()
		var startDate, endDate time.Time

		if len(ep.Commits) > 0 {
			startDate = ep.Commits[0].CommittedAt
			endDate = ep.Commits[len(ep.Commits)-1].CommittedAt
		}

		summaries[i] = EpisodeSummary{
			ID:          ep.ID,
			AuthorCount: len(authors),
			CommitCount: len(ep.Commits),
			StartDate:   startDate,
			EndDate:     endDate,
			Duration:    ep.GetDuration().String(),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summaries)
}

func outputTable(episodes []cluster.Episode) error {
	// LipGloss signature purple/pink palette
	var (
		// Colors
		headerColor  = lipgloss.Color("#F780FF") // Bright pink/magenta
		episodeColor = lipgloss.Color("#BD93F9") // Purple
		numberColor  = lipgloss.Color("#FF79C6") // Pink
		dateColor    = lipgloss.Color("#E9E9F4") // Light purple/white
		borderColor  = lipgloss.Color("#6272A4") // Muted purple
		summaryColor = lipgloss.Color("#8BE9FD") // Cyan accent
	)

	// Column widths
	const (
		idWidth     = 16
		authorWidth = 10
		commitWidth = 10
		dateWidth   = 42
	)

	// Header style
	headerStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Padding(0, 1)

	// Border separator
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Print header
	headers := []string{
		headerStyle.Width(idWidth).Render("EPISODE"),
		headerStyle.Width(authorWidth).Render("AUTHORS"),
		headerStyle.Width(commitWidth).Render("COMMITS"),
		headerStyle.Width(dateWidth).Render("DATE RANGE"),
	}
	fmt.Println(strings.Join(headers, borderStyle.Render("│")))

	// Print separator line - create separator sections and join with ┼
	separatorParts := []string{
		strings.Repeat("─", idWidth),
		strings.Repeat("─", authorWidth),
		strings.Repeat("─", commitWidth),
		strings.Repeat("─", dateWidth),
	}
	fmt.Println(borderStyle.Render(strings.Join(separatorParts, "┼")))

	// Print data rows - no alternating backgrounds
	for _, ep := range episodes {
		authors := ep.GetCommitAuthors()
		var dateRange string

		if len(ep.Commits) > 0 {
			startDate := ep.Commits[0].CommittedAt
			endDate := ep.Commits[len(ep.Commits)-1].CommittedAt

			if startDate.Equal(endDate) {
				dateRange = startDate.Format("Jan 02, 15:04")
			} else {
				dateRange = fmt.Sprintf("%s → %s",
					startDate.Format("Jan 02, 15:04"),
					endDate.Format("Jan 02, 15:04"))
			}
		} else {
			dateRange = "No commits"
		}

		// Create cell styles
		idStyle := lipgloss.NewStyle().
			Foreground(episodeColor).
			Padding(0, 1).
			Width(idWidth)

		numStyle := lipgloss.NewStyle().
			Foreground(numberColor).
			Padding(0, 1).
			Width(authorWidth).
			Align(lipgloss.Right)

		commitStyle := lipgloss.NewStyle().
			Foreground(numberColor).
			Padding(0, 1).
			Width(commitWidth).
			Align(lipgloss.Right)

		dateStyled := lipgloss.NewStyle().
			Foreground(dateColor).
			Padding(0, 1).
			Width(dateWidth)

		cells := []string{
			idStyle.Render(ep.ID),
			numStyle.Render(fmt.Sprintf("%d", len(authors))),
			commitStyle.Render(fmt.Sprintf("%d", len(ep.Commits))),
			dateStyled.Render(dateRange),
		}

		fmt.Println(strings.Join(cells, borderStyle.Render("│")))
	}

	// Calculate and print summary
	fmt.Println()
	totalCommits := 0
	allAuthors := make(map[string]bool)

	for _, ep := range episodes {
		totalCommits += len(ep.Commits)
		authors := ep.GetCommitAuthors()
		for _, author := range authors {
			allAuthors[author.Email] = true
		}
	}

	summaryStyle := lipgloss.NewStyle().
		Foreground(summaryColor).
		Italic(true)

	summary := fmt.Sprintf("Total: %d episodes, %d commits, %d unique authors",
		len(episodes), totalCommits, len(allAuthors))
	fmt.Println(summaryStyle.Render(summary))

	return nil
}
