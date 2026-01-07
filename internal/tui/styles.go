package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	ColorPrimary   = lipgloss.Color("99")  // Purple
	ColorSecondary = lipgloss.Color("39")  // Cyan
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorError     = lipgloss.Color("196") // Red
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorMuted     = lipgloss.Color("245") // Gray
	ColorHighlight = lipgloss.Color("212") // Pink
)

// Base styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginBottom(1)

	// Status indicators
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Text styles
	BoldStyle = lipgloss.NewStyle().Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// Labels
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	LogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)
)

// Progress bar styles
var (
	ProgressBarFilled = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Background(ColorSuccess)

	ProgressBarEmpty = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(lipgloss.Color("238"))

	ProgressBarText = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Status badge styles
func StatusBadge(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().
			Background(ColorSuccess).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true).
			Render(" RUNNING ")
	case "paused":
		return lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true).
			Render(" PAUSED ")
	case "complete":
		return lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true).
			Render(" COMPLETE ")
	case "error":
		return lipgloss.NewStyle().
			Background(ColorError).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true).
			Render(" ERROR ")
	default:
		return lipgloss.NewStyle().
			Background(ColorMuted).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Render(" IDLE ")
	}
}

// Help style
var HelpStyle = lipgloss.NewStyle().
	Foreground(ColorMuted).
	MarginTop(1)
