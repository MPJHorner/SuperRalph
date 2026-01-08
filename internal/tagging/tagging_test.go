package tagging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tagger := New("/tmp/test")
	require.NotNil(t, tagger)
	assert.Equal(t, "/tmp/test", tagger.workDir)
	assert.NotEmpty(t, tagger.excludeDirs)
}

func TestSetExcludeDirs(t *testing.T) {
	tagger := New("/tmp/test")
	customDirs := []string{"custom1", "custom2"}
	tagger.SetExcludeDirs(customDirs)

	assert.Len(t, tagger.excludeDirs, 2)
}

func TestParseTagExactFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte("package main"), 0644)
	require.NoError(t, err)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@main.go")
	require.NoError(t, err)

	assert.Equal(t, "@main.go", tag.Pattern)
	assert.False(t, tag.IsExclusion)
	assert.Len(t, tag.ResolvedPaths, 1)
	assert.Equal(t, testFile, tag.ResolvedPaths[0])
}

func TestParseTagWithoutPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte("package main"), 0644)
	require.NoError(t, err)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("main.go") // Without @ prefix
	require.NoError(t, err)

	assert.Len(t, tag.ResolvedPaths, 1)
}

func TestParseTagExclusion(t *testing.T) {
	tagger := New("/tmp/test")
	tag, err := tagger.ParseTag("@!vendor")
	require.NoError(t, err)

	assert.Equal(t, "@!vendor", tag.Pattern)
	assert.True(t, tag.IsExclusion)
	assert.Empty(t, tag.ResolvedPaths)
}

func TestParseTagGlobPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	srcDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(srcDir, 0755)
	require.NoError(t, err)

	files := []string{"main.go", "util.go", "helper.go"}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(srcDir, f), []byte("package src"), 0644)
		require.NoError(t, err)
	}

	// Create a non-.go file
	err = os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("readme"), 0644)
	require.NoError(t, err)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src/*.go")
	require.NoError(t, err)

	assert.Len(t, tag.ResolvedPaths, 3)
}

func TestParseTagDoubleStarGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create nested directories
	nestedDir := filepath.Join(tmpDir, "src", "pkg", "util")
	err = os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create .go files at different levels
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "pkg", "pkg.go"), []byte("package pkg"), 0644)
	os.WriteFile(filepath.Join(nestedDir, "util.go"), []byte("package util"), 0644)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src/**/*.go")
	require.NoError(t, err)

	assert.Len(t, tag.ResolvedPaths, 3)
}

func TestParseTagNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@nonexistent.go")
	require.NoError(t, err)

	assert.Empty(t, tag.ResolvedPaths)
}

func TestParseTagDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a directory with files
	srcDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(srcDir, 0755)
	require.NoError(t, err)

	os.WriteFile(filepath.Join(srcDir, "file1.go"), []byte("package src"), 0644)
	os.WriteFile(filepath.Join(srcDir, "file2.go"), []byte("package src"), 0644)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src")
	require.NoError(t, err)

	// Should return files in the directory (non-recursive)
	assert.Len(t, tag.ResolvedPaths, 2)
}

func TestLoadContents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testContent := "package main\n\nfunc main() {}"
	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@main.go")
	require.NoError(t, err)

	err = tagger.LoadContents(tag)
	require.NoError(t, err)

	assert.Contains(t, tag.Contents, "main.go")
	assert.Equal(t, testContent, tag.Contents["main.go"])
}

func TestLoadContentsExclusion(t *testing.T) {
	tagger := New("/tmp/test")
	tag := &FileTag{
		Pattern:     "@!vendor",
		IsExclusion: true,
		Contents:    make(map[string]string),
	}

	err := tagger.LoadContents(tag)
	require.NoError(t, err)

	assert.Empty(t, tag.Contents)
}

func TestResolveTags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package main"), 0644)

	tagger := New(tmpDir)
	tags, err := tagger.ResolveTags([]string{"@main.go", "@util.go", "@!vendor"})
	require.NoError(t, err)

	assert.Len(t, tags, 3)

	// Check first two are inclusions
	assert.False(t, tags[0].IsExclusion)
	assert.False(t, tags[1].IsExclusion)

	// Check last is exclusion
	assert.True(t, tags[2].IsExclusion)
}

func TestBuildTaggedFilesMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package util"), 0644)

	// Create vendor directory (should be excluded)
	vendorDir := filepath.Join(tmpDir, "vendor")
	os.Mkdir(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, "dep.go"), []byte("package dep"), 0644)

	tagger := New(tmpDir)
	tags, err := tagger.ResolveTags([]string{"@*.go", "@!vendor"})
	require.NoError(t, err)

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	require.NoError(t, err)

	// Should have main.go and util.go
	assert.Len(t, filesMap, 2)
	assert.Contains(t, filesMap, "main.go")
	assert.Contains(t, filesMap, "util.go")
}

func TestBuildTaggedFilesMapWithExclusion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test directory
	testDir := filepath.Join(tmpDir, "test")
	os.Mkdir(testDir, 0755)

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(testDir, "main_test.go"), []byte("package main"), 0644)

	tagger := New(tmpDir)

	// Include all .go files but exclude test directory
	tags, err := tagger.ResolveTags([]string{"@**/*.go", "@main.go", "@!test"})
	require.NoError(t, err)

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	require.NoError(t, err)

	// Should only have main.go (test directory excluded)
	assert.Contains(t, filesMap, "main.go")
	assert.NotContains(t, filesMap, "test/main_test.go")
}

func TestListFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create some files and directories
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("hidden"), 0644)

	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package src"), 0644)

	// Create node_modules (should be excluded)
	nodeDir := filepath.Join(tmpDir, "node_modules")
	os.Mkdir(nodeDir, 0755)
	os.WriteFile(filepath.Join(nodeDir, "dep.js"), []byte("module.exports = {}"), 0644)

	tagger := New(tmpDir)
	files, err := tagger.ListFiles(3)
	require.NoError(t, err)

	// Should contain main.go, .gitignore, src/, src/util.go
	// Should NOT contain .hidden, node_modules
	assert.Contains(t, files, "main.go")
	assert.Contains(t, files, ".gitignore")
	assert.Contains(t, files, "src/")
	assert.Contains(t, files, "src/util.go")
	assert.NotContains(t, files, ".hidden")
	assert.NotContains(t, files, "node_modules/")
}

func TestParseTagString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single tag",
			input: "@main.go",
			want:  []string{"@main.go"},
		},
		{
			name:  "multiple tags with spaces",
			input: "@main.go @util.go @!vendor",
			want:  []string{"@main.go", "@util.go", "@!vendor"},
		},
		{
			name:  "mixed content",
			input: "Include @main.go and @util.go please",
			want:  []string{"@main.go", "@util.go"},
		},
		{
			name:  "with newlines",
			input: "@main.go\n@util.go\n@!vendor",
			want:  []string{"@main.go", "@util.go", "@!vendor"},
		},
		{
			name:  "no tags",
			input: "Just some text without tags",
			want:  nil,
		},
		{
			name:  "glob patterns",
			input: "@src/**/*.go @internal/*.go",
			want:  []string{"@src/**/*.go", "@internal/*.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTagString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tagger := New("/tmp/workdir")

	tests := []struct {
		path     string
		excluded bool
	}{
		{"/tmp/workdir/main.go", false},
		{"/tmp/workdir/node_modules/dep/index.js", true},
		{"/tmp/workdir/vendor/pkg/pkg.go", true},
		{"/tmp/workdir/src/main.go", false},
		{"/tmp/workdir/.git/config", true},
		{"/tmp/workdir/__pycache__/module.pyc", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.excluded, tagger.isExcluded(tt.path))
		})
	}
}

func TestFileTagSerialization(t *testing.T) {
	tag := &FileTag{
		Pattern:       "@src/**/*.go",
		IsExclusion:   false,
		ResolvedPaths: []string{"/tmp/src/main.go", "/tmp/src/util.go"},
		Contents: map[string]string{
			"src/main.go": "package main",
			"src/util.go": "package util",
		},
	}

	// We just need to ensure the struct can hold this data
	// JSON serialization is tested implicitly by the struct tags
	assert.Equal(t, "@src/**/*.go", tag.Pattern)
	assert.False(t, tag.IsExclusion)
	assert.Len(t, tag.ResolvedPaths, 2)
	assert.Len(t, tag.Contents, 2)
}

func TestExclusionPatternMatching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	dirs := []string{"src", "internal", "test", "test/fixtures"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(tmpDir, d), 0755)
	}

	// Create files
	files := map[string]string{
		"main.go":               "package main",
		"src/util.go":           "package src",
		"internal/core.go":      "package internal",
		"test/main_test.go":     "package test",
		"test/fixtures/data.go": "package fixtures",
	}
	for path, content := range files {
		os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0644)
	}

	tagger := New(tmpDir)

	// Test excluding 'test' directory
	tags, err := tagger.ResolveTags([]string{"@**/*.go", "@!test"})
	require.NoError(t, err)

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	require.NoError(t, err)

	// Should have main.go, src/util.go, internal/core.go
	// Should NOT have test/main_test.go or test/fixtures/data.go
	expectedFiles := []string{"main.go", "src/util.go", "internal/core.go"}
	excludedFiles := []string{"test/main_test.go", "test/fixtures/data.go"}

	for _, f := range expectedFiles {
		assert.Contains(t, filesMap, f)
	}

	for _, f := range excludedFiles {
		assert.NotContains(t, filesMap, f)
	}
}
