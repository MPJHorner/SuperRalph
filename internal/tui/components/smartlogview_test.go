package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSmartLogView(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	require.NotNil(t, slv, "NewSmartLogView should return non-nil")
	assert.Equal(t, 80, slv.width)
	assert.Equal(t, 24, slv.height)
	assert.True(t, slv.autoScroll, "AutoScroll should be enabled by default")
	assert.False(t, slv.userScrolled, "userScrolled should be false initially")
	assert.Empty(t, slv.entries, "entries should be empty initially")
}

func TestSmartLogViewAddEntry(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddEntry(LogTypeText, "Test message")
	slv.AddEntry(LogTypeToolUse, "Read: main.go")
	slv.AddEntry(LogTypeSuccess, "Done")

	assert.Equal(t, 3, slv.GetEntryCount())
}

func TestSmartLogViewAddLine(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")

	assert.Equal(t, 2, slv.GetEntryCount())
	lines := slv.Lines()
	assert.Equal(t, "Line 1", lines[0])
	assert.Equal(t, "Line 2", lines[1])
}

func TestSmartLogViewRingBuffer(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add more than RingBufferSize entries
	for i := 0; i < RingBufferSize+100; i++ {
		slv.AddLine("Line")
	}

	// Should only keep RingBufferSize entries
	assert.Equal(t, RingBufferSize, slv.GetEntryCount())
}

func TestSmartLogViewClear(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")
	assert.Equal(t, 2, slv.GetEntryCount())

	slv.Clear()
	assert.Equal(t, 0, slv.GetEntryCount())
}

func TestSmartLogViewAutoScroll(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Default state
	assert.True(t, slv.IsAutoScrollEnabled())

	// Disable
	slv.SetAutoScroll(false)
	assert.False(t, slv.IsAutoScrollEnabled())

	// Toggle
	slv.ToggleAutoScroll()
	assert.True(t, slv.IsAutoScrollEnabled())

	slv.ToggleAutoScroll()
	assert.False(t, slv.IsAutoScrollEnabled())
}

func TestSmartLogViewResize(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.Resize(100, 40)

	assert.Equal(t, 100, slv.width)
	assert.Equal(t, 40, slv.height)
	assert.Equal(t, 100, slv.viewport.Width)
	assert.Equal(t, 36, slv.viewport.Height) // 40 - 4
}

func TestSmartLogViewView(t *testing.T) {
	slv := NewSmartLogView(80, 24)
	slv.AddLine("Test log line")

	view := slv.View()

	// Should contain title
	assert.Contains(t, view, "Claude Output")
	// Should contain auto-scroll indicator
	assert.Contains(t, view, "Auto-Scroll ON")
}

func TestSmartLogViewViewWithAutoScrollOff(t *testing.T) {
	slv := NewSmartLogView(80, 24)
	slv.SetAutoScroll(false)

	view := slv.View()

	assert.Contains(t, view, "Auto-Scroll OFF")
}

func TestSmartLogViewGetLastLines(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")
	slv.AddLine("Line 3")
	slv.AddLine("Line 4")
	slv.AddLine("Line 5")

	last3 := slv.GetLastLines(3)
	require.Len(t, last3, 3)
	assert.Equal(t, "Line 3", last3[0])
	assert.Equal(t, "Line 4", last3[1])
	assert.Equal(t, "Line 5", last3[2])
}

func TestSmartLogViewGetLastLinesMoreThanAvailable(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")

	// Request more lines than available
	lines := slv.GetLastLines(10)
	require.Len(t, lines, 2)
	assert.Equal(t, "Line 1", lines[0])
	assert.Equal(t, "Line 2", lines[1])
}

func TestSmartLogViewLines(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")

	lines := slv.Lines()
	require.Len(t, lines, 2)
	assert.Equal(t, "Line 1", lines[0])
	assert.Equal(t, "Line 2", lines[1])
}

func TestSmartLogViewScrolling(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add enough content to scroll
	for i := 0; i < 50; i++ {
		slv.AddLine("Line content")
	}

	// Initially at bottom with auto-scroll
	assert.True(t, slv.AtBottom())
	assert.True(t, slv.autoScroll)

	// Scroll up should disable auto-scroll
	slv.ScrollUp(5)
	assert.False(t, slv.autoScroll)

	// Go to bottom should re-enable auto-scroll
	slv.GotoBottom()
	assert.True(t, slv.autoScroll)

	// Go to top should disable auto-scroll
	slv.GotoTop()
	assert.False(t, slv.autoScroll)
	assert.True(t, slv.AtTop())
}

func TestSmartLogViewScrollDown(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content
	for i := 0; i < 50; i++ {
		slv.AddLine("Line content")
	}

	// Go to top first
	slv.GotoTop()
	assert.False(t, slv.autoScroll)

	// Scroll down to bottom should re-enable auto-scroll
	for !slv.AtBottom() {
		slv.ScrollDown(1)
	}
	assert.True(t, slv.autoScroll)
}

func TestSmartLogViewUpdate(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content
	for i := 0; i < 50; i++ {
		slv.AddLine("Line content")
	}

	// Update should return a command (may be nil)
	updated, cmd := slv.Update(tea.KeyMsg{Type: tea.KeyUp})
	require.NotNil(t, updated)
	// Command may or may not be nil depending on viewport state
	_ = cmd
}

func TestSmartLogViewMouseWheelEnabled(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Verify mouse wheel is enabled
	assert.True(t, slv.viewport.MouseWheelEnabled)
	assert.Equal(t, 3, slv.viewport.MouseWheelDelta)
}

func TestSmartLogViewHandleMouseWheel(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content
	for i := 0; i < 50; i++ {
		slv.AddLine("Line content")
	}

	// Handle mouse wheel up
	slv.HandleMouseWheel(true) // up
	// Should have scrolled up, disabling auto-scroll
	assert.False(t, slv.autoScroll)
}

func TestSmartLogViewScrollPercent(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content
	for i := 0; i < 100; i++ {
		slv.AddLine("Line content")
	}

	// At bottom, scroll percent should be 1.0 or close to it
	percent := slv.ScrollPercent()
	assert.True(t, percent >= 0.9, "At bottom, scroll percent should be near 1.0")

	// Go to top
	slv.GotoTop()
	percent = slv.ScrollPercent()
	assert.True(t, percent <= 0.1, "At top, scroll percent should be near 0.0")
}

func TestSmartLogViewTotalLineCount(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Line 1")
	slv.AddLine("Line 2")
	slv.AddLine("Line 3")

	// Total line count reflects rendered content
	count := slv.TotalLineCount()
	assert.True(t, count >= 3, "Should have at least 3 lines")
}

func TestSmartLogViewSyntaxHighlightingJSON(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add JSON-like content
	slv.AddEntry(LogTypeToolInput, `{"key": "value", "count": 42}`)

	// The content should be processed (we can't easily verify highlighting in a test,
	// but we can verify the entry was added)
	assert.Equal(t, 1, slv.GetEntryCount())
}

func TestIsLikelyJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`  { "spaced": true }  `, true},
		{`not json`, false},
		{`{incomplete`, false},
		{``, false},
	}

	for _, tt := range tests {
		result := isLikelyJSON(tt.input)
		assert.Equal(t, tt.expected, result, "isLikelyJSON(%q)", tt.input)
	}
}

func TestSmartLogViewMultilineContent(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content with newlines
	slv.AddLine("Line 1\nLine 2\nLine 3")

	// Should have 1 entry but multiple rendered lines
	assert.Equal(t, 1, slv.GetEntryCount())
}

func TestSmartLogViewEntryTypes(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add entries of each type
	slv.AddEntry(LogTypeText, "Plain text")
	slv.AddEntry(LogTypeToolUse, "Read: file.go")
	slv.AddEntry(LogTypeToolInput, "input data")
	slv.AddEntry(LogTypeToolResult, "result data")
	slv.AddEntry(LogTypePhase, "PLANNING")
	slv.AddEntry(LogTypeSuccess, "Success!")
	slv.AddEntry(LogTypeError, "Error occurred")
	slv.AddEntry(LogTypeInfo, "Info message")

	assert.Equal(t, 8, slv.GetEntryCount())
}

func TestSmartLogViewRenderFooterPosition(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add content
	for i := 0; i < 50; i++ {
		slv.AddLine("Line content")
	}

	view := slv.View()

	// Should contain position info (lines count)
	assert.Contains(t, view, "lines")
}

func TestRingBufferSizeConstant(t *testing.T) {
	assert.Equal(t, 1000, RingBufferSize, "RingBufferSize should be 1000")
}

func TestSmartLogViewHeaderContainsEntryCount(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	slv.AddLine("Entry 1")
	slv.AddLine("Entry 2")
	slv.AddLine("Entry 3")

	view := slv.View()
	assert.Contains(t, view, "3 entries")
}

func TestSmartLogViewEmptyView(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	view := slv.View()

	// Should still render even when empty
	assert.Contains(t, view, "Claude Output")
	assert.Contains(t, view, "0 entries")
}

func TestSmartLogViewUserScrolledState(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Initially not scrolled
	assert.False(t, slv.userScrolled)

	// Add content and scroll up
	for i := 0; i < 50; i++ {
		slv.AddLine("Line")
	}
	slv.ScrollUp(5)

	// Should be marked as user scrolled
	assert.True(t, slv.userScrolled)

	// Go back to bottom
	slv.GotoBottom()

	// Should reset user scrolled state
	assert.False(t, slv.userScrolled)
}

func TestSmartLogViewAutoScrollOnNewEntry(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add initial content
	for i := 0; i < 50; i++ {
		slv.AddLine("Line")
	}

	// With auto-scroll enabled, should be at bottom
	assert.True(t, slv.AtBottom())

	// Add more content
	slv.AddLine("New line")

	// Should still be at bottom
	assert.True(t, slv.AtBottom())
}

func TestSmartLogViewTruncation(t *testing.T) {
	slv := NewSmartLogView(80, 24)

	// Add a very long line
	longLine := strings.Repeat("A", 200)
	slv.AddLine(longLine)

	// Should have the entry
	assert.Equal(t, 1, slv.GetEntryCount())

	// The raw line should be preserved in entries
	lines := slv.Lines()
	assert.Equal(t, longLine, lines[0])
}
