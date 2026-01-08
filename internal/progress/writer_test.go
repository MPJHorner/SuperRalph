package progress

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterAppend(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
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
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, writer.Exists())

	// Read the file content
	content, err := Read(writer.Path())
	require.NoError(t, err)

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
		assert.Contains(t, content, part)
	}
}

func TestWriterAppendMultiple(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	// Read and verify both iterations are present
	content, err := Read(writer.Path())
	require.NoError(t, err)

	assert.Contains(t, content, "Iteration: 1")
	assert.Contains(t, content, "Iteration: 2")
}

func TestGetPath(t *testing.T) {
	path := GetPath("/some/dir")
	assert.Equal(t, "/some/dir/progress.txt", path)
}

func TestExistsInDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Should not exist initially
	assert.False(t, ExistsInDir(tmpDir))

	// Create the file
	progressPath := filepath.Join(tmpDir, DefaultFilename)
	err = os.WriteFile(progressPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Should exist now
	assert.True(t, ExistsInDir(tmpDir))
}

func TestReadEmpty(t *testing.T) {
	content, err := Read("/nonexistent/path/progress.txt")
	assert.NoError(t, err)
	assert.Empty(t, content)
}
