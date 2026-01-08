package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogTab(t *testing.T) {
	lt := NewLogTab(80, 24)

	require.NotNil(t, lt, "NewLogTab should return non-nil")
	assert.Equal(t, 80, lt.Width)
	assert.Equal(t, 24, lt.Height)
	assert.True(t, lt.AutoScroll, "AutoScroll should be enabled by default")
	require.NotNil(t, lt.LogView, "LogView should be initialized")
}

func TestLogTabAddLine(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddLine("Test line 1")
	lt.AddLine("Test line 2")

	lines := lt.GetLastLines(2)
	require.Len(t, lines, 2)
	assert.Equal(t, "Test line 1", lines[0])
	assert.Equal(t, "Test line 2", lines[1])
}

func TestLogTabAddEntry(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddEntry(LogTypeToolUse, "Using Read tool")
	lt.AddEntry(LogTypeSuccess, "Completed successfully")

	assert.Equal(t, 2, lt.GetEntryCount())
}

func TestLogTabClear(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddLine("Line 1")
	lt.AddLine("Line 2")
	assert.Equal(t, 2, lt.GetEntryCount())

	lt.Clear()
	assert.Equal(t, 0, lt.GetEntryCount())
}

func TestLogTabGetEntryCount(t *testing.T) {
	lt := NewLogTab(80, 24)

	assert.Equal(t, 0, lt.GetEntryCount())

	lt.AddLine("Line 1")
	assert.Equal(t, 1, lt.GetEntryCount())

	lt.AddLine("Line 2")
	lt.AddLine("Line 3")
	assert.Equal(t, 3, lt.GetEntryCount())
}

func TestLogTabSetAutoScroll(t *testing.T) {
	lt := NewLogTab(80, 24)

	assert.True(t, lt.IsAutoScrollEnabled())

	lt.SetAutoScroll(false)
	assert.False(t, lt.IsAutoScrollEnabled())

	lt.SetAutoScroll(true)
	assert.True(t, lt.IsAutoScrollEnabled())
}

func TestLogTabToggleAutoScroll(t *testing.T) {
	lt := NewLogTab(80, 24)

	assert.True(t, lt.AutoScroll)

	lt.ToggleAutoScroll()
	assert.False(t, lt.AutoScroll)

	lt.ToggleAutoScroll()
	assert.True(t, lt.AutoScroll)
}

func TestLogTabIsAutoScrollEnabled(t *testing.T) {
	lt := NewLogTab(80, 24)

	assert.True(t, lt.IsAutoScrollEnabled())

	lt.AutoScroll = false
	assert.False(t, lt.IsAutoScrollEnabled())
}

func TestLogTabResize(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.Resize(100, 40)

	assert.Equal(t, 100, lt.Width)
	assert.Equal(t, 40, lt.Height)
	assert.Equal(t, 100, lt.LogView.Width)
	assert.Equal(t, 36, lt.LogView.Height) // 40 - 4
}

func TestLogTabRender(t *testing.T) {
	lt := NewLogTab(80, 24)
	lt.AddLine("Test log line")

	rendered := lt.Render()

	// Should contain title
	assert.Contains(t, rendered, "Claude Output")

	// Should contain auto-scroll indicator
	assert.Contains(t, rendered, "Auto-Scroll")
}

func TestLogTabRenderWithAutoScrollOff(t *testing.T) {
	lt := NewLogTab(80, 24)
	lt.SetAutoScroll(false)

	rendered := lt.Render()

	assert.Contains(t, rendered, "Auto-Scroll OFF")
}

func TestLogTabRenderWithAutoScrollOn(t *testing.T) {
	lt := NewLogTab(80, 24)
	lt.SetAutoScroll(true)

	rendered := lt.Render()

	assert.Contains(t, rendered, "Auto-Scroll ON")
}

func TestLogTabGetLastLines(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddLine("Line 1")
	lt.AddLine("Line 2")
	lt.AddLine("Line 3")
	lt.AddLine("Line 4")
	lt.AddLine("Line 5")

	last3 := lt.GetLastLines(3)
	require.Len(t, last3, 3)
	assert.Equal(t, "Line 3", last3[0])
	assert.Equal(t, "Line 4", last3[1])
	assert.Equal(t, "Line 5", last3[2])
}

func TestLogTabLines(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddLine("Line 1")
	lt.AddLine("Line 2")

	lines := lt.Lines()
	require.Len(t, lines, 2)
	assert.Equal(t, "Line 1", lines[0])
	assert.Equal(t, "Line 2", lines[1])
}

func TestLogTabRenderWithEntries(t *testing.T) {
	lt := NewLogTab(80, 24)

	lt.AddEntry(LogTypePhase, "Starting PLAN phase")
	lt.AddEntry(LogTypeToolUse, "Read: main.go")
	lt.AddEntry(LogTypeSuccess, "Phase complete")

	rendered := lt.Render()

	// Should contain the entry count
	assert.Contains(t, rendered, "3 entries")
}

func TestLogTabRenderEmpty(t *testing.T) {
	lt := NewLogTab(80, 24)

	rendered := lt.Render()

	// Should contain title
	assert.Contains(t, rendered, "Claude Output")
	// Should show 0 entries
	assert.Contains(t, rendered, "0 entries")
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{12345, "12345"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		assert.Equal(t, tt.expected, result, "itoa(%d)", tt.input)
	}
}

func TestLogTabResizeUpdatesLogView(t *testing.T) {
	lt := NewLogTab(80, 24)

	initialLogViewWidth := lt.LogView.Width
	initialLogViewHeight := lt.LogView.Height

	lt.Resize(120, 50)

	assert.NotEqual(t, initialLogViewWidth, lt.LogView.Width, "LogView width should change")
	assert.NotEqual(t, initialLogViewHeight, lt.LogView.Height, "LogView height should change")
	assert.Equal(t, 120, lt.LogView.Width)
	assert.Equal(t, 46, lt.LogView.Height) // 50 - 4
}
