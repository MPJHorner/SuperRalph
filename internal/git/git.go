package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo checks if the given directory is a git repository
func IsRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsRepoCurrentDir checks if the current directory is a git repository
func IsRepoCurrentDir() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	return IsRepo(cwd)
}

// Init initializes a new git repository in the given directory
func Init(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %s: %w", string(output), err)
	}
	return nil
}

// InitCurrentDir initializes a git repository in the current directory
func InitCurrentDir() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	return Init(cwd)
}

// EnsureRepo ensures a git repository exists, creating one if necessary
func EnsureRepo(dir string) (bool, error) {
	if IsRepo(dir) {
		return false, nil // Already a repo
	}
	if err := Init(dir); err != nil {
		return false, err
	}
	return true, nil // Created new repo
}

// EnsureRepoCurrentDir ensures current directory is a git repository
func EnsureRepoCurrentDir() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}
	return EnsureRepo(cwd)
}

// GetRecentCommits returns recent commit messages
func GetRecentCommits(dir string, count int) ([]string, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--oneline")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// If there are no commits yet, that's fine
		if strings.Contains(string(output), "does not have any commits") {
			return nil, nil
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

// GetRecentCommitsCurrentDir returns recent commits from current directory
func GetRecentCommitsCurrentDir(count int) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return GetRecentCommits(cwd, count)
}

// HasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// HasUncommittedChangesCurrentDir checks current directory for uncommitted changes
func HasUncommittedChangesCurrentDir() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	return HasUncommittedChanges(cwd)
}

// GetStatus returns the git status output
func GetStatus(dir string) (string, error) {
	cmd := exec.Command("git", "status", "-s")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return string(output), nil
}

// GetStatusCurrentDir returns git status for current directory
func GetStatusCurrentDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return GetStatus(cwd)
}
