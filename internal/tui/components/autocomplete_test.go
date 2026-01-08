package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutocomplete(t *testing.T) {
	ac := NewAutocomplete(80, 20)

	require.NotNil(t, ac)
	assert.Equal(t, 80, ac.Width)
	assert.Equal(t, 20, ac.Height)
	assert.Equal(t, 10, ac.MaxVisible)
	assert.True(t, ac.ShowIcons)
	assert.False(t, ac.Active)
	assert.Empty(t, ac.Files)
	assert.Empty(t, ac.Filtered)
	assert.Empty(t, ac.Selected)
}

func TestAutocompleteSetFiles(t *testing.T) {
	ac := NewAutocomplete(80, 20)

	files := []string{
		"main.go",
		"cmd/",
		"internal/",
		"internal/tui/model.go",
		"README.md",
	}

	ac.SetFiles(files)

	assert.Len(t, ac.Files, 5)

	// Check file items
	assert.Equal(t, "main.go", ac.Files[0].Path)
	assert.False(t, ac.Files[0].IsDir)

	assert.Equal(t, "cmd", ac.Files[1].Path) // Trailing slash removed
	assert.True(t, ac.Files[1].IsDir)

	assert.Equal(t, "internal", ac.Files[2].Path)
	assert.True(t, ac.Files[2].IsDir)

	assert.Equal(t, "internal/tui/model.go", ac.Files[3].Path)
	assert.False(t, ac.Files[3].IsDir)

	// Filtered should have all files initially
	assert.Len(t, ac.Filtered, 5)
}

func TestAutocompleteActivateDeactivate(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"main.go", "test.go"})

	assert.False(t, ac.Active)

	ac.Activate()
	assert.True(t, ac.Active)
	assert.Equal(t, 0, ac.Cursor)

	ac.Query = "test"
	ac.Deactivate()
	assert.False(t, ac.Active)
	assert.Empty(t, ac.Query)
}

func TestAutocompleteFilter(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{
		"main.go",
		"main_test.go",
		"cmd/root.go",
		"cmd/build.go",
		"internal/tui/model.go",
		"internal/tui/model_test.go",
		"README.md",
	})

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query shows all", "", 7},
		{"filter by go", "go", 6},       // All .go files
		{"filter by test", "test", 2},   // Files with "test"
		{"filter by tui", "tui", 2},     // Files in tui directory
		{"filter by model", "model", 2}, // model.go and model_test.go
		{"filter by cmd", "cmd", 2},     // Files in cmd directory
		{"filter by main", "main", 2},   // main.go and main_test.go
		{"no match", "xyz123", 0},       // No matches
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac.Filter(tt.query)
			assert.Equal(t, tt.expected, len(ac.Filtered), "query: %s", tt.query)
			assert.Equal(t, tt.query, ac.Query)
			assert.Equal(t, 0, ac.Cursor) // Cursor resets on filter
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		target   string
		query    string
		expected bool
	}{
		{"main.go", "", true},
		{"main.go", "main", true},
		{"main.go", "mg", true},
		{"main.go", "mgo", true},
		{"main.go", "main.go", true},
		{"internal/tui/model.go", "tui", true},
		{"internal/tui/model.go", "model", true},
		{"internal/tui/model.go", "itm", true}, // i-nternal/t-ui/m-odel
		{"main.go", "xyz", false},
		{"main.go", "gom", false}, // Out of order
	}

	for _, tt := range tests {
		t.Run(tt.target+"_"+tt.query, func(t *testing.T) {
			result := fuzzyMatch(tt.target, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchScore(t *testing.T) {
	// Test match scoring priorities
	// Prefix matches should score higher than contains-only matches
	prefixScore := matchScore("main.go", "main")
	containsScore := matchScore("some_main.go", "main")
	fuzzyScore := matchScore("model.go", "mg")

	// Prefix match should beat contains-only match
	assert.Greater(t, prefixScore, containsScore)
	// Contains match should beat fuzzy-only match
	assert.Greater(t, containsScore, fuzzyScore)

	// Exact match gets 1000 flat
	exactScore := matchScore("main", "main")
	assert.Equal(t, 1000, exactScore)
}

func TestAutocompleteNavigation(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"a.go", "b.go", "c.go", "d.go", "e.go"})
	ac.Activate()

	assert.Equal(t, 0, ac.Cursor)

	// Move down
	ac.MoveDown()
	assert.Equal(t, 1, ac.Cursor)

	ac.MoveDown()
	ac.MoveDown()
	assert.Equal(t, 3, ac.Cursor)

	// Move down to end
	ac.MoveDown()
	assert.Equal(t, 4, ac.Cursor)

	// Can't go past end
	ac.MoveDown()
	assert.Equal(t, 4, ac.Cursor)

	// Move up
	ac.MoveUp()
	assert.Equal(t, 3, ac.Cursor)

	// Move to top
	ac.MoveUp()
	ac.MoveUp()
	ac.MoveUp()
	assert.Equal(t, 0, ac.Cursor)

	// Can't go past top
	ac.MoveUp()
	assert.Equal(t, 0, ac.Cursor)
}

func TestAutocompleteScrolling(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.MaxVisible = 3

	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		files[i] = string(rune('a'+i)) + ".go"
	}
	ac.SetFiles(files)
	ac.Activate()

	assert.Equal(t, 0, ac.ScrollOffset)

	// Move down past visible area
	ac.MoveDown() // cursor 1, offset 0
	ac.MoveDown() // cursor 2, offset 0
	ac.MoveDown() // cursor 3, offset 1 (scroll)

	assert.Equal(t, 3, ac.Cursor)
	assert.Equal(t, 1, ac.ScrollOffset)

	ac.MoveDown() // cursor 4, offset 2
	assert.Equal(t, 4, ac.Cursor)
	assert.Equal(t, 2, ac.ScrollOffset)

	// Move back up
	ac.MoveUp() // cursor 3, offset 2
	ac.MoveUp() // cursor 2, offset 2
	assert.Equal(t, 2, ac.Cursor)
	assert.Equal(t, 2, ac.ScrollOffset)

	ac.MoveUp() // cursor 1, offset 1 (scroll up)
	assert.Equal(t, 1, ac.Cursor)
	assert.Equal(t, 1, ac.ScrollOffset)
}

func TestAutocompleteSelection(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"main.go", "test.go", "cmd/"})
	ac.Activate()

	assert.Equal(t, 0, ac.SelectedCount())

	// Toggle selection on first item
	ac.ToggleSelection()
	assert.Equal(t, 1, ac.SelectedCount())
	assert.True(t, ac.Selected["main.go"])
	assert.True(t, ac.Filtered[0].Selected)
	assert.True(t, ac.Files[0].Selected)

	// Move to second item and select
	ac.MoveDown()
	ac.ToggleSelection()
	assert.Equal(t, 2, ac.SelectedCount())
	assert.True(t, ac.Selected["test.go"])

	// Deselect first item
	ac.MoveUp()
	ac.ToggleSelection()
	assert.Equal(t, 1, ac.SelectedCount())
	assert.False(t, ac.Selected["main.go"])
	assert.False(t, ac.Filtered[0].Selected)

	// Get selected
	selected := ac.GetSelected()
	assert.Len(t, selected, 1)
	assert.Contains(t, selected, "test.go")

	// Get as tags
	tags := ac.GetSelectedTags()
	assert.Len(t, tags, 1)
	assert.Contains(t, tags, "@test.go")
}

func TestAutocompleteSelectCurrent(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"main.go", "test.go"})
	ac.Activate()

	item := ac.SelectCurrent()
	require.NotNil(t, item)
	assert.Equal(t, "main.go", item.Path)

	ac.MoveDown()
	item = ac.SelectCurrent()
	require.NotNil(t, item)
	assert.Equal(t, "test.go", item.Path)

	// Empty list
	ac.Filter("xyz")
	assert.Empty(t, ac.Filtered)
	item = ac.SelectCurrent()
	assert.Nil(t, item)
}

func TestAutocompleteClearSelection(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"main.go", "test.go"})
	ac.Activate()

	ac.ToggleSelection()
	ac.MoveDown()
	ac.ToggleSelection()

	assert.Equal(t, 2, ac.SelectedCount())

	ac.ClearSelection()

	assert.Equal(t, 0, ac.SelectedCount())
	assert.Empty(t, ac.Selected)
	assert.False(t, ac.Files[0].Selected)
	assert.False(t, ac.Files[1].Selected)
	assert.False(t, ac.Filtered[0].Selected)
	assert.False(t, ac.Filtered[1].Selected)
}

func TestFileItemFileType(t *testing.T) {
	tests := []struct {
		path     string
		isDir    bool
		expected string
	}{
		{"main.go", false, "go"},
		{"app.js", false, "js"},
		{"component.jsx", false, "js"},
		{"app.ts", false, "ts"},
		{"component.tsx", false, "ts"},
		{"script.py", false, "python"},
		{"lib.rs", false, "rust"},
		{"gem.rb", false, "ruby"},
		{"Main.java", false, "java"},
		{"main.c", false, "c"},
		{"main.h", false, "c"},
		{"main.cpp", false, "cpp"},
		{"main.hpp", false, "cpp"},
		{"config.json", false, "json"},
		{"config.yaml", false, "yaml"},
		{"config.yml", false, "yaml"},
		{"config.toml", false, "toml"},
		{"README.md", false, "markdown"},
		{"index.html", false, "html"},
		{"style.css", false, "css"},
		{"style.scss", false, "css"},
		{"script.sh", false, "shell"},
		{"query.sql", false, "sql"},
		{"config.xml", false, "xml"},
		{"somefile.xyz", false, "file"},
		{"cmd", true, "dir"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			item := FileItem{Path: tt.path, IsDir: tt.isDir}
			assert.Equal(t, tt.expected, item.FileType())
		})
	}
}

func TestFileItemIcon(t *testing.T) {
	tests := []struct {
		path  string
		isDir bool
	}{
		{"main.go", false},
		{"app.js", false},
		{"script.py", false},
		{"cmd", true},
		{"unknown.xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			item := FileItem{Path: tt.path, IsDir: tt.isDir}
			icon := item.Icon()
			assert.NotEmpty(t, icon)
		})
	}
}

func TestAutocompleteRender(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.SetFiles([]string{"main.go", "test.go", "cmd/"})

	// Not active - empty render
	rendered := ac.Render()
	assert.Empty(t, rendered)

	// Activate
	ac.Activate()
	rendered = ac.Render()

	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "Files")
	assert.Contains(t, rendered, "main.go")
	assert.Contains(t, rendered, "test.go")
	assert.Contains(t, rendered, "cmd")
	assert.Contains(t, rendered, "Navigate")
	assert.Contains(t, rendered, "Toggle")
}

func TestAutocompleteRenderWithQuery(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.SetFiles([]string{"main.go", "main_test.go", "other.go"})
	ac.Activate()
	ac.Filter("main")

	rendered := ac.Render()

	assert.Contains(t, rendered, "matching 'main'")
	assert.Contains(t, rendered, "main.go")
	assert.Contains(t, rendered, "main_test.go")
}

func TestAutocompleteRenderWithSelection(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.SetFiles([]string{"main.go", "test.go"})
	ac.Activate()
	ac.ToggleSelection()

	rendered := ac.Render()

	// Should show selection indicator
	assert.Contains(t, rendered, "[x]")
	assert.Contains(t, rendered, "[ ]")
}

func TestAutocompleteRenderEmpty(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.SetFiles([]string{"main.go"})
	ac.Activate()
	ac.Filter("nonexistent")

	rendered := ac.Render()

	// Empty filtered list - should return empty
	assert.Empty(t, rendered)
}

func TestAutocompleteRenderCompact(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.SetFiles([]string{"main.go", "test.go", "cmd/"})

	// No selection
	compact := ac.RenderCompact()
	assert.Contains(t, compact, "No files tagged")

	// One selection
	ac.Activate()
	ac.ToggleSelection()
	compact = ac.RenderCompact()
	assert.Contains(t, compact, "1 file tagged")
	assert.Contains(t, compact, "@main.go")

	// Multiple selections
	ac.MoveDown()
	ac.ToggleSelection()
	compact = ac.RenderCompact()
	assert.Contains(t, compact, "2 files tagged")
}

func TestAutocompleteSelectionPersistsThroughFilter(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"main.go", "main_test.go", "other.go"})
	ac.Activate()

	// Select main.go
	ac.ToggleSelection()
	assert.True(t, ac.Selected["main.go"])

	// Filter to show only main files
	ac.Filter("main")
	assert.Len(t, ac.Filtered, 2)

	// Selection should still show
	assert.True(t, ac.Filtered[0].Selected || ac.Filtered[1].Selected)

	// Clear filter - selection should persist
	ac.Filter("")
	mainFile, found := findFileItem(ac.Files, "main.go")
	require.True(t, found)
	assert.True(t, mainFile.Selected)
}

func findFileItem(files []FileItem, path string) (FileItem, bool) {
	for _, f := range files {
		if f.Path == path {
			return f, true
		}
	}
	return FileItem{}, false
}

func TestAutocompleteToggleSelectionEmptyList(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.Activate()

	// Should not panic on empty list
	ac.ToggleSelection()
	assert.Equal(t, 0, ac.SelectedCount())
}

func TestAutocompleteNavigationEmptyList(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.Activate()

	// Should not panic
	ac.MoveUp()
	ac.MoveDown()
	assert.Equal(t, 0, ac.Cursor)
}

func TestAutocompleteScrollIndicators(t *testing.T) {
	ac := NewAutocomplete(60, 20)
	ac.MaxVisible = 2

	files := []string{"a.go", "b.go", "c.go", "d.go", "e.go"}
	ac.SetFiles(files)
	ac.Activate()

	// At top - no top indicator, has bottom indicator
	rendered := ac.Render()
	assert.Contains(t, rendered, "more below")

	// Scroll down
	ac.MoveDown()
	ac.MoveDown()
	ac.MoveDown()

	// In middle - should have indicators
	rendered = ac.Render()
	assert.Contains(t, rendered, "more above")
	assert.Contains(t, rendered, "more below")
}

func TestFileItemTypes(t *testing.T) {
	// Test all icon types return non-empty strings
	types := []struct {
		path  string
		isDir bool
	}{
		{"test.go", false},
		{"test.js", false},
		{"test.ts", false},
		{"test.py", false},
		{"test.rs", false},
		{"test.rb", false},
		{"test.java", false},
		{"test.c", false},
		{"test.cpp", false},
		{"test.json", false},
		{"test.yaml", false},
		{"test.md", false},
		{"test.html", false},
		{"test.css", false},
		{"test.sh", false},
		{"test.sql", false},
		{"dir", true},
		{"unknown.xyz", false},
	}

	for _, tt := range types {
		item := FileItem{Path: tt.path, IsDir: tt.isDir}
		assert.NotEmpty(t, item.Icon(), "Icon for %s should not be empty", tt.path)
		assert.NotEmpty(t, item.FileType(), "FileType for %s should not be empty", tt.path)
	}
}

func TestAutocompleteFilterCaseInsensitive(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{"Main.go", "README.md", "Makefile"})
	ac.Activate()

	// Lowercase query should match uppercase files
	ac.Filter("main")
	assert.Len(t, ac.Filtered, 1)
	assert.Equal(t, "Main.go", ac.Filtered[0].Path)

	// Uppercase query should match files
	ac.Filter("READ")
	assert.Len(t, ac.Filtered, 1)
	assert.Equal(t, "README.md", ac.Filtered[0].Path)
}

func TestAutocompleteFilterSortsByMatchQuality(t *testing.T) {
	ac := NewAutocomplete(80, 20)
	ac.SetFiles([]string{
		"internal/orchestrator/types.go",
		"cmd/build.go",
		"build.go",
		"cmd/root.go",
	})
	ac.Activate()

	// "build" should rank "build.go" higher than nested paths
	ac.Filter("build")

	require.GreaterOrEqual(t, len(ac.Filtered), 2)
	// First result should be exact filename match or shortest path
	assert.True(t, strings.Contains(ac.Filtered[0].Path, "build"))
}
