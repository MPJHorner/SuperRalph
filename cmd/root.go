package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mpjhorner/superralph/internal/version"
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
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip update check for version and update commands
		if cmd.Name() == "version" || cmd.Name() == "update" {
			return
		}
		checkForUpdateInBackground()
	},
}

// updateCheckResult holds the result of an async update check
var updateCheckResult chan *version.GitHubRelease

// Execute runs the root command
func Execute() error {
	// Start update check in background before executing command
	startUpdateCheck()
	return rootCmd.Execute()
}

// startUpdateCheck begins an async update check
func startUpdateCheck() {
	updateCheckResult = make(chan *version.GitHubRelease, 1)
	go func() {
		info, err := version.CheckForUpdate()
		if err == nil && info != nil && version.IsNewer(info) {
			updateCheckResult <- info
		}
		close(updateCheckResult)
	}()
}

// checkForUpdateInBackground displays update message if available
func checkForUpdateInBackground() {
	// Wait briefly for the result (don't block too long)
	select {
	case info := <-updateCheckResult:
		if info != nil {
			fmt.Fprintf(os.Stderr, "\n"+
				"╭────────────────────────────────────────────────╮\n"+
				"│  A new version of superralph is available!    │\n"+
				"│  Current: %-10s  Latest: %-10s     │\n"+
				"│  Run 'superralph update' to upgrade           │\n"+
				"╰────────────────────────────────────────────────╯\n",
				version.Version, info.TagName)
		}
	case <-time.After(500 * time.Millisecond):
		// Don't wait too long, skip the message
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// exitWithError prints an error message and exits
func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
