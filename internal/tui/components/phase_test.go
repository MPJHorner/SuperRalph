package components

import (
	"strings"
	"testing"
)

func TestNewPhaseIndicator(t *testing.T) {
	p := NewPhaseIndicator()

	if p == nil {
		t.Fatal("NewPhaseIndicator returned nil")
	}
	if p.CurrentPhase != PhaseNone {
		t.Errorf("Expected initial phase to be PhaseNone, got %s", p.CurrentPhase)
	}
	if p.Width != 60 {
		t.Errorf("Expected default width 60, got %d", p.Width)
	}
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
		if p.CurrentPhase != tc.expected {
			t.Errorf("After SetPhase(%s), expected %s, got %s", tc.phase, tc.expected, p.CurrentPhase)
		}
	}
}

func TestPhaseIndicatorRenderEmpty(t *testing.T) {
	p := NewPhaseIndicator()
	p.SetPhase(PhaseNone)

	result := p.Render()
	if result != "" {
		t.Errorf("Expected empty render for PhaseNone, got %q", result)
	}
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
		if !strings.Contains(result, "PLAN") {
			t.Errorf("Phase %s render missing 'PLAN' label", phase)
		}
		if !strings.Contains(result, "VALIDATE") {
			t.Errorf("Phase %s render missing 'VALIDATE' label", phase)
		}
		if !strings.Contains(result, "EXECUTE") {
			t.Errorf("Phase %s render missing 'EXECUTE' label", phase)
		}

		// Check arrows are present
		if !strings.Contains(result, "->") {
			t.Errorf("Phase %s render missing '->' arrows", phase)
		}
	}
}

func TestPhaseConstants(t *testing.T) {
	// Verify phase constants have expected values
	if PhasePlanning != "planning" {
		t.Errorf("PhasePlanning expected 'planning', got %q", PhasePlanning)
	}
	if PhaseValidating != "validating" {
		t.Errorf("PhaseValidating expected 'validating', got %q", PhaseValidating)
	}
	if PhaseExecuting != "executing" {
		t.Errorf("PhaseExecuting expected 'executing', got %q", PhaseExecuting)
	}
	if PhaseComplete != "complete" {
		t.Errorf("PhaseComplete expected 'complete', got %q", PhaseComplete)
	}
	if PhaseNone != "" {
		t.Errorf("PhaseNone expected empty string, got %q", PhaseNone)
	}
}
