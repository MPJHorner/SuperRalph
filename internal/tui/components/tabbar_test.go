package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTabBar(t *testing.T) {
	tb := NewTabBar()

	require.NotNil(t, tb, "NewTabBar should return non-nil")
	assert.Equal(t, TabDashboard, tb.ActiveTab, "Default active tab should be TabDashboard")
	assert.Equal(t, 80, tb.Width, "Default width should be 80")
}

func TestTabBarSetActiveTab(t *testing.T) {
	tb := NewTabBar()

	tb.SetActiveTab(TabFeatures)
	assert.Equal(t, TabFeatures, tb.ActiveTab, "ActiveTab should be TabFeatures")

	tb.SetActiveTab(TabLogs)
	assert.Equal(t, TabLogs, tb.ActiveTab, "ActiveTab should be TabLogs")

	tb.SetActiveTab(TabDashboard)
	assert.Equal(t, TabDashboard, tb.ActiveTab, "ActiveTab should be TabDashboard")
}

func TestTabBarNextTab(t *testing.T) {
	tb := NewTabBar()
	assert.Equal(t, TabDashboard, tb.ActiveTab)

	tb.NextTab()
	assert.Equal(t, TabFeatures, tb.ActiveTab, "NextTab from Dashboard should be Features")

	tb.NextTab()
	assert.Equal(t, TabLogs, tb.ActiveTab, "NextTab from Features should be Logs")

	tb.NextTab()
	assert.Equal(t, TabDashboard, tb.ActiveTab, "NextTab from Logs should wrap to Dashboard")
}

func TestTabBarPrevTab(t *testing.T) {
	tb := NewTabBar()
	assert.Equal(t, TabDashboard, tb.ActiveTab)

	tb.PrevTab()
	assert.Equal(t, TabLogs, tb.ActiveTab, "PrevTab from Dashboard should wrap to Logs")

	tb.PrevTab()
	assert.Equal(t, TabFeatures, tb.ActiveTab, "PrevTab from Logs should be Features")

	tb.PrevTab()
	assert.Equal(t, TabDashboard, tb.ActiveTab, "PrevTab from Features should be Dashboard")
}

func TestTabBarGetActiveTab(t *testing.T) {
	tb := NewTabBar()

	assert.Equal(t, TabDashboard, tb.GetActiveTab())

	tb.SetActiveTab(TabLogs)
	assert.Equal(t, TabLogs, tb.GetActiveTab())
}

func TestTabBarRender(t *testing.T) {
	tb := NewTabBar()
	tb.Width = 80

	rendered := tb.Render()

	// Should contain all tab labels
	assert.Contains(t, rendered, "Dashboard", "Render should contain Dashboard")
	assert.Contains(t, rendered, "Features", "Render should contain Features")
	assert.Contains(t, rendered, "Logs", "Render should contain Logs")

	// Should contain shortcut keys
	assert.Contains(t, rendered, "1", "Render should contain shortcut 1")
	assert.Contains(t, rendered, "2", "Render should contain shortcut 2")
	assert.Contains(t, rendered, "3", "Render should contain shortcut 3")
}

func TestTabBarRenderCompact(t *testing.T) {
	tb := NewTabBar()

	rendered := tb.RenderCompact()

	// Should contain shortcut keys
	assert.Contains(t, rendered, "1", "RenderCompact should contain 1")
	assert.Contains(t, rendered, "2", "RenderCompact should contain 2")
	assert.Contains(t, rendered, "3", "RenderCompact should contain 3")
}

func TestTabFromKey(t *testing.T) {
	tests := []struct {
		key      string
		expected Tab
	}{
		{"1", TabDashboard},
		{"2", TabFeatures},
		{"3", TabLogs},
		{"4", Tab(-1)},
		{"a", Tab(-1)},
		{"", Tab(-1)},
	}

	for _, tt := range tests {
		result := TabFromKey(tt.key)
		assert.Equal(t, tt.expected, result, "TabFromKey(%q)", tt.key)
	}
}

func TestTabString(t *testing.T) {
	tests := []struct {
		tab      Tab
		expected string
	}{
		{TabDashboard, "Dashboard"},
		{TabFeatures, "Features"},
		{TabLogs, "Logs"},
		{Tab(99), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.tab.String()
		assert.Equal(t, tt.expected, result, "Tab(%d).String()", tt.tab)
	}
}

func TestTabShortKey(t *testing.T) {
	tests := []struct {
		tab      Tab
		expected string
	}{
		{TabDashboard, "1"},
		{TabFeatures, "2"},
		{TabLogs, "3"},
		{Tab(99), "?"},
	}

	for _, tt := range tests {
		result := tt.tab.ShortKey()
		assert.Equal(t, tt.expected, result, "Tab(%d).ShortKey()", tt.tab)
	}
}

func TestAllTabs(t *testing.T) {
	tabs := AllTabs()

	require.Len(t, tabs, 3, "AllTabs should return 3 tabs")
	assert.Equal(t, TabDashboard, tabs[0])
	assert.Equal(t, TabFeatures, tabs[1])
	assert.Equal(t, TabLogs, tabs[2])
}

func TestTabBarRenderWithDifferentActiveTabs(t *testing.T) {
	tb := NewTabBar()
	tb.Width = 80

	// Test rendering with each tab active
	for _, tab := range AllTabs() {
		tb.SetActiveTab(tab)
		rendered := tb.Render()

		// Should always contain all tabs
		assert.Contains(t, rendered, "Dashboard")
		assert.Contains(t, rendered, "Features")
		assert.Contains(t, rendered, "Logs")
	}
}

func TestTabConstants(t *testing.T) {
	// Ensure tab constants have expected values
	assert.Equal(t, Tab(0), TabDashboard)
	assert.Equal(t, Tab(1), TabFeatures)
	assert.Equal(t, Tab(2), TabLogs)
}
