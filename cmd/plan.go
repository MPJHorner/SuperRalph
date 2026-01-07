package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mpjhorner/superralph/internal/orchestrator"
	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/spf13/cobra"
)

var planDebug bool

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Interactively create a new PRD with Claude's help",
	Long: `Plan launches an interactive session with Claude to help you create a PRD.

Claude will:
  1. Ask what you're building
  2. Explore your existing code
  3. Help you think through features
  4. Create a well-structured prd.json

If a prd.json already exists, you'll be asked to confirm before replacing it.`,
	Run: runPlan,
}

func init() {
	planCmd.Flags().BoolVar(&planDebug, "debug", false, "Show Claude's thinking process")
	rootCmd.AddCommand(planCmd)
}

// Styles for the planning session
var (
	claudeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	userStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("79")).Bold(true)
	thinkingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	actionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

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
	fmt.Println(boldStyle.Render("Starting PRD Planning Session"))
	fmt.Println(dimStyle.Render("Claude will ask questions and explore your codebase to create a PRD."))
	fmt.Println(dimStyle.Render("Press Ctrl+C to cancel at any time."))
	fmt.Println()

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to get current directory")
		os.Exit(1)
	}

	// Create the orchestrator
	orch := orchestrator.New(cwd).
		SetDebug(planDebug).
		OnMessage(func(role, content string) {
			if role == "assistant" {
				fmt.Println()
				fmt.Println(claudeStyle.Render("Claude:"))
				fmt.Println(content)
			}
		}).
		OnThinking(func(thinking string) {
			if planDebug {
				fmt.Println()
				fmt.Println(thinkingStyle.Render("Thinking: " + thinking))
			}
		}).
		OnAction(func(action orchestrator.Action, params orchestrator.ActionParams) {
			switch action {
			case orchestrator.ActionReadFiles:
				fmt.Println()
				fmt.Println(actionStyle.Render("Reading files: " + fmt.Sprintf("%v", params.Paths)))
			case orchestrator.ActionWriteFile:
				fmt.Println()
				fmt.Println(actionStyle.Render("Writing file: " + params.Path))
			case orchestrator.ActionDone:
				fmt.Println()
				fmt.Println(successStyle.Render("✓") + " Planning complete!")
			}
		}).
		SetPromptUser(func(question string) (string, error) {
			fmt.Println()
			fmt.Println(userStyle.Render("You:"))
			return orchestrator.DefaultPromptUser("")
		})

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nCancelling...")
		cancel()
	}()

	// Run the planning session
	err = orch.RunPlan(ctx)
	if err != nil {
		if ctx.Err() != nil {
			fmt.Println(warnStyle.Render("⚠") + " Planning session cancelled")
			os.Exit(0)
		}
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
		fmt.Println(dimStyle.Render("  The planning session ended without creating a PRD"))
	}
}
