package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/progress"
)

func TestResponseParsing(t *testing.T) {
	jsonStr := `{
		"thinking": "I should ask what they're building",
		"action": "ask_user",
		"action_params": {
			"question": "What are you building?"
		},
		"message": "Let's start by understanding your project.",
		"state": {"phase": "gathering"}
	}`

	var response Response
	err := json.Unmarshal([]byte(jsonStr), &response)
	require.NoError(t, err)

	assert.Equal(t, ActionAskUser, response.Action)
	assert.Equal(t, "What are you building?", response.ActionParams.Question)
	assert.Equal(t, "I should ask what they're building", response.Thinking)
}

func TestActionTypes(t *testing.T) {
	tests := []struct {
		action Action
		valid  bool
	}{
		{ActionAskUser, true},
		{ActionReadFiles, true},
		{ActionWriteFile, true},
		{ActionRunCommand, true},
		{ActionDone, true},
		{Action("invalid"), false},
	}

	validActions := map[Action]bool{
		ActionAskUser:    true,
		ActionReadFiles:  true,
		ActionWriteFile:  true,
		ActionRunCommand: true,
		ActionDone:       true,
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			_, exists := validActions[tt.action]
			assert.Equal(t, tt.valid, exists, "action %s", tt.action)
		})
	}
}

func TestSessionSerialization(t *testing.T) {
	session := &Session{
		ID:      "test-123",
		Mode:    "plan",
		WorkDir: "/tmp/test",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		State: &PlanState{
			Phase: "gathering",
			DraftPRD: &DraftPRD{
				Name: "Test Project",
			},
		},
	}

	data, err := json.Marshal(session)
	require.NoError(t, err)

	var restored Session
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, session.ID, restored.ID)
	assert.Len(t, restored.Messages, len(session.Messages))
}

func TestOrchestratorNew(t *testing.T) {
	orch := New("/tmp/test")
	require.NotNil(t, orch)
	assert.Equal(t, "/tmp/test", orch.workDir)
	require.NotNil(t, orch.session)
	assert.NotEmpty(t, orch.session.ID)
}

func TestOrchestratorSetDebug(t *testing.T) {
	orch := New("/tmp/test")
	orch.SetDebug(true)
	assert.True(t, orch.debug)
}

func TestIterationContextBuildPrompt(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:      `{"name": "Test Project", "features": []}`,
		ProgressContent: "Feature 1 completed",
		DirectoryTree:   "├── main.go\n└── go.mod",
		TaggedFiles: map[string]string{
			"main.go": "package main\n\nfunc main() {}",
		},
		CurrentFeature: &FeatureContext{
			ID:          "feat-001",
			Description: "Test feature",
			Steps:       []string{"Step 1", "Step 2"},
			Priority:    "high",
			Category:    "functional",
		},
		Phase:     PhasePlanning,
		Iteration: 1,
	}

	prompt := ctx.BuildPrompt()

	// Check that prompt contains all expected sections
	assert.Contains(t, prompt, "## prd.json")
	assert.Contains(t, prompt, `{"name": "Test Project"`)
	assert.Contains(t, prompt, "## progress.txt")
	assert.Contains(t, prompt, "Feature 1 completed")
	assert.Contains(t, prompt, "## Directory Structure")
	assert.Contains(t, prompt, "## Tagged Files")
	assert.Contains(t, prompt, "### main.go")
	assert.Contains(t, prompt, "## Current Feature")
	assert.Contains(t, prompt, "feat-001")
	assert.Contains(t, prompt, "## Current Phase: planning")
	// When phase is set, we get phase-specific instructions instead of generic task instructions
	assert.Contains(t, prompt, "Planning Phase Instructions")
}

func TestIterationContextEmptyProgress(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:      `{"name": "Test"}`,
		ProgressContent: "",
		Iteration:       1,
	}

	prompt := ctx.BuildPrompt()

	assert.Contains(t, prompt, "(empty)")
}

func TestIterationContextNoOptionalFields(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should not contain optional sections when not set
	assert.NotContains(t, prompt, "## Directory Structure")
	assert.NotContains(t, prompt, "## Tagged Files")
	assert.NotContains(t, prompt, "## Current Feature")
	assert.NotContains(t, prompt, "## Current Phase")
}

func TestPhaseConstants(t *testing.T) {
	tests := []struct {
		phase Phase
		want  string
	}{
		{PhasePlanning, "planning"},
		{PhaseValidating, "validating"},
		{PhaseExecuting, "executing"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, string(tt.phase))
	}
}

func TestIterationContextSerialization(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:      `{"name": "Test"}`,
		ProgressContent: "progress",
		TaggedFiles:     map[string]string{"file.go": "content"},
		DirectoryTree:   "tree",
		CurrentFeature: &FeatureContext{
			ID:          "feat-001",
			Description: "desc",
			Steps:       []string{"step1"},
			Priority:    "high",
			Category:    "functional",
		},
		Phase:     PhasePlanning,
		Iteration: 1,
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var restored IterationContext
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, ctx.PRDContent, restored.PRDContent)
	assert.Equal(t, ctx.ProgressContent, restored.ProgressContent)
	assert.Equal(t, ctx.Iteration, restored.Iteration)
	assert.Equal(t, ctx.Phase, restored.Phase)
	require.NotNil(t, restored.CurrentFeature)
	assert.Equal(t, ctx.CurrentFeature.ID, restored.CurrentFeature.ID)
	assert.Len(t, restored.TaggedFiles, 1)
	assert.Equal(t, "content", restored.TaggedFiles["file.go"])
}

func TestBuildIterationContext(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create prd.json
	prdContent := `{"name": "Test Project", "features": []}`
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644)
	require.NoError(t, err)

	// Create progress.txt
	progressContent := "Feature completed"
	err = os.WriteFile(filepath.Join(tmpDir, "progress.txt"), []byte(progressContent), 0644)
	require.NoError(t, err)

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, PhasePlanning, nil)
	require.NoError(t, err)

	assert.Equal(t, prdContent, ctx.PRDContent)
	assert.Equal(t, progressContent, ctx.ProgressContent)
	assert.Equal(t, 1, ctx.Iteration)
	assert.Equal(t, PhasePlanning, ctx.Phase)
	assert.NotEmpty(t, ctx.DirectoryTree)
	assert.Contains(t, ctx.DirectoryTree, "src/")
}

func TestBuildIterationContextWithFeature(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	feature := &FeatureContext{
		ID:          "feat-001",
		Description: "Test feature",
		Steps:       []string{"Step 1"},
		Priority:    "high",
		Category:    "functional",
	}

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(2, PhaseExecuting, feature)
	require.NoError(t, err)

	require.NotNil(t, ctx.CurrentFeature)
	assert.Equal(t, "feat-001", ctx.CurrentFeature.ID)
}

func TestBuildIterationContextMissingPRD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	orch := New(tmpDir)
	_, err = orch.BuildIterationContext(1, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prd.json")
}

func TestBuildIterationContextNoProgress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Empty(t, ctx.ProgressContent)
}

func TestAddTaggedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test file
	testContent := "package main\n\nfunc main() {}"
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(testContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	// Test with relative path
	err = orch.AddTaggedFile(ctx, "main.go")
	require.NoError(t, err)

	content, ok := ctx.TaggedFiles["main.go"]
	assert.True(t, ok, "TaggedFiles should contain main.go")
	assert.Equal(t, testContent, content)
}

func TestAddTaggedFileAbsolutePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testContent := "test content"
	absPath := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(absPath, []byte(testContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	err = orch.AddTaggedFile(ctx, absPath)
	require.NoError(t, err)

	content, ok := ctx.TaggedFiles["test.txt"]
	assert.True(t, ok, "TaggedFiles should contain test.txt with relative key")
	assert.Equal(t, testContent, content)
}

func TestAddTaggedFileMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	err = orch.AddTaggedFile(ctx, "nonexistent.go")
	require.Error(t, err)
}

func TestIterationIndependence(t *testing.T) {
	// This test verifies that each iteration context is independent
	// and doesn't carry forward state from previous iterations
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)

	// Build first iteration context
	ctx1, err := orch.BuildIterationContext(1, PhasePlanning, nil)
	require.NoError(t, err)

	// Modify the context (simulating what might happen during processing)
	ctx1.TaggedFiles["added.go"] = "some content"

	// Build second iteration context
	ctx2, err := orch.BuildIterationContext(2, PhaseExecuting, nil)
	require.NoError(t, err)

	// Verify ctx2 doesn't have the modifications from ctx1
	assert.Len(t, ctx2.TaggedFiles, 0)
	assert.Equal(t, 2, ctx2.Iteration)
	assert.Equal(t, PhaseExecuting, ctx2.Phase)
}

// Tests for three-phase loop (feat-003)

func TestExtractPlan(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid plan block",
			input: "Some preamble\n<plan>\n## Implementation Plan\n\n### Steps\n1. Do something\n</plan>\nSome epilogue",
			want:  "## Implementation Plan\n\n### Steps\n1. Do something",
		},
		{
			name:  "no plan block",
			input: "Just some regular output without a plan",
			want:  "",
		},
		{
			name:  "empty plan block",
			input: "<plan></plan>",
			want:  "",
		},
		{
			name:  "unclosed plan block",
			input: "<plan>Some content without closing tag",
			want:  "",
		},
		{
			name:  "plan with whitespace",
			input: "<plan>  \n  Trimmed content  \n  </plan>",
			want:  "Trimmed content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlan(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantValid    bool
		wantIssues   int
		wantFeedback bool
	}{
		{
			name:         "valid plan",
			input:        "<validation>\nvalid: true\nissues:\nfeedback:\n</validation>",
			wantValid:    true,
			wantIssues:   0,
			wantFeedback: false,
		},
		{
			name:         "invalid plan with issues",
			input:        "<validation>\nvalid: false\nissues:\n- Missing tests\n- No error handling\nfeedback: Please add tests and error handling\n</validation>",
			wantValid:    false,
			wantIssues:   2,
			wantFeedback: true,
		},
		{
			name:         "no validation block - defaults to valid",
			input:        "Some output without validation block",
			wantValid:    true,
			wantIssues:   0,
			wantFeedback: false,
		},
		{
			name:         "valid true case insensitive",
			input:        "<validation>\nvalid: TRUE\n</validation>",
			wantValid:    true,
			wantIssues:   0,
			wantFeedback: false,
		},
		{
			name:         "valid false explicit",
			input:        "<validation>\nvalid: false\n</validation>",
			wantValid:    false,
			wantIssues:   0,
			wantFeedback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValidation(tt.input)
			assert.Equal(t, tt.wantValid, result.Valid)
			assert.Len(t, result.Issues, tt.wantIssues)
			hasFeedback := result.Feedback != ""
			assert.Equal(t, tt.wantFeedback, hasFeedback)
		})
	}
}

func TestValidationResultSerialization(t *testing.T) {
	result := ValidationResult{
		Valid:    false,
		Issues:   []string{"Issue 1", "Issue 2"},
		Feedback: "Please fix these issues",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var restored ValidationResult
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, result.Valid, restored.Valid)
	assert.Len(t, restored.Issues, len(result.Issues))
	assert.Equal(t, result.Feedback, restored.Feedback)
}

func TestPhaseConfigDefaults(t *testing.T) {
	config := PhaseConfig{}
	assert.Equal(t, 0, config.MaxValidationAttempts)

	config = PhaseConfig{MaxValidationAttempts: 5}
	assert.Equal(t, 5, config.MaxValidationAttempts)
}

func TestIterationContextWithPlanAndFeedback(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:         `{"name": "Test"}`,
		Phase:              PhasePlanning,
		ValidationFeedback: "Missing error handling",
		ValidationAttempt:  2,
		Iteration:          2,
	}

	prompt := ctx.BuildPrompt()

	// Should contain validation feedback
	assert.Contains(t, prompt, "Missing error handling")
	assert.Contains(t, prompt, "Attempt 2/3")
}

func TestIterationContextValidatingPhase(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:   `{"name": "Test"}`,
		Phase:        PhaseValidating,
		PreviousPlan: "## My Plan\n\n1. Do something",
		Iteration:    1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain the plan to validate
	assert.Contains(t, prompt, "## My Plan")
	assert.Contains(t, prompt, "Validation Checklist")
}

func TestIterationContextExecutingPhase(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:   `{"name": "Test"}`,
		Phase:        PhaseExecuting,
		PreviousPlan: "## My Validated Plan\n\n1. Step one",
		Iteration:    1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain the plan to execute
	assert.Contains(t, prompt, "## My Validated Plan")
	assert.Contains(t, prompt, "Execution Rules")
	assert.Contains(t, prompt, "execution_complete")
}

func TestBuildPlanningInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhasePlanning,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain planning-specific instructions
	assert.Contains(t, prompt, "Planning Phase Instructions")
	assert.Contains(t, prompt, "<plan>")
	assert.Contains(t, prompt, "Do NOT implement anything yet")
}

func TestBuildValidatingInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhaseValidating,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain validation-specific instructions
	assert.Contains(t, prompt, "Validation Phase Instructions")
	assert.Contains(t, prompt, "<validation>")
	assert.Contains(t, prompt, "valid:")
}

func TestBuildExecutingInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhaseExecuting,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain execution-specific instructions
	assert.Contains(t, prompt, "Execution Phase Instructions")
	assert.Contains(t, prompt, "Follow the plan")
	assert.Contains(t, prompt, "tests_passing")
}

func TestIterationContextNewFields(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:         `{"name": "Test"}`,
		Phase:              PhasePlanning,
		PreviousPlan:       "The plan",
		ValidationFeedback: "Some feedback",
		ValidationAttempt:  2,
		Iteration:          2,
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var restored IterationContext
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, ctx.PreviousPlan, restored.PreviousPlan)
	assert.Equal(t, ctx.ValidationFeedback, restored.ValidationFeedback)
	assert.Equal(t, ctx.ValidationAttempt, restored.ValidationAttempt)
}

// Tests for file tagging integration (feat-004)

func TestOrchestratorHasTagger(t *testing.T) {
	orch := New("/tmp/test")
	require.NotNil(t, orch.tagger)
	require.NotNil(t, orch.GetTagger())
}

func TestAddTaggedFilesFromTags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-tag-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package util"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	// Create a vendor directory (should be excludable)
	vendorDir := filepath.Join(tmpDir, "vendor")
	os.Mkdir(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, "dep.go"), []byte("package dep"), 0644)

	orch := New(tmpDir)
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TaggedFiles: make(map[string]string),
		Iteration:   1,
	}

	// Test adding files with tags, excluding vendor
	err = orch.AddTaggedFilesFromTags(ctx, []string{"@main.go", "@util.go", "@!vendor"})
	require.NoError(t, err)

	assert.Contains(t, ctx.TaggedFiles, "main.go")
	assert.Contains(t, ctx.TaggedFiles, "util.go")
	assert.Len(t, ctx.TaggedFiles, 2)
}

func TestAddTaggedFilesFromTagsGlobPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-glob-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package util"), 0644)
	os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("readme"), 0644) // Not .go
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	orch := New(tmpDir)
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TaggedFiles: make(map[string]string),
		Iteration:   1,
	}

	// Test glob pattern
	err = orch.AddTaggedFilesFromTags(ctx, []string{"@src/*.go"})
	require.NoError(t, err)

	// Should have both .go files
	assert.Len(t, ctx.TaggedFiles, 2)
	assert.Contains(t, ctx.TaggedFiles, "src/main.go")
	assert.Contains(t, ctx.TaggedFiles, "src/util.go")
}

func TestAddTaggedFilesFromTagsDoubleStarGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-doublestar-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create nested directories
	nestedDir := filepath.Join(tmpDir, "src", "pkg", "util")
	os.MkdirAll(nestedDir, 0755)

	// Create .go files at different levels
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "pkg", "pkg.go"), []byte("package pkg"), 0644)
	os.WriteFile(filepath.Join(nestedDir, "util.go"), []byte("package util"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	orch := New(tmpDir)
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TaggedFiles: make(map[string]string),
		Iteration:   1,
	}

	// Test ** glob pattern
	err = orch.AddTaggedFilesFromTags(ctx, []string{"@src/**/*.go"})
	require.NoError(t, err)

	// Should have all 3 .go files
	assert.Len(t, ctx.TaggedFiles, 3)
}

func TestAddTaggedFilesFromTagsWithExclusion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-excl-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create directories
	testDir := filepath.Join(tmpDir, "test")
	os.Mkdir(testDir, 0755)

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(testDir, "main_test.go"), []byte("package test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	orch := New(tmpDir)
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TaggedFiles: make(map[string]string),
		Iteration:   1,
	}

	// Include all .go files but exclude test directory
	err = orch.AddTaggedFilesFromTags(ctx, []string{"@**/*.go", "@main.go", "@!test"})
	require.NoError(t, err)

	// Should only have main.go
	assert.Contains(t, ctx.TaggedFiles, "main.go")
	assert.NotContains(t, ctx.TaggedFiles, "test/main_test.go")
}

func TestListFilesForAutocomplete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-autocomplete-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create files and directories
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log"), 0644)

	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package src"), 0644)

	orch := New(tmpDir)
	files, err := orch.ListFilesForAutocomplete(3)
	require.NoError(t, err)

	// Should contain main.go, .gitignore, src/, src/util.go
	assert.Contains(t, files, "main.go")
	assert.Contains(t, files, ".gitignore")
	assert.Contains(t, files, "src/")
}

func TestIterationContextTagPatterns(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TagPatterns: []string{"@src/**/*.go", "@!vendor", "@main.go"},
		TaggedFiles: map[string]string{"main.go": "package main"},
		Iteration:   1,
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var restored IterationContext
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Len(t, restored.TagPatterns, 3)
	assert.Equal(t, "@src/**/*.go", restored.TagPatterns[0])
}

// Tests for parallel action execution (feat-005)

func TestOrchestratorHasParallelExecutor(t *testing.T) {
	orch := New("/tmp/test")
	require.NotNil(t, orch.parallel)
	require.NotNil(t, orch.GetParallelExecutor())
}

func TestOrchestratorSetParallelLimits(t *testing.T) {
	orch := New("/tmp/test")
	orch.SetParallelLimits(ParallelLimits{MaxReads: 5, MaxCommands: 2})

	pe := orch.GetParallelExecutor()
	assert.Equal(t, 5, pe.limits.MaxReads)
	assert.Equal(t, 2, pe.limits.MaxCommands)
}

func TestOrchestratorExecuteParallelReads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-parallel-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	orch := New(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"file1.txt"}}},
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"file2.txt"}}},
	}

	ctx := context.Background()
	result := orch.ExecuteParallel(ctx, actions)

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, 2)
}

func TestOrchestratorExecuteParallelMixed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-parallel-mixed-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create initial file
	os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("existing"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)

	orch := New(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"existing.txt"}}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "new.txt", Content: "new content"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo hello"}},
	}

	ctx := context.Background()
	result := orch.ExecuteParallel(ctx, actions)

	assert.True(t, result.AllSucceeded)

	// Verify the file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "new.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestActionParallelType(t *testing.T) {
	// Verify ActionParallel is a valid action type
	assert.Equal(t, Action("parallel"), ActionParallel)

	// Verify it's in the list of valid actions
	validActions := map[Action]bool{
		ActionAskUser:    true,
		ActionReadFiles:  true,
		ActionWriteFile:  true,
		ActionRunCommand: true,
		ActionDone:       true,
		ActionParallel:   true,
	}

	assert.True(t, validActions[ActionParallel])
}

func TestSetInitialTags(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Initially should be empty
	assert.Empty(t, orch.GetInitialTags())

	// Set some tags
	tags := []string{"@main.go", "@cmd/", "@internal/**/*.go"}
	orch.SetInitialTags(tags)

	// Should return the same tags
	assert.Equal(t, tags, orch.GetInitialTags())

	// Should support chaining
	newTags := []string{"@README.md"}
	result := orch.SetInitialTags(newTags)
	assert.Equal(t, orch, result)
	assert.Equal(t, newTags, orch.GetInitialTags())
}

func TestSetInitialTagsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Set empty tags
	orch.SetInitialTags([]string{})
	assert.Empty(t, orch.GetInitialTags())

	// Set nil tags
	orch.SetInitialTags(nil)
	assert.Nil(t, orch.GetInitialTags())
}

// Tests for codebase snapshot feature (feat-009)

func TestDefaultSnapshotConfig(t *testing.T) {
	config := DefaultSnapshotConfig()
	assert.Equal(t, 4, config.MaxTreeDepth)
	assert.Equal(t, int64(50*1024), config.MaxFileSizeBytes)
	assert.True(t, config.IncludeKeyFiles)
}

func TestOrchestratorDefaultSnapshotConfig(t *testing.T) {
	orch := New("/tmp/test")
	config := orch.GetSnapshotConfig()

	assert.Equal(t, 4, config.MaxTreeDepth)
	assert.Equal(t, int64(50*1024), config.MaxFileSizeBytes)
	assert.True(t, config.IncludeKeyFiles)
}

func TestSetSnapshotConfig(t *testing.T) {
	orch := New("/tmp/test")

	config := SnapshotConfig{
		MaxTreeDepth:     6,
		MaxFileSizeBytes: 100 * 1024,
		IncludeKeyFiles:  false,
	}

	result := orch.SetSnapshotConfig(config)
	assert.Equal(t, orch, result) // Test chaining

	retrieved := orch.GetSnapshotConfig()
	assert.Equal(t, 6, retrieved.MaxTreeDepth)
	assert.Equal(t, int64(100*1024), retrieved.MaxFileSizeBytes)
	assert.False(t, retrieved.IncludeKeyFiles)
}

func TestSetMaxTreeDepth(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetMaxTreeDepth(8)
	assert.Equal(t, orch, result) // Test chaining
	assert.Equal(t, 8, orch.GetSnapshotConfig().MaxTreeDepth)
}

func TestSetMaxFileSizeBytes(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetMaxFileSizeBytes(25 * 1024)
	assert.Equal(t, orch, result) // Test chaining
	assert.Equal(t, int64(25*1024), orch.GetSnapshotConfig().MaxFileSizeBytes)
}

func TestSetIncludeKeyFiles(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetIncludeKeyFiles(false)
	assert.Equal(t, orch, result) // Test chaining
	assert.False(t, orch.GetSnapshotConfig().IncludeKeyFiles)

	orch.SetIncludeKeyFiles(true)
	assert.True(t, orch.GetSnapshotConfig().IncludeKeyFiles)
}

func TestDetectKeyFilesGoProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go project structure
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.21"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Project"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("build:\n\tgo build"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	// Should have detected key files
	assert.NotEmpty(t, ctx.KeyFiles)
	assert.Contains(t, ctx.KeyFiles, "go.mod")
	assert.Contains(t, ctx.KeyFiles, "main.go")
	assert.Contains(t, ctx.KeyFiles, "README.md")
	assert.Contains(t, ctx.KeyFiles, "Makefile")
}

func TestDetectKeyFilesNodeProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Node.js project structure
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test", "version": "1.0.0"}`), 0644)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "src", "index.js"), []byte("console.log('hello')"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Contains(t, ctx.KeyFiles, "package.json")
	assert.Contains(t, ctx.KeyFiles, "src/index.js")
}

func TestDetectKeyFilesRustProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Rust project structure
	err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "src", "main.rs"), []byte("fn main() {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Contains(t, ctx.KeyFiles, "Cargo.toml")
	assert.Contains(t, ctx.KeyFiles, "src/main.rs")
}

func TestDetectKeyFilesPythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Python project structure
	err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests==2.28.0"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("print('hello')"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Contains(t, ctx.KeyFiles, "requirements.txt")
	assert.Contains(t, ctx.KeyFiles, "main.py")
}

func TestKeyFileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file larger than the limit
	largeContent := strings.Repeat("x", 60*1024) // 60KB
	err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(largeContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	// The file should be listed but with a truncation note
	content, exists := ctx.KeyFiles["README.md"]
	assert.True(t, exists)
	assert.Contains(t, content, "File too large")
	assert.Contains(t, content, "61440 bytes") // 60 * 1024
}

func TestKeyFileSizeLimitCustom(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file of 30KB
	mediumContent := strings.Repeat("y", 30*1024)
	err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(mediumContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	// Default (50KB limit) should include it
	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)
	assert.Equal(t, mediumContent, ctx.KeyFiles["README.md"])

	// With 25KB limit, should show truncation note
	orch.SetMaxFileSizeBytes(25 * 1024)
	ctx, err = orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)
	assert.Contains(t, ctx.KeyFiles["README.md"], "File too large")
}

func TestDisableKeyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	orch.SetIncludeKeyFiles(false)

	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	// KeyFiles should be empty when disabled
	assert.Empty(t, ctx.KeyFiles)
}

func TestConfigurableTreeDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directories
	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "f")
	err := os.MkdirAll(deepDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("deep"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	// With depth 2, should not see deep directories
	orch := New(tmpDir)
	orch.SetMaxTreeDepth(2)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Contains(t, ctx.DirectoryTree, "a/")
	// Depth 2 should include a/b but not a/b/c
	assert.Contains(t, ctx.DirectoryTree, "b/")
	assert.NotContains(t, ctx.DirectoryTree, "deep.txt")

	// With depth 6, should see everything
	orch.SetMaxTreeDepth(7)
	ctx, err = orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	assert.Contains(t, ctx.DirectoryTree, "deep.txt")
}

func TestKeyFilesInPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	prompt := ctx.BuildPrompt()

	assert.Contains(t, prompt, "## Key Files")
	assert.Contains(t, prompt, "automatically detected important project files")
	assert.Contains(t, prompt, "### go.mod")
	assert.Contains(t, prompt, "module test")
}

func TestIterationContextKeyFilesSerialization(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		KeyFiles: map[string]string{
			"go.mod":  "module test",
			"main.go": "package main",
		},
		Iteration: 1,
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var restored IterationContext
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Len(t, restored.KeyFiles, 2)
	assert.Equal(t, "module test", restored.KeyFiles["go.mod"])
	assert.Equal(t, "package main", restored.KeyFiles["main.go"])
}

func TestSnapshotConfigSerialization(t *testing.T) {
	config := SnapshotConfig{
		MaxTreeDepth:     5,
		MaxFileSizeBytes: 100 * 1024,
		IncludeKeyFiles:  false,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var restored SnapshotConfig
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, 5, restored.MaxTreeDepth)
	assert.Equal(t, int64(100*1024), restored.MaxFileSizeBytes)
	assert.False(t, restored.IncludeKeyFiles)
}

func TestKeyFilePatternsConstant(t *testing.T) {
	// Verify key patterns include expected files
	expectedPatterns := []string{
		"go.mod",
		"package.json",
		"Cargo.toml",
		"README.md",
		"Makefile",
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, p := range keyFilePatterns {
			if p == pattern {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected pattern %s to be in keyFilePatterns", pattern)
	}
}

func TestMainEntryPatternsConstant(t *testing.T) {
	// Verify main entry patterns include expected files
	expectedPatterns := []string{
		"main.go",
		"src/index.js",
		"main.py",
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, p := range mainEntryPatterns {
			if p == pattern {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected pattern %s to be in mainEntryPatterns", pattern)
	}
}

func TestNoKeyFilesIfNoneExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only prd.json (required) and a random file
	err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "random.xyz"), []byte("random"), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	// Should have empty key files (random.xyz is not a key file)
	assert.Empty(t, ctx.KeyFiles)
}

func TestCmdMainGoPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cmd/myapp/main.go structure
	cmdDir := filepath.Join(tmpDir, "cmd", "myapp")
	err := os.MkdirAll(cmdDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	require.NoError(t, err)

	// Should detect cmd/myapp/main.go
	assert.Contains(t, ctx.KeyFiles, "cmd/myapp/main.go")
}

// Tests for native Claude tools integration (feat-010)

func TestDefaultToolConfig(t *testing.T) {
	config := DefaultToolConfig()

	assert.True(t, config.AllowRead)
	assert.True(t, config.AllowWrite)
	assert.True(t, config.AllowEdit)
	assert.NotEmpty(t, config.AllowedBashCommands)

	// Verify default bash commands
	expectedCommands := []string{"go", "npm", "yarn", "pnpm", "cargo", "python", "pytest", "git", "make"}
	assert.Equal(t, expectedCommands, config.AllowedBashCommands)
}

func TestDefaultAllowedBashCommandsConstant(t *testing.T) {
	// Verify the constant contains expected commands
	expectedCommands := []string{"go", "npm", "yarn", "pnpm", "cargo", "python", "pytest", "git", "make"}

	assert.Equal(t, len(expectedCommands), len(DefaultAllowedBashCommands))
	for i, expected := range expectedCommands {
		assert.Equal(t, expected, DefaultAllowedBashCommands[i])
	}
}

func TestOrchestratorDefaultToolConfig(t *testing.T) {
	orch := New("/tmp/test")
	config := orch.GetToolConfig()

	assert.True(t, config.AllowRead)
	assert.True(t, config.AllowWrite)
	assert.True(t, config.AllowEdit)
	assert.NotEmpty(t, config.AllowedBashCommands)
}

func TestSetToolConfig(t *testing.T) {
	orch := New("/tmp/test")

	config := ToolConfig{
		AllowRead:           true,
		AllowWrite:          false,
		AllowEdit:           true,
		AllowedBashCommands: []string{"go", "npm"},
	}

	result := orch.SetToolConfig(config)
	assert.Equal(t, orch, result) // Test chaining

	retrieved := orch.GetToolConfig()
	assert.True(t, retrieved.AllowRead)
	assert.False(t, retrieved.AllowWrite)
	assert.True(t, retrieved.AllowEdit)
	assert.Equal(t, []string{"go", "npm"}, retrieved.AllowedBashCommands)
}

func TestSetAllowedBashCommands(t *testing.T) {
	orch := New("/tmp/test")

	commands := []string{"go", "cargo", "make"}
	result := orch.SetAllowedBashCommands(commands)
	assert.Equal(t, orch, result) // Test chaining

	config := orch.GetToolConfig()
	assert.Equal(t, commands, config.AllowedBashCommands)
}

func TestAddAllowedBashCommand(t *testing.T) {
	orch := New("/tmp/test")

	// Start with default commands
	originalLen := len(orch.GetToolConfig().AllowedBashCommands)

	// Add a new command
	result := orch.AddAllowedBashCommand("ruby")
	assert.Equal(t, orch, result) // Test chaining

	config := orch.GetToolConfig()
	assert.Len(t, config.AllowedBashCommands, originalLen+1)
	assert.Contains(t, config.AllowedBashCommands, "ruby")
}

func TestSetAllowRead(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetAllowRead(false)
	assert.Equal(t, orch, result) // Test chaining
	assert.False(t, orch.GetToolConfig().AllowRead)

	orch.SetAllowRead(true)
	assert.True(t, orch.GetToolConfig().AllowRead)
}

func TestSetAllowWrite(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetAllowWrite(false)
	assert.Equal(t, orch, result) // Test chaining
	assert.False(t, orch.GetToolConfig().AllowWrite)

	orch.SetAllowWrite(true)
	assert.True(t, orch.GetToolConfig().AllowWrite)
}

func TestSetAllowEdit(t *testing.T) {
	orch := New("/tmp/test")

	result := orch.SetAllowEdit(false)
	assert.Equal(t, orch, result) // Test chaining
	assert.False(t, orch.GetToolConfig().AllowEdit)

	orch.SetAllowEdit(true)
	assert.True(t, orch.GetToolConfig().AllowEdit)
}

func TestBuildAllowedToolsFlagDefault(t *testing.T) {
	config := DefaultToolConfig()
	flag := config.BuildAllowedToolsFlag()

	// Should start with Read,Write,Edit
	assert.True(t, strings.HasPrefix(flag, "Read,Write,Edit"))

	// Should contain bash commands in format Bash(cmd:*)
	assert.Contains(t, flag, "Bash(go:*)")
	assert.Contains(t, flag, "Bash(npm:*)")
	assert.Contains(t, flag, "Bash(yarn:*)")
	assert.Contains(t, flag, "Bash(pnpm:*)")
	assert.Contains(t, flag, "Bash(cargo:*)")
	assert.Contains(t, flag, "Bash(python:*)")
	assert.Contains(t, flag, "Bash(pytest:*)")
	assert.Contains(t, flag, "Bash(git:*)")
	assert.Contains(t, flag, "Bash(make:*)")
}

func TestBuildAllowedToolsFlagDisabledTools(t *testing.T) {
	config := ToolConfig{
		AllowRead:           false,
		AllowWrite:          true,
		AllowEdit:           false,
		AllowedBashCommands: []string{"go"},
	}
	flag := config.BuildAllowedToolsFlag()

	assert.NotContains(t, flag, "Read")
	assert.Contains(t, flag, "Write")
	assert.NotContains(t, flag, "Edit")
	assert.Contains(t, flag, "Bash(go:*)")
}

func TestBuildAllowedToolsFlagNoBash(t *testing.T) {
	config := ToolConfig{
		AllowRead:           true,
		AllowWrite:          true,
		AllowEdit:           true,
		AllowedBashCommands: []string{},
	}
	flag := config.BuildAllowedToolsFlag()

	assert.Equal(t, "Read,Write,Edit", flag)
}

func TestBuildAllowedToolsFlagOnlyBash(t *testing.T) {
	config := ToolConfig{
		AllowRead:           false,
		AllowWrite:          false,
		AllowEdit:           false,
		AllowedBashCommands: []string{"echo", "ls"},
	}
	flag := config.BuildAllowedToolsFlag()

	assert.Equal(t, "Bash(echo:*),Bash(ls:*)", flag)
}

func TestToolConfigSerialization(t *testing.T) {
	config := ToolConfig{
		AllowRead:           true,
		AllowWrite:          false,
		AllowEdit:           true,
		AllowedBashCommands: []string{"go", "npm", "git"},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var restored ToolConfig
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.True(t, restored.AllowRead)
	assert.False(t, restored.AllowWrite)
	assert.True(t, restored.AllowEdit)
	assert.Equal(t, []string{"go", "npm", "git"}, restored.AllowedBashCommands)
}

func TestToolConfigEmptyAllowedCommands(t *testing.T) {
	config := ToolConfig{
		AllowRead:           true,
		AllowWrite:          true,
		AllowEdit:           true,
		AllowedBashCommands: nil,
	}

	flag := config.BuildAllowedToolsFlag()
	assert.Equal(t, "Read,Write,Edit", flag)
}

func TestOrchestratorToolConfigChaining(t *testing.T) {
	orch := New("/tmp/test")

	// Test method chaining
	result := orch.
		SetAllowRead(false).
		SetAllowWrite(true).
		SetAllowEdit(false).
		SetAllowedBashCommands([]string{"go", "npm"}).
		AddAllowedBashCommand("cargo")

	assert.Equal(t, orch, result)

	config := orch.GetToolConfig()
	assert.False(t, config.AllowRead)
	assert.True(t, config.AllowWrite)
	assert.False(t, config.AllowEdit)
	assert.Equal(t, []string{"go", "npm", "cargo"}, config.AllowedBashCommands)
}

func TestToolConfigBuildFlagOrder(t *testing.T) {
	config := ToolConfig{
		AllowRead:           true,
		AllowWrite:          true,
		AllowEdit:           true,
		AllowedBashCommands: []string{"go", "npm"},
	}
	flag := config.BuildAllowedToolsFlag()

	// Verify the order is Read, Write, Edit, then Bash commands
	expected := "Read,Write,Edit,Bash(go:*),Bash(npm:*)"
	assert.Equal(t, expected, flag)
}

// ============================================================================
// Progress File Integration Tests (feat-011)
// ============================================================================

func TestOrchestratorHasProgressWriter(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	require.NotNil(t, orch.GetProgressWriter())
}

func TestProgressEntryBuilderNew(t *testing.T) {
	builder := NewProgressEntryBuilder(5)

	require.NotNil(t, builder)
	assert.Equal(t, 5, builder.Iteration)
	assert.NotZero(t, builder.Timestamp)
	assert.Empty(t, builder.WorkDone)
	assert.Empty(t, builder.Commits)
}

func TestProgressEntryBuilderSetStartingState(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	feature := &progress.FeatureRef{
		ID:          "feat-001",
		Description: "Test feature",
	}

	result := builder.SetStartingState(10, 3, feature)

	// Verify chaining
	assert.Equal(t, builder, result)

	assert.Equal(t, 10, builder.StartingState.FeaturesTotal)
	assert.Equal(t, 3, builder.StartingState.FeaturesPassing)
	assert.Equal(t, feature, builder.StartingState.WorkingOn)
	assert.Equal(t, feature, builder.CurrentFeature)
}

func TestProgressEntryBuilderAddWorkDone(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	builder.AddWorkDone("Implemented feature X").
		AddWorkDone("Added tests for X").
		AddWorkDone("Updated documentation")

	require.Len(t, builder.WorkDone, 3)
	assert.Equal(t, "Implemented feature X", builder.WorkDone[0])
	assert.Equal(t, "Added tests for X", builder.WorkDone[1])
	assert.Equal(t, "Updated documentation", builder.WorkDone[2])
}

func TestProgressEntryBuilderSetTestResult(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	result := builder.SetTestResult("go test ./...", true, "47 tests passed")

	assert.Equal(t, builder, result)
	assert.Equal(t, "go test ./...", builder.Testing.Command)
	assert.True(t, builder.Testing.Passed)
	assert.Equal(t, "47 tests passed", builder.Testing.Details)
}

func TestProgressEntryBuilderAddCommit(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	builder.AddCommit("abc1234", "feat: add feature X").
		AddCommit("def5678", "test: add tests for X")

	require.Len(t, builder.Commits, 2)
	assert.Equal(t, "abc1234", builder.Commits[0].Hash)
	assert.Equal(t, "feat: add feature X", builder.Commits[0].Message)
	assert.Equal(t, "def5678", builder.Commits[1].Hash)
}

func TestProgressEntryBuilderAddNote(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	builder.AddNote("Consider refactoring later").
		AddNote("Performance could be improved")

	require.Len(t, builder.NotesForNextSession, 2)
	assert.Equal(t, "Consider refactoring later", builder.NotesForNextSession[0])
}

func TestProgressEntryBuilderBuild(t *testing.T) {
	builder := NewProgressEntryBuilder(3)

	feature := &progress.FeatureRef{
		ID:          "feat-005",
		Description: "User authentication",
	}

	builder.SetStartingState(15, 4, feature).
		AddWorkDone("Added login endpoint").
		AddWorkDone("Added JWT validation").
		SetTestResult("go test ./...", true, "All tests passed").
		AddCommit("abc123", "feat: add authentication").
		AddNote("Consider adding refresh tokens")

	entry := builder.Build(15, 5, true)

	// Check entry fields
	assert.Equal(t, 3, entry.Iteration)
	assert.NotZero(t, entry.Timestamp)

	// Starting state
	assert.Equal(t, 15, entry.StartingState.FeaturesTotal)
	assert.Equal(t, 4, entry.StartingState.FeaturesPassing)
	assert.Equal(t, "feat-005", entry.StartingState.WorkingOn.ID)

	// Work done
	require.Len(t, entry.WorkDone, 2)
	assert.Equal(t, "Added login endpoint", entry.WorkDone[0])

	// Testing
	assert.Equal(t, "go test ./...", entry.Testing.Command)
	assert.True(t, entry.Testing.Passed)

	// Commits
	require.Len(t, entry.Commits, 1)
	assert.Equal(t, "abc123", entry.Commits[0].Hash)

	// Ending state
	assert.Equal(t, 15, entry.EndingState.FeaturesTotal)
	assert.Equal(t, 5, entry.EndingState.FeaturesPassing)
	assert.Equal(t, "feat-005", entry.EndingState.WorkingOn.ID)
	assert.True(t, entry.EndingState.AllTestsPassing)

	// Notes
	require.Len(t, entry.NotesForNextSession, 1)
	assert.Equal(t, "Consider adding refresh tokens", entry.NotesForNextSession[0])
}

func TestOrchestratorStartProgressEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a PRD file
	prdContent := `{
		"name": "Test Project",
		"description": "Test",
		"testCommand": "go test ./...",
		"features": [
			{"id": "feat-001", "category": "functional", "priority": "high", "description": "First feature", "steps": ["step1"], "passes": true},
			{"id": "feat-002", "category": "functional", "priority": "high", "description": "Second feature", "steps": ["step1"], "passes": false}
		]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)

	// Load PRD
	currentPRD, err := prd.LoadFromDir(tmpDir)
	require.NoError(t, err)

	// Start progress entry
	orch.StartProgressEntry(1, currentPRD)

	require.True(t, orch.HasProgressEntry())
	entry := orch.GetCurrentProgressEntry()
	require.NotNil(t, entry)

	assert.Equal(t, 1, entry.Iteration)
	assert.Equal(t, 2, entry.StartingState.FeaturesTotal)
	assert.Equal(t, 1, entry.StartingState.FeaturesPassing)
	require.NotNil(t, entry.CurrentFeature)
	assert.Equal(t, "feat-002", entry.CurrentFeature.ID)
}

func TestOrchestratorAddProgressWork(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Without an entry, should not panic
	orch.AddProgressWork("This should not crash")

	// Create an entry
	orch.currentEntry = NewProgressEntryBuilder(1)

	orch.AddProgressWork("Implemented feature")
	orch.AddProgressWork("Added tests")

	require.Len(t, orch.currentEntry.WorkDone, 2)
}

func TestOrchestratorSetProgressTestResult(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Without an entry, should not panic
	orch.SetProgressTestResult("go test ./...", true, "passed")

	// Create an entry
	orch.currentEntry = NewProgressEntryBuilder(1)

	orch.SetProgressTestResult("go test ./...", true, "47 tests passed")

	assert.Equal(t, "go test ./...", orch.currentEntry.Testing.Command)
	assert.True(t, orch.currentEntry.Testing.Passed)
}

func TestOrchestratorAddProgressCommit(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Without an entry, should not panic
	orch.AddProgressCommit("abc123", "test")

	// Create an entry
	orch.currentEntry = NewProgressEntryBuilder(1)

	orch.AddProgressCommit("abc123", "feat: add feature")
	orch.AddProgressCommit("def456", "test: add tests")

	require.Len(t, orch.currentEntry.Commits, 2)
	assert.Equal(t, "abc123", orch.currentEntry.Commits[0].Hash)
}

func TestOrchestratorAddProgressNote(t *testing.T) {
	tmpDir := t.TempDir()
	orch := New(tmpDir)

	// Without an entry, should not panic
	orch.AddProgressNote("test note")

	// Create an entry
	orch.currentEntry = NewProgressEntryBuilder(1)

	orch.AddProgressNote("Consider refactoring")

	require.Len(t, orch.currentEntry.NotesForNextSession, 1)
}

func TestOrchestratorFinishProgressEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a PRD file
	prdContent := `{
		"name": "Test Project",
		"description": "Test",
		"testCommand": "go test ./...",
		"features": [
			{"id": "feat-001", "category": "functional", "priority": "high", "description": "First feature", "steps": ["step1"], "passes": true},
			{"id": "feat-002", "category": "functional", "priority": "high", "description": "Second feature", "steps": ["step1"], "passes": true}
		]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)

	// Load PRD
	currentPRD, err := prd.LoadFromDir(tmpDir)
	require.NoError(t, err)

	// Without an entry, should return nil (no-op)
	err = orch.FinishProgressEntry(currentPRD, true)
	require.NoError(t, err)

	// Create and populate an entry
	orch.currentEntry = NewProgressEntryBuilder(1)
	orch.currentEntry.SetStartingState(2, 1, &progress.FeatureRef{ID: "feat-002", Description: "Second"})
	orch.AddProgressWork("Implemented feature")
	orch.SetProgressTestResult("go test ./...", true, "passed")
	orch.AddProgressCommit("abc123", "feat: add feature")

	// Finish the entry
	err = orch.FinishProgressEntry(currentPRD, true)
	require.NoError(t, err)

	// Entry should be cleared
	assert.False(t, orch.HasProgressEntry())

	// Progress file should exist and contain the entry
	progressPath := filepath.Join(tmpDir, "progress.txt")
	assert.FileExists(t, progressPath)

	content, err := os.ReadFile(progressPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "Iteration: 1")
	assert.Contains(t, contentStr, "Features passing: 1/2")
	assert.Contains(t, contentStr, "Implemented feature")
	assert.Contains(t, contentStr, "abc123")
	assert.Contains(t, contentStr, "All tests passing: YES")
}

func TestOrchestratorProgressFileGrowsAcrossIterations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a PRD file with multiple features
	prdContent := `{
		"name": "Test Project",
		"description": "Test",
		"testCommand": "go test ./...",
		"features": [
			{"id": "feat-001", "category": "functional", "priority": "high", "description": "Feature 1", "steps": ["step1"], "passes": false},
			{"id": "feat-002", "category": "functional", "priority": "high", "description": "Feature 2", "steps": ["step1"], "passes": false},
			{"id": "feat-003", "category": "functional", "priority": "high", "description": "Feature 3", "steps": ["step1"], "passes": false}
		]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)

	// Simulate 3 iterations
	for i := 1; i <= 3; i++ {
		currentPRD, err := prd.LoadFromDir(tmpDir)
		require.NoError(t, err)

		orch.StartProgressEntry(i, currentPRD)
		orch.AddProgressWork("Implemented feature " + currentPRD.NextFeature().ID)
		orch.SetProgressTestResult("go test ./...", true, "passed")
		orch.AddProgressCommit("commit"+string(rune('0'+i)), "feat: add feature")

		err = orch.FinishProgressEntry(currentPRD, true)
		require.NoError(t, err)

		// Mark the feature as passing for next iteration
		nextFeature := currentPRD.NextFeature()
		if nextFeature != nil {
			nextFeature.Passes = true
			prdData, _ := json.MarshalIndent(currentPRD, "", "  ")
			os.WriteFile(filepath.Join(tmpDir, "prd.json"), prdData, 0644)
		}
	}

	// Verify progress file contains all 3 iterations
	content, err := os.ReadFile(filepath.Join(tmpDir, "progress.txt"))
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "Iteration: 1")
	assert.Contains(t, contentStr, "Iteration: 2")
	assert.Contains(t, contentStr, "Iteration: 3")

	// Count occurrences of session separator
	separatorCount := strings.Count(contentStr, "================================================================================")
	assert.GreaterOrEqual(t, separatorCount, 6) // 2 separators per entry minimum
}

func TestProgressEntryBuilderNilFeature(t *testing.T) {
	builder := NewProgressEntryBuilder(1)

	// Setting starting state with nil feature should work
	result := builder.SetStartingState(5, 2, nil)
	assert.Equal(t, builder, result)

	assert.Nil(t, builder.StartingState.WorkingOn)
	assert.Nil(t, builder.CurrentFeature)

	// Build should work with nil feature
	entry := builder.Build(5, 3, true)
	assert.Nil(t, entry.EndingState.WorkingOn)
}

func TestOrchestratorStartProgressEntryAllComplete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a PRD where all features are complete
	prdContent := `{
		"name": "Test Project",
		"description": "Test",
		"testCommand": "go test ./...",
		"features": [
			{"id": "feat-001", "category": "functional", "priority": "high", "description": "Feature 1", "steps": ["step1"], "passes": true}
		]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644)
	require.NoError(t, err)

	orch := New(tmpDir)

	currentPRD, err := prd.LoadFromDir(tmpDir)
	require.NoError(t, err)

	// Start progress entry when no features are pending
	orch.StartProgressEntry(1, currentPRD)

	assert.True(t, orch.HasProgressEntry())
	entry := orch.GetCurrentProgressEntry()
	// Feature should be nil since NextFeature() returns nil
	assert.Nil(t, entry.CurrentFeature)
}
