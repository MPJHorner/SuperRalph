package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// LogTab is an enhanced log viewer component for the Logs tab
// It wraps SmartLogView for high-performance scrollable viewing
type LogTab struct {
	SmartLog *SmartLogView
	Width    int
	Height   int

	// Legacy field for backward compatibility with tests
	LogView *LogView
}

// NewLogTab creates a new log tab component
func NewLogTab(width, height int) *LogTab {
	return &LogTab{
		SmartLog: NewSmartLogView(width, height),
		Width:    width,
		Height:   height,
		LogView:  NewLogView(width, height-4), // Keep for backward compat
	}
}

// AddLine adds a line to the log (plain text)
func (lt *LogTab) AddLine(line string) {
	lt.SmartLog.AddLine(line)
	lt.LogView.AddLine(line) // Keep LogView in sync for tests
}

// AddEntry adds a typed entry to the log
func (lt *LogTab) AddEntry(entryType LogEntryType, content string) {
	lt.SmartLog.AddEntry(entryType, content)
	lt.LogView.AddEntry(entryType, content) // Keep LogView in sync for tests
}

// Clear clears all log entries
func (lt *LogTab) Clear() {
	lt.SmartLog.Clear()
	lt.LogView.Clear()
}

// GetEntryCount returns the number of entries in the log
func (lt *LogTab) GetEntryCount() int {
	return lt.SmartLog.GetEntryCount()
}

// SetAutoScroll enables or disables auto-scrolling
func (lt *LogTab) SetAutoScroll(enabled bool) {
	lt.SmartLog.SetAutoScroll(enabled)
}

// ToggleAutoScroll toggles auto-scroll mode
func (lt *LogTab) ToggleAutoScroll() {
	lt.SmartLog.ToggleAutoScroll()
}

// IsAutoScrollEnabled returns whether auto-scroll is enabled
func (lt *LogTab) IsAutoScrollEnabled() bool {
	return lt.SmartLog.IsAutoScrollEnabled()
}

// AutoScroll returns whether auto-scroll is enabled (legacy accessor)
func (lt *LogTab) AutoScroll() bool {
	return lt.SmartLog.IsAutoScrollEnabled()
}

// Resize updates the dimensions
func (lt *LogTab) Resize(width, height int) {
	lt.Width = width
	lt.Height = height
	lt.SmartLog.Resize(width, height)
	lt.LogView.Width = width
	lt.LogView.Height = height - 4
}

// Update handles tea.Msg for viewport interaction
func (lt *LogTab) Update(msg tea.Msg) (*LogTab, tea.Cmd) {
	var cmd tea.Cmd
	lt.SmartLog, cmd = lt.SmartLog.Update(msg)
	return lt, cmd
}

// Render renders the log tab using the smart viewport
func (lt *LogTab) Render() string {
	return lt.SmartLog.View()
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
	return lt.SmartLog.GetLastLines(n)
}

// Lines returns all entries as plain strings
func (lt *LogTab) Lines() []string {
	return lt.SmartLog.Lines()
}

// ScrollUp scrolls the viewport up
func (lt *LogTab) ScrollUp(n int) {
	lt.SmartLog.ScrollUp(n)
}

// ScrollDown scrolls the viewport down
func (lt *LogTab) ScrollDown(n int) {
	lt.SmartLog.ScrollDown(n)
}

// GotoTop scrolls to the top
func (lt *LogTab) GotoTop() {
	lt.SmartLog.GotoTop()
}

// GotoBottom scrolls to the bottom
func (lt *LogTab) GotoBottom() {
	lt.SmartLog.GotoBottom()
}

// HandleMouseWheel handles mouse wheel events
func (lt *LogTab) HandleMouseWheel(up bool) {
	lt.SmartLog.HandleMouseWheel(up)
}
