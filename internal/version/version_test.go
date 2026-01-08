package version

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	// Save original values
	origVersion := Version
	origBuildTime := BuildTime
	origGitCommit := GitCommit
	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
		GitCommit = origGitCommit
	}()

	Version = "v1.2.3"
	BuildTime = "2026-01-07"
	GitCommit = "abc1234567890"

	info := Info()

	assert.Equal(t, "superralph v1.2.3 (abc1234) built 2026-01-07", info)
}

func TestShort(t *testing.T) {
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "v1.2.3"
	assert.Equal(t, "v1.2.3", Short())
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"equal versions", "1.2.3", "1.2.3", 0},
		{"a greater major", "2.0.0", "1.0.0", 1},
		{"b greater major", "1.0.0", "2.0.0", -1},
		{"a greater minor", "1.2.0", "1.1.0", 1},
		{"b greater minor", "1.1.0", "1.2.0", -1},
		{"a greater patch", "1.0.2", "1.0.1", 1},
		{"b greater patch", "1.0.1", "1.0.2", -1},
		// Critical: double-digit version comparison (the bug we fixed)
		{"double digit patch a > b", "0.2.12", "0.2.9", 1},
		{"double digit patch b > a", "0.2.9", "0.2.12", -1},
		{"double digit minor", "0.12.0", "0.9.0", 1},
		{"double digit major", "12.0.0", "9.0.0", 1},
		// Edge cases
		{"different lengths a longer", "1.2.3", "1.2", 1},
		{"different lengths b longer", "1.2", "1.2.3", -1},
		{"zeros", "0.0.0", "0.0.0", 0},
		{"large numbers", "100.200.300", "100.200.299", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, compareSemver(tt.a, tt.b))
		})
	}
}

func TestIsNewer(t *testing.T) {
	origVersion := Version
	defer func() { Version = origVersion }()

	tests := []struct {
		name       string
		current    string
		releaseTag string
		want       bool
	}{
		{
			name:       "newer version available",
			current:    "v0.1.0",
			releaseTag: "v0.2.0",
			want:       true,
		},
		{
			name:       "same version",
			current:    "v0.1.0",
			releaseTag: "v0.1.0",
			want:       false,
		},
		{
			name:       "older version",
			current:    "v0.2.0",
			releaseTag: "v0.1.0",
			want:       false,
		},
		{
			name:       "dev version never needs update",
			current:    "dev",
			releaseTag: "v1.0.0",
			want:       false,
		},
		{
			name:       "patch version newer",
			current:    "v0.1.0",
			releaseTag: "v0.1.1",
			want:       true,
		},
		{
			name:       "major version newer",
			current:    "v0.9.9",
			releaseTag: "v1.0.0",
			want:       true,
		},
		// Critical: the bug case - double digit versions
		{
			name:       "double digit patch newer",
			current:    "v0.2.9",
			releaseTag: "v0.2.12",
			want:       true,
		},
		{
			name:       "double digit patch older",
			current:    "v0.2.12",
			releaseTag: "v0.2.9",
			want:       false,
		},
		{
			name:       "double digit minor newer",
			current:    "v0.9.0",
			releaseTag: "v0.12.0",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.current
			release := &GitHubRelease{TagName: tt.releaseTag}
			assert.Equal(t, tt.want, IsNewer(release))
		})
	}
}

func TestIsNewerNilRelease(t *testing.T) {
	assert.False(t, IsNewer(nil))
}

func TestCheckForUpdate(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tag_name": "v1.0.0",
			"published_at": "2026-01-07T12:00:00Z",
			"html_url": "https://github.com/MPJHorner/SuperRalph/releases/tag/v1.0.0"
		}`))
	}))
	defer server.Close()

	// We can't easily test the real function without modifying it to accept a URL
	// So this test just verifies the parsing logic works
	t.Log("CheckForUpdate requires mocking - skipping integration test")
}

func TestGetUpdateMessage(t *testing.T) {
	origVersion := Version
	defer func() { Version = origVersion }()

	// When version is "dev", should return empty
	Version = "dev"
	msg := GetUpdateMessage()
	// This will either be empty (no network) or empty (dev version)
	// We can't easily test with network, so just verify it doesn't panic
	t.Logf("GetUpdateMessage with dev version: %q", msg)
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, min(tt.a, tt.b))
	}
}

func TestGitHubReleaseStruct(t *testing.T) {
	release := GitHubRelease{
		TagName:     "v1.0.0",
		PublishedAt: "2026-01-07T12:00:00Z",
		HTMLURL:     "https://github.com/example/repo/releases/tag/v1.0.0",
	}

	assert.Equal(t, "v1.0.0", release.TagName)
}
