package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogEntryType represents the type of log entry for coloring
type LogEntryType string

const (
	LogTypeText       LogEntryType = "text"        // White - Claude's explanations
	LogTypeToolUse    LogEntryType = "tool_use"    // Cyan - Tool invocations
	LogTypeToolInput  LogEntryType = "tool_input"  // Cyan - Tool input/args
	LogTypeToolResult LogEntryType = "tool_result" // Gray - Tool output (truncated)
	LogTypePhase      LogEntryType = "phase"       // Purple - Phase changes
	LogTypeSuccess    LogEntryType = "success"     // Green - Success messages
	LogTypeError      LogEntryType = "error"       // Red - Errors
	LogTypeInfo       LogEntryType = "info"        // Muted - Info/status
)

// LogEntry represents a single log entry with type information
type LogEntry struct {
	Type    LogEntryType
	Content string
}

// Color constants matching the TUI theme
var (
	logColorText       = lipgloss.Color("255") // White
	logColorToolUse    = lipgloss.Color("39")  // Cyan
	logColorToolResult = lipgloss.Color("245") // Gray
	logColorPhase      = lipgloss.Color("99")  // Purple
	logColorSuccess    = lipgloss.Color("42")  // Green
	logColorError      = lipgloss.Color("196") // Red
	logColorInfo       = lipgloss.Color("245") // Muted gray
)

// LogView displays a scrolling log of output with colored entries
type LogView struct {
	Entries  []LogEntry
	MaxLines int
	Width    int
	Height   int
	Title    string
}

// NewLogView creates a new log view
func NewLogView(width, height int) *LogView {
	return &LogView{
		Entries:  make([]LogEntry, 0),
		MaxLines: 1000, // Keep last 1000 entries in memory
		Width:    width,
		Height:   height,
	}
}

// WithTitle sets the title
func (l *LogView) WithTitle(title string) *LogView {
	l.Title = title
	return l
}

// AddLine adds a new plain text line to the log (backward compatible)
func (l *LogView) AddLine(line string) {
	l.AddEntry(LogTypeText, line)
}

// AddEntry adds a new typed entry to the log
func (l *LogView) AddEntry(entryType LogEntryType, content string) {
	l.Entries = append(l.Entries, LogEntry{
		Type:    entryType,
		Content: content,
	})
	if len(l.Entries) > l.MaxLines {
		l.Entries = l.Entries[len(l.Entries)-l.MaxLines:]
	}
}

// AddLines adds multiple plain text lines
func (l *LogView) AddLines(lines []string) {
	for _, line := range lines {
		l.AddLine(line)
	}
}

// Clear clears all entries
func (l *LogView) Clear() {
	l.Entries = make([]LogEntry, 0)
}

// styleForType returns the lipgloss style for a given entry type
func styleForType(entryType LogEntryType) lipgloss.Style {
	switch entryType {
	case LogTypeText:
		return lipgloss.NewStyle().Foreground(logColorText)
	case LogTypeToolUse:
		return lipgloss.NewStyle().Foreground(logColorToolUse).Bold(true)
	case LogTypeToolInput:
		return lipgloss.NewStyle().Foreground(logColorToolUse)
	case LogTypeToolResult:
		return lipgloss.NewStyle().Foreground(logColorToolResult)
	case LogTypePhase:
		return lipgloss.NewStyle().Foreground(logColorPhase).Bold(true)
	case LogTypeSuccess:
		return lipgloss.NewStyle().Foreground(logColorSuccess).Bold(true)
	case LogTypeError:
		return lipgloss.NewStyle().Foreground(logColorError).Bold(true)
	case LogTypeInfo:
		return lipgloss.NewStyle().Foreground(logColorInfo)
	default:
		return lipgloss.NewStyle().Foreground(logColorText)
	}
}

// prefixForType returns a prefix icon/symbol for a given entry type
func prefixForType(entryType LogEntryType) string {
	switch entryType {
	case LogTypeToolUse:
		return "> "
	case LogTypeToolInput:
		return "  "
	case LogTypeToolResult:
		return "  "
	case LogTypePhase:
		return "# "
	case LogTypeSuccess:
		return "+ "
	case LogTypeError:
		return "! "
	case LogTypeInfo:
		return "- "
	default:
		return ""
	}
}

// Render returns the log view as a string
func (l *LogView) Render() string {
	displayHeight := l.Height - 2 // Account for border
	if displayHeight < 1 {
		displayHeight = 1
	}

	// Get visible entries (last N entries that fit)
	startIdx := len(l.Entries) - displayHeight
	if startIdx < 0 {
		startIdx = 0
	}
	visibleEntries := l.Entries[startIdx:]

	// Calculate content width accounting for border and padding
	contentWidth := l.Width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render each entry with its style
	var renderedLines []string
	for _, entry := range visibleEntries {
		style := styleForType(entry.Type)
		prefix := prefixForType(entry.Type)
		content := entry.Content

		// Truncate if too wide (accounting for prefix)
		maxContentLen := contentWidth - len(prefix)
		if len(content) > maxContentLen {
			content = content[:maxContentLen-3] + "..."
		}

		renderedLines = append(renderedLines, style.Render(prefix+content))
	}

	// Pad to fill height with empty lines
	for len(renderedLines) < displayHeight {
		renderedLines = append(renderedLines, "")
	}

	content := strings.Join(renderedLines, "\n")

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

// GetLastLines returns the last n lines as plain text (backward compatible)
func (l *LogView) GetLastLines(n int) []string {
	if n >= len(l.Entries) {
		lines := make([]string, len(l.Entries))
		for i, e := range l.Entries {
			lines[i] = e.Content
		}
		return lines
	}
	entries := l.Entries[len(l.Entries)-n:]
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.Content
	}
	return lines
}

// Lines returns all entries as plain strings (for backward compatibility)
func (l *LogView) Lines() []string {
	lines := make([]string, len(l.Entries))
	for i, e := range l.Entries {
		lines[i] = e.Content
	}
	return lines
}
