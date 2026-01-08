package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/tui/components"
)

func createTestPRD() *prd.PRD {
	return &prd.PRD{
		Name:        "Test Project",
		Description: "A test project",
		TestCommand: "go test ./...",
		Features: []prd.Feature{
			{
				ID:          "feat-001",
				Category:    "functional",
				Priority:    "high",
				Description: "First feature",
				Passes:      true,
			},
			{
				ID:          "feat-002",
				Category:    "functional",
				Priority:    "medium",
				Description: "Second feature",
				Passes:      false,
			},
		},
	}
}

func TestNewModel(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	assert.Equal(t, p, m.PRD, "PRD not set correctly")
	assert.Equal(t, "prd.json", m.PRDPath)
	assert.Equal(t, 10, m.MaxIterations)
	assert.Equal(t, StateIdle, m.State, "Initial state should be StateIdle")
	require.NotNil(t, m.LogView, "LogView should be initialized")
	require.NotNil(t, m.PhaseIndicator, "PhaseIndicator should be initialized")
	require.NotNil(t, m.ActionPanel, "ActionPanel should be initialized")
	assert.False(t, m.DebugMode, "DebugMode should be false initially")
	assert.Equal(t, components.PhaseNone, m.CurrentPhase, "CurrentPhase should be PhaseNone")
}

func TestModelInit(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return a command")
}

func TestModelUpdateQuit(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	quitCalled := false
	m.OnQuit = func() { quitCalled = true }

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	assert.True(t, quitCalled, "OnQuit callback should be called")
	require.NotNil(t, cmd, "Quit should return tea.Quit command")
	_ = newModel
}

func TestModelUpdatePause(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.State = StateRunning

	pauseCalled := false
	m.OnPause = func() { pauseCalled = true }

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m2 := newModel.(Model)

	assert.True(t, pauseCalled, "OnPause callback should be called")
	assert.Equal(t, StatePaused, m2.State, "State should be StatePaused")
}

func TestModelUpdateResume(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.State = StatePaused

	resumeCalled := false
	m.OnResume = func() { resumeCalled = true }

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := newModel.(Model)

	assert.True(t, resumeCalled, "OnResume callback should be called")
	assert.Equal(t, StateRunning, m2.State, "State should be StateRunning")
}

func TestModelUpdateDebugToggle(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	debugCalled := false
	var debugValue bool
	m.OnDebug = func(enabled bool) {
		debugCalled = true
		debugValue = enabled
	}

	// Toggle on
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m2 := newModel.(Model)

	assert.True(t, debugCalled, "OnDebug callback should be called")
	assert.True(t, debugValue, "Debug should be enabled")
	assert.True(t, m2.DebugMode, "DebugMode should be true")

	// Toggle off
	newModel, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m3 := newModel.(Model)

	assert.False(t, m3.DebugMode, "DebugMode should be false after second toggle")
}

func TestModelUpdateWindowSize(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m2 := newModel.(Model)

	assert.Equal(t, 100, m2.Width)
	assert.Equal(t, 50, m2.Height)
	assert.Equal(t, 96, m2.LogView.Width) // 100 - 4
}

func TestModelUpdateLogMsg(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.Update(LogMsg("Test log message"))

	lines := m.LogView.GetLastLines(1)
	require.Len(t, lines, 1)
	assert.Equal(t, "Test log message", lines[0])
}

func TestModelUpdateStateChange(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(StateChangeMsg(StateRunning))
	m2 := newModel.(Model)

	assert.Equal(t, StateRunning, m2.State, "State should be StateRunning")
	assert.False(t, m2.StartTime.IsZero(), "StartTime should be set when entering running state")
}

func TestModelUpdateIterationStart(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	feature := &prd.Feature{ID: "feat-001", Description: "Test"}

	newModel, _ := m.Update(IterationStartMsg{Iteration: 5, Feature: feature})
	m2 := newModel.(Model)

	assert.Equal(t, 5, m2.CurrentIteration)
	assert.Equal(t, feature, m2.CurrentFeature, "CurrentFeature not set correctly")
	assert.Equal(t, 0, m2.RetryCount, "RetryCount should be reset to 0")
}

func TestModelUpdatePhaseChange(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(PhaseChangeMsg{Phase: components.PhasePlanning})
	m2 := newModel.(Model)

	assert.Equal(t, components.PhasePlanning, m2.CurrentPhase, "CurrentPhase should be PhasePlanning")
	assert.Equal(t, components.PhasePlanning, m2.PhaseIndicator.CurrentPhase, "PhaseIndicator phase should be updated")
}

func TestModelUpdateActionAdd(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	action := components.ActionItem{
		ID:          "test-action",
		Type:        "read",
		Description: "Reading file",
		Status:      components.StatusPending,
	}

	newModel, _ := m.Update(ActionAddMsg{Action: action})
	m2 := newModel.(Model)

	require.Len(t, m2.ActionPanel.Actions, 1)
	assert.Equal(t, "test-action", m2.ActionPanel.Actions[0].ID, "Action not added correctly")
}

func TestModelUpdateActionUpdate(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.ActionPanel.AddAction(components.ActionItem{
		ID:     "test-action",
		Status: components.StatusPending,
	})

	newModel, _ := m.Update(ActionUpdateMsg{
		ID:     "test-action",
		Status: components.StatusDone,
		Output: "Success",
	})
	m2 := newModel.(Model)

	assert.Equal(t, components.StatusDone, m2.ActionPanel.Actions[0].Status, "Action status should be StatusDone")
	assert.Equal(t, "Success", m2.ActionPanel.Actions[0].Output, "Action output should be 'Success'")
}

func TestModelUpdateActionClear(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.ActionPanel.AddAction(components.ActionItem{ID: "1", Status: components.StatusPending})
	m.ActionPanel.AddAction(components.ActionItem{ID: "2", Status: components.StatusRunning})

	newModel, _ := m.Update(ActionClearMsg{})
	m2 := newModel.(Model)

	assert.Len(t, m2.ActionPanel.Actions, 0, "Expected 0 actions after clear")
}

func TestModelView(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	view := m.View()

	// Should contain header with project name
	assert.Contains(t, view, "SuperRalph", "View should contain 'SuperRalph' header")

	// Should contain project name
	assert.Contains(t, view, "Test Project", "View should contain project name")

	// Should contain progress
	assert.Contains(t, view, "Progress", "View should contain Progress section")

	// Should contain help keys
	assert.Contains(t, view, "[q] Quit", "View should contain quit help")
}

func TestModelViewWithPhase(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.CurrentPhase = components.PhasePlanning
	m.PhaseIndicator.SetPhase(components.PhasePlanning)

	view := m.View()

	// Should contain phase indicator with labels
	assert.Contains(t, view, "PLAN", "View should contain PLAN in phase indicator")
}

func TestModelViewWithActions(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.ActionPanel.AddAction(components.ActionItem{
		ID:          "1",
		Type:        "read",
		Description: "Reading test.go",
		Status:      components.StatusRunning,
	})

	view := m.View()

	// Should contain action description
	assert.Contains(t, view, "Reading test.go", "View should contain action description")
}

func TestModelHelperMethods(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	// Test AddLog
	m.AddLog("Test message")
	lines := m.LogView.GetLastLines(1)
	require.Len(t, lines, 1, "AddLog not working")
	assert.Equal(t, "Test message", lines[0], "AddLog not working")

	// Test SetState
	m.SetState(StateRunning)
	assert.Equal(t, StateRunning, m.State, "SetState not working")

	// Test SetPhase
	m.SetPhase(components.PhaseValidating)
	assert.Equal(t, components.PhaseValidating, m.CurrentPhase, "SetPhase not working")

	// Test AddAction
	m.AddAction(components.ActionItem{ID: "test", Status: components.StatusPending})
	assert.Len(t, m.ActionPanel.Actions, 1, "AddAction not working")

	// Test UpdateAction
	m.UpdateAction("test", components.StatusDone, "output")
	assert.Equal(t, components.StatusDone, m.ActionPanel.Actions[0].Status, "UpdateAction not working")

	// Test ClearActions
	m.ClearActions()
	assert.Len(t, m.ActionPanel.Actions, 0, "ClearActions not working")

	// Test SetDebugMode
	m.SetDebugMode(true)
	assert.True(t, m.DebugMode, "SetDebugMode not working")

	// Test IsDebugMode
	assert.True(t, m.IsDebugMode(), "IsDebugMode not working")
}

func TestRunStateString(t *testing.T) {
	testCases := []struct {
		state    RunState
		expected string
	}{
		{StateIdle, "idle"},
		{StateRunning, "running"},
		{StatePaused, "paused"},
		{StateComplete, "complete"},
		{StateError, "error"},
		{RunState(99), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.state.String()
		assert.Equal(t, tc.expected, result, "RunState(%d).String()", tc.state)
	}
}

func TestModelUpdatePRD(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newPRD := &prd.PRD{
		Name: "Updated Project",
		Features: []prd.Feature{
			{ID: "feat-001", Passes: true},
			{ID: "feat-002", Passes: true},
			{ID: "feat-003", Passes: true},
		},
	}

	m.UpdatePRD(newPRD)

	assert.Equal(t, "Updated Project", m.PRD.Name, "PRD not updated")
	assert.Equal(t, 3, m.PRDStats.TotalFeatures, "PRDStats not updated")
}

func TestModelUpdateError(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(ErrorMsgType{Error: "Something went wrong"})
	m2 := newModel.(Model)

	assert.Equal(t, StateError, m2.State, "State should be StateError")
	assert.Equal(t, "Something went wrong", m2.ErrorMsg, "ErrorMsg should be 'Something went wrong'")
}

func TestModelUpdateBuildComplete(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.State = StateRunning

	// Test successful completion
	newModel, _ := m.Update(BuildCompleteMsg{Success: true, Error: nil})
	m2 := newModel.(Model)

	assert.Equal(t, StateComplete, m2.State, "State should be StateComplete on success")
	assert.Empty(t, m2.ErrorMsg, "ErrorMsg should be empty on success")

	// Test failed completion
	m3 := NewModel(p, "prd.json", 10)
	m3.State = StateRunning

	newModel, _ = m3.Update(BuildCompleteMsg{Success: false, Error: fmt.Errorf("build failed")})
	m4 := newModel.(Model)

	assert.Equal(t, StateError, m4.State, "State should be StateError on failure")
	assert.Equal(t, "build failed", m4.ErrorMsg, "ErrorMsg should contain error message")
}

func TestModelUpdateBuildCompleteNoError(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.State = StateRunning

	// Test failed completion without error (e.g., canceled)
	newModel, _ := m.Update(BuildCompleteMsg{Success: false, Error: nil})
	m2 := newModel.(Model)

	assert.Equal(t, StateError, m2.State, "State should be StateError on failure")
	assert.Empty(t, m2.ErrorMsg, "ErrorMsg should be empty when no error provided")
}

// Tab navigation tests

func TestModelHasTabBar(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	require.NotNil(t, m.TabBar, "TabBar should be initialized")
	assert.Equal(t, components.TabDashboard, m.ActiveTab, "Default active tab should be Dashboard")
}

func TestModelHasLogTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	require.NotNil(t, m.LogTab, "LogTab should be initialized")
}

func TestModelHasDashboard(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	require.NotNil(t, m.Dashboard, "Dashboard should be initialized")
	assert.Equal(t, "Test Project", m.Dashboard.PRDName, "Dashboard should have PRD name")
}

func TestModelUpdateTabViaNumberKey(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	// Press "2" to switch to Features tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m2 := newModel.(Model)

	assert.Equal(t, components.TabFeatures, m2.ActiveTab, "ActiveTab should be Features after pressing 2")
	assert.Equal(t, components.TabFeatures, m2.TabBar.GetActiveTab(), "TabBar should also be updated")

	// Press "3" to switch to Logs tab
	newModel, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m3 := newModel.(Model)

	assert.Equal(t, components.TabLogs, m3.ActiveTab, "ActiveTab should be Logs after pressing 3")

	// Press "1" to switch back to Dashboard tab
	newModel, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m4 := newModel.(Model)

	assert.Equal(t, components.TabDashboard, m4.ActiveTab, "ActiveTab should be Dashboard after pressing 1")
}

func TestModelUpdateTabViaTabKey(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	assert.Equal(t, components.TabDashboard, m.ActiveTab)

	// Press Tab to go to next tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m2 := newModel.(Model)

	assert.Equal(t, components.TabFeatures, m2.ActiveTab, "Tab key should move to next tab")
}

func TestModelUpdateTabViaShiftTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	assert.Equal(t, components.TabDashboard, m.ActiveTab)

	// Press Shift+Tab to go to previous tab (wraps around)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m2 := newModel.(Model)

	assert.Equal(t, components.TabLogs, m2.ActiveTab, "Shift+Tab should wrap to last tab")
}

func TestModelUpdateTabChangeMsg(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(TabChangeMsg{Tab: components.TabLogs})
	m2 := newModel.(Model)

	assert.Equal(t, components.TabLogs, m2.ActiveTab, "TabChangeMsg should update ActiveTab")
	assert.Equal(t, components.TabLogs, m2.TabBar.GetActiveTab(), "TabBar should also be updated")
}

func TestModelAutoScrollToggleOnLogsTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	// Switch to Logs tab
	m.ActiveTab = components.TabLogs

	// Toggle auto-scroll
	assert.True(t, m.LogTab.IsAutoScrollEnabled())

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m2 := newModel.(Model)

	assert.False(t, m2.LogTab.IsAutoScrollEnabled(), "Auto-scroll should be disabled after pressing 'a'")

	// Toggle again
	newModel, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m3 := newModel.(Model)

	assert.True(t, m3.LogTab.IsAutoScrollEnabled(), "Auto-scroll should be enabled after pressing 'a' again")
}

func TestModelAutoScrollToggleNotOnOtherTabs(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	// On Dashboard tab
	m.ActiveTab = components.TabDashboard
	initialAutoScroll := m.LogTab.IsAutoScrollEnabled()

	// Press 'a' should not affect auto-scroll on Dashboard tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m2 := newModel.(Model)

	assert.Equal(t, initialAutoScroll, m2.LogTab.IsAutoScrollEnabled(), "Auto-scroll should not change on Dashboard tab")
}

func TestModelViewContainsTabBar(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	view := m.View()

	// Should contain tab labels
	assert.Contains(t, view, "Dashboard", "View should contain Dashboard tab")
	assert.Contains(t, view, "Features", "View should contain Features tab")
	assert.Contains(t, view, "Logs", "View should contain Logs tab")
}

func TestModelViewRendersDashboardTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.ActiveTab = components.TabDashboard

	view := m.View()

	// Dashboard should show progress
	assert.Contains(t, view, "Progress", "Dashboard view should contain Progress")
	// Dashboard should show status
	assert.Contains(t, view, "Status", "Dashboard view should contain Status")
}

func TestModelViewRendersLogsTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.ActiveTab = components.TabLogs

	view := m.View()

	// Logs tab should show Claude Output
	assert.Contains(t, view, "Claude Output", "Logs view should contain Claude Output")
	// Logs tab should show auto-scroll status
	assert.Contains(t, view, "Auto-Scroll", "Logs view should show auto-scroll status")
}

func TestModelViewRendersFeaturesTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.ActiveTab = components.TabFeatures

	view := m.View()

	// Features tab should show feature list
	assert.Contains(t, view, "Features", "Features view should contain Features")
}

func TestModelLogMsgUpdatesLogTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.Update(LogMsg("Test log entry"))

	// Check LogView has the entry
	lines := m.LogView.GetLastLines(1)
	require.Len(t, lines, 1)
	assert.Equal(t, "Test log entry", lines[0])

	// Check LogTab also has the entry
	ltLines := m.LogTab.GetLastLines(1)
	require.Len(t, ltLines, 1)
	assert.Equal(t, "Test log entry", ltLines[0])
}

func TestModelTypedLogMsgUpdatesLogTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.Update(TypedLogMsg{Type: components.LogTypeToolUse, Content: "Using Read tool"})

	// Check both LogView and LogTab have the entry
	assert.Equal(t, 1, len(m.LogView.Entries))
	assert.Equal(t, 1, m.LogTab.GetEntryCount())
}

func TestModelWindowSizeUpdatesTabBar(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := newModel.(Model)

	assert.Equal(t, 120, m2.TabBar.Width, "TabBar width should be updated")
}

func TestModelHelpShowsTabNavigation(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	view := m.View()

	assert.Contains(t, view, "Switch tabs", "Help should mention tab switching")
}

func TestModelHelpShowsAutoScrollOnLogsTab(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.ActiveTab = components.TabLogs

	view := m.View()

	assert.Contains(t, view, "[a] Auto-scroll", "Help should show auto-scroll toggle on Logs tab")
}

func TestRunStateToDashboardState(t *testing.T) {
	tests := []struct {
		input    RunState
		expected components.DashboardState
	}{
		{StateIdle, components.DashboardStateIdle},
		{StateRunning, components.DashboardStateRunning},
		{StatePaused, components.DashboardStatePaused},
		{StateComplete, components.DashboardStateComplete},
		{StateError, components.DashboardStateError},
		{RunState(99), components.DashboardStateIdle}, // Unknown defaults to Idle
	}

	for _, tt := range tests {
		result := runStateToDashboardState(tt.input)
		assert.Equal(t, tt.expected, result, "runStateToDashboardState(%v)", tt.input)
	}
}
