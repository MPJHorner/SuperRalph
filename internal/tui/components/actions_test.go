package components

import (
	"strings"
	"testing"
)

func TestNewActionPanel(t *testing.T) {
	p := NewActionPanel(80, 10)

	if p == nil {
		t.Fatal("NewActionPanel returned nil")
	}
	if p.Width != 80 {
		t.Errorf("Expected width 80, got %d", p.Width)
	}
	if p.Height != 10 {
		t.Errorf("Expected height 10, got %d", p.Height)
	}
	if len(p.Actions) != 0 {
		t.Errorf("Expected empty actions, got %d", len(p.Actions))
	}
	if p.MaxActions != 10 {
		t.Errorf("Expected MaxActions 10, got %d", p.MaxActions)
	}
	if p.Title != "Actions" {
		t.Errorf("Expected title 'Actions', got %q", p.Title)
	}
}

func TestActionPanelAddAction(t *testing.T) {
	p := NewActionPanel(80, 10)

	action := ActionItem{
		ID:          "action-1",
		Type:        "read",
		Description: "Reading file.go",
		Status:      StatusPending,
	}

	p.AddAction(action)

	if len(p.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(p.Actions))
	}
	if p.Actions[0].ID != "action-1" {
		t.Errorf("Expected action ID 'action-1', got %q", p.Actions[0].ID)
	}
}

func TestActionPanelMaxActions(t *testing.T) {
	p := NewActionPanel(80, 10)
	p.MaxActions = 3

	for i := 0; i < 5; i++ {
		p.AddAction(ActionItem{
			ID:          string(rune('a' + i)),
			Type:        "read",
			Description: "test",
			Status:      StatusPending,
		})
	}

	if len(p.Actions) != 3 {
		t.Errorf("Expected 3 actions (max), got %d", len(p.Actions))
	}

	// Should keep the last 3 actions
	expectedIDs := []string{"c", "d", "e"}
	for i, expected := range expectedIDs {
		if p.Actions[i].ID != expected {
			t.Errorf("Expected action[%d].ID to be %q, got %q", i, expected, p.Actions[i].ID)
		}
	}
}

func TestActionPanelUpdateAction(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{
		ID:          "action-1",
		Type:        "command",
		Description: "Running tests",
		Status:      StatusRunning,
	})

	p.UpdateAction("action-1", StatusDone, "All tests passed")

	if p.Actions[0].Status != StatusDone {
		t.Errorf("Expected status StatusDone, got %s", p.Actions[0].Status)
	}
	if p.Actions[0].Output != "All tests passed" {
		t.Errorf("Expected output 'All tests passed', got %q", p.Actions[0].Output)
	}
}

func TestActionPanelUpdateActionNotFound(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{
		ID:     "action-1",
		Status: StatusPending,
	})

	// Should not panic when updating non-existent action
	p.UpdateAction("non-existent", StatusDone, "output")

	// Original action should be unchanged
	if p.Actions[0].Status != StatusPending {
		t.Errorf("Original action should be unchanged")
	}
}

func TestActionPanelClear(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})

	p.Clear()

	if len(p.Actions) != 0 {
		t.Errorf("Expected 0 actions after clear, got %d", len(p.Actions))
	}
}

func TestActionPanelGetPendingCount(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusPending})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})

	count := p.GetPendingCount()
	if count != 2 {
		t.Errorf("Expected 2 pending, got %d", count)
	}
}

func TestActionPanelGetRunningCount(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})

	count := p.GetRunningCount()
	if count != 2 {
		t.Errorf("Expected 2 running, got %d", count)
	}
}

func TestStatusIcon(t *testing.T) {
	testCases := []struct {
		status   ActionStatus
		expected string
	}{
		{StatusPending, "○"},
		{StatusRunning, "◐"},
		{StatusDone, "●"},
		{StatusFailed, "✗"},
		{StatusSkipped, "◌"},
		{ActionStatus("unknown"), "?"},
	}

	for _, tc := range testCases {
		result := statusIcon(tc.status)
		if result != tc.expected {
			t.Errorf("statusIcon(%s) expected %q, got %q", tc.status, tc.expected, result)
		}
	}
}

func TestTypeIcon(t *testing.T) {
	testCases := []struct {
		actionType string
		expected   string
	}{
		{"read", "📖"},
		{"read_files", "📖"},
		{"write", "📝"},
		{"write_file", "📝"},
		{"command", "⚡"},
		{"run_command", "⚡"},
		{"bash", "⚡"},
		{"edit", "✏️"},
		{"parallel", "⇶"},
		{"unknown", "•"},
	}

	for _, tc := range testCases {
		result := typeIcon(tc.actionType)
		if result != tc.expected {
			t.Errorf("typeIcon(%s) expected %q, got %q", tc.actionType, tc.expected, result)
		}
	}
}

func TestActionPanelRenderEmpty(t *testing.T) {
	p := NewActionPanel(80, 10)

	result := p.Render()
	if result != "" {
		t.Errorf("Expected empty render for empty panel, got %q", result)
	}
}

func TestActionPanelRenderWithActions(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{
		ID:          "1",
		Type:        "read",
		Description: "Reading main.go",
		Status:      StatusDone,
	})
	p.AddAction(ActionItem{
		ID:          "2",
		Type:        "command",
		Description: "Running tests",
		Status:      StatusRunning,
	})

	result := p.Render()

	// Should contain title
	if !strings.Contains(result, "Actions") {
		t.Error("Render should contain title")
	}

	// Should contain descriptions
	if !strings.Contains(result, "Reading main.go") {
		t.Error("Render should contain first action description")
	}
	if !strings.Contains(result, "Running tests") {
		t.Error("Render should contain second action description")
	}
}

func TestActionPanelSummary(t *testing.T) {
	p := NewActionPanel(80, 10)

	// Empty panel
	if p.Summary() != "" {
		t.Error("Empty panel should have empty summary")
	}

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})
	p.AddAction(ActionItem{ID: "5", Status: StatusFailed})

	summary := p.Summary()

	if !strings.Contains(summary, "2 running") {
		t.Errorf("Summary should contain '2 running', got %q", summary)
	}
	if !strings.Contains(summary, "1 pending") {
		t.Errorf("Summary should contain '1 pending', got %q", summary)
	}
	if !strings.Contains(summary, "1 done") {
		t.Errorf("Summary should contain '1 done', got %q", summary)
	}
	if !strings.Contains(summary, "1 failed") {
		t.Errorf("Summary should contain '1 failed', got %q", summary)
	}
}

func TestActionStatusConstants(t *testing.T) {
	// Verify status constants have expected values
	if StatusPending != "pending" {
		t.Errorf("StatusPending expected 'pending', got %q", StatusPending)
	}
	if StatusRunning != "running" {
		t.Errorf("StatusRunning expected 'running', got %q", StatusRunning)
	}
	if StatusDone != "done" {
		t.Errorf("StatusDone expected 'done', got %q", StatusDone)
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed expected 'failed', got %q", StatusFailed)
	}
	if StatusSkipped != "skipped" {
		t.Errorf("StatusSkipped expected 'skipped', got %q", StatusSkipped)
	}
}
