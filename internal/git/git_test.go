package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsRepo(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not be a repo initially
	if IsRepo(tmpDir) {
		t.Error("IsRepo() = true for non-git dir")
	}

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Should be a repo now
	if !IsRepo(tmpDir) {
		t.Error("IsRepo() = false for git dir")
	}
}

func TestInit(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not be a repo initially
	if IsRepo(tmpDir) {
		t.Error("IsRepo() = true before Init()")
	}

	// Initialize
	err = Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Should be a repo now
	if !IsRepo(tmpDir) {
		t.Error("IsRepo() = false after Init()")
	}
}

func TestEnsureRepo(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First call should create repo
	created, err := EnsureRepo(tmpDir)
	if err != nil {
		t.Fatalf("EnsureRepo() error = %v", err)
	}
	if !created {
		t.Error("EnsureRepo() created = false, want true")
	}

	// Second call should not create repo
	created, err = EnsureRepo(tmpDir)
	if err != nil {
		t.Fatalf("EnsureRepo() second call error = %v", err)
	}
	if created {
		t.Error("EnsureRepo() second call created = true, want false")
	}
}

func TestGetRecentCommitsNoCommits(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Should return empty slice for repo with no commits
	commits, err := GetRecentCommits(tmpDir, 5)
	if err != nil {
		// This might error on some git versions, that's ok
		t.Logf("GetRecentCommits() error = %v (may be expected)", err)
	}
	if len(commits) != 0 {
		t.Errorf("GetRecentCommits() = %v, want empty slice", commits)
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Should have no uncommitted changes initially (empty repo)
	// Note: new repos have no changes because there's nothing to commit yet
	hasChanges, err := HasUncommittedChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() error = %v", err)
	}
	if hasChanges {
		t.Error("HasUncommittedChanges() = true for empty repo")
	}

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should have uncommitted changes now
	hasChanges, err = HasUncommittedChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() error = %v", err)
	}
	if !hasChanges {
		t.Error("HasUncommittedChanges() = false after creating file")
	}
}

func TestGetStatus(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Get status (should be empty)
	status, err := GetStatus(tmpDir)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status != "" {
		t.Errorf("GetStatus() = %q for empty repo, want empty", status)
	}

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get status (should show untracked file)
	status, err = GetStatus(tmpDir)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status == "" {
		t.Error("GetStatus() = empty after creating file")
	}
}
