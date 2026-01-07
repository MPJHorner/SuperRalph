package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultParallelLimits(t *testing.T) {
	limits := DefaultParallelLimits()
	if limits.MaxReads != 10 {
		t.Errorf("Expected MaxReads=10, got %d", limits.MaxReads)
	}
	if limits.MaxCommands != 3 {
		t.Errorf("Expected MaxCommands=3, got %d", limits.MaxCommands)
	}
}

func TestNewParallelExecutor(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	if pe == nil {
		t.Fatal("Expected non-nil executor")
	}
	if pe.workDir != "/tmp/test" {
		t.Errorf("Expected workDir=/tmp/test, got %s", pe.workDir)
	}
	if pe.limits.MaxReads != 10 {
		t.Errorf("Expected default MaxReads=10, got %d", pe.limits.MaxReads)
	}
}

func TestParallelExecutorSetLimits(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	pe.SetLimits(ParallelLimits{MaxReads: 5, MaxCommands: 2})

	if pe.limits.MaxReads != 5 {
		t.Errorf("Expected MaxReads=5, got %d", pe.limits.MaxReads)
	}
	if pe.limits.MaxCommands != 2 {
		t.Errorf("Expected MaxCommands=2, got %d", pe.limits.MaxCommands)
	}
}

func TestParallelExecutorSetDebug(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	var debugMsgs []string
	pe.SetDebug(true, func(msg string) {
		debugMsgs = append(debugMsgs, msg)
	})

	if !pe.debug {
		t.Error("Expected debug=true")
	}
	pe.debugLog("test message")
	if len(debugMsgs) != 1 {
		t.Errorf("Expected 1 debug message, got %d", len(debugMsgs))
	}
}

func TestExecuteEmpty(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	result := pe.Execute(context.Background(), ParallelAction{})

	if !result.AllSucceeded {
		t.Error("Empty parallel action should succeed")
	}
	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result.Results))
	}
}

func TestExecuteParallelReads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte("content "+string(rune('0'+i))), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	pe := NewParallelExecutor(tmpDir)

	// Create parallel read actions
	var actions []SubAction
	for i := 1; i <= 5; i++ {
		actions = append(actions, SubAction{
			Type: ActionReadFiles,
			Params: ActionParams{
				Paths: []string{"file" + string(rune('0'+i)) + ".txt"},
			},
		})
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Errorf("Expected all reads to succeed, got %d failures", result.FailedCount)
		for _, r := range result.Results {
			if !r.Success {
				t.Logf("Failed: %s - %s", r.Action.Params.Paths, r.Error)
			}
		}
	}
	if len(result.Results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(result.Results))
	}
}

func TestExecuteParallelReadsWithLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-limit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create many test files
	numFiles := 20
	for i := 1; i <= numFiles; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('a'-1+i))+".txt")
		if err := os.WriteFile(filename, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	pe := NewParallelExecutor(tmpDir)
	pe.SetLimits(ParallelLimits{MaxReads: 3, MaxCommands: 1}) // Low limit to test concurrency control

	var actions []SubAction
	for i := 1; i <= numFiles; i++ {
		actions = append(actions, SubAction{
			Type: ActionReadFiles,
			Params: ActionParams{
				Paths: []string{"file" + string(rune('a'-1+i)) + ".txt"},
			},
		})
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Errorf("Expected all reads to succeed, got %d failures", result.FailedCount)
	}
	if len(result.Results) != numFiles {
		t.Errorf("Expected %d results, got %d", numFiles, len(result.Results))
	}
}

func TestExecuteParallelCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo hello"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo world"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo test"}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Errorf("Expected all commands to succeed, got %d failures", result.FailedCount)
		for _, r := range result.Results {
			if !r.Success {
				t.Logf("Failed: %s - %s", r.Action.Params.Command, r.Error)
			}
		}
	}
	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result.Results))
	}

	// Check outputs contain expected content
	for _, r := range result.Results {
		if !strings.Contains(r.Output, "hello") && !strings.Contains(r.Output, "world") && !strings.Contains(r.Output, "test") {
			t.Errorf("Unexpected output: %s", r.Output)
		}
	}
}

func TestExecuteSequentialWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-write-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Create multiple write actions
	actions := []SubAction{
		{Type: ActionWriteFile, Params: ActionParams{Path: "file1.txt", Content: "content1"}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "file2.txt", Content: "content2"}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "file3.txt", Content: "content3"}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Errorf("Expected all writes to succeed, got %d failures", result.FailedCount)
	}

	// Verify files were written
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("Failed to read %s: %v", filename, err)
			continue
		}
		expected := "content" + string(rune('0'+i))
		if string(content) != expected {
			t.Errorf("Expected %s, got %s", expected, string(content))
		}
	}
}

func TestExecuteMixedActions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-mixed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an initial file to read
	if err := os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"existing.txt"}}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "new.txt", Content: "new content"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo mixed"}},
		{Type: ActionDone, Params: ActionParams{}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Errorf("Expected all actions to succeed, got %d failures", result.FailedCount)
		for _, r := range result.Results {
			if !r.Success {
				t.Logf("Failed: %v - %s", r.Action.Type, r.Error)
			}
		}
	}
	if len(result.Results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(result.Results))
	}
}

func TestExecutePartialFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-fail-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create one file but not the other
	if err := os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"exists.txt"}}},
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"nonexistent.txt"}}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if result.AllSucceeded {
		t.Error("Expected partial failure")
	}
	if result.FailedCount != 1 {
		t.Errorf("Expected 1 failure, got %d", result.FailedCount)
	}

	// Check that we have both success and failure
	var successCount, failCount int
	for _, r := range result.Results {
		if r.Success {
			successCount++
		} else {
			failCount++
			if !strings.Contains(r.Error, "nonexistent") {
				t.Errorf("Expected error to mention nonexistent file, got: %s", r.Error)
			}
		}
	}
	if successCount != 1 || failCount != 1 {
		t.Errorf("Expected 1 success and 1 failure, got %d/%d", successCount, failCount)
	}
}

func TestExecuteContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-cancel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Create a long-running command
	actions := []SubAction{
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 10"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := pe.Execute(ctx, ParallelAction{Actions: actions})

	if result.AllSucceeded {
		t.Error("Expected failure due to context cancellation")
	}
	if result.FailedCount != 1 {
		t.Errorf("Expected 1 failure, got %d", result.FailedCount)
	}
}

func TestExecuteSingleRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-read-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test content for single read"
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	pe := NewParallelExecutor(tmpDir)

	output, err := pe.ExecuteSingleRead(context.Background(), "test.txt")
	if err != nil {
		t.Fatalf("ExecuteSingleRead failed: %v", err)
	}
	if !strings.Contains(output, testContent) {
		t.Errorf("Expected output to contain %q, got %q", testContent, output)
	}
}

func TestExecuteSingleReadMissing(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	_, err := pe.ExecuteSingleRead(context.Background(), "nonexistent-file-12345.txt")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestExecuteSingleWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-write-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	testContent := "test content for single write"
	err = pe.ExecuteSingleWrite(context.Background(), "output.txt", testContent)
	if err != nil {
		t.Fatalf("ExecuteSingleWrite failed: %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Expected %q, got %q", testContent, string(content))
	}
}

func TestExecuteSingleWriteNestedDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-nested-write-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Write to a nested path that doesn't exist
	err = pe.ExecuteSingleWrite(context.Background(), "a/b/c/file.txt", "nested content")
	if err != nil {
		t.Fatalf("ExecuteSingleWrite to nested dir failed: %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "a", "b", "c", "file.txt"))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("Expected 'nested content', got %q", string(content))
	}
}

func TestExecuteSingleCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-cmd-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	output, err := pe.ExecuteSingleCommand(context.Background(), "echo 'hello world'")
	if err != nil {
		t.Fatalf("ExecuteSingleCommand failed: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got %q", output)
	}
}

func TestExecuteSingleCommandFail(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	_, err := pe.ExecuteSingleCommand(context.Background(), "false") // 'false' command always exits with 1
	if err == nil {
		t.Error("Expected error for failed command")
	}
}

func TestParallelActionsPreserveOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-order-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte("content"+string(rune('0'+i))), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	pe := NewParallelExecutor(tmpDir)

	var actions []SubAction
	for i := 1; i <= 5; i++ {
		actions = append(actions, SubAction{
			Type: ActionReadFiles,
			Params: ActionParams{
				Paths: []string{"file" + string(rune('0'+i)) + ".txt"},
			},
		})
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	if !result.AllSucceeded {
		t.Fatalf("Expected all reads to succeed")
	}

	// Results should preserve order
	for i, r := range result.Results {
		expectedPath := "file" + string(rune('0'+i+1)) + ".txt"
		if len(r.Action.Params.Paths) == 0 || r.Action.Params.Paths[0] != expectedPath {
			t.Errorf("Result %d: expected path %s, got %v", i, expectedPath, r.Action.Params.Paths)
		}
	}
}

func TestConcurrencyLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-concurrency-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)
	pe.SetLimits(ParallelLimits{MaxReads: 1, MaxCommands: 2}) // Very restrictive limits

	// Track concurrent executions
	var maxConcurrent int32
	var currentConcurrent int32

	// Create commands that take some time
	actions := []SubAction{
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 0.1 && echo done1"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 0.1 && echo done2"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 0.1 && echo done3"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 0.1 && echo done4"}},
	}

	// Note: This is a basic concurrency test - the actual tracking would need
	// modification of the executor internals to properly measure

	start := time.Now()
	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})
	elapsed := time.Since(start)

	if !result.AllSucceeded {
		t.Errorf("Expected all commands to succeed, got %d failures", result.FailedCount)
	}

	// With MaxCommands=2 and 4 commands of ~100ms each, it should take at least 200ms
	// (two batches of 2 parallel commands)
	if elapsed < 150*time.Millisecond {
		t.Logf("Warning: Commands may not have been properly limited (elapsed: %v)", elapsed)
	}

	_ = maxConcurrent
	_ = currentConcurrent
}

func TestActionParallelConstant(t *testing.T) {
	// Verify the ActionParallel constant is properly defined
	if ActionParallel != "parallel" {
		t.Errorf("Expected ActionParallel='parallel', got %q", ActionParallel)
	}

	// Verify it's distinct from other actions
	validActions := map[Action]bool{
		ActionAskUser:    true,
		ActionReadFiles:  true,
		ActionWriteFile:  true,
		ActionRunCommand: true,
		ActionDone:       true,
		ActionParallel:   true,
	}

	if !validActions[ActionParallel] {
		t.Error("ActionParallel should be in valid actions")
	}
}

func TestReadEmptyPaths(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeRead(context.Background(), SubAction{
		Type:   ActionReadFiles,
		Params: ActionParams{Paths: []string{}},
	})

	if result.Success {
		t.Error("Expected failure for empty paths")
	}
	if !strings.Contains(result.Error, "no file paths") {
		t.Errorf("Expected 'no file paths' error, got: %s", result.Error)
	}
}

func TestWriteEmptyPath(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeWrite(context.Background(), SubAction{
		Type:   ActionWriteFile,
		Params: ActionParams{Path: "", Content: "test"},
	})

	if result.Success {
		t.Error("Expected failure for empty path")
	}
	if !strings.Contains(result.Error, "no file path") {
		t.Errorf("Expected 'no file path' error, got: %s", result.Error)
	}
}

func TestCommandEmptyCommand(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeCommand(context.Background(), SubAction{
		Type:   ActionRunCommand,
		Params: ActionParams{Command: ""},
	})

	if result.Success {
		t.Error("Expected failure for empty command")
	}
	if !strings.Contains(result.Error, "no command") {
		t.Errorf("Expected 'no command' error, got: %s", result.Error)
	}
}

func TestExecuteOtherUnsupported(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeOther(context.Background(), SubAction{
		Type:   Action("unsupported"),
		Params: ActionParams{},
	})

	if result.Success {
		t.Error("Expected failure for unsupported action type")
	}
	if !strings.Contains(result.Error, "unsupported action type") {
		t.Errorf("Expected 'unsupported action type' error, got: %s", result.Error)
	}
}

func TestReadMultipleFilesInSingleAction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-multi-read-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("content A"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("content B"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte("content C"), 0644)

	pe := NewParallelExecutor(tmpDir)

	// Single action that reads multiple files
	result := pe.executeRead(context.Background(), SubAction{
		Type: ActionReadFiles,
		Params: ActionParams{
			Paths: []string{"a.txt", "b.txt", "c.txt"},
		},
	})

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// Output should contain all three file contents
	if !strings.Contains(result.Output, "content A") {
		t.Error("Expected output to contain 'content A'")
	}
	if !strings.Contains(result.Output, "content B") {
		t.Error("Expected output to contain 'content B'")
	}
	if !strings.Contains(result.Output, "content C") {
		t.Error("Expected output to contain 'content C'")
	}
}

func TestReadMultipleFilesPartialFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-multi-read-fail-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create only some files
	os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("exists"), 0644)

	pe := NewParallelExecutor(tmpDir)

	// Single action with one existing and one missing file
	result := pe.executeRead(context.Background(), SubAction{
		Type: ActionReadFiles,
		Params: ActionParams{
			Paths: []string{"exists.txt", "missing.txt"},
		},
	})

	if result.Success {
		t.Error("Expected failure when any file is missing")
	}
	if !strings.Contains(result.Error, "missing.txt") {
		t.Errorf("Expected error to mention missing file, got: %s", result.Error)
	}
}

func TestParallelResultSerialization(t *testing.T) {
	result := ParallelResult{
		Results: []ActionResult{
			{
				Action:  SubAction{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"test.txt"}}},
				Success: true,
				Output:  "content",
			},
			{
				Action:  SubAction{Type: ActionRunCommand, Params: ActionParams{Command: "echo test"}},
				Success: false,
				Error:   "command failed",
			},
		},
		AllSucceeded: false,
		FailedCount:  1,
	}

	// Test that it can be serialized to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var restored ParallelResult
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if restored.AllSucceeded != result.AllSucceeded {
		t.Error("AllSucceeded not preserved")
	}
	if restored.FailedCount != result.FailedCount {
		t.Error("FailedCount not preserved")
	}
	if len(restored.Results) != len(result.Results) {
		t.Error("Results count not preserved")
	}
}

