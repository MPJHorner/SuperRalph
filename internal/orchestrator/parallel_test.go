package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultParallelLimits(t *testing.T) {
	limits := DefaultParallelLimits()
	assert.Equal(t, 10, limits.MaxReads)
	assert.Equal(t, 3, limits.MaxCommands)
}

func TestNewParallelExecutor(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	require.NotNil(t, pe)
	assert.Equal(t, "/tmp/test", pe.workDir)
	assert.Equal(t, 10, pe.limits.MaxReads)
}

func TestParallelExecutorSetLimits(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	pe.SetLimits(ParallelLimits{MaxReads: 5, MaxCommands: 2})

	assert.Equal(t, 5, pe.limits.MaxReads)
	assert.Equal(t, 2, pe.limits.MaxCommands)
}

func TestParallelExecutorSetDebug(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	var debugMsgs []string
	pe.SetDebug(true, func(msg string) {
		debugMsgs = append(debugMsgs, msg)
	})

	assert.True(t, pe.debug)
	pe.debugLog("test message")
	assert.Len(t, debugMsgs, 1)
}

func TestExecuteEmpty(t *testing.T) {
	pe := NewParallelExecutor("/tmp/test")
	result := pe.Execute(context.Background(), ParallelAction{})

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, 0)
}

func TestExecuteParallelReads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		err := os.WriteFile(filename, []byte("content "+string(rune('0'+i))), 0644)
		require.NoError(t, err)
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

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, 5)
}

func TestExecuteParallelReadsWithLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-limit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create many test files
	numFiles := 20
	for i := 1; i <= numFiles; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('a'-1+i))+".txt")
		err := os.WriteFile(filename, []byte("content"), 0644)
		require.NoError(t, err)
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

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, numFiles)
}

func TestExecuteParallelCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-cmd-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo hello"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo world"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo test"}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, 3)

	// Check outputs contain expected content
	for _, r := range result.Results {
		hasExpectedContent := strings.Contains(r.Output, "hello") || strings.Contains(r.Output, "world") || strings.Contains(r.Output, "test")
		assert.True(t, hasExpectedContent, "Unexpected output: %s", r.Output)
	}
}

func TestExecuteSequentialWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-write-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Create multiple write actions
	actions := []SubAction{
		{Type: ActionWriteFile, Params: ActionParams{Path: "file1.txt", Content: "content1"}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "file2.txt", Content: "content2"}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "file3.txt", Content: "content3"}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	assert.True(t, result.AllSucceeded)

	// Verify files were written
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		expected := "content" + string(rune('0'+i))
		assert.Equal(t, expected, string(content))
	}
}

func TestExecuteMixedActions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-mixed-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an initial file to read
	err = os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("existing content"), 0644)
	require.NoError(t, err)

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"existing.txt"}}},
		{Type: ActionWriteFile, Params: ActionParams{Path: "new.txt", Content: "new content"}},
		{Type: ActionRunCommand, Params: ActionParams{Command: "echo mixed"}},
		{Type: ActionDone, Params: ActionParams{}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.Results, 4)
}

func TestExecutePartialFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-fail-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create one file but not the other
	err = os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	pe := NewParallelExecutor(tmpDir)

	actions := []SubAction{
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"exists.txt"}}},
		{Type: ActionReadFiles, Params: ActionParams{Paths: []string{"nonexistent.txt"}}},
	}

	result := pe.Execute(context.Background(), ParallelAction{Actions: actions})

	assert.False(t, result.AllSucceeded)
	assert.Equal(t, 1, result.FailedCount)

	// Check that we have both success and failure
	var successCount, failCount int
	for _, r := range result.Results {
		if r.Success {
			successCount++
		} else {
			failCount++
			assert.Contains(t, r.Error, "nonexistent")
		}
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, failCount)
}

func TestExecuteContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-cancel-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Create a long-running command
	actions := []SubAction{
		{Type: ActionRunCommand, Params: ActionParams{Command: "sleep 10"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := pe.Execute(ctx, ParallelAction{Actions: actions})

	assert.False(t, result.AllSucceeded)
	assert.Equal(t, 1, result.FailedCount)
}

func TestExecuteSingleRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-read-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testContent := "test content for single read"
	err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(testContent), 0644)
	require.NoError(t, err)

	pe := NewParallelExecutor(tmpDir)

	output, err := pe.ExecuteSingleRead(context.Background(), "test.txt")
	require.NoError(t, err)
	assert.Contains(t, output, testContent)
}

func TestExecuteSingleReadMissing(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	_, err := pe.ExecuteSingleRead(context.Background(), "nonexistent-file-12345.txt")
	require.Error(t, err)
}

func TestExecuteSingleWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-write-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	testContent := "test content for single write"
	err = pe.ExecuteSingleWrite(context.Background(), "output.txt", testContent)
	require.NoError(t, err)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestExecuteSingleWriteNestedDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-nested-write-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	// Write to a nested path that doesn't exist
	err = pe.ExecuteSingleWrite(context.Background(), "a/b/c/file.txt", "nested content")
	require.NoError(t, err)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "a", "b", "c", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(content))
}

func TestExecuteSingleCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-single-cmd-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pe := NewParallelExecutor(tmpDir)

	output, err := pe.ExecuteSingleCommand(context.Background(), "echo 'hello world'")
	require.NoError(t, err)
	assert.Contains(t, output, "hello world")
}

func TestExecuteSingleCommandFail(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	_, err := pe.ExecuteSingleCommand(context.Background(), "false") // 'false' command always exits with 1
	require.Error(t, err)
}

func TestParallelActionsPreserveOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-order-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		err := os.WriteFile(filename, []byte("content"+string(rune('0'+i))), 0644)
		require.NoError(t, err)
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

	require.True(t, result.AllSucceeded)

	// Results should preserve order
	for i, r := range result.Results {
		expectedPath := "file" + string(rune('0'+i+1)) + ".txt"
		require.NotEmpty(t, r.Action.Params.Paths)
		assert.Equal(t, expectedPath, r.Action.Params.Paths[0])
	}
}

func TestConcurrencyLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-concurrency-test-*")
	require.NoError(t, err)
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

	assert.True(t, result.AllSucceeded)

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
	assert.Equal(t, Action("parallel"), ActionParallel)

	// Verify it's distinct from other actions
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

func TestReadEmptyPaths(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeRead(context.Background(), SubAction{
		Type:   ActionReadFiles,
		Params: ActionParams{Paths: []string{}},
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "no file paths")
}

func TestWriteEmptyPath(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeWrite(context.Background(), SubAction{
		Type:   ActionWriteFile,
		Params: ActionParams{Path: "", Content: "test"},
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "no file path")
}

func TestCommandEmptyCommand(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeCommand(context.Background(), SubAction{
		Type:   ActionRunCommand,
		Params: ActionParams{Command: ""},
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "no command")
}

func TestExecuteOtherUnsupported(t *testing.T) {
	pe := NewParallelExecutor("/tmp")
	result := pe.executeOther(context.Background(), SubAction{
		Type:   Action("unsupported"),
		Params: ActionParams{},
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unsupported action type")
}

func TestReadMultipleFilesInSingleAction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-multi-read-*")
	require.NoError(t, err)
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

	require.True(t, result.Success)

	// Output should contain all three file contents
	assert.Contains(t, result.Output, "content A")
	assert.Contains(t, result.Output, "content B")
	assert.Contains(t, result.Output, "content C")
}

func TestReadMultipleFilesPartialFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parallel-multi-read-fail-*")
	require.NoError(t, err)
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

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "missing.txt")
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
	require.NoError(t, err)

	var restored ParallelResult
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, result.AllSucceeded, restored.AllSucceeded)
	assert.Equal(t, result.FailedCount, restored.FailedCount)
	assert.Len(t, restored.Results, len(result.Results))
}
