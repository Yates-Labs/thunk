package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "thunk",
	Short: "Thunk - Repository episode analysis tool",
	Long: `Thunk analyzes Git repositories and groups commits into narrative episodes.
	
It ingests repository data, applies clustering algorithms, and presents
development activity as coherent episodes with timing and authorship details.`,
}

// Execute runs the root command
func Execute() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
