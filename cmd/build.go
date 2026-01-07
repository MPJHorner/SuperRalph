package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/mpjhorner/superralph/internal/agent"
	"github.com/mpjhorner/superralph/internal/git"
	"github.com/mpjhorner/superralph/internal/notify"
	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/tui"
	"github.com/spf13/cobra"
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
  7. Repeat until all features pass or iterations exhausted

Tests MUST pass before any commit. This is non-negotiable.`,
	Run: runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
}

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

	// Prompt for number of iterations
	var iterationsStr string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("How many iterations?").
				Description("Number of agent loops to run (default: 10)").
				Placeholder("10").
				Value(&iterationsStr),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Println("Cancelled")
		os.Exit(0)
	}

	iterations := 10
	if iterationsStr != "" {
		if n, err := strconv.Atoi(iterationsStr); err == nil && n > 0 {
			iterations = n
		}
	}

	fmt.Printf("\nStarting build with %d iterations...\n\n", iterations)

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to get current directory")
		os.Exit(1)
	}

	// Get PRD path
	prdPath, _ := prd.GetPath()

	// Create the TUI model
	model := tui.NewModel(p, prdPath, iterations)

	// Create the agent runner
	runner := agent.NewRunner(cwd)

	// Create a channel to communicate between TUI and agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build model with callbacks
	buildModel := &buildTUIModel{
		Model:      model,
		runner:     runner,
		ctx:        ctx,
		cancel:     cancel,
		iterations: iterations,
		maxRetries: 3,
		cwd:        cwd,
	}

	// Set up callbacks
	model.OnQuit = func() {
		cancel()
	}
	model.OnPause = func() {
		runner.Pause()
	}
	model.OnResume = func() {
		runner.Resume()
	}

	buildModel.Model = model

	// Run the TUI
	program := tea.NewProgram(
		buildModel,
		tea.WithAltScreen(),
	)

	// Start the agent loop in background
	go buildModel.runAgentLoop(program)

	if _, err := program.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

type buildTUIModel struct {
	tui.Model
	runner     *agent.Runner
	ctx        context.Context
	cancel     context.CancelFunc
	iterations int
	maxRetries int
	cwd        string
	completed  bool
	errored    bool
}

func (m *buildTUIModel) Init() tea.Cmd {
	return m.Model.Init()
}

func (m *buildTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit
		}
	}

	// Handle the underlying model update
	updated, cmd := m.Model.Update(msg)
	if model, ok := updated.(tui.Model); ok {
		m.Model = model
	}
	return m, cmd
}

func (m *buildTUIModel) View() string {
	return m.Model.View()
}

func (m *buildTUIModel) runAgentLoop(program *tea.Program) {
	m.Model.State = tui.StateRunning
	program.Send(tui.StateChangeMsg(tui.StateRunning))

	for i := 1; i <= m.iterations; i++ {
		// Check if cancelled
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		// Check if paused
		for m.runner.IsPaused() {
			select {
			case <-m.ctx.Done():
				return
			default:
			}
		}

		// Reload PRD to get latest state
		p, err := prd.LoadFromCurrentDir()
		if err != nil {
			program.Send(tui.LogMsg("Error loading PRD: " + err.Error()))
			continue
		}

		// Check if complete
		if p.IsComplete() {
			program.Send(tui.LogMsg("All features complete!"))
			program.Send(tui.StateChangeMsg(tui.StateComplete))
			m.completed = true
			notify.SendSuccess("PRD complete after " + fmt.Sprintf("%d", i-1) + " iterations")
			return
		}

		// Get next feature
		feature := p.NextFeature()
		if feature == nil {
			program.Send(tui.LogMsg("No more features to implement"))
			program.Send(tui.StateChangeMsg(tui.StateComplete))
			return
		}

		// Update iteration info
		program.Send(tui.IterationStartMsg{
			Iteration: i,
			Feature:   feature,
		})
		program.Send(tui.LogMsg(fmt.Sprintf("=== Iteration %d/%d ===", i, m.iterations)))
		program.Send(tui.LogMsg(fmt.Sprintf("Working on: %s - %s", feature.ID, feature.Description)))

		// Build prompt
		prompt := agent.BuildPrompt(p, i)

		// Run with retries
		success := false
		for retry := 0; retry < m.maxRetries; retry++ {
			if retry > 0 {
				program.Send(tui.LogMsg(fmt.Sprintf("Retry %d/%d...", retry, m.maxRetries)))
			}

			// Set up output handler
			m.runner.ClearOutput()
			m.runner.OnOutput(func(line string) {
				program.Send(tui.LogMsg(line))
			})

			// Run the agent
			err := m.runner.Run(m.ctx, prompt)

			output := m.runner.GetOutput()

			// Check for completion signal
			if agent.ContainsCompletionSignal(output) {
				program.Send(tui.LogMsg("Received completion signal"))
				program.Send(tui.StateChangeMsg(tui.StateComplete))
				m.completed = true
				notify.SendSuccess("PRD complete!")
				return
			}

			if err == nil {
				success = true
				break
			}

			program.Send(tui.LogMsg(fmt.Sprintf("Error: %v", err)))
		}

		if !success {
			program.Send(tui.LogMsg(fmt.Sprintf("Failed after %d retries", m.maxRetries)))
			program.Send(tui.StateChangeMsg(tui.StateError))
			m.errored = true
			notify.SendError("Build failed after " + fmt.Sprintf("%d", m.maxRetries) + " retries")
			return
		}

		// Reload PRD to update stats
		p, err = prd.LoadFromCurrentDir()
		if err == nil {
			program.Send(tui.PRDUpdateMsg{
				PRD:   p,
				Stats: p.Stats(),
			})
		}

		program.Send(tui.IterationCompleteMsg{
			Iteration: i,
			Success:   true,
		})
	}

	// Finished all iterations
	program.Send(tui.LogMsg(fmt.Sprintf("Completed %d iterations", m.iterations)))

	// Check final state
	p, err := prd.LoadFromCurrentDir()
	if err == nil && p.IsComplete() {
		program.Send(tui.StateChangeMsg(tui.StateComplete))
		notify.SendSuccess("PRD complete!")
	} else {
		program.Send(tui.StateChangeMsg(tui.StateIdle))
		stats := p.Stats()
		notify.Send("SuperRalph", fmt.Sprintf("Build finished: %d/%d features complete", stats.PassingFeatures, stats.TotalFeatures))
	}
}
