package orchestrator

import (
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
	if !strings.Contains(prompt, "## Your Task") {
		t.Error("prompt should contain task instructions")
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
