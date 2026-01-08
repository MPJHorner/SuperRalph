package agent

import (
	"testing"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/stretchr/testify/assert"
)

func TestBuildPrompt(t *testing.T) {
	p := &prd.PRD{
		Name:        "Test Project",
		Description: "Test description",
		TestCommand: "npm test",
		Features: []prd.Feature{
			{
				ID:          "feat-001",
				Category:    prd.CategoryFunctional,
				Priority:    prd.PriorityHigh,
				Description: "Test feature",
				Steps:       []string{"Step 1"},
				Passes:      false,
			},
		},
	}

	prompt := BuildPrompt(p, 5)

	// Verify prompt contains key elements
	expectedParts := []string{
		"@prd.json",
		"@progress.txt",
		"npm test",                    // Test command should appear multiple times
		"Iteration: 5",                // Iteration number
		"TESTS MUST PASS",             // Critical rule
		"SMART FEATURE SELECTION",     // Feature selection rule
		"NEVER COMMIT",                // No commit without tests
		"<promise>COMPLETE</promise>", // Completion signal
	}

	for _, part := range expectedParts {
		assert.Contains(t, prompt, part, "Prompt missing expected part")
	}
}

func TestBuildPlanPrompt(t *testing.T) {
	prompt := BuildPlanPrompt()

	// Verify prompt contains key elements
	expectedParts := []string{
		"PRD",
		"prd.json",
		"testCommand",
		"features",
		"category",
		"priority",
		"functional",
		"ui",
		"integration",
		"high",
		"medium",
		"low",
		"Write tool", // Must use Write tool to create file
	}

	for _, part := range expectedParts {
		assert.Contains(t, prompt, part, "Plan prompt missing expected part")
	}
}

func TestContainsCompletionSignal(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"<promise>COMPLETE</promise>", true},
		{"Some text before <promise>COMPLETE</promise> and after", true},
		{"No completion signal here", false},
		{"<promise>INCOMPLETE</promise>", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got := ContainsCompletionSignal(tt.output)
			assert.Equal(t, tt.want, got, "ContainsCompletionSignal(%q)", tt.output)
		})
	}
}

func TestContainsError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"error lowercase", "error: something went wrong", true},
		{"Error mixed case", "Error: something went wrong", true},
		{"ERROR uppercase", "ERROR: something went wrong", true},
		{"fatal", "fatal: git error", true},
		{"FAILED", "Tests FAILED", true},
		{"failed to", "failed to compile", true},
		{"panic", "panic: runtime error", true},
		{"no error", "All tests passed successfully", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsError(tt.output)
			assert.Equal(t, tt.want, got, "ContainsError(%q)", tt.output)
		})
	}
}
