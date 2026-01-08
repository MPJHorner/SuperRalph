package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/orchestrator"
	"github.com/mpjhorner/superralph/internal/prd"
)

// DashboardState represents the build run state for the dashboard
type DashboardState string

const (
	DashboardStateIdle     DashboardState = "idle"
	DashboardStateRunning  DashboardState = "running"
	DashboardStatePaused   DashboardState = "paused"
	DashboardStateComplete DashboardState = "complete"
	DashboardStateError    DashboardState = "error"
)

// Dashboard displays the main build status including progress, phase, and current activity
type Dashboard struct {
	// PRD data
	PRDName  string
	PRDPath  string
	PRDStats prd.PRDStats

	// Build state
	State            DashboardState
	CurrentIteration int
	MaxIterations    int
	CurrentFeature   *prd.Feature
	StartTime        time.Time
	ErrorMsg         string
	RetryCount       int
	MaxRetries       int

	// Phase and step tracking
	CurrentPhase    Phase
	CurrentStep     orchestrator.Step
	CurrentActivity string

	// UI components (sub-components)
	PhaseIndicator *PhaseIndicator
	StepIndicator  *StepIndicator
	ActionPanel    *ActionPanel

	// Dimensions
	Width  int
	Height int

	// Styles
	titleStyle     lipgloss.Style
	labelStyle     lipgloss.Style
	valueStyle     lipgloss.Style
	mutedStyle     lipgloss.Style
	successStyle   lipgloss.Style
	errorStyle     lipgloss.Style
	warningStyle   lipgloss.Style
	highlightStyle lipgloss.Style
}

// NewDashboard creates a new dashboard component
func NewDashboard(width, height int) *Dashboard {
	return &Dashboard{
		State:          DashboardStateIdle,
		MaxRetries:     3,
		CurrentPhase:   PhaseNone,
		CurrentStep:    orchestrator.StepIdle,
		PhaseIndicator: NewPhaseIndicator(),
		StepIndicator:  NewStepIndicator(),
		ActionPanel:    NewActionPanel(width, 8),
		Width:          width,
		Height:         height,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		labelStyle: lipgloss.NewStyle().
			Bold(true),
		valueStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),
		mutedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
		highlightStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true),
	}
}

// SetPRD updates the PRD information displayed
func (d *Dashboard) SetPRD(p *prd.PRD, path string) {
	d.PRDName = p.Name
	d.PRDPath = path
	d.PRDStats = p.Stats()
}

// SetState sets the current build state
func (d *Dashboard) SetState(state DashboardState) {
	d.State = state
	if state == DashboardStateRunning && d.StartTime.IsZero() {
		d.StartTime = time.Now()
	}
}

// SetIteration sets the current iteration info
func (d *Dashboard) SetIteration(current, maxIter int) {
	d.CurrentIteration = current
	d.MaxIterations = maxIter
}

// SetFeature sets the current feature being worked on
func (d *Dashboard) SetFeature(f *prd.Feature) {
	d.CurrentFeature = f
}

// SetPhase sets the current phase
func (d *Dashboard) SetPhase(phase Phase) {
	d.CurrentPhase = phase
	d.PhaseIndicator.SetPhase(phase)
}

// SetStep sets the current step
func (d *Dashboard) SetStep(step orchestrator.Step) {
	d.CurrentStep = step
	d.StepIndicator.SetStep(step)
}

// SetActivity sets the current activity description
func (d *Dashboard) SetActivity(activity string) {
	d.CurrentActivity = activity
}

// SetError sets the error message
func (d *Dashboard) SetError(msg string) {
	d.ErrorMsg = msg
}

// SetRetry sets the retry count
func (d *Dashboard) SetRetry(count, maxRetries int) {
	d.RetryCount = count
	d.MaxRetries = maxRetries
}

// UpdateStats updates the PRD stats
func (d *Dashboard) UpdateStats(stats prd.PRDStats) {
	d.PRDStats = stats
}

// Render renders the dashboard
func (d *Dashboard) Render() string {
	var b strings.Builder

	// Progress section
	b.WriteString(d.renderProgress())
	b.WriteString("\n")

	// Phase indicator (if in a phase)
	if d.CurrentPhase != PhaseNone {
		b.WriteString(d.labelStyle.Render("Phase: "))
		b.WriteString(d.PhaseIndicator.Render())
		b.WriteString("\n")
	}

	// Step indicator (if running)
	if d.State == DashboardStateRunning {
		b.WriteString(d.labelStyle.Render("Step: "))
		b.WriteString(d.StepIndicator.Render())
		b.WriteString("\n")
	}

	// Status section
	b.WriteString(d.renderStatus())
	b.WriteString("\n")

	// Action panel (if there are actions)
	if len(d.ActionPanel.Actions) > 0 {
		b.WriteString(d.ActionPanel.Render())
	}

	return b.String()
}

// renderProgress renders the progress section with bars
func (d *Dashboard) renderProgress() string {
	stats := d.PRDStats
	pb := NewProgressBar(stats.PassingFeatures, stats.TotalFeatures, 40)

	var b strings.Builder
	b.WriteString(d.labelStyle.Render("Progress: "))
	b.WriteString(pb.Render())
	b.WriteString("\n\n")

	// Category breakdown
	b.WriteString(d.mutedStyle.Render("By Category:") + "                    ")
	b.WriteString(d.mutedStyle.Render("By Priority:") + "\n")

	categories := prd.ValidCategories()
	priorities := prd.ValidPriorities()
	maxRows := len(categories)
	if len(priorities) > maxRows {
		maxRows = len(priorities)
	}

	for i := 0; i < maxRows; i++ {
		// Category column
		if i < len(categories) {
			cat := categories[i]
			cs := stats.ByCategory[cat]
			mini := NewMiniProgressBar(cs.Passing, cs.Total, 10)
			b.WriteString(fmt.Sprintf("  %-12s %s %d/%d", cat, mini.Render(), cs.Passing, cs.Total))
		} else {
			b.WriteString(strings.Repeat(" ", 32))
		}

		b.WriteString("    ")

		// Priority column
		if i < len(priorities) {
			pri := priorities[i]
			ps := stats.ByPriority[pri]
			mini := NewMiniProgressBar(ps.Passing, ps.Total, 10)
			b.WriteString(fmt.Sprintf("%-8s %s %d/%d", pri, mini.Render(), ps.Passing, ps.Total))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderStatus renders the status section
func (d *Dashboard) renderStatus() string {
	var b strings.Builder

	// Status badge
	b.WriteString(d.labelStyle.Render("Status: "))
	b.WriteString(d.statusBadge())
	b.WriteString("\n")

	// Iteration info
	if d.MaxIterations > 0 {
		b.WriteString(fmt.Sprintf("Iteration: %d/%d", d.CurrentIteration, d.MaxIterations))
		if d.RetryCount > 0 {
			b.WriteString(d.warningStyle.Render(fmt.Sprintf(" (retry %d/%d)", d.RetryCount, d.MaxRetries)))
		}
		b.WriteString("\n")
	}

	// Current feature
	if d.CurrentFeature != nil {
		b.WriteString(fmt.Sprintf("Feature: %s ", d.highlightStyle.Render(d.CurrentFeature.ID)))
		b.WriteString(fmt.Sprintf("\"%s\"\n", d.CurrentFeature.Description))
	}

	// Current activity
	if d.CurrentActivity != "" && d.State == DashboardStateRunning {
		activityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
		b.WriteString(activityStyle.Render("Activity: " + d.CurrentActivity))
		b.WriteString("\n")
	}

	// Elapsed time
	if !d.StartTime.IsZero() {
		elapsed := time.Since(d.StartTime).Round(time.Second)
		b.WriteString(d.mutedStyle.Render(fmt.Sprintf("Elapsed: %s\n", elapsed)))
	}

	// Error message
	if d.ErrorMsg != "" {
		b.WriteString(d.errorStyle.Render("Error: " + d.ErrorMsg))
		b.WriteString("\n")
	}

	return b.String()
}

// statusBadge returns a styled status badge
func (d *Dashboard) statusBadge() string {
	switch d.State {
	case DashboardStateRunning:
		return lipgloss.NewStyle().
			Background(lipgloss.Color("42")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true).
			Render(" RUNNING ")
	case DashboardStatePaused:
		return lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true).
			Render(" PAUSED ")
	case DashboardStateComplete:
		return lipgloss.NewStyle().
			Background(lipgloss.Color("99")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true).
			Render(" COMPLETE ")
	case DashboardStateError:
		return lipgloss.NewStyle().
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true).
			Render(" ERROR ")
	default:
		return lipgloss.NewStyle().
			Background(lipgloss.Color("245")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Render(" IDLE ")
	}
}

// AddAction adds an action to the action panel
func (d *Dashboard) AddAction(action ActionItem) {
	d.ActionPanel.AddAction(action)
}

// UpdateAction updates an action's status
func (d *Dashboard) UpdateAction(id string, status ActionStatus, output string) {
	d.ActionPanel.UpdateAction(id, status, output)
}

// ClearActions clears all actions
func (d *Dashboard) ClearActions() {
	d.ActionPanel.Clear()
}
