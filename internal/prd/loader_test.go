package prd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndSave(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test PRD
	original := &PRD{
		Name:        "Test Project",
		Description: "Test description",
		TestCommand: "go test ./...",
		Features: []Feature{
			{
				ID:          "feat-001",
				Category:    CategoryFunctional,
				Priority:    PriorityHigh,
				Description: "Test feature",
				Steps:       []string{"Step 1", "Step 2"},
				Passes:      false,
			},
		},
	}

	// Save it
	prdPath := filepath.Join(tmpDir, "prd.json")
	err = Save(original, prdPath)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, Exists(prdPath))

	// Load it back
	loaded, err := Load(prdPath)
	require.NoError(t, err)

	// Verify contents
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)
	assert.Equal(t, original.TestCommand, loaded.TestCommand)
	require.Len(t, loaded.Features, len(original.Features))
	assert.Equal(t, original.Features[0].ID, loaded.Features[0].ID)
}

func TestLoadFromDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test PRD
	original := &PRD{
		Name:        "Test Project",
		Description: "Test description",
		TestCommand: "go test ./...",
		Features: []Feature{
			{
				ID:          "feat-001",
				Category:    CategoryFunctional,
				Priority:    PriorityHigh,
				Description: "Test feature",
				Steps:       []string{"Step 1"},
				Passes:      false,
			},
		},
	}

	// Save using SaveToDir
	err = SaveToDir(original, tmpDir)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, ExistsInDir(tmpDir))

	// Load using LoadFromDir
	loaded, err := LoadFromDir(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, original.Name, loaded.Name)
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/prd.json")
	require.Error(t, err)
}

func TestLoadInvalidJSON(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Write invalid JSON
	prdPath := filepath.Join(tmpDir, "prd.json")
	err = os.WriteFile(prdPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	_, err = Load(prdPath)
	require.Error(t, err)
}

func TestExistsNotFound(t *testing.T) {
	assert.False(t, Exists("/nonexistent/path/prd.json"))
}
