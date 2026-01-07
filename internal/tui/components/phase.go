package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Phase represents a phase in the three-phase loop
type Phase string

const (
	PhasePlanning   Phase = "planning"
	PhaseValidating Phase = "validating"
	PhaseExecuting  Phase = "executing"
	PhaseComplete   Phase = "complete"
	PhaseNone       Phase = ""
)

// PhaseIndicator shows the current phase in the PLAN -> VALIDATE -> EXECUTE loop
type PhaseIndicator struct {
	CurrentPhase Phase
	Width        int
}

// NewPhaseIndicator creates a new phase indicator
func NewPhaseIndicator() *PhaseIndicator {
	return &PhaseIndicator{
		CurrentPhase: PhaseNone,
		Width:        60,
	}
}

// SetPhase updates the current phase
func (p *PhaseIndicator) SetPhase(phase Phase) {
	p.CurrentPhase = phase
}

// Render returns the phase indicator as a string
func (p *PhaseIndicator) Render() string {
	if p.CurrentPhase == PhaseNone {
		return ""
	}

	// Colors
	activeColor := lipgloss.Color("42")   // Green
	pendingColor := lipgloss.Color("245") // Gray
	completeColor := lipgloss.Color("99") // Purple

	// Styles
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(activeColor).
		Bold(true).
		Padding(0, 1)

	pendingStyle := lipgloss.NewStyle().
		Foreground(pendingColor).
		Padding(0, 1)

	completeStyle := lipgloss.NewStyle().
		Foreground(completeColor).
		Bold(true).
		Padding(0, 1)

	arrowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// Determine state of each phase
	var planStyle, validateStyle, executeStyle lipgloss.Style

	switch p.CurrentPhase {
	case PhasePlanning:
		planStyle = activeStyle
		validateStyle = pendingStyle
		executeStyle = pendingStyle
	case PhaseValidating:
		planStyle = completeStyle
		validateStyle = activeStyle
		executeStyle = pendingStyle
	case PhaseExecuting:
		planStyle = completeStyle
		validateStyle = completeStyle
		executeStyle = activeStyle
	case PhaseComplete:
		planStyle = completeStyle
		validateStyle = completeStyle
		executeStyle = completeStyle
	default:
		planStyle = pendingStyle
		validateStyle = pendingStyle
		executeStyle = pendingStyle
	}

	// Build the indicator
	var parts []string

	parts = append(parts, planStyle.Render("PLAN"))
	parts = append(parts, arrowStyle.Render(" -> "))
	parts = append(parts, validateStyle.Render("VALIDATE"))
	parts = append(parts, arrowStyle.Render(" -> "))
	parts = append(parts, executeStyle.Render("EXECUTE"))

	return strings.Join(parts, "")
}
