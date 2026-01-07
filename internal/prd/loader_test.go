package prd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSave(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if !Exists(prdPath) {
		t.Error("Exists() = false after Save()")
	}

	// Load it back
	loaded, err := Load(prdPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify contents
	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if loaded.Description != original.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, original.Description)
	}
	if loaded.TestCommand != original.TestCommand {
		t.Errorf("TestCommand = %q, want %q", loaded.TestCommand, original.TestCommand)
	}
	if len(loaded.Features) != len(original.Features) {
		t.Errorf("len(Features) = %d, want %d", len(loaded.Features), len(original.Features))
	}
	if len(loaded.Features) > 0 {
		if loaded.Features[0].ID != original.Features[0].ID {
			t.Errorf("Features[0].ID = %q, want %q", loaded.Features[0].ID, original.Features[0].ID)
		}
	}
}

func TestLoadFromDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("SaveToDir() error = %v", err)
	}

	// Verify file exists
	if !ExistsInDir(tmpDir) {
		t.Error("ExistsInDir() = false after SaveToDir()")
	}

	// Load using LoadFromDir
	loaded, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/prd.json")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid JSON
	prdPath := filepath.Join(tmpDir, "prd.json")
	err = os.WriteFile(prdPath, []byte("not valid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = Load(prdPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestExistsNotFound(t *testing.T) {
	if Exists("/nonexistent/path/prd.json") {
		t.Error("Exists() = true for nonexistent file")
	}
}
