package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPhaseIndicator(t *testing.T) {
	p := NewPhaseIndicator()

	require.NotNil(t, p, "NewPhaseIndicator returned nil")
	assert.Equal(t, PhaseNone, p.CurrentPhase, "Expected initial phase to be PhaseNone")
	assert.Equal(t, 60, p.Width, "Expected default width 60")
}

func TestPhaseIndicatorSetPhase(t *testing.T) {
	p := NewPhaseIndicator()

	testCases := []struct {
		phase    Phase
		expected Phase
	}{
		{PhasePlanning, PhasePlanning},
		{PhaseValidating, PhaseValidating},
		{PhaseExecuting, PhaseExecuting},
		{PhaseComplete, PhaseComplete},
		{PhaseNone, PhaseNone},
	}

	for _, tc := range testCases {
		p.SetPhase(tc.phase)
		assert.Equal(t, tc.expected, p.CurrentPhase, "After SetPhase(%s)", tc.phase)
	}
}

func TestPhaseIndicatorRenderEmpty(t *testing.T) {
	p := NewPhaseIndicator()
	p.SetPhase(PhaseNone)

	result := p.Render()
	assert.Equal(t, "", result, "Expected empty render for PhaseNone")
}

func TestPhaseIndicatorRenderContainsLabels(t *testing.T) {
	p := NewPhaseIndicator()

	testCases := []Phase{
		PhasePlanning,
		PhaseValidating,
		PhaseExecuting,
		PhaseComplete,
	}

	for _, phase := range testCases {
		p.SetPhase(phase)
		result := p.Render()

		// Check that all three phase labels are present
		assert.Contains(t, result, "PLAN", "Phase %s render missing 'PLAN' label", phase)
		assert.Contains(t, result, "VALIDATE", "Phase %s render missing 'VALIDATE' label", phase)
		assert.Contains(t, result, "EXECUTE", "Phase %s render missing 'EXECUTE' label", phase)

		// Check arrows are present
		assert.Contains(t, result, "->", "Phase %s render missing '->' arrows", phase)
	}
}

func TestPhaseConstants(t *testing.T) {
	// Verify phase constants have expected values
	assert.Equal(t, Phase("planning"), PhasePlanning)
	assert.Equal(t, Phase("validating"), PhaseValidating)
	assert.Equal(t, Phase("executing"), PhaseExecuting)
	assert.Equal(t, Phase("complete"), PhaseComplete)
	assert.Equal(t, Phase(""), PhaseNone)
}
