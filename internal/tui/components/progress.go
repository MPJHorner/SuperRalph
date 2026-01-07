package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a progress bar
type ProgressBar struct {
	Width   int
	Total   int
	Current int
	Label   string
	ShowPct bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(current, total, width int) ProgressBar {
	return ProgressBar{
		Width:   width,
		Total:   total,
		Current: current,
		ShowPct: true,
	}
}

// WithLabel adds a label to the progress bar
func (p ProgressBar) WithLabel(label string) ProgressBar {
	p.Label = label
	return p
}

// Render returns the string representation of the progress bar
func (p ProgressBar) Render() string {
	if p.Total == 0 {
		return ""
	}

	pct := float64(p.Current) / float64(p.Total)
	filledWidth := int(pct * float64(p.Width))
	if filledWidth > p.Width {
		filledWidth = p.Width
	}
	emptyWidth := p.Width - filledWidth

	filled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Render(strings.Repeat("█", filledWidth))

	empty := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("░", emptyWidth))

	bar := filled + empty

	var result string
	if p.Label != "" {
		result = fmt.Sprintf("%s %s", p.Label, bar)
	} else {
		result = bar
	}

	if p.ShowPct {
		pctStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render(fmt.Sprintf(" %d/%d (%.0f%%)", p.Current, p.Total, pct*100))
		result += pctStr
	}

	return result
}

// MiniProgressBar renders a compact progress bar for category/priority breakdown
type MiniProgressBar struct {
	Width   int
	Total   int
	Current int
}

// NewMiniProgressBar creates a new mini progress bar
func NewMiniProgressBar(current, total, width int) MiniProgressBar {
	return MiniProgressBar{
		Width:   width,
		Total:   total,
		Current: current,
	}
}

// Render returns the mini progress bar
func (p MiniProgressBar) Render() string {
	if p.Total == 0 {
		return strings.Repeat("░", p.Width)
	}

	pct := float64(p.Current) / float64(p.Total)
	filledWidth := int(pct * float64(p.Width))
	if filledWidth > p.Width {
		filledWidth = p.Width
	}
	emptyWidth := p.Width - filledWidth

	filled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Render(strings.Repeat("█", filledWidth))

	empty := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("░", emptyWidth))

	return filled + empty
}
