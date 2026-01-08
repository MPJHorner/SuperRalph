package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/orchestrator"
)

// StepIndicator shows the current step in the iteration
type StepIndicator struct {
	CurrentStep orchestrator.Step
	Width       int

	// Styles
	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	completeStyle lipgloss.Style
}

// NewStepIndicator creates a new step indicator
func NewStepIndicator() *StepIndicator {
	return &StepIndicator{
		CurrentStep: orchestrator.StepIdle,
		Width:       80,
		activeStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42")), // Green
		inactiveStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Muted gray
		completeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")), // Purple
	}
}

// SetStep sets the current step
func (si *StepIndicator) SetStep(step orchestrator.Step) {
	si.CurrentStep = step
}

// Render renders the step indicator
func (si *StepIndicator) Render() string {
	steps := []orchestrator.Step{
		orchestrator.StepReading,
		orchestrator.StepCoding,
		orchestrator.StepTesting,
		orchestrator.StepCommitting,
		orchestrator.StepUpdating,
	}

	var parts []string
	currentIdx := -1

	// Find current step index
	for i, s := range steps {
		if s == si.CurrentStep {
			currentIdx = i
			break
		}
	}

	// If complete, mark all as complete
	if si.CurrentStep == orchestrator.StepComplete {
		currentIdx = len(steps)
	}

	for i, step := range steps {
		var style lipgloss.Style
		if currentIdx == -1 || i > currentIdx {
			// Future step
			style = si.inactiveStyle
		} else if i == currentIdx {
			// Current step
			style = si.activeStyle
		} else {
			// Past step
			style = si.completeStyle
		}

		parts = append(parts, style.Render(step.String()))
	}

	// Join with arrows
	arrow := si.inactiveStyle.Render(" → ")
	return strings.Join(parts, arrow)
}

// RenderCompact renders a compact version showing just the current step
func (si *StepIndicator) RenderCompact() string {
	if si.CurrentStep == orchestrator.StepIdle {
		return si.inactiveStyle.Render("Idle")
	}
	if si.CurrentStep == orchestrator.StepComplete {
		return si.completeStyle.Render("Complete")
	}
	return si.activeStyle.Render(si.CurrentStep.String())
}
