package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/mpjhorner/superralph/internal/agent"
	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Interactively create a new PRD with Claude's help",
	Long: `Plan launches an interactive session with Claude to help you create a PRD.

Claude will:
  1. Ask what you're building
  2. Help you think through features
  3. Ask clarifying questions
  4. Create a well-structured prd.json

If a prd.json already exists, you'll be asked to confirm before replacing it.`,
	Run: runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) {
	// Check if prd.json already exists
	if prd.ExistsInCurrentDir() {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("prd.json already exists").
					Description("Do you want to replace it with a new PRD?").
					Affirmative("Yes, start fresh").
					Negative("No, cancel").
					Value(&confirm),
			),
		)

		err := form.Run()
		if err != nil || !confirm {
			fmt.Println("Cancelled")
			os.Exit(0)
		}

		// Back up existing PRD
		existingPRD, err := prd.LoadFromCurrentDir()
		if err == nil {
			backupPath := "prd.json.backup"
			if err := prd.Save(existingPRD, backupPath); err == nil {
				fmt.Println(dimStyle.Render("  Backed up existing PRD to " + backupPath))
			}
		}
	}

	fmt.Println()
	fmt.Println(boldStyle.Render("Starting interactive PRD planning session..."))
	fmt.Println(dimStyle.Render("Claude will help you define your project features."))
	fmt.Println(dimStyle.Render("When done, Claude will create the prd.json file."))
	fmt.Println()

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to get current directory")
		os.Exit(1)
	}

	// Create the agent runner
	runner := agent.NewRunner(cwd)

	// Build the planning prompt
	prompt := agent.BuildPlanPrompt()

	// Run in interactive mode
	ctx := context.Background()
	err = runner.RunInteractive(ctx, prompt)
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗") + " Planning session ended with error")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	fmt.Println()

	// Check if prd.json was created
	if prd.ExistsInCurrentDir() {
		p, err := prd.LoadFromCurrentDir()
		if err != nil {
			fmt.Println(warnStyle.Render("⚠") + " prd.json was created but has errors")
			fmt.Println(dimStyle.Render("  Run 'superralph validate' to see details"))
			os.Exit(1)
		}

		// Validate
		result := prd.Validate(p)
		if !result.Valid {
			fmt.Println(warnStyle.Render("⚠") + " prd.json was created but has validation errors:")
			for _, e := range result.Errors {
				fmt.Printf("  %s %s\n", errorStyle.Render("•"), e.Error())
			}
			fmt.Println()
			fmt.Println(dimStyle.Render("  You may need to manually fix the PRD or run 'superralph plan' again"))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("✓") + " prd.json created successfully!")
		fmt.Println()

		stats := p.Stats()
		fmt.Printf("  %s %s\n", boldStyle.Render("Project:"), p.Name)
		fmt.Printf("  %s %s\n", boldStyle.Render("Test Command:"), p.TestCommand)
		fmt.Printf("  %s %d features defined\n", boldStyle.Render("Features:"), stats.TotalFeatures)
		fmt.Println()
		fmt.Println(dimStyle.Render("  Run 'superralph build' to start implementing features"))
	} else {
		fmt.Println(warnStyle.Render("⚠") + " No prd.json was created")
		fmt.Println(dimStyle.Render("  Make sure Claude creates the prd.json file during the session"))
	}
}
