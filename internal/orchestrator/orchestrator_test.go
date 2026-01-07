package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Action != ActionAskUser {
		t.Errorf("expected action %s, got %s", ActionAskUser, response.Action)
	}
	if response.ActionParams.Question != "What are you building?" {
		t.Errorf("unexpected question: %s", response.ActionParams.Question)
	}
	if response.Thinking != "I should ask what they're building" {
		t.Errorf("unexpected thinking: %s", response.Thinking)
	}
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
			if exists != tt.valid {
				t.Errorf("action %s: expected valid=%v, got valid=%v", tt.action, tt.valid, exists)
			}
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
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}

	var restored Session
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal session: %v", err)
	}

	if restored.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, restored.ID)
	}
	if len(restored.Messages) != len(session.Messages) {
		t.Errorf("expected %d messages, got %d", len(session.Messages), len(restored.Messages))
	}
}

func TestOrchestratorNew(t *testing.T) {
	orch := New("/tmp/test")
	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.workDir != "/tmp/test" {
		t.Errorf("expected workDir /tmp/test, got %s", orch.workDir)
	}
	if orch.session == nil {
		t.Fatal("expected non-nil session")
	}
	if orch.session.ID == "" {
		t.Error("expected non-empty session ID")
	}
}

func TestOrchestratorSetDebug(t *testing.T) {
	orch := New("/tmp/test")
	orch.SetDebug(true)
	if !orch.debug {
		t.Error("expected debug to be true")
	}
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
	if !strings.Contains(prompt, "## prd.json") {
		t.Error("prompt should contain prd.json section")
	}
	if !strings.Contains(prompt, `{"name": "Test Project"`) {
		t.Error("prompt should contain PRD content")
	}
	if !strings.Contains(prompt, "## progress.txt") {
		t.Error("prompt should contain progress.txt section")
	}
	if !strings.Contains(prompt, "Feature 1 completed") {
		t.Error("prompt should contain progress content")
	}
	if !strings.Contains(prompt, "## Directory Structure") {
		t.Error("prompt should contain directory structure section")
	}
	if !strings.Contains(prompt, "## Tagged Files") {
		t.Error("prompt should contain tagged files section")
	}
	if !strings.Contains(prompt, "### main.go") {
		t.Error("prompt should contain tagged file path")
	}
	if !strings.Contains(prompt, "## Current Feature") {
		t.Error("prompt should contain current feature section")
	}
	if !strings.Contains(prompt, "feat-001") {
		t.Error("prompt should contain feature ID")
	}
	if !strings.Contains(prompt, "## Current Phase: planning") {
		t.Error("prompt should contain current phase")
	}
	// When phase is set, we get phase-specific instructions instead of generic task instructions
	if !strings.Contains(prompt, "Planning Phase Instructions") {
		t.Error("prompt should contain planning phase instructions when phase is planning")
	}
}

func TestIterationContextEmptyProgress(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:      `{"name": "Test"}`,
		ProgressContent: "",
		Iteration:       1,
	}

	prompt := ctx.BuildPrompt()

	if !strings.Contains(prompt, "(empty)") {
		t.Error("prompt should show (empty) for empty progress")
	}
}

func TestIterationContextNoOptionalFields(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should not contain optional sections when not set
	if strings.Contains(prompt, "## Directory Structure") {
		t.Error("prompt should not contain directory structure when not set")
	}
	if strings.Contains(prompt, "## Tagged Files") {
		t.Error("prompt should not contain tagged files when not set")
	}
	if strings.Contains(prompt, "## Current Feature") {
		t.Error("prompt should not contain current feature when not set")
	}
	if strings.Contains(prompt, "## Current Phase") {
		t.Error("prompt should not contain phase when not set")
	}
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
		if string(tt.phase) != tt.want {
			t.Errorf("Phase %v should be %q, got %q", tt.phase, tt.want, string(tt.phase))
		}
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
	if err != nil {
		t.Fatalf("failed to marshal context: %v", err)
	}

	var restored IterationContext
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal context: %v", err)
	}

	if restored.PRDContent != ctx.PRDContent {
		t.Error("PRDContent not preserved")
	}
	if restored.ProgressContent != ctx.ProgressContent {
		t.Error("ProgressContent not preserved")
	}
	if restored.Iteration != ctx.Iteration {
		t.Error("Iteration not preserved")
	}
	if restored.Phase != ctx.Phase {
		t.Error("Phase not preserved")
	}
	if restored.CurrentFeature == nil || restored.CurrentFeature.ID != ctx.CurrentFeature.ID {
		t.Error("CurrentFeature not preserved")
	}
	if len(restored.TaggedFiles) != 1 || restored.TaggedFiles["file.go"] != "content" {
		t.Error("TaggedFiles not preserved")
	}
}

func TestBuildIterationContext(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create prd.json
	prdContent := `{"name": "Test Project", "features": []}`
	if err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(prdContent), 0644); err != nil {
		t.Fatalf("failed to write prd.json: %v", err)
	}

	// Create progress.txt
	progressContent := "Feature completed"
	if err := os.WriteFile(filepath.Join(tmpDir, "progress.txt"), []byte(progressContent), 0644); err != nil {
		t.Fatalf("failed to write progress.txt: %v", err)
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, PhasePlanning, nil)
	if err != nil {
		t.Fatalf("BuildIterationContext failed: %v", err)
	}

	if ctx.PRDContent != prdContent {
		t.Errorf("PRDContent mismatch: got %q, want %q", ctx.PRDContent, prdContent)
	}
	if ctx.ProgressContent != progressContent {
		t.Errorf("ProgressContent mismatch: got %q, want %q", ctx.ProgressContent, progressContent)
	}
	if ctx.Iteration != 1 {
		t.Errorf("Iteration mismatch: got %d, want 1", ctx.Iteration)
	}
	if ctx.Phase != PhasePlanning {
		t.Errorf("Phase mismatch: got %q, want %q", ctx.Phase, PhasePlanning)
	}
	if ctx.DirectoryTree == "" {
		t.Error("DirectoryTree should not be empty")
	}
	if !strings.Contains(ctx.DirectoryTree, "src/") {
		t.Error("DirectoryTree should contain src/ directory")
	}
}

func TestBuildIterationContextWithFeature(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644); err != nil {
		t.Fatalf("failed to write prd.json: %v", err)
	}

	feature := &FeatureContext{
		ID:          "feat-001",
		Description: "Test feature",
		Steps:       []string{"Step 1"},
		Priority:    "high",
		Category:    "functional",
	}

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(2, PhaseExecuting, feature)
	if err != nil {
		t.Fatalf("BuildIterationContext failed: %v", err)
	}

	if ctx.CurrentFeature == nil {
		t.Fatal("CurrentFeature should not be nil")
	}
	if ctx.CurrentFeature.ID != "feat-001" {
		t.Errorf("Feature ID mismatch: got %q, want %q", ctx.CurrentFeature.ID, "feat-001")
	}
}

func TestBuildIterationContextMissingPRD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	orch := New(tmpDir)
	_, err = orch.BuildIterationContext(1, "", nil)
	if err == nil {
		t.Error("BuildIterationContext should fail when prd.json is missing")
	}
	if !strings.Contains(err.Error(), "prd.json") {
		t.Errorf("Error should mention prd.json: %v", err)
	}
}

func TestBuildIterationContextNoProgress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644); err != nil {
		t.Fatalf("failed to write prd.json: %v", err)
	}

	orch := New(tmpDir)
	ctx, err := orch.BuildIterationContext(1, "", nil)
	if err != nil {
		t.Fatalf("BuildIterationContext failed: %v", err)
	}

	if ctx.ProgressContent != "" {
		t.Errorf("ProgressContent should be empty when file doesn't exist, got %q", ctx.ProgressContent)
	}
}

func TestAddTaggedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testContent := "package main\n\nfunc main() {}"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	// Test with relative path
	err = orch.AddTaggedFile(ctx, "main.go")
	if err != nil {
		t.Fatalf("AddTaggedFile failed: %v", err)
	}

	if content, ok := ctx.TaggedFiles["main.go"]; !ok {
		t.Error("TaggedFiles should contain main.go")
	} else if content != testContent {
		t.Errorf("Content mismatch: got %q, want %q", content, testContent)
	}
}

func TestAddTaggedFileAbsolutePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test content"
	absPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(absPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test.txt: %v", err)
	}

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	err = orch.AddTaggedFile(ctx, absPath)
	if err != nil {
		t.Fatalf("AddTaggedFile failed: %v", err)
	}

	if content, ok := ctx.TaggedFiles["test.txt"]; !ok {
		t.Error("TaggedFiles should contain test.txt with relative key")
	} else if content != testContent {
		t.Errorf("Content mismatch: got %q, want %q", content, testContent)
	}
}

func TestAddTaggedFileMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	orch := New(tmpDir)
	ctx := &IterationContext{TaggedFiles: make(map[string]string)}

	err = orch.AddTaggedFile(ctx, "nonexistent.go")
	if err == nil {
		t.Error("AddTaggedFile should fail for missing file")
	}
}

func TestIterationIndependence(t *testing.T) {
	// This test verifies that each iteration context is independent
	// and doesn't carry forward state from previous iterations
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "prd.json"), []byte(`{"name": "Test"}`), 0644); err != nil {
		t.Fatalf("failed to write prd.json: %v", err)
	}

	orch := New(tmpDir)

	// Build first iteration context
	ctx1, err := orch.BuildIterationContext(1, PhasePlanning, nil)
	if err != nil {
		t.Fatalf("BuildIterationContext 1 failed: %v", err)
	}

	// Modify the context (simulating what might happen during processing)
	ctx1.TaggedFiles["added.go"] = "some content"

	// Build second iteration context
	ctx2, err := orch.BuildIterationContext(2, PhaseExecuting, nil)
	if err != nil {
		t.Fatalf("BuildIterationContext 2 failed: %v", err)
	}

	// Verify ctx2 doesn't have the modifications from ctx1
	if len(ctx2.TaggedFiles) != 0 {
		t.Error("Second context should not have tagged files from first context")
	}
	if ctx2.Iteration != 2 {
		t.Errorf("Second context Iteration should be 2, got %d", ctx2.Iteration)
	}
	if ctx2.Phase != PhaseExecuting {
		t.Errorf("Second context Phase should be executing, got %q", ctx2.Phase)
	}
}

// Tests for three-phase loop (feat-003)

func TestExtractPlan(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
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
			if got != tt.want {
				t.Errorf("extractPlan() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantIssues int
		wantFeedback bool
	}{
		{
			name:      "valid plan",
			input:     "<validation>\nvalid: true\nissues:\nfeedback:\n</validation>",
			wantValid: true,
			wantIssues: 0,
			wantFeedback: false,
		},
		{
			name:      "invalid plan with issues",
			input:     "<validation>\nvalid: false\nissues:\n- Missing tests\n- No error handling\nfeedback: Please add tests and error handling\n</validation>",
			wantValid: false,
			wantIssues: 2,
			wantFeedback: true,
		},
		{
			name:      "no validation block - defaults to valid",
			input:     "Some output without validation block",
			wantValid: true,
			wantIssues: 0,
			wantFeedback: false,
		},
		{
			name:      "valid true case insensitive",
			input:     "<validation>\nvalid: TRUE\n</validation>",
			wantValid: true,
			wantIssues: 0,
			wantFeedback: false,
		},
		{
			name:      "valid false explicit",
			input:     "<validation>\nvalid: false\n</validation>",
			wantValid: false,
			wantIssues: 0,
			wantFeedback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValidation(tt.input)
			if result.Valid != tt.wantValid {
				t.Errorf("parseValidation().Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.Issues) != tt.wantIssues {
				t.Errorf("parseValidation().Issues count = %d, want %d", len(result.Issues), tt.wantIssues)
			}
			hasFeedback := result.Feedback != ""
			if hasFeedback != tt.wantFeedback {
				t.Errorf("parseValidation() has feedback = %v, want %v", hasFeedback, tt.wantFeedback)
			}
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
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var restored ValidationResult
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if restored.Valid != result.Valid {
		t.Error("Valid not preserved")
	}
	if len(restored.Issues) != len(result.Issues) {
		t.Error("Issues not preserved")
	}
	if restored.Feedback != result.Feedback {
		t.Error("Feedback not preserved")
	}
}

func TestPhaseConfigDefaults(t *testing.T) {
	config := PhaseConfig{}
	if config.MaxValidationAttempts != 0 {
		t.Errorf("Default MaxValidationAttempts should be 0 (unset), got %d", config.MaxValidationAttempts)
	}

	config = PhaseConfig{MaxValidationAttempts: 5}
	if config.MaxValidationAttempts != 5 {
		t.Errorf("MaxValidationAttempts should be 5, got %d", config.MaxValidationAttempts)
	}
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
	if !strings.Contains(prompt, "Missing error handling") {
		t.Error("prompt should contain validation feedback")
	}
	if !strings.Contains(prompt, "Attempt 2/3") {
		t.Error("prompt should contain attempt number")
	}
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
	if !strings.Contains(prompt, "## My Plan") {
		t.Error("prompt should contain the plan to validate")
	}
	if !strings.Contains(prompt, "Validation Checklist") {
		t.Error("prompt should contain validation checklist")
	}
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
	if !strings.Contains(prompt, "## My Validated Plan") {
		t.Error("prompt should contain the validated plan")
	}
	if !strings.Contains(prompt, "Execution Rules") {
		t.Error("prompt should contain execution rules")
	}
	if !strings.Contains(prompt, "execution_complete") {
		t.Error("prompt should contain completion signal instructions")
	}
}

func TestBuildPlanningInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhasePlanning,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain planning-specific instructions
	if !strings.Contains(prompt, "Planning Phase Instructions") {
		t.Error("prompt should contain planning phase instructions")
	}
	if !strings.Contains(prompt, "<plan>") {
		t.Error("prompt should contain plan output format")
	}
	if !strings.Contains(prompt, "Do NOT implement anything yet") {
		t.Error("prompt should warn against implementing during planning")
	}
}

func TestBuildValidatingInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhaseValidating,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain validation-specific instructions
	if !strings.Contains(prompt, "Validation Phase Instructions") {
		t.Error("prompt should contain validation phase instructions")
	}
	if !strings.Contains(prompt, "<validation>") {
		t.Error("prompt should contain validation output format")
	}
	if !strings.Contains(prompt, "valid:") {
		t.Error("prompt should show valid field format")
	}
}

func TestBuildExecutingInstructions(t *testing.T) {
	ctx := &IterationContext{
		PRDContent: `{"name": "Test"}`,
		Phase:      PhaseExecuting,
		Iteration:  1,
	}

	prompt := ctx.BuildPrompt()

	// Should contain execution-specific instructions
	if !strings.Contains(prompt, "Execution Phase Instructions") {
		t.Error("prompt should contain execution phase instructions")
	}
	if !strings.Contains(prompt, "Follow the plan") {
		t.Error("prompt should instruct to follow the plan")
	}
	if !strings.Contains(prompt, "tests_passing") {
		t.Error("prompt should mention test verification")
	}
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
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var restored IterationContext
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if restored.PreviousPlan != ctx.PreviousPlan {
		t.Error("PreviousPlan not preserved")
	}
	if restored.ValidationFeedback != ctx.ValidationFeedback {
		t.Error("ValidationFeedback not preserved")
	}
	if restored.ValidationAttempt != ctx.ValidationAttempt {
		t.Error("ValidationAttempt not preserved")
	}
}

// Tests for file tagging integration (feat-004)

func TestOrchestratorHasTagger(t *testing.T) {
	orch := New("/tmp/test")
	if orch.tagger == nil {
		t.Fatal("orchestrator should have a tagger")
	}
	if orch.GetTagger() == nil {
		t.Fatal("GetTagger should return the tagger")
	}
}

func TestAddTaggedFilesFromTags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-tag-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("AddTaggedFilesFromTags failed: %v", err)
	}

	if _, ok := ctx.TaggedFiles["main.go"]; !ok {
		t.Error("TaggedFiles should contain main.go")
	}
	if _, ok := ctx.TaggedFiles["util.go"]; !ok {
		t.Error("TaggedFiles should contain util.go")
	}
	if len(ctx.TaggedFiles) != 2 {
		t.Errorf("Expected 2 files, got %d", len(ctx.TaggedFiles))
	}
}

func TestAddTaggedFilesFromTagsGlobPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-glob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("AddTaggedFilesFromTags failed: %v", err)
	}

	// Should have both .go files
	if len(ctx.TaggedFiles) != 2 {
		t.Errorf("Expected 2 .go files, got %d: %v", len(ctx.TaggedFiles), ctx.TaggedFiles)
	}
	if _, ok := ctx.TaggedFiles["src/main.go"]; !ok {
		t.Error("TaggedFiles should contain src/main.go")
	}
	if _, ok := ctx.TaggedFiles["src/util.go"]; !ok {
		t.Error("TaggedFiles should contain src/util.go")
	}
}

func TestAddTaggedFilesFromTagsDoubleStarGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-doublestar-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("AddTaggedFilesFromTags failed: %v", err)
	}

	// Should have all 3 .go files
	if len(ctx.TaggedFiles) != 3 {
		t.Errorf("Expected 3 .go files with **, got %d: %v", len(ctx.TaggedFiles), ctx.TaggedFiles)
	}
}

func TestAddTaggedFilesFromTagsWithExclusion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-excl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("AddTaggedFilesFromTags failed: %v", err)
	}

	// Should only have main.go
	if _, ok := ctx.TaggedFiles["main.go"]; !ok {
		t.Error("TaggedFiles should contain main.go")
	}
	if _, ok := ctx.TaggedFiles["test/main_test.go"]; ok {
		t.Error("TaggedFiles should NOT contain test/main_test.go (excluded)")
	}
}

func TestListFilesForAutocomplete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-autocomplete-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files and directories
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log"), 0644)

	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package src"), 0644)

	orch := New(tmpDir)
	files, err := orch.ListFilesForAutocomplete(3)
	if err != nil {
		t.Fatalf("ListFilesForAutocomplete failed: %v", err)
	}

	// Should contain main.go, .gitignore, src/, src/util.go
	hasMain := false
	hasGitignore := false
	hasSrc := false

	for _, f := range files {
		switch f {
		case "main.go":
			hasMain = true
		case ".gitignore":
			hasGitignore = true
		case "src/":
			hasSrc = true
		}
	}

	if !hasMain {
		t.Error("files should contain main.go")
	}
	if !hasGitignore {
		t.Error("files should contain .gitignore")
	}
	if !hasSrc {
		t.Error("files should contain src/")
	}
}

func TestIterationContextTagPatterns(t *testing.T) {
	ctx := &IterationContext{
		PRDContent:  `{"name": "Test"}`,
		TagPatterns: []string{"@src/**/*.go", "@!vendor", "@main.go"},
		TaggedFiles: map[string]string{"main.go": "package main"},
		Iteration:   1,
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var restored IterationContext
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(restored.TagPatterns) != 3 {
		t.Errorf("Expected 3 tag patterns, got %d", len(restored.TagPatterns))
	}
	if restored.TagPatterns[0] != "@src/**/*.go" {
		t.Errorf("First tag pattern should be @src/**/*.go, got %s", restored.TagPatterns[0])
	}
}

// Tests for parallel action execution (feat-005)

func TestOrchestratorHasParallelExecutor(t *testing.T) {
	orch := New("/tmp/test")
	if orch.parallel == nil {
		t.Fatal("orchestrator should have a parallel executor")
	}
	if orch.GetParallelExecutor() == nil {
		t.Fatal("GetParallelExecutor should return the parallel executor")
	}
}

func TestOrchestratorSetParallelLimits(t *testing.T) {
	orch := New("/tmp/test")
	orch.SetParallelLimits(ParallelLimits{MaxReads: 5, MaxCommands: 2})

	pe := orch.GetParallelExecutor()
	if pe.limits.MaxReads != 5 {
		t.Errorf("Expected MaxReads=5, got %d", pe.limits.MaxReads)
	}
	if pe.limits.MaxCommands != 2 {
		t.Errorf("Expected MaxCommands=2, got %d", pe.limits.MaxCommands)
	}
}

func TestOrchestratorExecuteParallelReads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-parallel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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

	if !result.AllSucceeded {
		t.Errorf("Expected all reads to succeed, got %d failures", result.FailedCount)
		for _, r := range result.Results {
			if !r.Success {
				t.Logf("Failed: %v - %s", r.Action.Params.Paths, r.Error)
			}
		}
	}
	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}
}

func TestOrchestratorExecuteParallelMixed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-parallel-mixed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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

	if !result.AllSucceeded {
		t.Errorf("Expected all actions to succeed, got %d failures", result.FailedCount)
	}

	// Verify the file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "new.txt"))
	if err != nil {
		t.Errorf("Failed to read new.txt: %v", err)
	} else if string(content) != "new content" {
		t.Errorf("Expected 'new content', got %q", string(content))
	}
}

func TestActionParallelType(t *testing.T) {
	// Verify ActionParallel is a valid action type
	if ActionParallel != "parallel" {
		t.Errorf("Expected ActionParallel='parallel', got %q", ActionParallel)
	}

	// Verify it's in the list of valid actions
	validActions := map[Action]bool{
		ActionAskUser:    true,
		ActionReadFiles:  true,
		ActionWriteFile:  true,
		ActionRunCommand: true,
		ActionDone:       true,
		ActionParallel:   true,
	}

	if !validActions[ActionParallel] {
		t.Error("ActionParallel should be a valid action")
	}
}
