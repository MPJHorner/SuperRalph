package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActionPanel(t *testing.T) {
	p := NewActionPanel(80, 10)

	require.NotNil(t, p, "NewActionPanel returned nil")
	assert.Equal(t, 80, p.Width)
	assert.Equal(t, 10, p.Height)
	assert.Len(t, p.Actions, 0)
	assert.Equal(t, 10, p.MaxActions)
	assert.Equal(t, "Actions", p.Title)
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

	require.Len(t, p.Actions, 1)
	assert.Equal(t, "action-1", p.Actions[0].ID)
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

	require.Len(t, p.Actions, 3)

	// Should keep the last 3 actions
	expectedIDs := []string{"c", "d", "e"}
	for i, expected := range expectedIDs {
		assert.Equal(t, expected, p.Actions[i].ID, "action[%d].ID", i)
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

	assert.Equal(t, StatusDone, p.Actions[0].Status)
	assert.Equal(t, "All tests passed", p.Actions[0].Output)
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
	assert.Equal(t, StatusPending, p.Actions[0].Status, "Original action should be unchanged")
}

func TestActionPanelClear(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})

	p.Clear()

	assert.Len(t, p.Actions, 0, "Expected 0 actions after clear")
}

func TestActionPanelGetPendingCount(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusPending})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})

	count := p.GetPendingCount()
	assert.Equal(t, 2, count, "Expected 2 pending")
}

func TestActionPanelGetRunningCount(t *testing.T) {
	p := NewActionPanel(80, 10)

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})

	count := p.GetRunningCount()
	assert.Equal(t, 2, count, "Expected 2 running")
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
		assert.Equal(t, tc.expected, result, "statusIcon(%s)", tc.status)
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
		assert.Equal(t, tc.expected, result, "typeIcon(%s)", tc.actionType)
	}
}

func TestActionPanelRenderEmpty(t *testing.T) {
	p := NewActionPanel(80, 10)

	result := p.Render()
	assert.Equal(t, "", result, "Expected empty render for empty panel")
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
	assert.Contains(t, result, "Actions", "Render should contain title")

	// Should contain descriptions
	assert.Contains(t, result, "Reading main.go", "Render should contain first action description")
	assert.Contains(t, result, "Running tests", "Render should contain second action description")
}

func TestActionPanelSummary(t *testing.T) {
	p := NewActionPanel(80, 10)

	// Empty panel
	assert.Equal(t, "", p.Summary(), "Empty panel should have empty summary")

	p.AddAction(ActionItem{ID: "1", Status: StatusPending})
	p.AddAction(ActionItem{ID: "2", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "3", Status: StatusRunning})
	p.AddAction(ActionItem{ID: "4", Status: StatusDone})
	p.AddAction(ActionItem{ID: "5", Status: StatusFailed})

	summary := p.Summary()

	assert.Contains(t, summary, "2 running")
	assert.Contains(t, summary, "1 pending")
	assert.Contains(t, summary, "1 done")
	assert.Contains(t, summary, "1 failed")
}

func TestActionStatusConstants(t *testing.T) {
	// Verify status constants have expected values
	assert.Equal(t, ActionStatus("pending"), StatusPending)
	assert.Equal(t, ActionStatus("running"), StatusRunning)
	assert.Equal(t, ActionStatus("done"), StatusDone)
	assert.Equal(t, ActionStatus("failed"), StatusFailed)
	assert.Equal(t, ActionStatus("skipped"), StatusSkipped)
}
