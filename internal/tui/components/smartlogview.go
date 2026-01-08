package components

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RingBufferSize is the maximum number of log entries to keep in memory
const RingBufferSize = 1000

// SmartLogView is a high-performance scrollable log viewer using viewport
// It features:
// - Ring buffer to prevent memory bloat (max 1000 lines)
// - Auto-scroll toggle (locks to bottom when enabled, unlocks on user scroll)
// - Syntax highlighting for JSON and code blocks
// - Mouse wheel support
type SmartLogView struct {
	viewport viewport.Model
	entries  []LogEntry
	width    int
	height   int

	// Auto-scroll state
	autoScroll    bool
	userScrolled  bool // Track if user manually scrolled
	lastLineCount int  // Track last line count to detect new entries

	// Syntax highlighting patterns
	jsonPattern *regexp.Regexp
	codePattern *regexp.Regexp

	// Styles
	titleStyle   lipgloss.Style
	mutedStyle   lipgloss.Style
	scrollStyle  lipgloss.Style
	jsonKeyStyle lipgloss.Style
	jsonValStyle lipgloss.Style
}

// NewSmartLogView creates a new smart log view with the given dimensions
func NewSmartLogView(width, height int) *SmartLogView {
	vp := viewport.New(width, height-4) // Account for header/footer
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return &SmartLogView{
		viewport:     vp,
		entries:      make([]LogEntry, 0, RingBufferSize),
		width:        width,
		height:       height,
		autoScroll:   true,
		userScrolled: false,
		jsonPattern:  regexp.MustCompile(`"([^"]+)":\s*`),
		codePattern:  regexp.MustCompile("```(\\w+)?([\\s\\S]*?)```"),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		mutedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		scrollStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		jsonKeyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")), // Cyan for keys
		jsonValStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("186")), // Yellow for values
	}
}

// AddEntry adds a new typed entry to the log with ring buffer management
func (s *SmartLogView) AddEntry(entryType LogEntryType, content string) {
	s.entries = append(s.entries, LogEntry{
		Type:    entryType,
		Content: content,
	})

	// Ring buffer: remove oldest entries if we exceed the limit
	if len(s.entries) > RingBufferSize {
		s.entries = s.entries[len(s.entries)-RingBufferSize:]
	}

	s.refreshContent()
}

// AddLine adds a plain text line to the log (convenience method)
func (s *SmartLogView) AddLine(line string) {
	s.AddEntry(LogTypeText, line)
}

// Clear clears all log entries
func (s *SmartLogView) Clear() {
	s.entries = make([]LogEntry, 0, RingBufferSize)
	s.viewport.SetContent("")
	s.lastLineCount = 0
}

// GetEntryCount returns the number of entries in the log
func (s *SmartLogView) GetEntryCount() int {
	return len(s.entries)
}

// SetAutoScroll enables or disables auto-scrolling
func (s *SmartLogView) SetAutoScroll(enabled bool) {
	s.autoScroll = enabled
	if enabled {
		s.userScrolled = false
		s.viewport.GotoBottom()
	}
}

// ToggleAutoScroll toggles auto-scroll mode
func (s *SmartLogView) ToggleAutoScroll() {
	s.SetAutoScroll(!s.autoScroll)
}

// IsAutoScrollEnabled returns whether auto-scroll is enabled
func (s *SmartLogView) IsAutoScrollEnabled() bool {
	return s.autoScroll
}

// Resize updates the dimensions of the viewport
func (s *SmartLogView) Resize(width, height int) {
	s.width = width
	s.height = height
	s.viewport.Width = width
	s.viewport.Height = height - 4 // Account for header/footer
	s.refreshContent()
}

// Update handles tea.Msg for viewport interaction
func (s *SmartLogView) Update(msg tea.Msg) (*SmartLogView, tea.Cmd) {
	var cmd tea.Cmd

	// Track position before update
	wasAtBottom := s.viewport.AtBottom()

	// Update viewport
	s.viewport, cmd = s.viewport.Update(msg)

	// Detect user scrolling
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		// If user scrolled up from bottom, disable auto-scroll
		if wasAtBottom && !s.viewport.AtBottom() {
			s.userScrolled = true
			s.autoScroll = false
		}
		// If user scrolled to bottom, re-enable auto-scroll
		if s.viewport.AtBottom() && s.userScrolled {
			s.userScrolled = false
			s.autoScroll = true
		}
	}

	return s, cmd
}

// View renders the smart log view
func (s *SmartLogView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(s.renderHeader())
	b.WriteString("\n")

	// Viewport content
	b.WriteString(s.viewport.View())
	b.WriteString("\n")

	// Footer with scroll info
	b.WriteString(s.renderFooter())

	return b.String()
}

// renderHeader renders the header with title and entry count
func (s *SmartLogView) renderHeader() string {
	title := s.titleStyle.Render("Claude Output")
	count := s.mutedStyle.Render("  (" + itoa(len(s.entries)) + " entries)")
	return title + count
}

// renderFooter renders the footer with scroll status and position
func (s *SmartLogView) renderFooter() string {
	var parts []string

	// Auto-scroll indicator
	if s.autoScroll {
		parts = append(parts, s.scrollStyle.Render("[Auto-Scroll ON]"))
	} else {
		parts = append(parts, s.mutedStyle.Render("[Auto-Scroll OFF]"))
	}

	// Scroll position indicator
	scrollPercent := int(s.viewport.ScrollPercent() * 100)
	if s.viewport.TotalLineCount() > 0 {
		posInfo := s.mutedStyle.Render(
			itoa(scrollPercent) + "% | " +
				itoa(s.viewport.YOffset+1) + "-" +
				itoa(min(s.viewport.YOffset+s.viewport.Height, s.viewport.TotalLineCount())) +
				"/" + itoa(s.viewport.TotalLineCount()) + " lines",
		)
		parts = append(parts, posInfo)
	}

	// Help hint
	parts = append(parts, s.mutedStyle.Render("[a] toggle | [j/k] scroll | mouse wheel"))

	return strings.Join(parts, "  ")
}

// refreshContent rebuilds the viewport content from entries
func (s *SmartLogView) refreshContent() {
	contentWidth := s.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	var lines []string
	for _, entry := range s.entries {
		// Apply styling based on entry type
		style := styleForType(entry.Type)
		prefix := prefixForType(entry.Type)
		content := entry.Content

		// Apply syntax highlighting for certain types
		content = s.applySyntaxHighlighting(entry.Type, content)

		// Handle multi-line content
		contentLines := strings.Split(content, "\n")
		for i, line := range contentLines {
			p := ""
			if i == 0 {
				p = prefix
			} else {
				p = strings.Repeat(" ", len(prefix))
			}

			// Truncate if too wide
			maxLen := contentWidth - len(p)
			if maxLen > 0 && len(line) > maxLen {
				line = line[:maxLen-3] + "..."
			}

			lines = append(lines, style.Render(p+line))
		}
	}

	s.viewport.SetContent(strings.Join(lines, "\n"))

	// Auto-scroll to bottom if enabled
	if s.autoScroll && !s.userScrolled {
		s.viewport.GotoBottom()
	}
}

// applySyntaxHighlighting applies syntax highlighting to content based on type
func (s *SmartLogView) applySyntaxHighlighting(entryType LogEntryType, content string) string {
	switch entryType {
	case LogTypeToolInput, LogTypeToolResult:
		// Try to highlight JSON
		if isLikelyJSON(content) {
			return s.highlightJSON(content)
		}
	}
	return content
}

// highlightJSON applies basic JSON syntax highlighting
func (s *SmartLogView) highlightJSON(content string) string {
	// Simple JSON key highlighting
	result := s.jsonPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the key (everything between quotes before the colon)
		keyMatch := s.jsonPattern.FindStringSubmatch(match)
		if len(keyMatch) > 1 {
			key := keyMatch[1]
			return s.jsonKeyStyle.Render("\""+key+"\"") + ": "
		}
		return match
	})
	return result
}

// isLikelyJSON checks if content looks like JSON
func isLikelyJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

// ScrollUp scrolls the viewport up by n lines
func (s *SmartLogView) ScrollUp(n int) {
	s.viewport.SetYOffset(s.viewport.YOffset - n)
	if !s.viewport.AtBottom() {
		s.userScrolled = true
		s.autoScroll = false
	}
}

// ScrollDown scrolls the viewport down by n lines
func (s *SmartLogView) ScrollDown(n int) {
	s.viewport.SetYOffset(s.viewport.YOffset + n)
	if s.viewport.AtBottom() {
		s.userScrolled = false
		s.autoScroll = true
	}
}

// GotoTop scrolls to the top of the log
func (s *SmartLogView) GotoTop() {
	s.viewport.GotoTop()
	s.userScrolled = true
	s.autoScroll = false
}

// GotoBottom scrolls to the bottom of the log
func (s *SmartLogView) GotoBottom() {
	s.viewport.GotoBottom()
	s.userScrolled = false
	s.autoScroll = true
}

// GetLastLines returns the last n lines as plain text
func (s *SmartLogView) GetLastLines(n int) []string {
	if n >= len(s.entries) {
		lines := make([]string, len(s.entries))
		for i, e := range s.entries {
			lines[i] = e.Content
		}
		return lines
	}
	entries := s.entries[len(s.entries)-n:]
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.Content
	}
	return lines
}

// Lines returns all entries as plain strings
func (s *SmartLogView) Lines() []string {
	lines := make([]string, len(s.entries))
	for i, e := range s.entries {
		lines[i] = e.Content
	}
	return lines
}

// AtBottom returns true if the viewport is at the bottom
func (s *SmartLogView) AtBottom() bool {
	return s.viewport.AtBottom()
}

// AtTop returns true if the viewport is at the top
func (s *SmartLogView) AtTop() bool {
	return s.viewport.AtTop()
}

// ScrollPercent returns the current scroll position as a percentage (0-1)
func (s *SmartLogView) ScrollPercent() float64 {
	return s.viewport.ScrollPercent()
}

// TotalLineCount returns the total number of rendered lines
func (s *SmartLogView) TotalLineCount() int {
	return s.viewport.TotalLineCount()
}

// HandleMouseWheel processes mouse wheel events and updates auto-scroll state
func (s *SmartLogView) HandleMouseWheel(up bool) {
	if up {
		s.ScrollUp(3)
	} else {
		s.ScrollDown(3)
	}
}
