package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiffViewer(t *testing.T) {
	dv := NewDiffViewer(80)

	assert.NotNil(t, dv)
	assert.Equal(t, 80, dv.Width)
}

func TestDiffViewerGenerateDiff_EmptyToContent(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "", "line1\nline2\nline3")

	require.NotNil(t, diff)
	assert.Equal(t, "test.go", diff.FilePath)
	assert.Equal(t, "", diff.OldContent)
	assert.Equal(t, "line1\nline2\nline3", diff.NewContent)
	assert.Equal(t, 3, diff.AddedCount)
	assert.Equal(t, 0, diff.RemovedCount)
}

func TestDiffViewerGenerateDiff_ContentToEmpty(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "line1\nline2\nline3", "")

	require.NotNil(t, diff)
	assert.Equal(t, 0, diff.AddedCount)
	assert.Equal(t, 3, diff.RemovedCount)
}

func TestDiffViewerGenerateDiff_NoChanges(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "line1\nline2", "line1\nline2")

	require.NotNil(t, diff)
	assert.Equal(t, 0, diff.AddedCount)
	assert.Equal(t, 0, diff.RemovedCount)
	assert.Len(t, diff.Lines, 0) // No changes = no diff lines
}

func TestDiffViewerGenerateDiff_SingleLineAdded(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "line1\nline2", "line1\nline2\nline3")

	require.NotNil(t, diff)
	assert.Equal(t, 1, diff.AddedCount)
	assert.Equal(t, 0, diff.RemovedCount)
}

func TestDiffViewerGenerateDiff_SingleLineRemoved(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "line1\nline2\nline3", "line1\nline2")

	require.NotNil(t, diff)
	assert.Equal(t, 0, diff.AddedCount)
	assert.Equal(t, 1, diff.RemovedCount)
}

func TestDiffViewerGenerateDiff_LineModified(t *testing.T) {
	dv := NewDiffViewer(80)

	diff := dv.GenerateDiff("test.go", "line1\nold line\nline3", "line1\nnew line\nline3")

	require.NotNil(t, diff)
	assert.Equal(t, 1, diff.AddedCount)
	assert.Equal(t, 1, diff.RemovedCount)
}

func TestDiffViewerRenderDiff_Nil(t *testing.T) {
	dv := NewDiffViewer(80)

	result := dv.RenderDiff(nil)

	assert.Equal(t, "", result)
}

func TestDiffViewerRenderDiff_NoLines(t *testing.T) {
	dv := NewDiffViewer(80)
	diff := &FileDiff{
		FilePath: "test.go",
		Lines:    []DiffLine{},
	}

	result := dv.RenderDiff(diff)

	assert.Equal(t, "", result)
}

func TestDiffViewerRenderDiff_WithChanges(t *testing.T) {
	dv := NewDiffViewer(80)
	diff := &FileDiff{
		FilePath:     "test.go",
		AddedCount:   2,
		RemovedCount: 1,
		Lines: []DiffLine{
			{Type: DiffLineHeader, Content: "@@ ... @@"},
			{Type: DiffLineRemoved, Content: "-old line"},
			{Type: DiffLineAdded, Content: "+new line 1"},
			{Type: DiffLineAdded, Content: "+new line 2"},
		},
	}

	result := dv.RenderDiff(diff)

	// Should contain file path
	assert.Contains(t, result, "test.go")
	// Should contain stats
	assert.Contains(t, result, "+2")
	assert.Contains(t, result, "-1")
}

func TestDiffViewerRenderCompact(t *testing.T) {
	dv := NewDiffViewer(80)
	diff := &FileDiff{
		FilePath:     "internal/foo/bar.go",
		AddedCount:   5,
		RemovedCount: 2,
	}

	result := dv.RenderCompact(diff)

	assert.Contains(t, result, "internal/foo/bar.go")
	assert.Contains(t, result, "+5")
	assert.Contains(t, result, "-2")
}

func TestDiffViewerRenderCompact_Nil(t *testing.T) {
	dv := NewDiffViewer(80)

	result := dv.RenderCompact(nil)

	assert.Equal(t, "", result)
}

func TestDiffViewerRenderCompact_NoChanges(t *testing.T) {
	dv := NewDiffViewer(80)
	diff := &FileDiff{
		FilePath:     "test.go",
		AddedCount:   0,
		RemovedCount: 0,
	}

	result := dv.RenderCompact(diff)

	assert.Contains(t, result, "(no changes)")
}

func TestDiffLineTypes(t *testing.T) {
	tests := []struct {
		name     string
		lineType DiffLineType
		expected int
	}{
		{"Context", DiffLineContext, 0},
		{"Added", DiffLineAdded, 1},
		{"Removed", DiffLineRemoved, 2},
		{"Header", DiffLineHeader, 3},
		{"FilePath", DiffLineFilePath, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, DiffLineType(tt.expected), tt.lineType)
		})
	}
}

func TestFileDiffStruct(t *testing.T) {
	diff := FileDiff{
		FilePath:     "src/main.go",
		OldContent:   "old",
		NewContent:   "new",
		Collapsed:    true,
		Lines:        []DiffLine{{Type: DiffLineAdded, Content: "+test"}},
		AddedCount:   1,
		RemovedCount: 0,
	}

	assert.Equal(t, "src/main.go", diff.FilePath)
	assert.Equal(t, "old", diff.OldContent)
	assert.Equal(t, "new", diff.NewContent)
	assert.True(t, diff.Collapsed)
	assert.Len(t, diff.Lines, 1)
	assert.Equal(t, 1, diff.AddedCount)
	assert.Equal(t, 0, diff.RemovedCount)
}

func TestDiffViewerGenerateDiff_MultiLineChange(t *testing.T) {
	dv := NewDiffViewer(80)

	old := `package main

func hello() {
	fmt.Println("Hello")
}
`
	new := `package main

import "fmt"

func hello() {
	fmt.Println("Hello, World!")
}

func main() {
	hello()
}
`

	diff := dv.GenerateDiff("main.go", old, new)

	require.NotNil(t, diff)
	assert.Equal(t, "main.go", diff.FilePath)
	// Should have multiple additions (import, modified line, main func)
	assert.Greater(t, diff.AddedCount, 0)
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Empty", "", []string{}},
		{"SingleLine", "hello", []string{"hello"}},
		{"TwoLines", "line1\nline2", []string{"line1", "line2"}},
		{"TrailingNewline", "line1\nline2\n", []string{"line1", "line2"}},
		{"EmptyLines", "line1\n\nline2", []string{"line1", "", "line2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLongestCommonSubsequence(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{"BothEmpty", []string{}, []string{}, nil},
		{"FirstEmpty", []string{}, []string{"a", "b"}, nil},
		{"SecondEmpty", []string{"a", "b"}, []string{}, nil},
		{"Identical", []string{"a", "b", "c"}, []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"NoCommon", []string{"a", "b"}, []string{"c", "d"}, []string{}},
		{"PartialMatch", []string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
		{"Interleaved", []string{"a", "b", "c", "d"}, []string{"a", "c", "e"}, []string{"a", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := longestCommonSubsequence(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatStats(t *testing.T) {
	tests := []struct {
		name     string
		added    int
		removed  int
		contains []string
	}{
		{"NoChanges", 0, 0, []string{"(no changes)"}},
		{"OnlyAdded", 5, 0, []string{"+5"}},
		{"OnlyRemoved", 0, 3, []string{"-3"}},
		{"Both", 10, 4, []string{"+10", "-4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStats(tt.added, tt.removed)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestDiffViewerCollapsedDiff(t *testing.T) {
	dv := NewDiffViewer(80)
	diff := &FileDiff{
		FilePath:     "test.go",
		Collapsed:    true,
		AddedCount:   5,
		RemovedCount: 2,
		Lines: []DiffLine{
			{Type: DiffLineAdded, Content: "+line"},
		},
	}

	result := dv.RenderDiff(diff)

	// Should show collapsed indicator
	assert.Contains(t, result, "collapsed")
}

func TestDiffViewerRenderWithLongLines(t *testing.T) {
	dv := NewDiffViewer(50) // Small width
	diff := &FileDiff{
		FilePath:     "test.go",
		AddedCount:   1,
		RemovedCount: 0,
		Lines: []DiffLine{
			{Type: DiffLineAdded, Content: "+this is a very long line that should be truncated when rendered in the diff viewer"},
		},
	}

	result := dv.RenderDiff(diff)

	// Should contain truncation indicator
	assert.Contains(t, result, "...")
}

func TestMaxIntFunction(t *testing.T) {
	assert.Equal(t, 5, maxInt(3, 5))
	assert.Equal(t, 5, maxInt(5, 3))
	assert.Equal(t, 0, maxInt(0, 0))
	assert.Equal(t, 10, maxInt(10, -5))
}

func TestComputeUnifiedDiff(t *testing.T) {
	// Test basic diff computation
	old := []string{"line1", "line2", "line3"}
	new := []string{"line1", "modified", "line3", "line4"}

	lines, added, removed := computeUnifiedDiff(old, new)

	assert.Greater(t, len(lines), 0)
	assert.Equal(t, 2, added)   // "modified" and "line4"
	assert.Equal(t, 1, removed) // "line2"
}

func TestComputeUnifiedDiff_Empty(t *testing.T) {
	lines, added, removed := computeUnifiedDiff([]string{}, []string{})

	assert.Len(t, lines, 0)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, removed)
}

func TestFormatHunkHeader(t *testing.T) {
	result := formatHunkHeader(1, 10)
	assert.Contains(t, result, "@@")
}

func TestDiffViewerWidthUpdate(t *testing.T) {
	dv := NewDiffViewer(80)
	assert.Equal(t, 80, dv.Width)

	dv.Width = 120
	assert.Equal(t, 120, dv.Width)
}

func TestGenerateDiffPreservesFilePath(t *testing.T) {
	dv := NewDiffViewer(80)

	paths := []string{
		"simple.go",
		"path/to/file.go",
		"internal/tui/components/test.go",
		"./relative.go",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			diff := dv.GenerateDiff(path, "a", "b")
			assert.Equal(t, path, diff.FilePath)
		})
	}
}

func TestDiffLineContent(t *testing.T) {
	line := DiffLine{
		Type:    DiffLineAdded,
		Content: "+func main() {}",
	}

	assert.Equal(t, DiffLineAdded, line.Type)
	assert.True(t, strings.HasPrefix(line.Content, "+"))
}
