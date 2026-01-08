package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab in the navigation
type Tab int

const (
	TabDashboard Tab = iota
	TabFeatures
	TabLogs
)

// String returns the display name for a tab
func (t Tab) String() string {
	switch t {
	case TabDashboard:
		return "Dashboard"
	case TabFeatures:
		return "Features"
	case TabLogs:
		return "Logs"
	default:
		return "Unknown"
	}
}

// ShortKey returns the keyboard shortcut number for a tab
func (t Tab) ShortKey() string {
	switch t {
	case TabDashboard:
		return "1"
	case TabFeatures:
		return "2"
	case TabLogs:
		return "3"
	default:
		return "?"
	}
}

// AllTabs returns all available tabs in order
func AllTabs() []Tab {
	return []Tab{TabDashboard, TabFeatures, TabLogs}
}

// TabBar represents a navigation bar with tabs
type TabBar struct {
	ActiveTab Tab
	Width     int

	// Styles
	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	barStyle      lipgloss.Style
}

// NewTabBar creates a new tab bar component
func NewTabBar() *TabBar {
	return &TabBar{
		ActiveTab: TabDashboard,
		Width:     80,
		activeStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("99")). // Purple background
			Padding(0, 2),
		inactiveStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 2),
		barStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("236")),
	}
}

// SetActiveTab sets the currently active tab
func (tb *TabBar) SetActiveTab(tab Tab) {
	tb.ActiveTab = tab
}

// NextTab moves to the next tab (wrapping around)
func (tb *TabBar) NextTab() {
	tabs := AllTabs()
	for i, t := range tabs {
		if t == tb.ActiveTab {
			tb.ActiveTab = tabs[(i+1)%len(tabs)]
			return
		}
	}
}

// PrevTab moves to the previous tab (wrapping around)
func (tb *TabBar) PrevTab() {
	tabs := AllTabs()
	for i, t := range tabs {
		if t == tb.ActiveTab {
			prevIdx := i - 1
			if prevIdx < 0 {
				prevIdx = len(tabs) - 1
			}
			tb.ActiveTab = tabs[prevIdx]
			return
		}
	}
}

// GetActiveTab returns the currently active tab
func (tb *TabBar) GetActiveTab() Tab {
	return tb.ActiveTab
}

// Render renders the tab bar
func (tb *TabBar) Render() string {
	var tabs []string

	for _, tab := range AllTabs() {
		label := "[" + tab.ShortKey() + "] " + tab.String()

		var style lipgloss.Style
		if tab == tb.ActiveTab {
			style = tb.activeStyle
		} else {
			style = tb.inactiveStyle
		}

		tabs = append(tabs, style.Render(label))
	}

	// Join tabs with a small gap
	gap := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Render(" ")

	content := strings.Join(tabs, gap)

	// Add padding to fill width
	return tb.barStyle.Width(tb.Width).Render(content)
}

// RenderCompact renders a compact version of the tab bar (for narrow terminals)
func (tb *TabBar) RenderCompact() string {
	var tabs []string

	for _, tab := range AllTabs() {
		label := tab.ShortKey()

		var style lipgloss.Style
		if tab == tb.ActiveTab {
			style = tb.activeStyle
		} else {
			style = tb.inactiveStyle
		}

		tabs = append(tabs, style.Render(label))
	}

	gap := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Render(" ")

	return strings.Join(tabs, gap)
}

// TabFromKey returns the tab for a given key press, or -1 if not a tab key
func TabFromKey(key string) Tab {
	switch key {
	case "1":
		return TabDashboard
	case "2":
		return TabFeatures
	case "3":
		return TabLogs
	default:
		return Tab(-1)
	}
}
