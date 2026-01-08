package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mpjhorner/superralph/internal/orchestrator"
	"github.com/mpjhorner/superralph/internal/prd"
)

func createTestPRDForDashboard() *prd.PRD {
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

func TestNewDashboard(t *testing.T) {
	d := NewDashboard(80, 24)

	require.NotNil(t, d, "NewDashboard should return non-nil")
	assert.Equal(t, 80, d.Width)
	assert.Equal(t, 24, d.Height)
	assert.Equal(t, DashboardStateIdle, d.State)
	assert.Equal(t, 3, d.MaxRetries)
	assert.Equal(t, PhaseNone, d.CurrentPhase)
	assert.Equal(t, orchestrator.StepIdle, d.CurrentStep)
	require.NotNil(t, d.PhaseIndicator)
	require.NotNil(t, d.StepIndicator)
	require.NotNil(t, d.ActionPanel)
}

func TestDashboardSetPRD(t *testing.T) {
	d := NewDashboard(80, 24)
	p := createTestPRDForDashboard()

	d.SetPRD(p, "test-prd.json")

	assert.Equal(t, "Test Project", d.PRDName)
	assert.Equal(t, "test-prd.json", d.PRDPath)
	assert.Equal(t, 2, d.PRDStats.TotalFeatures)
	assert.Equal(t, 1, d.PRDStats.PassingFeatures)
}

func TestDashboardSetState(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetState(DashboardStateRunning)
	assert.Equal(t, DashboardStateRunning, d.State)
	assert.False(t, d.StartTime.IsZero(), "StartTime should be set when state is Running")

	d.SetState(DashboardStatePaused)
	assert.Equal(t, DashboardStatePaused, d.State)

	d.SetState(DashboardStateComplete)
	assert.Equal(t, DashboardStateComplete, d.State)

	d.SetState(DashboardStateError)
	assert.Equal(t, DashboardStateError, d.State)
}

func TestDashboardSetIteration(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetIteration(5, 10)

	assert.Equal(t, 5, d.CurrentIteration)
	assert.Equal(t, 10, d.MaxIterations)
}

func TestDashboardSetFeature(t *testing.T) {
	d := NewDashboard(80, 24)
	f := &prd.Feature{
		ID:          "feat-001",
		Description: "Test feature",
	}

	d.SetFeature(f)

	assert.Equal(t, f, d.CurrentFeature)
}

func TestDashboardSetPhase(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetPhase(PhasePlanning)

	assert.Equal(t, PhasePlanning, d.CurrentPhase)
	assert.Equal(t, PhasePlanning, d.PhaseIndicator.CurrentPhase)
}

func TestDashboardSetStep(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetStep(orchestrator.StepCoding)

	assert.Equal(t, orchestrator.StepCoding, d.CurrentStep)
	assert.Equal(t, orchestrator.StepCoding, d.StepIndicator.CurrentStep)
}

func TestDashboardSetActivity(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetActivity("Reading files")

	assert.Equal(t, "Reading files", d.CurrentActivity)
}

func TestDashboardSetError(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetError("Something went wrong")

	assert.Equal(t, "Something went wrong", d.ErrorMsg)
}

func TestDashboardSetRetry(t *testing.T) {
	d := NewDashboard(80, 24)

	d.SetRetry(2, 5)

	assert.Equal(t, 2, d.RetryCount)
	assert.Equal(t, 5, d.MaxRetries)
}

func TestDashboardUpdateStats(t *testing.T) {
	d := NewDashboard(80, 24)
	stats := prd.PRDStats{
		TotalFeatures:   10,
		PassingFeatures: 7,
	}

	d.UpdateStats(stats)

	assert.Equal(t, 10, d.PRDStats.TotalFeatures)
	assert.Equal(t, 7, d.PRDStats.PassingFeatures)
}

func TestDashboardRender(t *testing.T) {
	d := NewDashboard(80, 24)
	p := createTestPRDForDashboard()
	d.SetPRD(p, "test-prd.json")

	rendered := d.Render()

	// Should contain progress section
	assert.Contains(t, rendered, "Progress")

	// Should contain status
	assert.Contains(t, rendered, "Status")
}

func TestDashboardRenderWithRunningState(t *testing.T) {
	d := NewDashboard(80, 24)
	p := createTestPRDForDashboard()
	d.SetPRD(p, "test-prd.json")
	d.SetState(DashboardStateRunning)
	d.SetIteration(3, 10)
	d.SetActivity("Writing code")
	d.SetPhase(PhaseExecuting)

	rendered := d.Render()

	// Should contain running indicator
	assert.Contains(t, rendered, "RUNNING")

	// Should contain iteration info
	assert.Contains(t, rendered, "3/10")

	// Should contain phase
	assert.Contains(t, rendered, "Phase")
}

func TestDashboardRenderWithError(t *testing.T) {
	d := NewDashboard(80, 24)
	p := createTestPRDForDashboard()
	d.SetPRD(p, "test-prd.json")
	d.SetState(DashboardStateError)
	d.SetError("Test error message")

	rendered := d.Render()

	assert.Contains(t, rendered, "ERROR")
	assert.Contains(t, rendered, "Test error message")
}

func TestDashboardRenderWithFeature(t *testing.T) {
	d := NewDashboard(80, 24)
	p := createTestPRDForDashboard()
	d.SetPRD(p, "test-prd.json")
	d.SetState(DashboardStateRunning)
	d.SetFeature(&prd.Feature{
		ID:          "feat-001",
		Description: "Test feature description",
	})

	rendered := d.Render()

	assert.Contains(t, rendered, "feat-001")
	assert.Contains(t, rendered, "Test feature description")
}

func TestDashboardActionPanel(t *testing.T) {
	d := NewDashboard(80, 24)

	action := ActionItem{
		ID:          "action-1",
		Type:        "read",
		Description: "Reading file",
		Status:      StatusRunning,
	}

	d.AddAction(action)
	assert.Len(t, d.ActionPanel.Actions, 1)

	d.UpdateAction("action-1", StatusDone, "Success")
	assert.Equal(t, StatusDone, d.ActionPanel.Actions[0].Status)

	d.ClearActions()
	assert.Len(t, d.ActionPanel.Actions, 0)
}

func TestDashboardStateConstants(t *testing.T) {
	assert.Equal(t, DashboardState("idle"), DashboardStateIdle)
	assert.Equal(t, DashboardState("running"), DashboardStateRunning)
	assert.Equal(t, DashboardState("paused"), DashboardStatePaused)
	assert.Equal(t, DashboardState("complete"), DashboardStateComplete)
	assert.Equal(t, DashboardState("error"), DashboardStateError)
}

func TestDashboardStartTimeOnlySetOnce(t *testing.T) {
	d := NewDashboard(80, 24)

	// First time setting to running should set StartTime
	d.SetState(DashboardStateRunning)
	firstStartTime := d.StartTime

	// Wait a tiny bit
	time.Sleep(time.Millisecond)

	// Setting to running again should not change StartTime
	d.SetState(DashboardStateRunning)
	assert.Equal(t, firstStartTime, d.StartTime, "StartTime should not change on subsequent running state")
}

func TestDashboardStatusBadge(t *testing.T) {
	d := NewDashboard(80, 24)

	tests := []struct {
		state    DashboardState
		contains string
	}{
		{DashboardStateIdle, "IDLE"},
		{DashboardStateRunning, "RUNNING"},
		{DashboardStatePaused, "PAUSED"},
		{DashboardStateComplete, "COMPLETE"},
		{DashboardStateError, "ERROR"},
	}

	for _, tt := range tests {
		d.SetState(tt.state)
		badge := d.statusBadge()
		assert.Contains(t, badge, tt.contains, "State %s should produce badge containing %s", tt.state, tt.contains)
	}
}
