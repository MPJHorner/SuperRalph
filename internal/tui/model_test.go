package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	if m.PRD != p {
		t.Error("PRD not set correctly")
	}
	if m.PRDPath != "prd.json" {
		t.Errorf("PRDPath expected 'prd.json', got %q", m.PRDPath)
	}
	if m.MaxIterations != 10 {
		t.Errorf("MaxIterations expected 10, got %d", m.MaxIterations)
	}
	if m.State != StateIdle {
		t.Errorf("Initial state should be StateIdle, got %s", m.State)
	}
	if m.LogView == nil {
		t.Error("LogView should be initialized")
	}
	if m.PhaseIndicator == nil {
		t.Error("PhaseIndicator should be initialized")
	}
	if m.ActionPanel == nil {
		t.Error("ActionPanel should be initialized")
	}
	if m.DebugMode != false {
		t.Error("DebugMode should be false initially")
	}
	if m.CurrentPhase != components.PhaseNone {
		t.Errorf("CurrentPhase should be PhaseNone, got %s", m.CurrentPhase)
	}
}

func TestModelInit(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModelUpdateQuit(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	quitCalled := false
	m.OnQuit = func() { quitCalled = true }

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if !quitCalled {
		t.Error("OnQuit callback should be called")
	}
	if cmd == nil {
		t.Error("Quit should return tea.Quit command")
	}
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

	if !pauseCalled {
		t.Error("OnPause callback should be called")
	}
	if m2.State != StatePaused {
		t.Errorf("State should be StatePaused, got %s", m2.State)
	}
}

func TestModelUpdateResume(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.State = StatePaused

	resumeCalled := false
	m.OnResume = func() { resumeCalled = true }

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := newModel.(Model)

	if !resumeCalled {
		t.Error("OnResume callback should be called")
	}
	if m2.State != StateRunning {
		t.Errorf("State should be StateRunning, got %s", m2.State)
	}
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

	if !debugCalled {
		t.Error("OnDebug callback should be called")
	}
	if !debugValue {
		t.Error("Debug should be enabled")
	}
	if !m2.DebugMode {
		t.Error("DebugMode should be true")
	}

	// Toggle off
	newModel, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m3 := newModel.(Model)

	if m3.DebugMode {
		t.Error("DebugMode should be false after second toggle")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m2 := newModel.(Model)

	if m2.Width != 100 {
		t.Errorf("Width expected 100, got %d", m2.Width)
	}
	if m2.Height != 50 {
		t.Errorf("Height expected 50, got %d", m2.Height)
	}
	if m2.LogView.Width != 96 { // 100 - 4
		t.Errorf("LogView width expected 96, got %d", m2.LogView.Width)
	}
}

func TestModelUpdateLogMsg(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.Update(LogMsg("Test log message"))

	lines := m.LogView.GetLastLines(1)
	if len(lines) != 1 || lines[0] != "Test log message" {
		t.Errorf("Log message not added correctly: %v", lines)
	}
}

func TestModelUpdateStateChange(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(StateChangeMsg(StateRunning))
	m2 := newModel.(Model)

	if m2.State != StateRunning {
		t.Errorf("State should be StateRunning, got %s", m2.State)
	}
	if m2.StartTime.IsZero() {
		t.Error("StartTime should be set when entering running state")
	}
}

func TestModelUpdateIterationStart(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	feature := &prd.Feature{ID: "feat-001", Description: "Test"}

	newModel, _ := m.Update(IterationStartMsg{Iteration: 5, Feature: feature})
	m2 := newModel.(Model)

	if m2.CurrentIteration != 5 {
		t.Errorf("CurrentIteration expected 5, got %d", m2.CurrentIteration)
	}
	if m2.CurrentFeature != feature {
		t.Error("CurrentFeature not set correctly")
	}
	if m2.RetryCount != 0 {
		t.Errorf("RetryCount should be reset to 0, got %d", m2.RetryCount)
	}
}

func TestModelUpdatePhaseChange(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(PhaseChangeMsg{Phase: components.PhasePlanning})
	m2 := newModel.(Model)

	if m2.CurrentPhase != components.PhasePlanning {
		t.Errorf("CurrentPhase should be PhasePlanning, got %s", m2.CurrentPhase)
	}
	if m2.PhaseIndicator.CurrentPhase != components.PhasePlanning {
		t.Error("PhaseIndicator phase should be updated")
	}
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

	if len(m2.ActionPanel.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(m2.ActionPanel.Actions))
	}
	if m2.ActionPanel.Actions[0].ID != "test-action" {
		t.Error("Action not added correctly")
	}
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

	if m2.ActionPanel.Actions[0].Status != components.StatusDone {
		t.Errorf("Action status should be StatusDone, got %s", m2.ActionPanel.Actions[0].Status)
	}
	if m2.ActionPanel.Actions[0].Output != "Success" {
		t.Errorf("Action output should be 'Success', got %q", m2.ActionPanel.Actions[0].Output)
	}
}

func TestModelUpdateActionClear(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	m.ActionPanel.AddAction(components.ActionItem{ID: "1", Status: components.StatusPending})
	m.ActionPanel.AddAction(components.ActionItem{ID: "2", Status: components.StatusRunning})

	newModel, _ := m.Update(ActionClearMsg{})
	m2 := newModel.(Model)

	if len(m2.ActionPanel.Actions) != 0 {
		t.Errorf("Expected 0 actions after clear, got %d", len(m2.ActionPanel.Actions))
	}
}

func TestModelView(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	view := m.View()

	// Should contain header with project name
	if !strings.Contains(view, "SuperRalph") {
		t.Error("View should contain 'SuperRalph' header")
	}

	// Should contain project name
	if !strings.Contains(view, "Test Project") {
		t.Error("View should contain project name")
	}

	// Should contain progress
	if !strings.Contains(view, "Progress") {
		t.Error("View should contain Progress section")
	}

	// Should contain help keys
	if !strings.Contains(view, "[q] Quit") {
		t.Error("View should contain quit help")
	}
}

func TestModelViewWithPhase(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)
	m.CurrentPhase = components.PhasePlanning
	m.PhaseIndicator.SetPhase(components.PhasePlanning)

	view := m.View()

	// Should contain phase indicator with labels
	if !strings.Contains(view, "PLAN") {
		t.Error("View should contain PLAN in phase indicator")
	}
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
	if !strings.Contains(view, "Reading test.go") {
		t.Error("View should contain action description")
	}
}

func TestModelHelperMethods(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	// Test AddLog
	m.AddLog("Test message")
	lines := m.LogView.GetLastLines(1)
	if len(lines) != 1 || lines[0] != "Test message" {
		t.Error("AddLog not working")
	}

	// Test SetState
	m.SetState(StateRunning)
	if m.State != StateRunning {
		t.Error("SetState not working")
	}

	// Test SetPhase
	m.SetPhase(components.PhaseValidating)
	if m.CurrentPhase != components.PhaseValidating {
		t.Error("SetPhase not working")
	}

	// Test AddAction
	m.AddAction(components.ActionItem{ID: "test", Status: components.StatusPending})
	if len(m.ActionPanel.Actions) != 1 {
		t.Error("AddAction not working")
	}

	// Test UpdateAction
	m.UpdateAction("test", components.StatusDone, "output")
	if m.ActionPanel.Actions[0].Status != components.StatusDone {
		t.Error("UpdateAction not working")
	}

	// Test ClearActions
	m.ClearActions()
	if len(m.ActionPanel.Actions) != 0 {
		t.Error("ClearActions not working")
	}

	// Test SetDebugMode
	m.SetDebugMode(true)
	if !m.DebugMode {
		t.Error("SetDebugMode not working")
	}

	// Test IsDebugMode
	if !m.IsDebugMode() {
		t.Error("IsDebugMode not working")
	}
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
		if result != tc.expected {
			t.Errorf("RunState(%d).String() expected %q, got %q", tc.state, tc.expected, result)
		}
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

	if m.PRD.Name != "Updated Project" {
		t.Error("PRD not updated")
	}
	if m.PRDStats.TotalFeatures != 3 {
		t.Errorf("PRDStats not updated, expected 3 features, got %d", m.PRDStats.TotalFeatures)
	}
}

func TestModelUpdateError(t *testing.T) {
	p := createTestPRD()
	m := NewModel(p, "prd.json", 10)

	newModel, _ := m.Update(ErrorMsgType{Error: "Something went wrong"})
	m2 := newModel.(Model)

	if m2.State != StateError {
		t.Errorf("State should be StateError, got %s", m2.State)
	}
	if m2.ErrorMsg != "Something went wrong" {
		t.Errorf("ErrorMsg should be 'Something went wrong', got %q", m2.ErrorMsg)
	}
}
