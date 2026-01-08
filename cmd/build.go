package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/mpjhorner/superralph/internal/git"
	"github.com/mpjhorner/superralph/internal/notify"
	"github.com/mpjhorner/superralph/internal/orchestrator"
	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/tui"
	"github.com/mpjhorner/superralph/internal/tui/components"
)

var (
	buildDebug  bool
	buildResume bool
)

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

Tests MUST pass before any commit. This is non-negotiable.

Graceful Shutdown:
  Press Ctrl+C to gracefully stop the build. The current action will complete
  before saving state. Use --resume to continue from where you left off.`,
	Run: runBuild,
}

func init() {
	buildCmd.Flags().BoolVar(&buildDebug, "debug", false, "Show Claude's thinking process")
	buildCmd.Flags().BoolVar(&buildResume, "resume", false, "Resume from saved state after interruption")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) {
	// Check if prd.json exists
	if !prd.ExistsInCurrentDir() {
		fmt.Println(errorStyle.Render("x") + " prd.json not found in current directory")
		fmt.Println(dimStyle.Render("  Run 'superralph plan' to create one"))
		os.Exit(1)
	}

	// Load the PRD
	p, err := prd.LoadFromCurrentDir()
	if err != nil {
		fmt.Println(errorStyle.Render("x") + " Failed to load prd.json")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	// Validate the PRD
	result := prd.Validate(p)
	if !result.Valid {
		fmt.Println(errorStyle.Render("x") + " prd.json has validation errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  %s %s\n", errorStyle.Render("*"), e.Error())
		}
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("ok") + " PRD validated: " + p.Name)
	stats := p.Stats()
	fmt.Printf("  %d/%d features passing\n\n", stats.PassingFeatures, stats.TotalFeatures)

	// Check if already complete
	if p.IsComplete() {
		fmt.Println(successStyle.Render("ok") + " All features already complete!")
		os.Exit(0)
	}

	// Ensure git repo exists
	created, err := git.EnsureRepoCurrentDir()
	if err != nil {
		fmt.Println(errorStyle.Render("x") + " Failed to initialize git repository")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}
	if created {
		fmt.Println(successStyle.Render("ok") + " Initialized git repository")
	}

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render("x") + " Failed to get current directory")
		os.Exit(1)
	}

	// Check for resume state
	var startIteration = 1
	var resumeFeature string
	var maxIterations = 50

	tempOrch := orchestrator.New(cwd)
	resumeState, err := tempOrch.LoadResumeState()
	if err != nil {
		fmt.Println(errorStyle.Render("x") + " Failed to load resume state")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	if resumeState != nil {
		if buildResume {
			// Resume from saved state
			fmt.Println(successStyle.Render("ok") + " Found saved state from " + resumeState.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Resuming from iteration %d (feature: %s)\n\n", resumeState.Iteration, resumeState.CurrentFeature)
			startIteration = resumeState.Iteration
			resumeFeature = resumeState.CurrentFeature
			maxIterations = resumeState.TotalIterations
		} else {
			// State exists but --resume not specified
			fmt.Println(dimStyle.Render("  Note: Previous build was interrupted. Use --resume to continue."))
			fmt.Printf("  Saved state: iteration %d, feature %s\n\n", resumeState.Iteration, resumeState.CurrentFeature)
		}
	} else if buildResume {
		// --resume specified but no state found
		fmt.Println(dimStyle.Render("  No saved state found. Starting fresh build."))
	}

	// Prompt for iterations only if not resuming
	if !buildResume || resumeState == nil {
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
			fmt.Println("Canceled")
			os.Exit(0)
		}

		if iterationsStr != "" {
			if n, err := strconv.Atoi(iterationsStr); err == nil && n > 0 {
				maxIterations = n
			}
		}
	}

	// Create the TUI model
	model := tui.NewModel(p, "prd.json", maxIterations)
	model.SetDebugMode(buildDebug)

	// Create the Bubble Tea program with alternate screen buffer
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Track current iteration for display
	currentIteration := 0

	// Create the orchestrator with callbacks that send messages to the TUI
	orch := orchestrator.New(cwd).
		SetDebug(buildDebug).
		OnMessage(func(role, content string) {
			if role == "assistant" && content != "" {
				program.Send(tui.LogMsg(content))
			}
		}).
		OnThinking(func(thinking string) {
			if buildDebug {
				program.Send(tui.TypedLogMsg{Type: components.LogTypeInfo, Content: "[thinking] " + thinking})
			}
		}).
		OnDebug(func(msg string) {
			if buildDebug {
				program.Send(tui.TypedLogMsg{Type: components.LogTypeInfo, Content: "[debug] " + msg})
			}
		}).
		OnOutput(func(line string) {
			program.Send(tui.LogMsg(line))
		}).
		OnTypedOutput(func(outputType orchestrator.OutputType, content string) {
			// Map orchestrator output types to TUI log entry types
			var logType components.LogEntryType
			switch outputType {
			case orchestrator.OutputText:
				logType = components.LogTypeText
			case orchestrator.OutputToolUse:
				logType = components.LogTypeToolUse
			case orchestrator.OutputToolInput:
				logType = components.LogTypeToolInput
			case orchestrator.OutputToolResult:
				logType = components.LogTypeToolResult
			case orchestrator.OutputPhase:
				logType = components.LogTypePhase
			case orchestrator.OutputSuccess:
				logType = components.LogTypeSuccess
			case orchestrator.OutputError:
				logType = components.LogTypeError
			case orchestrator.OutputInfo:
				logType = components.LogTypeInfo
			default:
				logType = components.LogTypeText
			}
			program.Send(tui.TypedLogMsg{Type: logType, Content: content})
		}).
		OnActivity(func(activity string) {
			program.Send(tui.ActivityMsg(activity))
		}).
		OnStep(func(step orchestrator.Step) {
			program.Send(tui.StepChangeMsg{Step: step})
		}).
		OnAction(func(action orchestrator.Action, params orchestrator.ActionParams) {
			switch action {
			case orchestrator.ActionReadFiles:
				for _, path := range params.Paths {
					program.Send(tui.ActionAddMsg{
						Action: components.ActionItem{
							ID:          fmt.Sprintf("read-%s", path),
							Type:        "read",
							Description: "Reading: " + path,
							Status:      components.StatusRunning,
						},
					})
				}
			case orchestrator.ActionWriteFile:
				program.Send(tui.ActionAddMsg{
					Action: components.ActionItem{
						ID:          fmt.Sprintf("write-%s", params.Path),
						Type:        "write",
						Description: "Writing: " + params.Path,
						Status:      components.StatusRunning,
					},
				})
			case orchestrator.ActionRunCommand:
				program.Send(tui.ActionAddMsg{
					Action: components.ActionItem{
						ID:          fmt.Sprintf("cmd-%d", currentIteration),
						Type:        "command",
						Description: "Running: " + params.Command,
						Status:      components.StatusRunning,
					},
				})
			case orchestrator.ActionDone:
				program.Send(tui.LogMsg("Build complete!"))
			}
		}).
		OnState(func(state any) {
			if bs, ok := state.(*orchestrator.BuildState); ok {
				// Update iteration
				if bs.Iteration != currentIteration {
					currentIteration = bs.Iteration

					// Find the feature in the PRD
					var feature *prd.Feature
					if bs.CurrentFeature != "" {
						// Reload PRD to get current state
						if currentPRD, err := prd.LoadFromCurrentDir(); err == nil {
							for i := range currentPRD.Features {
								if currentPRD.Features[i].ID == bs.CurrentFeature {
									feature = &currentPRD.Features[i]
									break
								}
							}
							// Also send PRD update
							program.Send(tui.PRDUpdateMsg{PRD: currentPRD, Stats: currentPRD.Stats()})
						}
					}

					program.Send(tui.IterationStartMsg{
						Iteration: bs.Iteration,
						Feature:   feature,
					})
				}

				// Map orchestrator phase to TUI phase
				var tuiPhase components.Phase
				switch bs.Phase {
				case "planning":
					tuiPhase = components.PhasePlanning
				case "validating":
					tuiPhase = components.PhaseValidating
				case "executing":
					tuiPhase = components.PhaseExecuting
				case "complete":
					tuiPhase = components.PhaseComplete
				default:
					tuiPhase = components.PhaseNone
				}
				program.Send(tui.PhaseChangeMsg{Phase: tuiPhase})

				// Send step change if available
				if bs.CurrentStep != "" {
					program.Send(tui.StepChangeMsg{Step: bs.CurrentStep})
				}
			}
		})

	// Set up TUI callbacks
	model.OnQuit = func() {
		cancel()
	}
	model.OnPause = func() {
		// Could implement pause logic here
		program.Send(tui.LogMsg("Build paused"))
	}
	model.OnResume = func() {
		// Could implement resume logic here
		program.Send(tui.LogMsg("Build resumed"))
	}
	model.OnDebug = func(enabled bool) {
		orch.SetDebug(enabled)
		if enabled {
			program.Send(tui.LogMsg("Debug mode enabled"))
		} else {
			program.Send(tui.LogMsg("Debug mode disabled"))
		}
	}

	// Run the orchestrator in a goroutine
	go func() {
		// Handle signals
		go func() {
			<-sigChan
			cancel()
			program.Send(tui.LogMsg("Canceling build..."))
			program.Send(tui.BuildCompleteMsg{Success: false, Error: fmt.Errorf("canceled by user")})
		}()

		// Set state to running
		program.Send(tui.StateChangeMsg(tui.StateRunning))

		// Build config with resume support
		buildConfig := orchestrator.BuildConfig{
			MaxIterations:          maxIterations,
			DelayBetweenIterations: 3 * time.Second,
			StartIteration:         startIteration,
			ResumeFeature:          resumeFeature,
		}

		// Run the build with config
		err := orch.RunBuildWithConfig(ctx, buildConfig)

		if err != nil {
			if ctx.Err() != nil {
				// Canceled
				program.Send(tui.LogMsg("Build canceled"))
				program.Send(tui.BuildCompleteMsg{Success: false, Error: nil})
				_ = notify.Send("SuperRalph", "Build canceled by user")
			} else {
				// Error
				program.Send(tui.LogMsg("Build failed: " + err.Error()))
				program.Send(tui.BuildCompleteMsg{Success: false, Error: err})
				_ = notify.SendError("Build failed: " + err.Error())
			}
		} else {
			// Success - reload PRD to check status
			p, err := prd.LoadFromCurrentDir()
			if err == nil {
				program.Send(tui.PRDUpdateMsg{PRD: p, Stats: p.Stats()})
				if p.IsComplete() {
					program.Send(tui.LogMsg("All features complete!"))
					_ = notify.SendSuccess("PRD complete! All features implemented.")
				} else {
					stats := p.Stats()
					program.Send(tui.LogMsg(fmt.Sprintf("%d/%d features complete", stats.PassingFeatures, stats.TotalFeatures)))
					_ = notify.Send("SuperRalph", fmt.Sprintf("Build paused: %d/%d features complete", stats.PassingFeatures, stats.TotalFeatures))
				}
			}
			program.Send(tui.BuildCompleteMsg{Success: true, Error: nil})
		}
	}()

	// Run the TUI (blocks until quit)
	if _, err := program.Run(); err != nil {
		fmt.Println(errorStyle.Render("x") + " TUI error: " + err.Error())
		os.Exit(1)
	}
}
