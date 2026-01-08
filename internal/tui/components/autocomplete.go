package components

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

// FileItem represents a file or directory in the autocomplete list
type FileItem struct {
	// Path is the relative path from the working directory
	Path string

	// IsDir indicates if this is a directory
	IsDir bool

	// Selected indicates if this item is selected for tagging
	Selected bool
}

// FileType returns the type of file based on extension
func (f FileItem) FileType() string {
	if f.IsDir {
		return "dir"
	}

	ext := strings.ToLower(filepath.Ext(f.Path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx", ".mjs":
		return "js"
	case ".ts", ".tsx":
		return "ts"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md", ".markdown":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".css", ".scss", ".sass":
		return "css"
	case ".sh", ".bash":
		return "shell"
	case ".sql":
		return "sql"
	case ".xml":
		return "xml"
	default:
		return "file"
	}
}

// Icon returns an icon for the file type
func (f FileItem) Icon() string {
	switch f.FileType() {
	case "dir":
		return "📁"
	case "go":
		return "🔷"
	case "js", "ts":
		return "🟨"
	case "python":
		return "🐍"
	case "rust":
		return "🦀"
	case "ruby":
		return "💎"
	case "java":
		return "☕"
	case "c", "cpp":
		return "⚙️"
	case "json", "yaml", "toml", "xml":
		return "📋"
	case "markdown":
		return "📝"
	case "html", "css":
		return "🌐"
	case "shell":
		return "🖥️"
	case "sql":
		return "🗃️"
	default:
		return "📄"
	}
}

// Autocomplete is a component for file/directory autocomplete
type Autocomplete struct {
	// All files available for selection
	Files []FileItem

	// Filtered files based on current query
	Filtered []FileItem

	// Current query string (after @)
	Query string

	// Currently highlighted index in filtered list
	Cursor int

	// Selected file paths (for multi-select)
	Selected map[string]bool

	// Whether autocomplete is active (showing dropdown)
	Active bool

	// Maximum number of items to show in dropdown
	MaxVisible int

	// Scroll offset for virtual scrolling
	ScrollOffset int

	// Width and height for rendering
	Width  int
	Height int

	// Style options
	ShowIcons bool
}

// NewAutocomplete creates a new Autocomplete component
func NewAutocomplete(width, height int) *Autocomplete {
	return &Autocomplete{
		Files:      make([]FileItem, 0),
		Filtered:   make([]FileItem, 0),
		Selected:   make(map[string]bool),
		MaxVisible: 10,
		Width:      width,
		Height:     height,
		ShowIcons:  true,
	}
}

// SetFiles sets the available files for autocomplete
func (a *Autocomplete) SetFiles(files []string) {
	a.Files = make([]FileItem, len(files))
	for i, path := range files {
		isDir := strings.HasSuffix(path, "/")
		a.Files[i] = FileItem{
			Path:     strings.TrimSuffix(path, "/"),
			IsDir:    isDir,
			Selected: a.Selected[strings.TrimSuffix(path, "/")],
		}
	}
	a.Filter("")
}

// Activate shows the autocomplete dropdown
func (a *Autocomplete) Activate() {
	a.Active = true
	a.Cursor = 0
	a.ScrollOffset = 0
	a.Filter(a.Query)
}

// Deactivate hides the autocomplete dropdown
func (a *Autocomplete) Deactivate() {
	a.Active = false
	a.Query = ""
}

// Filter filters the file list based on the query
func (a *Autocomplete) Filter(query string) {
	a.Query = query
	a.Cursor = 0
	a.ScrollOffset = 0

	if query == "" {
		// Show all files when no query
		a.Filtered = make([]FileItem, len(a.Files))
		copy(a.Filtered, a.Files)
	} else {
		// Fuzzy match
		a.Filtered = lo.Filter(a.Files, func(f FileItem, _ int) bool {
			return fuzzyMatch(strings.ToLower(f.Path), strings.ToLower(query))
		})

		// Sort by match quality (shorter paths that match are better)
		sort.Slice(a.Filtered, func(i, j int) bool {
			iScore := matchScore(a.Filtered[i].Path, query)
			jScore := matchScore(a.Filtered[j].Path, query)
			return iScore > jScore
		})
	}

	// Update selection state
	for i := range a.Filtered {
		a.Filtered[i].Selected = a.Selected[a.Filtered[i].Path]
	}
}

// fuzzyMatch checks if query characters appear in order in the target
func fuzzyMatch(target, query string) bool {
	if query == "" {
		return true
	}

	queryIdx := 0
	for _, c := range target {
		if queryIdx < len(query) && byte(c) == query[queryIdx] {
			queryIdx++
		}
	}
	return queryIdx == len(query)
}

// matchScore returns a score for how well the query matches the target
// Higher score = better match
func matchScore(target, query string) int {
	target = strings.ToLower(target)
	query = strings.ToLower(query)

	score := 0

	// Exact match is best
	if target == query {
		return 1000
	}

	// Starts with query is very good
	if strings.HasPrefix(target, query) {
		score += 500
	}

	// Contains query as substring is good
	if strings.Contains(target, query) {
		score += 300
	}

	// Filename (not path) starts with query is good
	base := filepath.Base(target)
	if strings.HasPrefix(base, query) {
		score += 200
	}

	// Shorter paths score higher (more specific)
	score += 100 - len(target)

	// Consecutive character matches score higher
	consecutive := 0
	maxConsecutive := 0
	queryIdx := 0
	for _, c := range target {
		if queryIdx < len(query) && byte(c) == query[queryIdx] {
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
			queryIdx++
		} else {
			consecutive = 0
		}
	}
	score += maxConsecutive * 10

	return score
}

// MoveUp moves the cursor up
func (a *Autocomplete) MoveUp() {
	if a.Cursor > 0 {
		a.Cursor--
		// Adjust scroll if cursor goes above visible area
		if a.Cursor < a.ScrollOffset {
			a.ScrollOffset = a.Cursor
		}
	}
}

// MoveDown moves the cursor down
func (a *Autocomplete) MoveDown() {
	if a.Cursor < len(a.Filtered)-1 {
		a.Cursor++
		// Adjust scroll if cursor goes below visible area
		if a.Cursor >= a.ScrollOffset+a.MaxVisible {
			a.ScrollOffset = a.Cursor - a.MaxVisible + 1
		}
	}
}

// ToggleSelection toggles selection of the current item
func (a *Autocomplete) ToggleSelection() {
	if len(a.Filtered) == 0 || a.Cursor >= len(a.Filtered) {
		return
	}

	path := a.Filtered[a.Cursor].Path
	if a.Selected[path] {
		delete(a.Selected, path)
		a.Filtered[a.Cursor].Selected = false
	} else {
		a.Selected[path] = true
		a.Filtered[a.Cursor].Selected = true
	}

	// Also update in main Files list
	for i := range a.Files {
		if a.Files[i].Path == path {
			a.Files[i].Selected = a.Selected[path]
			break
		}
	}
}

// SelectCurrent selects the current item and returns it (single-select mode)
func (a *Autocomplete) SelectCurrent() *FileItem {
	if len(a.Filtered) == 0 || a.Cursor >= len(a.Filtered) {
		return nil
	}
	return &a.Filtered[a.Cursor]
}

// GetSelected returns all selected file paths
func (a *Autocomplete) GetSelected() []string {
	return lo.Keys(a.Selected)
}

// GetSelectedTags returns selected paths formatted as @ tags
func (a *Autocomplete) GetSelectedTags() []string {
	paths := a.GetSelected()
	return lo.Map(paths, func(p string, _ int) string {
		return "@" + p
	})
}

// ClearSelection clears all selections
func (a *Autocomplete) ClearSelection() {
	a.Selected = make(map[string]bool)
	for i := range a.Files {
		a.Files[i].Selected = false
	}
	for i := range a.Filtered {
		a.Filtered[i].Selected = false
	}
}

// SelectedCount returns the number of selected items
func (a *Autocomplete) SelectedCount() int {
	return len(a.Selected)
}

// Render renders the autocomplete component
func (a *Autocomplete) Render() string {
	if !a.Active || len(a.Filtered) == 0 {
		return ""
	}

	var lines []string

	// Title with selection count
	title := "Files"
	if a.SelectedCount() > 0 {
		title = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render(strings.Repeat("*", a.SelectedCount()) + " ") + title
	}
	if a.Query != "" {
		title += " matching '" + a.Query + "'"
	}
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))
	lines = append(lines, titleStyle.Render(title))

	// Determine visible range
	start := a.ScrollOffset
	end := start + a.MaxVisible
	if end > len(a.Filtered) {
		end = len(a.Filtered)
	}

	// Show scroll indicator at top if needed
	if start > 0 {
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("  ... " + strings.Repeat(" ", 20) + "(" + string(rune('0'+start)) + " more above)")
		lines = append(lines, scrollIndicator)
	}

	// File items
	for i := start; i < end; i++ {
		item := a.Filtered[i]
		line := a.renderItem(item, i == a.Cursor)
		lines = append(lines, line)
	}

	// Show scroll indicator at bottom if needed
	remaining := len(a.Filtered) - end
	if remaining > 0 {
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("  ... " + strings.Repeat(" ", 20) + "(" + string(rune('0'+remaining)) + " more below)")
		lines = append(lines, scrollIndicator)
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	lines = append(lines, helpStyle.Render("  [↑/↓] Navigate  [Space] Toggle  [Enter] Confirm  [Esc] Cancel"))

	// Wrap in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Width(a.Width)

	return boxStyle.Render(strings.Join(lines, "\n"))
}

// renderItem renders a single file item
func (a *Autocomplete) renderItem(item FileItem, highlighted bool) string {
	var parts []string

	// Selection indicator
	if item.Selected {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("[x]"))
	} else {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("[ ]"))
	}

	// Icon
	if a.ShowIcons {
		parts = append(parts, item.Icon())
	}

	// Path
	pathStyle := lipgloss.NewStyle()
	if highlighted {
		pathStyle = pathStyle.
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("255")).
			Bold(true)
	} else if item.IsDir {
		pathStyle = pathStyle.Foreground(lipgloss.Color("39")) // Cyan for directories
	} else {
		pathStyle = pathStyle.Foreground(lipgloss.Color("255"))
	}

	path := item.Path
	if item.IsDir {
		path += "/"
	}
	parts = append(parts, pathStyle.Render(path))

	return strings.Join(parts, " ")
}

// RenderCompact renders a compact summary of selections
func (a *Autocomplete) RenderCompact() string {
	if a.SelectedCount() == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("No files tagged. Type @ to add files.")
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	count := a.SelectedCount()
	if count == 1 {
		return style.Render("1 file tagged: @" + a.GetSelected()[0])
	}

	return style.Render(fmt.Sprintf("%d files tagged", count))
}
