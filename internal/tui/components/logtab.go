package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogTab is an enhanced log viewer component for the Logs tab
// It wraps LogView but provides additional functionality for dedicated viewing
type LogTab struct {
	LogView    *LogView
	Width      int
	Height     int
	AutoScroll bool

	// Styles
	titleStyle  lipgloss.Style
	mutedStyle  lipgloss.Style
	scrollStyle lipgloss.Style
}

// NewLogTab creates a new log tab component
func NewLogTab(width, height int) *LogTab {
	return &LogTab{
		LogView:    NewLogView(width, height-4), // Account for header/footer
		Width:      width,
		Height:     height,
		AutoScroll: true,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		mutedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		scrollStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
	}
}

// AddLine adds a line to the log (plain text)
func (lt *LogTab) AddLine(line string) {
	lt.LogView.AddLine(line)
}

// AddEntry adds a typed entry to the log
func (lt *LogTab) AddEntry(entryType LogEntryType, content string) {
	lt.LogView.AddEntry(entryType, content)
}

// Clear clears all log entries
func (lt *LogTab) Clear() {
	lt.LogView.Clear()
}

// GetEntryCount returns the number of entries in the log
func (lt *LogTab) GetEntryCount() int {
	return len(lt.LogView.Entries)
}

// SetAutoScroll enables or disables auto-scrolling
func (lt *LogTab) SetAutoScroll(enabled bool) {
	lt.AutoScroll = enabled
}

// ToggleAutoScroll toggles auto-scroll mode
func (lt *LogTab) ToggleAutoScroll() {
	lt.AutoScroll = !lt.AutoScroll
}

// IsAutoScrollEnabled returns whether auto-scroll is enabled
func (lt *LogTab) IsAutoScrollEnabled() bool {
	return lt.AutoScroll
}

// Resize updates the dimensions
func (lt *LogTab) Resize(width, height int) {
	lt.Width = width
	lt.Height = height
	lt.LogView.Width = width
	lt.LogView.Height = height - 4
}

// Render renders the log tab
func (lt *LogTab) Render() string {
	var b strings.Builder

	// Header
	b.WriteString(lt.renderHeader())
	b.WriteString("\n")

	// Main log content
	lt.LogView.Width = lt.Width - 2
	lt.LogView.Height = lt.Height - 4
	b.WriteString(lt.LogView.Render())
	b.WriteString("\n")

	// Footer with controls
	b.WriteString(lt.renderFooter())

	return b.String()
}

// renderHeader renders the header with title and entry count
func (lt *LogTab) renderHeader() string {
	title := lt.titleStyle.Render("Claude Output")
	count := lt.mutedStyle.Render(strings.Repeat(" ", 2) + "(" + itoa(lt.GetEntryCount()) + " entries)")

	return title + count
}

// renderFooter renders the footer with scroll status and help
func (lt *LogTab) renderFooter() string {
	var parts []string

	// Auto-scroll indicator
	if lt.AutoScroll {
		parts = append(parts, lt.scrollStyle.Render("[Auto-Scroll ON]"))
	} else {
		parts = append(parts, lt.mutedStyle.Render("[Auto-Scroll OFF]"))
	}

	// Help
	parts = append(parts, lt.mutedStyle.Render("Press 'a' to toggle auto-scroll"))

	return strings.Join(parts, "  ")
}

// itoa is a helper to convert int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}

// GetLastLines returns the last n lines as plain text
func (lt *LogTab) GetLastLines(n int) []string {
	return lt.LogView.GetLastLines(n)
}

// Lines returns all entries as plain strings
func (lt *LogTab) Lines() []string {
	return lt.LogView.Lines()
}
