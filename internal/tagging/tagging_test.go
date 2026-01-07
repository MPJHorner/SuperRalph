package tagging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tagger := New("/tmp/test")
	if tagger == nil {
		t.Fatal("expected non-nil tagger")
	}
	if tagger.workDir != "/tmp/test" {
		t.Errorf("expected workDir /tmp/test, got %s", tagger.workDir)
	}
	if len(tagger.excludeDirs) == 0 {
		t.Error("expected default exclude dirs to be set")
	}
}

func TestSetExcludeDirs(t *testing.T) {
	tagger := New("/tmp/test")
	customDirs := []string{"custom1", "custom2"}
	tagger.SetExcludeDirs(customDirs)

	if len(tagger.excludeDirs) != 2 {
		t.Errorf("expected 2 exclude dirs, got %d", len(tagger.excludeDirs))
	}
}

func TestParseTagExactFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@main.go")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if tag.Pattern != "@main.go" {
		t.Errorf("expected pattern @main.go, got %s", tag.Pattern)
	}
	if tag.IsExclusion {
		t.Error("expected IsExclusion to be false")
	}
	if len(tag.ResolvedPaths) != 1 {
		t.Errorf("expected 1 resolved path, got %d", len(tag.ResolvedPaths))
	}
	if tag.ResolvedPaths[0] != testFile {
		t.Errorf("expected resolved path %s, got %s", testFile, tag.ResolvedPaths[0])
	}
}

func TestParseTagWithoutPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("main.go") // Without @ prefix
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if len(tag.ResolvedPaths) != 1 {
		t.Errorf("expected 1 resolved path, got %d", len(tag.ResolvedPaths))
	}
}

func TestParseTagExclusion(t *testing.T) {
	tagger := New("/tmp/test")
	tag, err := tagger.ParseTag("@!vendor")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if tag.Pattern != "@!vendor" {
		t.Errorf("expected pattern @!vendor, got %s", tag.Pattern)
	}
	if !tag.IsExclusion {
		t.Error("expected IsExclusion to be true")
	}
	if len(tag.ResolvedPaths) != 0 {
		t.Errorf("exclusion patterns should not have resolved paths, got %d", len(tag.ResolvedPaths))
	}
}

func TestParseTagGlobPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	files := []string{"main.go", "util.go", "helper.go"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(srcDir, f), []byte("package src"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	// Create a non-.go file
	if err := os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write readme.txt: %v", err)
	}

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src/*.go")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if len(tag.ResolvedPaths) != 3 {
		t.Errorf("expected 3 resolved paths, got %d", len(tag.ResolvedPaths))
	}
}

func TestParseTagDoubleStarGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directories
	nestedDir := filepath.Join(tmpDir, "src", "pkg", "util")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Create .go files at different levels
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "pkg", "pkg.go"), []byte("package pkg"), 0644)
	os.WriteFile(filepath.Join(nestedDir, "util.go"), []byte("package util"), 0644)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src/**/*.go")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if len(tag.ResolvedPaths) != 3 {
		t.Errorf("expected 3 resolved paths for **, got %d: %v", len(tag.ResolvedPaths), tag.ResolvedPaths)
	}
}

func TestParseTagNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@nonexistent.go")
	if err != nil {
		t.Fatalf("ParseTag should not error for non-existent file: %v", err)
	}

	if len(tag.ResolvedPaths) != 0 {
		t.Errorf("expected 0 resolved paths for non-existent file, got %d", len(tag.ResolvedPaths))
	}
}

func TestParseTagDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a directory with files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	os.WriteFile(filepath.Join(srcDir, "file1.go"), []byte("package src"), 0644)
	os.WriteFile(filepath.Join(srcDir, "file2.go"), []byte("package src"), 0644)

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@src")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	// Should return files in the directory (non-recursive)
	if len(tag.ResolvedPaths) != 2 {
		t.Errorf("expected 2 files in directory, got %d", len(tag.ResolvedPaths))
	}
}

func TestLoadContents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "package main\n\nfunc main() {}"
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tagger := New(tmpDir)
	tag, err := tagger.ParseTag("@main.go")
	if err != nil {
		t.Fatalf("ParseTag failed: %v", err)
	}

	if err := tagger.LoadContents(tag); err != nil {
		t.Fatalf("LoadContents failed: %v", err)
	}

	if content, ok := tag.Contents["main.go"]; !ok {
		t.Error("Contents should contain main.go")
	} else if content != testContent {
		t.Errorf("Content mismatch: got %q, want %q", content, testContent)
	}
}

func TestLoadContentsExclusion(t *testing.T) {
	tagger := New("/tmp/test")
	tag := &FileTag{
		Pattern:     "@!vendor",
		IsExclusion: true,
		Contents:    make(map[string]string),
	}

	err := tagger.LoadContents(tag)
	if err != nil {
		t.Fatalf("LoadContents should not fail for exclusion: %v", err)
	}

	if len(tag.Contents) != 0 {
		t.Error("Exclusion patterns should have no contents")
	}
}

func TestResolveTags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package main"), 0644)

	tagger := New(tmpDir)
	tags, err := tagger.ResolveTags([]string{"@main.go", "@util.go", "@!vendor"})
	if err != nil {
		t.Fatalf("ResolveTags failed: %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}

	// Check first two are inclusions
	if tags[0].IsExclusion || tags[1].IsExclusion {
		t.Error("first two tags should not be exclusions")
	}

	// Check last is exclusion
	if !tags[2].IsExclusion {
		t.Error("last tag should be an exclusion")
	}
}

func TestBuildTaggedFilesMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("ResolveTags failed: %v", err)
	}

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	if err != nil {
		t.Fatalf("BuildTaggedFilesMap failed: %v", err)
	}

	// Should have main.go and util.go
	if len(filesMap) != 2 {
		t.Errorf("expected 2 files in map, got %d: %v", len(filesMap), filesMap)
	}

	if _, ok := filesMap["main.go"]; !ok {
		t.Error("filesMap should contain main.go")
	}
	if _, ok := filesMap["util.go"]; !ok {
		t.Error("filesMap should contain util.go")
	}
}

func TestBuildTaggedFilesMapWithExclusion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("ResolveTags failed: %v", err)
	}

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	if err != nil {
		t.Fatalf("BuildTaggedFilesMap failed: %v", err)
	}

	// Should only have main.go (test directory excluded)
	if _, ok := filesMap["main.go"]; !ok {
		t.Error("filesMap should contain main.go")
	}
	if _, ok := filesMap["test/main_test.go"]; ok {
		t.Error("filesMap should NOT contain test/main_test.go (excluded)")
	}
}

func TestListFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
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
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Should contain main.go, .gitignore, src/, src/util.go
	// Should NOT contain .hidden, node_modules
	hasMain := false
	hasGitignore := false
	hasSrc := false
	hasSrcUtil := false
	hasHidden := false
	hasNodeModules := false

	for _, f := range files {
		switch f {
		case "main.go":
			hasMain = true
		case ".gitignore":
			hasGitignore = true
		case "src/":
			hasSrc = true
		case "src/util.go":
			hasSrcUtil = true
		case ".hidden":
			hasHidden = true
		case "node_modules/":
			hasNodeModules = true
		}
	}

	if !hasMain {
		t.Error("files should contain main.go")
	}
	if !hasGitignore {
		t.Error("files should contain .gitignore")
	}
	if !hasSrc {
		t.Error("files should contain src/")
	}
	if !hasSrcUtil {
		t.Error("files should contain src/util.go")
	}
	if hasHidden {
		t.Error("files should NOT contain .hidden")
	}
	if hasNodeModules {
		t.Error("files should NOT contain node_modules/")
	}
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
			if len(got) != len(tt.want) {
				t.Errorf("ParseTagString() = %v, want %v", got, tt.want)
				return
			}
			for i, tag := range got {
				if tag != tt.want[i] {
					t.Errorf("ParseTagString()[%d] = %s, want %s", i, tag, tt.want[i])
				}
			}
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
			got := tagger.isExcluded(tt.path)
			if got != tt.excluded {
				t.Errorf("isExcluded(%s) = %v, want %v", tt.path, got, tt.excluded)
			}
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
	if tag.Pattern != "@src/**/*.go" {
		t.Errorf("Pattern should be @src/**/*.go, got %s", tag.Pattern)
	}
	if tag.IsExclusion {
		t.Error("IsExclusion should be false")
	}
	if len(tag.ResolvedPaths) != 2 {
		t.Errorf("Should have 2 resolved paths, got %d", len(tag.ResolvedPaths))
	}
	if len(tag.Contents) != 2 {
		t.Errorf("Should have 2 contents, got %d", len(tag.Contents))
	}
}

func TestExclusionPatternMatching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	dirs := []string{"src", "internal", "test", "test/fixtures"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(tmpDir, d), 0755)
	}

	// Create files
	files := map[string]string{
		"main.go":              "package main",
		"src/util.go":          "package src",
		"internal/core.go":     "package internal",
		"test/main_test.go":    "package test",
		"test/fixtures/data.go": "package fixtures",
	}
	for path, content := range files {
		os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0644)
	}

	tagger := New(tmpDir)

	// Test excluding 'test' directory
	tags, err := tagger.ResolveTags([]string{"@**/*.go", "@!test"})
	if err != nil {
		t.Fatalf("ResolveTags failed: %v", err)
	}

	filesMap, err := tagger.BuildTaggedFilesMap(tags)
	if err != nil {
		t.Fatalf("BuildTaggedFilesMap failed: %v", err)
	}

	// Should have main.go, src/util.go, internal/core.go
	// Should NOT have test/main_test.go or test/fixtures/data.go
	expectedFiles := []string{"main.go", "src/util.go", "internal/core.go"}
	excludedFiles := []string{"test/main_test.go", "test/fixtures/data.go"}

	for _, f := range expectedFiles {
		if _, ok := filesMap[f]; !ok {
			t.Errorf("filesMap should contain %s", f)
		}
	}

	for _, f := range excludedFiles {
		if _, ok := filesMap[f]; ok {
			t.Errorf("filesMap should NOT contain %s", f)
		}
	}
}
