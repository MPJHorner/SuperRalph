package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/prd"
)

// FeatureStatus represents the status of a feature in the list
type FeatureStatus string

const (
	FeatureStatusComplete FeatureStatus = "complete"
	FeatureStatusCurrent  FeatureStatus = "current"
	FeatureStatusPending  FeatureStatus = "pending"
	FeatureStatusBlocked  FeatureStatus = "blocked"
)

// FeatureListItem represents a single item in the feature list
type FeatureListItem struct {
	ID          string
	Description string
	Priority    prd.Priority
	Status      FeatureStatus
}

// FeatureList is a scrollable list of features
type FeatureList struct {
	Items            []FeatureListItem
	CurrentFeatureID string
	Width            int
	Height           int
	ScrollOffset     int
	Focused          bool

	// Styles
	titleStyle     lipgloss.Style
	itemStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	completeStyle  lipgloss.Style
	blockedStyle   lipgloss.Style
	priorityStyles map[prd.Priority]lipgloss.Style
}

// NewFeatureList creates a new feature list component
func NewFeatureList(width, height int) *FeatureList {
	return &FeatureList{
		Items:        []FeatureListItem{},
		Width:        width,
		Height:       height,
		ScrollOffset: 0,
		Focused:      false,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		selectedStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42")), // Green for current
		completeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Muted for complete
		blockedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red for blocked
		priorityStyles: map[prd.Priority]lipgloss.Style{
			prd.PriorityHigh: lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")), // Red
			prd.PriorityMedium: lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")), // Orange
			prd.PriorityLow: lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")), // Blue
		},
	}
}

// UpdateFromPRD updates the feature list from a PRD
func (fl *FeatureList) UpdateFromPRD(p *prd.PRD, currentFeatureID string) {
	fl.CurrentFeatureID = currentFeatureID
	fl.Items = make([]FeatureListItem, 0, len(p.Features))

	for _, f := range p.Features {
		status := FeatureStatusPending
		if f.Passes {
			status = FeatureStatusComplete
		} else if f.ID == currentFeatureID {
			status = FeatureStatusCurrent
		} else if !p.DependenciesMet(&f) {
			status = FeatureStatusBlocked
		}

		fl.Items = append(fl.Items, FeatureListItem{
			ID:          f.ID,
			Description: f.Description,
			Priority:    f.Priority,
			Status:      status,
		})
	}

	// Auto-scroll to show current feature
	fl.scrollToCurrent()
}

// scrollToCurrent scrolls to show the current feature
func (fl *FeatureList) scrollToCurrent() {
	for i, item := range fl.Items {
		if item.Status == FeatureStatusCurrent {
			// Ensure the current item is visible
			visibleLines := fl.Height - 3 // Account for borders and title
			if i < fl.ScrollOffset {
				fl.ScrollOffset = i
			} else if i >= fl.ScrollOffset+visibleLines {
				fl.ScrollOffset = i - visibleLines + 1
			}
			break
		}
	}
}

// ScrollUp scrolls the list up
func (fl *FeatureList) ScrollUp() {
	if fl.ScrollOffset > 0 {
		fl.ScrollOffset--
	}
}

// ScrollDown scrolls the list down
func (fl *FeatureList) ScrollDown() {
	maxOffset := len(fl.Items) - (fl.Height - 3)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if fl.ScrollOffset < maxOffset {
		fl.ScrollOffset++
	}
}

// Render renders the feature list
func (fl *FeatureList) Render() string {
	var b strings.Builder

	// Title
	title := fl.titleStyle.Render("Features")
	b.WriteString(title)
	b.WriteString("\n")

	if len(fl.Items) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No features"))
		return fl.wrapInBox(b.String())
	}

	// Calculate visible area
	visibleLines := fl.Height - 3 // Account for borders and title
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Show scroll indicator at top if needed
	if fl.ScrollOffset > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  ↑ more"))
		b.WriteString("\n")
		visibleLines--
	}

	// Render visible items
	endIdx := fl.ScrollOffset + visibleLines
	if endIdx > len(fl.Items) {
		endIdx = len(fl.Items)
	}

	for i := fl.ScrollOffset; i < endIdx; i++ {
		item := fl.Items[i]
		line := fl.renderItem(item)
		b.WriteString(line)
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Show scroll indicator at bottom if needed
	if endIdx < len(fl.Items) {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  ↓ more"))
	}

	return fl.wrapInBox(b.String())
}

// renderItem renders a single feature item
func (fl *FeatureList) renderItem(item FeatureListItem) string {
	// Status icon
	var icon string
	var style lipgloss.Style

	switch item.Status {
	case FeatureStatusComplete:
		icon = "✓"
		style = fl.completeStyle
	case FeatureStatusCurrent:
		icon = "→"
		style = fl.selectedStyle
	case FeatureStatusBlocked:
		icon = "✗"
		style = fl.blockedStyle
	default:
		icon = "○"
		style = fl.itemStyle
	}

	// Priority indicator (colored dot)
	priorityStyle, ok := fl.priorityStyles[item.Priority]
	if !ok {
		priorityStyle = fl.itemStyle
	}
	priorityDot := priorityStyle.Render("●")

	// Truncate description to fit
	maxDescLen := fl.Width - 15 // Account for icon, ID, padding
	if maxDescLen < 10 {
		maxDescLen = 10
	}
	desc := item.Description
	if len(desc) > maxDescLen {
		desc = desc[:maxDescLen-3] + "..."
	}

	// Format: icon ID description
	idStyle := style
	if item.Status == FeatureStatusComplete {
		idStyle = idStyle.Strikethrough(true)
	}

	line := fmt.Sprintf("%s %s %s %s",
		style.Render(icon),
		priorityDot,
		idStyle.Render(item.ID),
		style.Render(desc),
	)

	return line
}

// wrapInBox wraps content in a box
func (fl *FeatureList) wrapInBox(content string) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Width(fl.Width)

	return boxStyle.Render(content)
}

// SetFocused sets whether the list is focused
func (fl *FeatureList) SetFocused(focused bool) {
	fl.Focused = focused
}

// GetStats returns stats about the feature list
func (fl *FeatureList) GetStats() (complete, current, pending, blocked int) {
	for _, item := range fl.Items {
		switch item.Status {
		case FeatureStatusComplete:
			complete++
		case FeatureStatusCurrent:
			current++
		case FeatureStatusBlocked:
			blocked++
		default:
			pending++
		}
	}
	return
}
