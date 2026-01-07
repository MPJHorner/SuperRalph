package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ActionStatus represents the status of an action
type ActionStatus string

const (
	StatusPending  ActionStatus = "pending"
	StatusRunning  ActionStatus = "running"
	StatusDone     ActionStatus = "done"
	StatusFailed   ActionStatus = "failed"
	StatusSkipped  ActionStatus = "skipped"
)

// ActionItem represents a single action in the action panel
type ActionItem struct {
	ID          string
	Type        string // e.g., "read", "write", "command"
	Description string
	Status      ActionStatus
	Output      string // truncated output for display
}

// ActionPanel displays current and parallel actions with their status
type ActionPanel struct {
	Actions     []ActionItem
	MaxActions  int // maximum actions to display
	Width       int
	Height      int
	Title       string
	ShowOutput  bool
}

// NewActionPanel creates a new action panel
func NewActionPanel(width, height int) *ActionPanel {
	return &ActionPanel{
		Actions:    make([]ActionItem, 0),
		MaxActions: 10,
		Width:      width,
		Height:     height,
		Title:      "Actions",
		ShowOutput: true,
	}
}

// AddAction adds an action to the panel
func (p *ActionPanel) AddAction(action ActionItem) {
	p.Actions = append(p.Actions, action)
	if len(p.Actions) > p.MaxActions {
		p.Actions = p.Actions[len(p.Actions)-p.MaxActions:]
	}
}

// UpdateAction updates the status of an action by ID
func (p *ActionPanel) UpdateAction(id string, status ActionStatus, output string) {
	for i := range p.Actions {
		if p.Actions[i].ID == id {
			p.Actions[i].Status = status
			if output != "" {
				p.Actions[i].Output = output
			}
			return
		}
	}
}

// Clear removes all actions
func (p *ActionPanel) Clear() {
	p.Actions = make([]ActionItem, 0)
}

// GetPendingCount returns the number of pending actions
func (p *ActionPanel) GetPendingCount() int {
	count := 0
	for _, a := range p.Actions {
		if a.Status == StatusPending {
			count++
		}
	}
	return count
}

// GetRunningCount returns the number of running actions
func (p *ActionPanel) GetRunningCount() int {
	count := 0
	for _, a := range p.Actions {
		if a.Status == StatusRunning {
			count++
		}
	}
	return count
}

// statusIcon returns an icon for the status
func statusIcon(status ActionStatus) string {
	switch status {
	case StatusPending:
		return "○"
	case StatusRunning:
		return "◐"
	case StatusDone:
		return "●"
	case StatusFailed:
		return "✗"
	case StatusSkipped:
		return "◌"
	default:
		return "?"
	}
}

// statusStyle returns the style for a status
func statusStyle(status ActionStatus) lipgloss.Style {
	switch status {
	case StatusPending:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	case StatusRunning:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	case StatusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case StatusFailed:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	case StatusSkipped:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	default:
		return lipgloss.NewStyle()
	}
}

// typeIcon returns an icon for the action type
func typeIcon(actionType string) string {
	switch actionType {
	case "read", "read_files":
		return "📖"
	case "write", "write_file":
		return "📝"
	case "command", "run_command", "bash":
		return "⚡"
	case "edit":
		return "✏️"
	case "parallel":
		return "⇶"
	default:
		return "•"
	}
}

// Render returns the action panel as a string
func (p *ActionPanel) Render() string {
	if len(p.Actions) == 0 {
		return ""
	}

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Bold(true)

	// Count running
	running := p.GetRunningCount()
	statusInfo := ""
	if running > 0 {
		statusInfo = fmt.Sprintf(" (%d running)", running)
	}

	sb.WriteString(titleStyle.Render(p.Title + statusInfo))
	sb.WriteString("\n")

	// Calculate visible actions
	displayHeight := p.Height - 2 // Account for title
	if displayHeight < 1 {
		displayHeight = 5
	}

	startIdx := len(p.Actions) - displayHeight
	if startIdx < 0 {
		startIdx = 0
	}
	visibleActions := p.Actions[startIdx:]

	// Render each action
	contentWidth := p.Width - 6 // Account for icon and spacing
	for _, action := range visibleActions {
		style := statusStyle(action.Status)
		icon := statusIcon(action.Status)
		tIcon := typeIcon(action.Type)

		// Truncate description if too long
		desc := action.Description
		if len(desc) > contentWidth {
			desc = desc[:contentWidth-3] + "..."
		}

		line := fmt.Sprintf(" %s %s %s", style.Render(icon), tIcon, desc)
		sb.WriteString(line)
		sb.WriteString("\n")

		// Show output snippet if available and requested
		if p.ShowOutput && action.Output != "" && action.Status == StatusRunning {
			outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			outputLines := strings.Split(action.Output, "\n")
			if len(outputLines) > 2 {
				outputLines = outputLines[:2]
			}
			for _, oLine := range outputLines {
				if len(oLine) > contentWidth-4 {
					oLine = oLine[:contentWidth-7] + "..."
				}
				sb.WriteString("     " + outputStyle.Render(oLine) + "\n")
			}
		}
	}

	// Render in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("245")).
		Width(p.Width - 2)

	return boxStyle.Render(sb.String())
}

// Summary returns a brief summary of action status
func (p *ActionPanel) Summary() string {
	if len(p.Actions) == 0 {
		return ""
	}

	var pending, running, done, failed int
	for _, a := range p.Actions {
		switch a.Status {
		case StatusPending:
			pending++
		case StatusRunning:
			running++
		case StatusDone:
			done++
		case StatusFailed:
			failed++
		}
	}

	var parts []string
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if done > 0 {
		parts = append(parts, fmt.Sprintf("%d done", done))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}

	return strings.Join(parts, ", ")
}
