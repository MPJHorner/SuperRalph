package progress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterAppend(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	// Create a test entry
	entry := Entry{
		Timestamp: time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
		Iteration: 1,
		StartingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 3,
			WorkingOn: &FeatureRef{
				ID:          "feat-004",
				Description: "Test feature",
			},
		},
		WorkDone: []string{
			"Implemented feature",
			"Added tests",
		},
		Testing: TestResult{
			Command: "go test ./...",
			Passed:  true,
			Details: "All tests passed",
		},
		Commits: []Commit{
			{Hash: "abc1234", Message: "feat: add feature"},
		},
		EndingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 4,
			WorkingOn: &FeatureRef{
				ID:          "feat-004",
				Description: "Test feature",
			},
			AllTestsPassing: true,
		},
		NotesForNextSession: []string{
			"Consider adding caching",
		},
	}

	// Append entry
	err = writer.Append(entry)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Verify file exists
	if !writer.Exists() {
		t.Error("Exists() = false after Append()")
	}

	// Read the file content
	content, err := Read(writer.Path())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Verify content contains expected parts
	expectedParts := []string{
		"Session: 2026-01-07T12:00:00Z",
		"Iteration: 1",
		"Features passing: 3/10",
		"Working on: feat-004",
		"Implemented feature",
		"Added tests",
		"go test ./...",
		"PASSED",
		"abc1234: feat: add feature",
		"Features passing: 4/10",
		"All tests passing: YES",
		"Consider adding caching",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("Content missing expected part: %q", part)
		}
	}
}

func TestWriterAppendMultiple(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	// Append first entry
	entry1 := Entry{
		Timestamp: time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
		Iteration: 1,
		StartingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 0,
		},
		EndingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 1,
		},
	}

	err = writer.Append(entry1)
	if err != nil {
		t.Fatalf("Append(entry1) error = %v", err)
	}

	// Append second entry
	entry2 := Entry{
		Timestamp: time.Date(2026, 1, 7, 13, 0, 0, 0, time.UTC),
		Iteration: 2,
		StartingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 1,
		},
		EndingState: State{
			FeaturesTotal:   10,
			FeaturesPassing: 2,
		},
	}

	err = writer.Append(entry2)
	if err != nil {
		t.Fatalf("Append(entry2) error = %v", err)
	}

	// Read and verify both iterations are present
	content, err := Read(writer.Path())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !strings.Contains(content, "Iteration: 1") {
		t.Error("Content missing Iteration: 1")
	}
	if !strings.Contains(content, "Iteration: 2") {
		t.Error("Content missing Iteration: 2")
	}
}

func TestGetPath(t *testing.T) {
	path := GetPath("/some/dir")
	expected := "/some/dir/progress.txt"
	if path != expected {
		t.Errorf("GetPath() = %q, want %q", path, expected)
	}
}

func TestExistsInDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not exist initially
	if ExistsInDir(tmpDir) {
		t.Error("ExistsInDir() = true for empty dir")
	}

	// Create the file
	progressPath := filepath.Join(tmpDir, DefaultFilename)
	err = os.WriteFile(progressPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should exist now
	if !ExistsInDir(tmpDir) {
		t.Error("ExistsInDir() = false after creating file")
	}
}

func TestReadEmpty(t *testing.T) {
	content, err := Read("/nonexistent/path/progress.txt")
	if err != nil {
		t.Errorf("Read() error = %v for nonexistent file", err)
	}
	if content != "" {
		t.Errorf("Read() = %q for nonexistent file, want empty string", content)
	}
}
