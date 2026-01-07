package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "superralph",
	Short: "PRD-driven agent harness for long-running Claude development sessions",
	Long: `SuperRalph is a CLI tool that validates PRD (Product Requirements Document) files
and orchestrates Claude to implement features incrementally with test-gated commits.

It provides a TUI for monitoring progress and ensures that all tests pass before
any code is committed.

Inspired by Matt Pocock's Ralph and Anthropic's research on effective agent harnesses.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// exitWithError prints an error message and exits
func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
