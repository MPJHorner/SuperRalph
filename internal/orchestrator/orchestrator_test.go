package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
