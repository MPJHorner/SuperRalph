package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// These are set at build time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

const (
	repoOwner = "MPJHorner"
	repoName  = "SuperRalph"
)

// Info returns formatted version information
func Info() string {
	return fmt.Sprintf("superralph %s (%s) built %s", Version, GitCommit[:min(7, len(GitCommit))], BuildTime)
}

// Short returns just the version string
func Short() string {
	return Version
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// CheckForUpdate checks if a newer version is available
func CheckForUpdate() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// IsNewer returns true if the release version is newer than current
func IsNewer(release *GitHubRelease) bool {
	if release == nil {
		return false
	}

	if Version == "dev" {
		return false
	}

	current := strings.TrimPrefix(Version, "v")
	latest := strings.TrimPrefix(release.TagName, "v")

	return compareSemver(latest, current) > 0
}

// compareSemver compares two semver strings, returning:
// -1 if a < b, 0 if a == b, 1 if a > b
func compareSemver(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}

		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}

	return 0
}

// GetUpdateMessage returns a message if an update is available
func GetUpdateMessage() string {
	release, err := CheckForUpdate()
	if err != nil {
		return ""
	}

	if IsNewer(release) {
		return fmt.Sprintf("\nUpdate available: %s → %s\nRun 'superralph update' to upgrade\n",
			Version, release.TagName)
	}

	return ""
}

// CheckForUpdateAsync checks for updates in the background and calls the callback
func CheckForUpdateAsync(callback func(msg string)) {
	go func() {
		msg := GetUpdateMessage()
		if msg != "" && callback != nil {
			callback(msg)
		}
	}()
}

// SelfUpdate downloads and installs the latest version
func SelfUpdate() error {
	release, err := CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !IsNewer(release) {
		return fmt.Errorf("already at latest version (%s)", Version)
	}

	// Determine binary name based on OS/arch
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binaryName := fmt.Sprintf("superralph-%s-%s", goos, goarch)

	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner, repoName, release.TagName, binaryName)

	// Get current binary path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "superralph-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Download new binary
	fmt.Printf("Downloading %s...\n", release.TagName)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = outFile.ReadFrom(resp.Body)
	outFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	// Replace current binary
	fmt.Printf("Installing to %s...\n", execPath)

	// Check if we need sudo
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try with sudo
		fmt.Println("Need elevated permissions, using sudo...")
		cmd := exec.Command("sudo", "mv", tmpPath, execPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install update: %w", err)
		}
	}

	fmt.Printf("Successfully updated to %s\n", release.TagName)
	return nil
}
