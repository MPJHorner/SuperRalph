package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRepo(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// Should not be a repo initially
	assert.False(t, IsRepo(tmpDir), "IsRepo() = true for non-git dir")

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	require.NoError(t, err, "Failed to create .git dir")

	// Should be a repo now
	assert.True(t, IsRepo(tmpDir), "IsRepo() = false for git dir")
}

func TestInit(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// Should not be a repo initially
	assert.False(t, IsRepo(tmpDir), "IsRepo() = true before Init()")

	// Initialize
	err = Init(tmpDir)
	require.NoError(t, err, "Init() error")

	// Should be a repo now
	assert.True(t, IsRepo(tmpDir), "IsRepo() = false after Init()")
}

func TestEnsureRepo(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// First call should create repo
	created, err := EnsureRepo(tmpDir)
	require.NoError(t, err, "EnsureRepo() error")
	assert.True(t, created, "EnsureRepo() created = false, want true")

	// Second call should not create repo
	created, err = EnsureRepo(tmpDir)
	require.NoError(t, err, "EnsureRepo() second call error")
	assert.False(t, created, "EnsureRepo() second call created = true, want false")
}

func TestGetRecentCommitsNoCommits(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	require.NoError(t, err, "Init() error")

	// Should return empty slice for repo with no commits
	commits, err := GetRecentCommits(tmpDir, 5)
	if err != nil {
		// This might error on some git versions, that's ok
		t.Logf("GetRecentCommits() error = %v (may be expected)", err)
	}
	assert.Empty(t, commits, "GetRecentCommits() should return empty slice")
}

func TestHasUncommittedChanges(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	require.NoError(t, err, "Init() error")

	// Should have no uncommitted changes initially (empty repo)
	// Note: new repos have no changes because there's nothing to commit yet
	hasChanges, err := HasUncommittedChanges(tmpDir)
	require.NoError(t, err, "HasUncommittedChanges() error")
	assert.False(t, hasChanges, "HasUncommittedChanges() = true for empty repo")

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Should have uncommitted changes now
	hasChanges, err = HasUncommittedChanges(tmpDir)
	require.NoError(t, err, "HasUncommittedChanges() error")
	assert.True(t, hasChanges, "HasUncommittedChanges() = false after creating file")
}

func TestGetStatus(t *testing.T) {
	// Create a temp directory and init git
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	err = Init(tmpDir)
	require.NoError(t, err, "Init() error")

	// Get status (should be empty)
	status, err := GetStatus(tmpDir)
	require.NoError(t, err, "GetStatus() error")
	assert.Empty(t, status, "GetStatus() should be empty for empty repo")

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Get status (should show untracked file)
	status, err = GetStatus(tmpDir)
	require.NoError(t, err, "GetStatus() error")
	assert.NotEmpty(t, status, "GetStatus() = empty after creating file")
}
