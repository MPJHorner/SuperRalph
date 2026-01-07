package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mpjhorner/superralph/internal/git"
	"github.com/mpjhorner/superralph/internal/notify"
	"github.com/mpjhorner/superralph/internal/orchestrator"
	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/spf13/cobra"
)

var buildDebug bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the Claude agent loop to implement PRD features",
	Long: `Build validates your PRD, then runs Claude in a loop to implement features.

The agent will:
  1. Read the PRD and progress file to understand current state
  2. Pick the highest-priority incomplete feature
  3. Implement the feature
  4. Run tests (must pass before committing)
  5. Update prd.json and progress.txt
  6. Commit changes
  7. Repeat until all features pass

Tests MUST pass before any commit. This is non-negotiable.`,
	Run: runBuild,
}

func init() {
	buildCmd.Flags().BoolVar(&buildDebug, "debug", false, "Show Claude's thinking process")
	rootCmd.AddCommand(buildCmd)
}

// Styles for the build session
var (
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	fileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	cmdStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("171"))
)

func runBuild(cmd *cobra.Command, args []string) {
	// Check if prd.json exists
	if !prd.ExistsInCurrentDir() {
		fmt.Println(errorStyle.Render("✗") + " prd.json not found in current directory")
		fmt.Println(dimStyle.Render("  Run 'superralph plan' to create one"))
		os.Exit(1)
	}

	// Load the PRD
	p, err := prd.LoadFromCurrentDir()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to load prd.json")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	// Validate the PRD
	result := prd.Validate(p)
	if !result.Valid {
		fmt.Println(errorStyle.Render("✗") + " prd.json has validation errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  %s %s\n", errorStyle.Render("•"), e.Error())
		}
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("✓") + " PRD validated: " + p.Name)
	stats := p.Stats()
	fmt.Printf("  %d/%d features passing\n\n", stats.PassingFeatures, stats.TotalFeatures)

	// Check if already complete
	if p.IsComplete() {
		fmt.Println(successStyle.Render("✓") + " All features already complete!")
		os.Exit(0)
	}

	// Ensure git repo exists
	created, err := git.EnsureRepoCurrentDir()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to initialize git repository")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}
	if created {
		fmt.Println(successStyle.Render("✓") + " Initialized git repository")
	}

	// Prompt for confirmation
	var iterationsStr string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Maximum iterations?").
				Description("Safety limit for agent loops (default: 50)").
				Placeholder("50").
				Value(&iterationsStr),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Println("Cancelled")
		os.Exit(0)
	}

	maxIterations := 50
	if iterationsStr != "" {
		if n, err := strconv.Atoi(iterationsStr); err == nil && n > 0 {
			maxIterations = n
		}
	}

	fmt.Println()
	fmt.Println(boldStyle.Render("Starting Build Session"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("Max iterations: %d | Test command: %s", maxIterations, p.TestCommand)))
	fmt.Println(dimStyle.Render("Press Ctrl+C to cancel at any time."))
	fmt.Println()

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to get current directory")
		os.Exit(1)
	}

	// Track state for display
	var currentFeature string
	var currentPhase string

	// Create the orchestrator
	orch := orchestrator.New(cwd).
		SetDebug(buildDebug).
		OnMessage(func(role, content string) {
			if role == "assistant" && content != "" {
				fmt.Println()
				fmt.Println(content)
			}
		}).
		OnThinking(func(thinking string) {
			if buildDebug {
				fmt.Println()
				fmt.Println(thinkingStyle.Render("Thinking: " + thinking))
			}
		}).
		OnDebug(func(msg string) {
			if buildDebug {
				fmt.Println(dimStyle.Render("[debug] " + msg))
			}
		}).
		OnAction(func(action orchestrator.Action, params orchestrator.ActionParams) {
			switch action {
			case orchestrator.ActionReadFiles:
				for _, path := range params.Paths {
					fmt.Println(dimStyle.Render("  Reading: ") + fileStyle.Render(path))
				}
			case orchestrator.ActionWriteFile:
				fmt.Println(dimStyle.Render("  Writing: ") + fileStyle.Render(params.Path))
			case orchestrator.ActionRunCommand:
				fmt.Println(dimStyle.Render("  Running: ") + cmdStyle.Render(params.Command))
			case orchestrator.ActionDone:
				fmt.Println()
				fmt.Println(successStyle.Render("✓") + " Build complete!")
			}
		}).
		OnState(func(state any) {
			if bs, ok := state.(*orchestrator.BuildState); ok {
				if bs.Phase != currentPhase {
					currentPhase = bs.Phase
					fmt.Println()
					fmt.Println(phaseStyle.Render("Phase: " + currentPhase))
				}
				if bs.CurrentFeature != currentFeature {
					currentFeature = bs.CurrentFeature
					fmt.Println(dimStyle.Render("Feature: ") + boldStyle.Render(currentFeature))
				}
			}
		})

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nCancelling build...")
		cancel()
	}()

	// Run the build session
	err = orch.RunBuild(ctx)
	if err != nil {
		if ctx.Err() != nil {
			fmt.Println()
			fmt.Println(warnStyle.Render("⚠") + " Build cancelled")
			notify.Send("SuperRalph", "Build cancelled by user")
			os.Exit(0)
		}
		fmt.Println()
		fmt.Println(errorStyle.Render("✗") + " Build failed")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		notify.SendError("Build failed: " + err.Error())
		os.Exit(1)
	}

	// Check final state
	p, err = prd.LoadFromCurrentDir()
	if err == nil {
		stats := p.Stats()
		if p.IsComplete() {
			fmt.Println()
			fmt.Println(successStyle.Render("✓") + " All features complete!")
			notify.SendSuccess("PRD complete! All features implemented.")
		} else {
			fmt.Println()
			fmt.Printf("%s %d/%d features complete\n",
				warnStyle.Render("⚠"),
				stats.PassingFeatures,
				stats.TotalFeatures)
			fmt.Println(dimStyle.Render("  Run 'superralph build' again to continue"))
			notify.Send("SuperRalph", fmt.Sprintf("Build paused: %d/%d features complete", stats.PassingFeatures, stats.TotalFeatures))
		}
	}
}
