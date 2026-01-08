package components

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/prd"
)

// FeatureItemStatus represents the status of a feature
type FeatureItemStatus int

const (
	FeatureItemStatusPending FeatureItemStatus = iota
	FeatureItemStatusInProgress
	FeatureItemStatusDone
	FeatureItemStatusBlocked
)

// FeatureItem implements list.Item for the bubbles/list component
type FeatureItem struct {
	feature *prd.Feature
	status  FeatureItemStatus
}

// FilterValue returns the value used for filtering
func (f FeatureItem) FilterValue() string {
	return f.feature.ID + " " + f.feature.Description
}

// Title returns the feature ID
func (f FeatureItem) Title() string {
	return f.feature.ID
}

// Description returns the feature description
func (f FeatureItem) Description() string {
	return f.feature.Description
}

// Status returns the status
func (f FeatureItem) Status() FeatureItemStatus {
	return f.status
}

// Feature returns the underlying feature
func (f FeatureItem) Feature() *prd.Feature {
	return f.feature
}

// StatusIcon returns the icon for the current status
func (f FeatureItem) StatusIcon() string {
	switch f.status {
	case FeatureItemStatusDone:
		return "✓"
	case FeatureItemStatusInProgress:
		return "◐"
	case FeatureItemStatusBlocked:
		return "✗"
	default:
		return "○"
	}
}

// PriorityIcon returns the colored priority indicator
func (f FeatureItem) PriorityIcon() string {
	switch f.feature.Priority {
	case prd.PriorityHigh:
		return "●" // Will be styled red
	case prd.PriorityMedium:
		return "●" // Will be styled orange
	case prd.PriorityLow:
		return "●" // Will be styled blue
	default:
		return "●"
	}
}

// FeatureDelegate is a custom delegate for rendering feature items
type FeatureDelegate struct {
	Styles         FeatureDelegateStyles
	ShowPriority   bool
	ShowCategory   bool
	CurrentFeature string // ID of the feature currently being worked on
}

// FeatureDelegateStyles contains styles for the delegate
type FeatureDelegateStyles struct {
	NormalTitle    lipgloss.Style
	NormalDesc     lipgloss.Style
	SelectedTitle  lipgloss.Style
	SelectedDesc   lipgloss.Style
	DoneTitle      lipgloss.Style
	DoneDesc       lipgloss.Style
	InProgressIcon lipgloss.Style
	DoneIcon       lipgloss.Style
	PendingIcon    lipgloss.Style
	BlockedIcon    lipgloss.Style
	HighPriority   lipgloss.Style
	MediumPriority lipgloss.Style
	LowPriority    lipgloss.Style
}

// NewFeatureDelegate creates a new delegate with default styles
func NewFeatureDelegate() FeatureDelegate {
	return FeatureDelegate{
		Styles: FeatureDelegateStyles{
			NormalTitle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")),
			NormalDesc: lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")),
			SelectedTitle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("99")).
				Bold(true),
			SelectedDesc: lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("99")),
			DoneTitle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Strikethrough(true),
			DoneDesc: lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")),
			InProgressIcon: lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")), // Green
			DoneIcon: lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")), // Gray
			PendingIcon: lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")), // Light gray
			BlockedIcon: lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")), // Red
			HighPriority: lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")), // Red
			MediumPriority: lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")), // Orange
			LowPriority: lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")), // Blue
		},
		ShowPriority: true,
		ShowCategory: false,
	}
}

// Height returns the height of each item
func (d FeatureDelegate) Height() int {
	return 2
}

// Spacing returns the spacing between items
func (d FeatureDelegate) Spacing() int {
	return 0
}

// Update handles item updates
func (d FeatureDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a feature item
func (d FeatureDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	fi, ok := item.(FeatureItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Build the status icon
	var iconStyle lipgloss.Style
	switch fi.status {
	case FeatureItemStatusDone:
		iconStyle = d.Styles.DoneIcon
	case FeatureItemStatusInProgress:
		iconStyle = d.Styles.InProgressIcon
	case FeatureItemStatusBlocked:
		iconStyle = d.Styles.BlockedIcon
	default:
		iconStyle = d.Styles.PendingIcon
	}
	icon := iconStyle.Render(fi.StatusIcon())

	// Priority icon
	var priorityIcon string
	if d.ShowPriority {
		var priorityStyle lipgloss.Style
		switch fi.feature.Priority {
		case prd.PriorityHigh:
			priorityStyle = d.Styles.HighPriority
		case prd.PriorityMedium:
			priorityStyle = d.Styles.MediumPriority
		case prd.PriorityLow:
			priorityStyle = d.Styles.LowPriority
		}
		priorityIcon = priorityStyle.Render("●") + " "
	}

	// Title and description styles based on selection and status
	var titleStyle, descStyle lipgloss.Style
	if isSelected {
		titleStyle = d.Styles.SelectedTitle
		descStyle = d.Styles.SelectedDesc
	} else if fi.status == FeatureItemStatusDone {
		titleStyle = d.Styles.DoneTitle
		descStyle = d.Styles.DoneDesc
	} else {
		titleStyle = d.Styles.NormalTitle
		descStyle = d.Styles.NormalDesc
	}

	// Calculate available width
	width := m.Width()
	if width <= 0 {
		width = 80
	}
	contentWidth := width - 8 // Account for icon, padding

	// Build title line
	titleLine := fmt.Sprintf("%s %s%s", icon, priorityIcon, titleStyle.Render(fi.Title()))

	// Build description line with truncation
	desc := fi.Description()
	if len(desc) > contentWidth-4 {
		desc = desc[:contentWidth-7] + "..."
	}
	descLine := "    " + descStyle.Render(desc)

	// Write output
	fmt.Fprint(w, titleLine+"\n"+descLine)
}

// InteractiveFeatureList is a full-featured interactive list component
type InteractiveFeatureList struct {
	List        list.Model
	Delegate    FeatureDelegate
	PRD         *prd.PRD
	Width       int
	Height      int
	Filtering   bool
	FilterInput textinput.Model

	// Current feature ID (the one being worked on)
	CurrentFeatureID string

	// Detail view state
	ShowDetail    bool
	DetailFeature *prd.Feature

	// Styles
	titleStyle      lipgloss.Style
	groupTitleStyle lipgloss.Style
	detailStyle     lipgloss.Style
}

// NewInteractiveFeatureList creates a new interactive feature list
func NewInteractiveFeatureList(width, height int) *InteractiveFeatureList {
	delegate := NewFeatureDelegate()
	items := []list.Item{}

	l := list.New(items, delegate, width, height)
	l.Title = "Features"
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))
	l.DisableQuitKeybindings()

	// Create filter input
	ti := textinput.New()
	ti.Placeholder = "Search features..."
	ti.CharLimit = 100
	ti.Width = 40

	return &InteractiveFeatureList{
		List:            l,
		Delegate:        delegate,
		Width:           width,
		Height:          height,
		FilterInput:     ti,
		titleStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
		groupTitleStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).MarginTop(1),
		detailStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2),
	}
}

// SetPRD updates the list with features from a PRD
func (ifl *InteractiveFeatureList) SetPRD(p *prd.PRD, currentFeatureID string) {
	ifl.PRD = p
	ifl.CurrentFeatureID = currentFeatureID
	ifl.Delegate.CurrentFeature = currentFeatureID

	// Convert features to items with proper status
	var items []list.Item
	for i := range p.Features {
		f := &p.Features[i]
		status := FeatureItemStatusPending

		if f.Passes {
			status = FeatureItemStatusDone
		} else if f.ID == currentFeatureID {
			status = FeatureItemStatusInProgress
		} else if !p.DependenciesMet(f) {
			status = FeatureItemStatusBlocked
		}

		items = append(items, FeatureItem{
			feature: f,
			status:  status,
		})
	}

	// Sort items: In Progress first, then Pending, then Done (dimmed at bottom)
	sort.SliceStable(items, func(i, j int) bool {
		fi := items[i].(FeatureItem)
		fj := items[j].(FeatureItem)

		// Sort order: InProgress < Blocked < Pending < Done
		statusOrder := map[FeatureItemStatus]int{
			FeatureItemStatusInProgress: 0,
			FeatureItemStatusBlocked:    1,
			FeatureItemStatusPending:    2,
			FeatureItemStatusDone:       3,
		}

		if statusOrder[fi.status] != statusOrder[fj.status] {
			return statusOrder[fi.status] < statusOrder[fj.status]
		}

		// Within same status, sort by priority
		priorityOrder := map[prd.Priority]int{
			prd.PriorityHigh:   0,
			prd.PriorityMedium: 1,
			prd.PriorityLow:    2,
		}

		return priorityOrder[fi.feature.Priority] < priorityOrder[fj.feature.Priority]
	})

	ifl.List.SetItems(items)
}

// Resize updates the dimensions
func (ifl *InteractiveFeatureList) Resize(width, height int) {
	ifl.Width = width
	ifl.Height = height
	// Reserve space for title and help
	listHeight := height - 4
	if listHeight < 5 {
		listHeight = 5
	}
	ifl.List.SetSize(width-4, listHeight)
}

// Update handles messages
func (ifl *InteractiveFeatureList) Update(msg tea.Msg) (*InteractiveFeatureList, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Handle detail view
		if ifl.ShowDetail {
			if keyMsg.String() == "esc" || keyMsg.String() == "enter" || keyMsg.String() == "q" {
				ifl.ShowDetail = false
				ifl.DetailFeature = nil
				return ifl, nil
			}
			return ifl, nil
		}

		// Handle filtering
		if ifl.Filtering {
			switch keyMsg.String() {
			case "esc":
				ifl.Filtering = false
				ifl.FilterInput.SetValue("")
				ifl.FilterInput.Blur()
				return ifl, nil
			case "enter":
				ifl.Filtering = false
				ifl.FilterInput.Blur()
				return ifl, nil
			default:
				var cmd tea.Cmd
				ifl.FilterInput, cmd = ifl.FilterInput.Update(msg)
				// Update list filtering
				ifl.applyFilter(ifl.FilterInput.Value())
				return ifl, cmd
			}
		}

		// Normal mode key handling
		switch keyMsg.String() {
		case "/":
			ifl.Filtering = true
			ifl.FilterInput.Focus()
			return ifl, textinput.Blink
		case "enter":
			// Show detail view for selected item
			if item := ifl.SelectedItem(); item != nil {
				ifl.ShowDetail = true
				ifl.DetailFeature = item.Feature()
			}
			return ifl, nil
		case "j", "down":
			ifl.List.CursorDown()
			return ifl, nil
		case "k", "up":
			ifl.List.CursorUp()
			return ifl, nil
		case "g":
			// Go to top
			ifl.List.Select(0)
			return ifl, nil
		case "G":
			// Go to bottom
			items := ifl.List.Items()
			if len(items) > 0 {
				ifl.List.Select(len(items) - 1)
			}
			return ifl, nil
		}
	}

	// Pass to list
	var cmd tea.Cmd
	ifl.List, cmd = ifl.List.Update(msg)
	cmds = append(cmds, cmd)

	return ifl, tea.Batch(cmds...)
}

// applyFilter filters the list based on query
func (ifl *InteractiveFeatureList) applyFilter(query string) {
	if ifl.PRD == nil {
		return
	}

	if query == "" {
		// Reset to full list
		ifl.SetPRD(ifl.PRD, ifl.CurrentFeatureID)
		return
	}

	query = strings.ToLower(query)

	// Filter features
	var items []list.Item
	for i := range ifl.PRD.Features {
		f := &ifl.PRD.Features[i]

		// Fuzzy match on ID and description
		matchTarget := strings.ToLower(f.ID + " " + f.Description)
		if !fuzzyMatchFeature(matchTarget, query) {
			continue
		}

		status := FeatureItemStatusPending
		if f.Passes {
			status = FeatureItemStatusDone
		} else if f.ID == ifl.CurrentFeatureID {
			status = FeatureItemStatusInProgress
		} else if !ifl.PRD.DependenciesMet(f) {
			status = FeatureItemStatusBlocked
		}

		items = append(items, FeatureItem{
			feature: f,
			status:  status,
		})
	}

	ifl.List.SetItems(items)
}

// fuzzyMatchFeature checks if query characters appear in order in target
func fuzzyMatchFeature(target, query string) bool {
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

// SelectedItem returns the currently selected feature item
func (ifl *InteractiveFeatureList) SelectedItem() *FeatureItem {
	item := ifl.List.SelectedItem()
	if item == nil {
		return nil
	}
	fi, ok := item.(FeatureItem)
	if !ok {
		return nil
	}
	return &fi
}

// View renders the component
func (ifl *InteractiveFeatureList) View() string {
	var b strings.Builder

	// If showing detail view, render that instead
	if ifl.ShowDetail && ifl.DetailFeature != nil {
		return ifl.renderDetailView()
	}

	// Filter input if active
	if ifl.Filtering {
		filterBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1).
			Render("🔍 " + ifl.FilterInput.View())
		b.WriteString(filterBox)
		b.WriteString("\n")
	}

	// Main list
	b.WriteString(ifl.List.View())

	// Help footer
	b.WriteString("\n")
	b.WriteString(ifl.renderHelp())

	return b.String()
}

// renderDetailView renders the feature detail panel
func (ifl *InteractiveFeatureList) renderDetailView() string {
	f := ifl.DetailFeature
	if f == nil {
		return ""
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	b.WriteString(titleStyle.Render(f.ID))
	b.WriteString("\n\n")

	// Status
	statusStyle := lipgloss.NewStyle().Bold(true)
	if f.Passes {
		statusStyle = statusStyle.Foreground(lipgloss.Color("42"))
		b.WriteString(statusStyle.Render("✓ COMPLETE"))
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("214"))
		b.WriteString(statusStyle.Render("○ PENDING"))
	}
	b.WriteString("\n\n")

	// Priority and Category
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	b.WriteString(labelStyle.Render("Priority: "))
	b.WriteString(valueStyle.Render(string(f.Priority)))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Category: "))
	b.WriteString(valueStyle.Render(string(f.Category)))
	b.WriteString("\n\n")

	// Description
	b.WriteString(labelStyle.Render("Description:"))
	b.WriteString("\n")
	b.WriteString(valueStyle.Render(f.Description))
	b.WriteString("\n\n")

	// Dependencies
	if len(f.DependsOn) > 0 {
		b.WriteString(labelStyle.Render("Depends On:"))
		b.WriteString("\n")
		for _, dep := range f.DependsOn {
			b.WriteString("  • ")
			b.WriteString(valueStyle.Render(dep))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Steps
	b.WriteString(labelStyle.Render("Steps:"))
	b.WriteString("\n")
	for i, step := range f.Steps {
		b.WriteString(fmt.Sprintf("  %d. ", i+1))
		b.WriteString(valueStyle.Render(step))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	b.WriteString(helpStyle.Render("Press Enter or Esc to close"))

	// Wrap in styled box
	return ifl.detailStyle.Width(ifl.Width - 4).Render(b.String())
}

// renderHelp renders the help text
func (ifl *InteractiveFeatureList) renderHelp() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	var keys []string
	if ifl.Filtering {
		keys = []string{"[Enter] Apply", "[Esc] Cancel"}
	} else {
		keys = []string{"[j/k] Navigate", "[Enter] Details", "[/] Search", "[g/G] Top/Bottom"}
	}

	return helpStyle.Render(strings.Join(keys, "  "))
}

// GetStats returns statistics about the features
func (ifl *InteractiveFeatureList) GetStats() (inProgress, pending, blocked, done int) {
	items := ifl.List.Items()
	for _, item := range items {
		fi, ok := item.(FeatureItem)
		if !ok {
			continue
		}
		switch fi.status {
		case FeatureItemStatusInProgress:
			inProgress++
		case FeatureItemStatusPending:
			pending++
		case FeatureItemStatusBlocked:
			blocked++
		case FeatureItemStatusDone:
			done++
		}
	}
	return
}

// GetGroupedCounts returns feature counts by group for display
func (ifl *InteractiveFeatureList) GetGroupedCounts() string {
	inProgress, pending, blocked, done := ifl.GetStats()

	var parts []string
	if inProgress > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render(fmt.Sprintf("◐ %d", inProgress)))
	}
	if blocked > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("✗ %d", blocked)))
	}
	if pending > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Render(fmt.Sprintf("○ %d", pending)))
	}
	if done > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render(fmt.Sprintf("✓ %d", done)))
	}

	return strings.Join(parts, "  ")
}

// IsFiltering returns whether filtering is active
func (ifl *InteractiveFeatureList) IsFiltering() bool {
	return ifl.Filtering
}

// IsShowingDetail returns whether detail view is active
func (ifl *InteractiveFeatureList) IsShowingDetail() bool {
	return ifl.ShowDetail
}

// SetShowPriority enables/disables priority display
func (ifl *InteractiveFeatureList) SetShowPriority(show bool) {
	ifl.Delegate.ShowPriority = show
}

// SetShowCategory enables/disables category display
func (ifl *InteractiveFeatureList) SetShowCategory(show bool) {
	ifl.Delegate.ShowCategory = show
}

// FilteredCount returns the number of items currently visible (after filtering)
func (ifl *InteractiveFeatureList) FilteredCount() int {
	return len(ifl.List.Items())
}

// TotalCount returns total features
func (ifl *InteractiveFeatureList) TotalCount() int {
	if ifl.PRD == nil {
		return 0
	}
	return len(ifl.PRD.Features)
}

// Helper to check if list has mouse support through wheel events
func (ifl *InteractiveFeatureList) HandleMouseWheel(delta int) {
	if delta < 0 {
		// Scroll up
		for i := 0; i < -delta; i++ {
			ifl.List.CursorUp()
		}
	} else {
		// Scroll down
		for i := 0; i < delta; i++ {
			ifl.List.CursorDown()
		}
	}
}

// compile-time check that FeatureItem implements list.Item and list.DefaultItem
var _ list.Item = FeatureItem{}
var _ list.DefaultItem = FeatureItem{}

// compile-time check that FeatureDelegate implements list.ItemDelegate
var _ list.ItemDelegate = FeatureDelegate{}
