package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    DiffLineType
	Content string
}

// DiffLineType represents the type of a diff line
type DiffLineType int

const (
	DiffLineContext  DiffLineType = iota // Unchanged line (starts with space)
	DiffLineAdded                        // Added line (starts with +)
	DiffLineRemoved                      // Removed line (starts with -)
	DiffLineHeader                       // Header line (@@ ... @@)
	DiffLineFilePath                     // File path line (--- or +++)
)

// FileDiff represents a diff for a single file
type FileDiff struct {
	FilePath     string
	OldContent   string
	NewContent   string
	Collapsed    bool // Whether the diff is collapsed in the UI
	Lines        []DiffLine
	AddedCount   int
	RemovedCount int
}

// DiffViewer renders file diffs with syntax highlighting
type DiffViewer struct {
	Width int

	// Styles
	addedStyle   lipgloss.Style
	removedStyle lipgloss.Style
	contextStyle lipgloss.Style
	headerStyle  lipgloss.Style
	pathStyle    lipgloss.Style
	statsStyle   lipgloss.Style
	borderStyle  lipgloss.Style
}

// NewDiffViewer creates a new diff viewer
func NewDiffViewer(width int) *DiffViewer {
	return &DiffViewer{
		Width: width,
		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")). // Green
			Background(lipgloss.Color("22")), // Dark green background
		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red
			Background(lipgloss.Color("52")),  // Dark red background
		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Muted gray
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // Cyan
			Bold(true),
		pathStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")). // Purple
			Bold(true),
		statsStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Muted
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
	}
}

// GenerateDiff creates a FileDiff from old and new content
func (d *DiffViewer) GenerateDiff(filePath, oldContent, newContent string) *FileDiff {
	diff := &FileDiff{
		FilePath:   filePath,
		OldContent: oldContent,
		NewContent: newContent,
		Collapsed:  false,
		Lines:      []DiffLine{},
	}

	// Generate unified diff lines
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	// Use simple line-by-line diff algorithm
	diff.Lines, diff.AddedCount, diff.RemovedCount = computeUnifiedDiff(oldLines, newLines)

	return diff
}

// splitLines splits content into lines, handling empty content
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}
	lines := strings.Split(content, "\n")
	// Remove trailing empty line if content ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// computeUnifiedDiff computes a unified diff between old and new lines
func computeUnifiedDiff(oldLines, newLines []string) ([]DiffLine, int, int) {
	var result []DiffLine
	var addedCount, removedCount int

	// Use Myers diff algorithm approximation via LCS
	lcs := longestCommonSubsequence(oldLines, newLines)

	// Build diff from LCS
	oi, ni, li := 0, 0, 0

	// Track where we are in terms of hunks
	var hunkLines []DiffLine
	var hunkStart int
	inHunk := false

	flushHunk := func() {
		if len(hunkLines) > 0 {
			// Add hunk header
			result = append(result, DiffLine{
				Type:    DiffLineHeader,
				Content: formatHunkHeader(hunkStart, len(hunkLines)),
			})
			result = append(result, hunkLines...)
			hunkLines = nil
		}
		inHunk = false
	}

	addToHunk := func(line DiffLine, lineNum int) {
		if !inHunk {
			inHunk = true
			hunkStart = lineNum
		}
		hunkLines = append(hunkLines, line)
	}

	for oi < len(oldLines) || ni < len(newLines) {
		if li < len(lcs) {
			// Check if current old line matches LCS
			if oi < len(oldLines) && oldLines[oi] == lcs[li] {
				// Check if current new line matches LCS
				if ni < len(newLines) && newLines[ni] == lcs[li] {
					// Context line - both match
					addToHunk(DiffLine{
						Type:    DiffLineContext,
						Content: " " + oldLines[oi],
					}, oi+1)
					oi++
					ni++
					li++
				} else {
					// New line added
					addToHunk(DiffLine{
						Type:    DiffLineAdded,
						Content: "+" + newLines[ni],
					}, ni+1)
					addedCount++
					ni++
				}
			} else {
				// Old line removed
				if oi < len(oldLines) {
					addToHunk(DiffLine{
						Type:    DiffLineRemoved,
						Content: "-" + oldLines[oi],
					}, oi+1)
					removedCount++
					oi++
				}
			}
		} else {
			// Past LCS - remaining lines are additions or removals
			if oi < len(oldLines) {
				addToHunk(DiffLine{
					Type:    DiffLineRemoved,
					Content: "-" + oldLines[oi],
				}, oi+1)
				removedCount++
				oi++
			}
			if ni < len(newLines) {
				addToHunk(DiffLine{
					Type:    DiffLineAdded,
					Content: "+" + newLines[ni],
				}, ni+1)
				addedCount++
				ni++
			}
		}
	}

	flushHunk()

	// If no changes, return empty
	if addedCount == 0 && removedCount == 0 {
		return nil, 0, 0
	}

	return result, addedCount, removedCount
}

// longestCommonSubsequence finds the LCS of two string slices
func longestCommonSubsequence(a, b []string) []string {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return nil
	}

	// DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = maxInt(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
	lcs := make([]string, dp[m][n])
	i, j, k := m, n, dp[m][n]-1
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[k] = a[i-1]
			i--
			j--
			k--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// formatHunkHeader creates a unified diff hunk header
func formatHunkHeader(start, count int) string {
	_ = start // unused but kept for potential future use
	_ = count // unused but kept for potential future use
	return "@@ ... @@"
}

// RenderDiff renders a FileDiff as a string
func (d *DiffViewer) RenderDiff(diff *FileDiff) string {
	if diff == nil || len(diff.Lines) == 0 {
		return ""
	}

	var b strings.Builder

	// Header with file path and stats
	header := d.pathStyle.Render("  " + diff.FilePath)
	stats := d.statsStyle.Render(formatStats(diff.AddedCount, diff.RemovedCount))
	b.WriteString(header + "  " + stats + "\n")

	if diff.Collapsed {
		b.WriteString(d.contextStyle.Render("  [collapsed - press enter to expand]") + "\n")
		return b.String()
	}

	// Calculate content width
	contentWidth := d.Width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Render each line with appropriate styling
	for _, line := range diff.Lines {
		content := line.Content
		// Truncate if too long
		if len(content) > contentWidth {
			content = content[:contentWidth-3] + "..."
		}

		switch line.Type {
		case DiffLineAdded:
			b.WriteString(d.addedStyle.Render(content))
		case DiffLineRemoved:
			b.WriteString(d.removedStyle.Render(content))
		case DiffLineContext:
			b.WriteString(d.contextStyle.Render(content))
		case DiffLineHeader:
			b.WriteString(d.headerStyle.Render(content))
		case DiffLineFilePath:
			b.WriteString(d.pathStyle.Render(content))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// RenderCompact renders a compact summary of the diff
func (d *DiffViewer) RenderCompact(diff *FileDiff) string {
	if diff == nil {
		return ""
	}

	path := d.pathStyle.Render(diff.FilePath)
	stats := formatStats(diff.AddedCount, diff.RemovedCount)

	return path + " " + stats
}

// formatStats formats the +/- statistics
func formatStats(added, removed int) string {
	var parts []string
	if added > 0 {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		parts = append(parts, addStyle.Render("+"+itoa(added)))
	}
	if removed > 0 {
		remStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		parts = append(parts, remStyle.Render("-"+itoa(removed)))
	}
	if len(parts) == 0 {
		return "(no changes)"
	}
	return strings.Join(parts, " ")
}

// maxInt returns the larger of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
