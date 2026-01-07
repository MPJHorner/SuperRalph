package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogView displays a scrolling log of output
type LogView struct {
	Lines    []string
	MaxLines int
	Width    int
	Height   int
	Title    string
}

// NewLogView creates a new log view
func NewLogView(width, height int) *LogView {
	return &LogView{
		Lines:    make([]string, 0),
		MaxLines: 1000, // Keep last 1000 lines in memory
		Width:    width,
		Height:   height,
	}
}

// WithTitle sets the title
func (l *LogView) WithTitle(title string) *LogView {
	l.Title = title
	return l
}

// AddLine adds a new line to the log
func (l *LogView) AddLine(line string) {
	l.Lines = append(l.Lines, line)
	if len(l.Lines) > l.MaxLines {
		l.Lines = l.Lines[len(l.Lines)-l.MaxLines:]
	}
}

// AddLines adds multiple lines
func (l *LogView) AddLines(lines []string) {
	for _, line := range lines {
		l.AddLine(line)
	}
}

// Clear clears all lines
func (l *LogView) Clear() {
	l.Lines = make([]string, 0)
}

// Render returns the log view as a string
func (l *LogView) Render() string {
	displayHeight := l.Height - 2 // Account for border
	if displayHeight < 1 {
		displayHeight = 1
	}

	// Get visible lines (last N lines that fit)
	var visibleLines []string
	startIdx := len(l.Lines) - displayHeight
	if startIdx < 0 {
		startIdx = 0
	}
	visibleLines = l.Lines[startIdx:]

	// Truncate lines that are too wide
	contentWidth := l.Width - 4 // Account for border and padding
	for i, line := range visibleLines {
		if len(line) > contentWidth {
			visibleLines[i] = line[:contentWidth-3] + "..."
		}
	}

	// Pad to fill height
	for len(visibleLines) < displayHeight {
		visibleLines = append(visibleLines, "")
	}

	content := strings.Join(visibleLines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("245")).
		Width(l.Width - 2).
		Height(displayHeight)

	if l.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Bold(true)
		return titleStyle.Render(l.Title) + "\n" + boxStyle.Render(content)
	}

	return boxStyle.Render(content)
}

// GetLastLines returns the last n lines
func (l *LogView) GetLastLines(n int) []string {
	if n >= len(l.Lines) {
		return l.Lines
	}
	return l.Lines[len(l.Lines)-n:]
}
