package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/progress"
	"github.com/mpjhorner/superralph/internal/tui"
)

const refreshInterval = 2 * time.Second

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live status of PRD progress",
	Long: `Status displays a live-updating TUI showing the current state of your PRD.

It shows:
  - Overall progress (features passing/total)
  - Breakdown by category and priority
  - Recent activity from progress.txt
  - Current working state`,
	Run: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
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
		fmt.Println(errorStyle.Render("✗") + " prd.json has validation errors")
		fmt.Println(dimStyle.Render("  Run 'superralph validate' for details"))
		os.Exit(1)
	}

	// Get PRD path
	prdPath, _ := prd.GetPath()

	// Create status model (read-only mode, no iterations)
	model := tui.NewModel(p, prdPath, 0)
	model.State = tui.StateIdle

	// Try to load recent progress
	progressContent, err := progress.ReadFromCurrentDir()
	if err == nil && progressContent != "" {
		// Add last few lines to log view
		lines := splitLines(progressContent)
		// Show last 20 lines
		start := len(lines) - 20
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			model.LogView.AddLine(line)
		}
	} else {
		model.LogView.AddLine("No progress.txt found - run 'superralph build' to start")
	}

	// Run the TUI with auto-refresh
	program := tea.NewProgram(
		statusModel{Model: model},
		tea.WithAltScreen(),
	)

	if _, err := program.Run(); err != nil {
		fmt.Println("Error running status TUI:", err)
		os.Exit(1)
	}
}

// statusModel wraps the TUI model for status-only mode
type statusModel struct {
	tui.Model
}

func (m statusModel) Init() tea.Cmd {
	return tea.Batch(
		m.Model.Init(),
		refreshTick(),
	)
}

func refreshTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

type refreshMsg struct{}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		// Reload PRD to get updated stats
		p, err := prd.LoadFromCurrentDir()
		if err == nil {
			m.PRD = p
			m.PRDStats = p.Stats()
		}
		return m, refreshTick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			// Manual refresh
			p, err := prd.LoadFromCurrentDir()
			if err == nil {
				m.PRD = p
				m.PRDStats = p.Stats()
			}
		}
	}

	// Handle other messages
	updated, cmd := m.Model.Update(msg)
	if model, ok := updated.(tui.Model); ok {
		m.Model = model
	}
	return m, cmd
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
