package tagging

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/samber/lo"
)

// FileTag represents a file tag with its resolved paths and contents
type FileTag struct {
	// Pattern is the original pattern (e.g., "@src/main.go", "@src/**/*.go", "@!vendor")
	Pattern string `json:"pattern"`

	// IsExclusion indicates if this is an exclusion pattern (starts with @!)
	IsExclusion bool `json:"is_exclusion"`

	// ResolvedPaths contains the absolute paths that match this pattern
	ResolvedPaths []string `json:"resolved_paths"`

	// Contents maps relative paths to their file contents (only for non-exclusion patterns)
	Contents map[string]string `json:"contents,omitempty"`
}

// Tagger handles file tagging operations
type Tagger struct {
	workDir     string
	excludeDirs []string // Directories to always exclude (e.g., .git, node_modules)
}

// New creates a new Tagger for the given working directory
func New(workDir string) *Tagger {
	return &Tagger{
		workDir: workDir,
		excludeDirs: []string{
			".git",
			"node_modules",
			"vendor",
			"__pycache__",
			".venv",
			"venv",
			"target",
			"build",
			"dist",
			".superralph",
		},
	}
}

// SetExcludeDirs sets the directories to always exclude
func (t *Tagger) SetExcludeDirs(dirs []string) {
	t.excludeDirs = dirs
}

// ParseTag parses a tag string and returns a FileTag
// Tags can be:
// - @filepath - exact file path
// - @glob/pattern/**/*.go - glob pattern
// - @!dirname - exclusion pattern
func (t *Tagger) ParseTag(tag string) (*FileTag, error) {
	if !strings.HasPrefix(tag, "@") {
		tag = "@" + tag
	}

	// Remove the @ prefix
	pattern := strings.TrimPrefix(tag, "@")

	fileTag := &FileTag{
		Pattern:  tag,
		Contents: make(map[string]string),
	}

	// Check for exclusion pattern
	if strings.HasPrefix(pattern, "!") {
		fileTag.IsExclusion = true
		fileTag.Pattern = "@!" + strings.TrimPrefix(pattern, "!")
		// Exclusion patterns don't resolve to files
		return fileTag, nil
	}

	// Resolve the pattern
	paths, err := t.resolvePattern(pattern)
	if err != nil {
		return nil, err
	}

	fileTag.ResolvedPaths = paths
	return fileTag, nil
}

// resolvePattern resolves a pattern to absolute file paths
func (t *Tagger) resolvePattern(pattern string) ([]string, error) {
	// Check if it's a glob pattern (contains *, ?, [, or **)
	isGlob := strings.ContainsAny(pattern, "*?[")

	if !isGlob {
		// Exact file path
		fullPath := pattern
		if !filepath.IsAbs(pattern) {
			fullPath = filepath.Join(t.workDir, pattern)
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return []string{}, nil // File doesn't exist, return empty
			}
			return nil, err
		}

		if info.IsDir() {
			// If it's a directory, return all files in it (non-recursive)
			return t.filesInDir(fullPath, false)
		}

		return []string{fullPath}, nil
	}

	// Glob pattern
	return t.resolveGlob(pattern)
}

// resolveGlob resolves a glob pattern to matching files
func (t *Tagger) resolveGlob(pattern string) ([]string, error) {
	// Make pattern absolute if it isn't
	fullPattern := pattern
	if !filepath.IsAbs(pattern) {
		fullPattern = filepath.Join(t.workDir, pattern)
	}

	// Use doublestar for ** support
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return nil, err
	}

	// Filter out directories and excluded paths
	var result []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		// Check if path contains excluded directory
		if t.isExcluded(match) {
			continue
		}

		result = append(result, match)
	}

	return result, nil
}

// filesInDir returns all files in a directory
func (t *Tagger) filesInDir(dir string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip excluded directories
			if info.IsDir() {
				if t.isExcludedDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			files = append(files, path)
			return nil
		})
		return files, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}

// isExcluded checks if a path should be excluded
func (t *Tagger) isExcluded(path string) bool {
	relPath, err := filepath.Rel(t.workDir, path)
	if err != nil {
		return false
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	return lo.SomeBy(parts, func(part string) bool {
		return t.isExcludedDir(part)
	})
}

// isExcludedDir checks if a directory name should be excluded
func (t *Tagger) isExcludedDir(name string) bool {
	return lo.Contains(t.excludeDirs, name)
}

// LoadContents loads the contents of all resolved paths into the FileTag
func (t *Tagger) LoadContents(tag *FileTag) error {
	if tag.IsExclusion {
		return nil // Exclusion patterns don't have contents
	}

	for _, path := range tag.ResolvedPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Store with relative path as key
		relPath, err := filepath.Rel(t.workDir, path)
		if err != nil {
			relPath = path
		}

		tag.Contents[relPath] = string(content)
	}

	return nil
}

// ResolveTags parses and resolves multiple tag strings
func (t *Tagger) ResolveTags(tags []string) ([]*FileTag, error) {
	var result []*FileTag

	for _, tag := range tags {
		fileTag, err := t.ParseTag(tag)
		if err != nil {
			return nil, err
		}
		result = append(result, fileTag)
	}

	return result, nil
}

// BuildTaggedFilesMap builds a map of relative paths to contents from multiple tags
// It respects exclusion patterns - files matching exclusions are not included
func (t *Tagger) BuildTaggedFilesMap(tags []*FileTag) (map[string]string, error) {
	result := make(map[string]string)
	var exclusions []string

	// First, collect all exclusion patterns
	for _, tag := range tags {
		if tag.IsExclusion {
			// Convert exclusion pattern to a directory/file prefix
			pattern := strings.TrimPrefix(tag.Pattern, "@!")
			exclusions = append(exclusions, pattern)
		}
	}

	// Then, process inclusion patterns
	for _, tag := range tags {
		if tag.IsExclusion {
			continue
		}

		// Load contents if not already loaded
		if len(tag.Contents) == 0 && len(tag.ResolvedPaths) > 0 {
			if err := t.LoadContents(tag); err != nil {
				return nil, err
			}
		}

		// Add to result, checking exclusions
		for relPath, content := range tag.Contents {
			if !t.isExcludedByPatterns(relPath, exclusions) {
				result[relPath] = content
			}
		}
	}

	return result, nil
}

// isExcludedByPatterns checks if a path matches any exclusion pattern
func (t *Tagger) isExcludedByPatterns(path string, exclusions []string) bool {
	sep := string(filepath.Separator)
	return lo.SomeBy(exclusions, func(excl string) bool {
		return strings.HasPrefix(path, excl) ||
			strings.Contains(path, sep+excl+sep) ||
			strings.HasSuffix(path, sep+excl)
	})
}

// ListFiles returns a list of files in the working directory for autocomplete
// Respects .gitignore if present, and always excludes default directories
func (t *Tagger) ListFiles(maxDepth int) ([]string, error) {
	var files []string

	err := filepath.Walk(t.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the work directory itself
		if path == t.workDir {
			return nil
		}

		relPath, err := filepath.Rel(t.workDir, path)
		if err != nil {
			return err
		}

		// Check depth
		depth := strings.Count(relPath, string(filepath.Separator)) + 1
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			if t.isExcludedDir(info.Name()) {
				return filepath.SkipDir
			}
			// Add directories with trailing slash
			files = append(files, relPath+"/")
			return nil
		}

		// Skip hidden files (except .gitignore)
		if strings.HasPrefix(info.Name(), ".") && info.Name() != ".gitignore" {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// ParseTagString parses a string containing multiple tags separated by spaces or newlines
// Tags are identified by the @ prefix
func ParseTagString(input string) []string {
	tags := lo.Filter(strings.Fields(input), func(part string, _ int) bool {
		return strings.HasPrefix(part, "@")
	})
	if len(tags) == 0 {
		return nil
	}
	return tags
}
