package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mpjhorner/superralph/internal/prd"
)

func TestNewInteractiveFeatureList(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	require.NotNil(t, ifl)
	assert.Equal(t, 80, ifl.Width)
	assert.Equal(t, 20, ifl.Height)
	assert.False(t, ifl.Filtering)
	assert.False(t, ifl.ShowDetail)
	assert.Nil(t, ifl.DetailFeature)
}

func TestInteractiveFeatureListSetPRD(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First feature", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-002", Description: "Second feature", Priority: prd.PriorityMedium, Passes: true},
			{ID: "feat-003", Description: "Third feature", Priority: prd.PriorityLow, Passes: false},
		},
	}

	ifl.SetPRD(p, "feat-001")

	assert.Equal(t, p, ifl.PRD)
	assert.Equal(t, "feat-001", ifl.CurrentFeatureID)
	assert.Equal(t, 3, len(ifl.List.Items()))
}

func TestInteractiveFeatureListGrouping(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityLow, Passes: true},
			{ID: "feat-002", Description: "Second", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-003", Description: "Third", Priority: prd.PriorityMedium, Passes: false},
		},
	}

	ifl.SetPRD(p, "feat-002")

	// Get items - should be sorted: InProgress first, then Pending by priority, then Done
	items := ifl.List.Items()
	require.Len(t, items, 3)

	// First item should be the in-progress one
	fi1 := items[0].(FeatureItem)
	assert.Equal(t, "feat-002", fi1.feature.ID)
	assert.Equal(t, FeatureItemStatusInProgress, fi1.status)

	// Second should be pending medium priority
	fi2 := items[1].(FeatureItem)
	assert.Equal(t, "feat-003", fi2.feature.ID)
	assert.Equal(t, FeatureItemStatusPending, fi2.status)

	// Third should be done
	fi3 := items[2].(FeatureItem)
	assert.Equal(t, "feat-001", fi3.feature.ID)
	assert.Equal(t, FeatureItemStatusDone, fi3.status)
}

func TestInteractiveFeatureListResize(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	ifl.Resize(100, 30)
	assert.Equal(t, 100, ifl.Width)
	assert.Equal(t, 30, ifl.Height)
}

func TestFeatureItem(t *testing.T) {
	f := &prd.Feature{
		ID:          "feat-001",
		Description: "Test feature",
		Priority:    prd.PriorityHigh,
	}

	fi := FeatureItem{feature: f, status: FeatureItemStatusPending}

	assert.Equal(t, "feat-001 Test feature", fi.FilterValue())
	assert.Equal(t, "feat-001", fi.Title())
	assert.Equal(t, "Test feature", fi.Description())
	assert.Equal(t, FeatureItemStatusPending, fi.Status())
	assert.Equal(t, f, fi.Feature())
}

func TestFeatureItemStatusIcon(t *testing.T) {
	tests := []struct {
		status   FeatureItemStatus
		expected string
	}{
		{FeatureItemStatusDone, "✓"},
		{FeatureItemStatusInProgress, "◐"},
		{FeatureItemStatusBlocked, "✗"},
		{FeatureItemStatusPending, "○"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			fi := FeatureItem{
				feature: &prd.Feature{ID: "test"},
				status:  tt.status,
			}
			assert.Equal(t, tt.expected, fi.StatusIcon())
		})
	}
}

func TestFeatureDelegate(t *testing.T) {
	delegate := NewFeatureDelegate()
	assert.Equal(t, 2, delegate.Height())
	assert.Equal(t, 0, delegate.Spacing())
	assert.True(t, delegate.ShowPriority)
	assert.False(t, delegate.ShowCategory)
}

func TestInteractiveFeatureListGetStats(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityHigh, Passes: true},
			{ID: "feat-002", Description: "Second", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-003", Description: "Third", Priority: prd.PriorityMedium, Passes: false},
			{ID: "feat-004", Description: "Fourth", Priority: prd.PriorityLow, Passes: true},
		},
	}

	ifl.SetPRD(p, "feat-002")

	inProgress, pending, blocked, done := ifl.GetStats()
	assert.Equal(t, 1, inProgress)
	assert.Equal(t, 1, pending)
	assert.Equal(t, 0, blocked)
	assert.Equal(t, 2, done)
}

func TestInteractiveFeatureListFilteredCount(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
			{ID: "feat-003", Description: "Third", Passes: false},
		},
	}

	ifl.SetPRD(p, "")
	assert.Equal(t, 3, ifl.FilteredCount())
	assert.Equal(t, 3, ifl.TotalCount())
}

func TestInteractiveFeatureListTotalCountNilPRD(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	assert.Equal(t, 0, ifl.TotalCount())
}

func TestInteractiveFeatureListSelectedItem(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
		},
	}

	ifl.SetPRD(p, "")
	selected := ifl.SelectedItem()
	require.NotNil(t, selected)
	assert.Contains(t, []string{"feat-001", "feat-002"}, selected.feature.ID)
}

func TestInteractiveFeatureListSelectedItemEmpty(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name:     "Test PRD",
		Features: []prd.Feature{},
	}

	ifl.SetPRD(p, "")
	selected := ifl.SelectedItem()
	assert.Nil(t, selected)
}

func TestInteractiveFeatureListFilterActivation(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	assert.False(t, ifl.IsFiltering())

	// Press / to activate filter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, ifl.IsFiltering())
}

func TestInteractiveFeatureListFilterDeactivation(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Activate filter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, ifl.IsFiltering())

	// Press Esc to deactivate
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, ifl.IsFiltering())
}

func TestInteractiveFeatureListDetailView(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	assert.False(t, ifl.IsShowingDetail())

	// Press Enter to show detail
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, ifl.IsShowingDetail())
	assert.NotNil(t, ifl.DetailFeature)
}

func TestInteractiveFeatureListDetailViewClose(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Open detail view
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, ifl.IsShowingDetail())

	// Close with Esc
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, ifl.IsShowingDetail())
	assert.Nil(t, ifl.DetailFeature)
}

func TestInteractiveFeatureListNavigation(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-002", Description: "Second", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-003", Description: "Third", Priority: prd.PriorityHigh, Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Initial position
	initialIdx := ifl.List.Index()

	// Navigate down with j
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, initialIdx+1, ifl.List.Index())

	// Navigate up with k
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, initialIdx, ifl.List.Index())
}

func TestInteractiveFeatureListView(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityHigh, Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	view := ifl.View()
	assert.NotEmpty(t, view)
	// Should contain navigation help
	assert.Contains(t, view, "Navigate")
	assert.Contains(t, view, "Search")
}

func TestInteractiveFeatureListViewWithDetail(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{
				ID:          "feat-001",
				Description: "First feature description",
				Priority:    prd.PriorityHigh,
				Category:    prd.CategoryFunctional,
				Steps:       []string{"Step 1", "Step 2"},
				Passes:      false,
			},
		},
	}
	ifl.SetPRD(p, "")

	// Show detail
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := ifl.View()
	assert.Contains(t, view, "feat-001")
	assert.Contains(t, view, "Description")
	assert.Contains(t, view, "Steps")
	assert.Contains(t, view, "Step 1")
}

func TestInteractiveFeatureListViewWithFilter(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Activate filter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	view := ifl.View()
	// Should show filter UI
	assert.Contains(t, view, "🔍")
}

func TestInteractiveFeatureListSetShowPriority(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	assert.True(t, ifl.Delegate.ShowPriority)

	ifl.SetShowPriority(false)
	assert.False(t, ifl.Delegate.ShowPriority)

	ifl.SetShowPriority(true)
	assert.True(t, ifl.Delegate.ShowPriority)
}

func TestInteractiveFeatureListSetShowCategory(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	assert.False(t, ifl.Delegate.ShowCategory)

	ifl.SetShowCategory(true)
	assert.True(t, ifl.Delegate.ShowCategory)

	ifl.SetShowCategory(false)
	assert.False(t, ifl.Delegate.ShowCategory)
}

func TestInteractiveFeatureListHandleMouseWheel(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
			{ID: "feat-003", Description: "Third", Passes: false},
			{ID: "feat-004", Description: "Fourth", Passes: false},
			{ID: "feat-005", Description: "Fifth", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	initialIdx := ifl.List.Index()

	// Scroll down
	ifl.HandleMouseWheel(2)
	assert.Equal(t, initialIdx+2, ifl.List.Index())

	// Scroll up
	ifl.HandleMouseWheel(-1)
	assert.Equal(t, initialIdx+1, ifl.List.Index())
}

func TestFuzzyMatchFeature(t *testing.T) {
	tests := []struct {
		target   string
		query    string
		expected bool
	}{
		{"feat-001 test feature", "", true},
		{"feat-001 test feature", "feat", true},
		{"feat-001 test feature", "001", true},
		{"feat-001 test feature", "f001", true},
		{"feat-001 test feature", "xyz", false},
		{"feat-001 test feature", "tst", true},
		{"authentication login", "auth", true},
		{"authentication login", "authlg", true},
	}

	for _, tt := range tests {
		t.Run(tt.target+"/"+tt.query, func(t *testing.T) {
			result := fuzzyMatchFeature(tt.target, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeatureItemStatusConstants(t *testing.T) {
	assert.Equal(t, FeatureItemStatus(0), FeatureItemStatusPending)
	assert.Equal(t, FeatureItemStatus(1), FeatureItemStatusInProgress)
	assert.Equal(t, FeatureItemStatus(2), FeatureItemStatusDone)
	assert.Equal(t, FeatureItemStatus(3), FeatureItemStatusBlocked)
}

func TestInteractiveFeatureListGetGroupedCounts(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityHigh, Passes: true},
			{ID: "feat-002", Description: "Second", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-003", Description: "Third", Priority: prd.PriorityMedium, Passes: false},
		},
	}

	ifl.SetPRD(p, "feat-002")

	counts := ifl.GetGroupedCounts()
	assert.NotEmpty(t, counts)
	// Should contain status icons
	assert.Contains(t, counts, "◐") // in progress
	assert.Contains(t, counts, "○") // pending
	assert.Contains(t, counts, "✓") // done
}

func TestInteractiveFeatureListBlockedStatus(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Priority: prd.PriorityHigh, Passes: false},
			{ID: "feat-002", Description: "Second", Priority: prd.PriorityHigh, Passes: false, DependsOn: []string{"feat-001"}},
		},
	}

	ifl.SetPRD(p, "")

	// feat-002 should be blocked because feat-001 hasn't passed
	items := ifl.List.Items()
	var blockedFound bool
	for _, item := range items {
		fi := item.(FeatureItem)
		if fi.feature.ID == "feat-002" && fi.status == FeatureItemStatusBlocked {
			blockedFound = true
			break
		}
	}
	assert.True(t, blockedFound, "feat-002 should be marked as blocked")
}

func TestFeatureDelegateUpdate(t *testing.T) {
	delegate := NewFeatureDelegate()
	model := list.New([]list.Item{}, delegate, 80, 20)

	// Update should return nil command
	cmd := delegate.Update(tea.KeyMsg{}, &model)
	assert.Nil(t, cmd)
}

func TestInteractiveFeatureListApplyFilter(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "Authentication feature", Passes: false},
			{ID: "feat-002", Description: "Dashboard feature", Passes: false},
			{ID: "feat-003", Description: "Settings feature", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	assert.Equal(t, 3, ifl.FilteredCount())

	// Apply filter
	ifl.applyFilter("auth")
	assert.Equal(t, 1, ifl.FilteredCount())

	// Clear filter
	ifl.applyFilter("")
	assert.Equal(t, 3, ifl.FilteredCount())
}

func TestInteractiveFeatureListApplyFilterNilPRD(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	// Should not panic
	ifl.applyFilter("test")
}

func TestInteractiveFeatureListGoToTop(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
			{ID: "feat-003", Description: "Third", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Move down
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Go to top with 'g'
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equal(t, 0, ifl.List.Index())
}

func TestInteractiveFeatureListGoToBottom(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
			{ID: "feat-003", Description: "Third", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Go to bottom with 'G'
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.Equal(t, 2, ifl.List.Index())
}

func TestFeatureItemPriorityIcon(t *testing.T) {
	f := &prd.Feature{ID: "test"}
	fi := FeatureItem{feature: f}

	// All priorities return the same icon (●), just styled differently
	assert.Equal(t, "●", fi.PriorityIcon())
}

func TestInteractiveFeatureListFilterConfirmWithEnter(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Activate filter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, ifl.IsFiltering())

	// Confirm with Enter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, ifl.IsFiltering())
}

func TestInteractiveFeatureListDetailViewCloseWithEnter(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Open detail
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, ifl.IsShowingDetail())

	// Close with Enter
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, ifl.IsShowingDetail())
}

func TestInteractiveFeatureListDetailViewCloseWithQ(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Open detail
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, ifl.IsShowingDetail())

	// Close with 'q'
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.False(t, ifl.IsShowingDetail())
}

func TestInteractiveFeatureListRenderDetailView(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	ifl.ShowDetail = true
	ifl.DetailFeature = &prd.Feature{
		ID:          "feat-001",
		Description: "Test feature with dependencies",
		Priority:    prd.PriorityHigh,
		Category:    prd.CategoryFunctional,
		DependsOn:   []string{"feat-000"},
		Steps:       []string{"Step 1", "Step 2", "Step 3"},
		Passes:      true,
	}

	view := ifl.View()
	assert.Contains(t, view, "feat-001")
	assert.Contains(t, view, "COMPLETE")
	assert.Contains(t, view, "high")
	assert.Contains(t, view, "functional")
	assert.Contains(t, view, "Depends On")
	assert.Contains(t, view, "feat-000")
	assert.Contains(t, view, "Steps")
	assert.Contains(t, view, "Step 1")
}

func TestInteractiveFeatureListRenderDetailViewPending(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	ifl.ShowDetail = true
	ifl.DetailFeature = &prd.Feature{
		ID:          "feat-001",
		Description: "Test feature",
		Passes:      false,
	}

	view := ifl.View()
	assert.Contains(t, view, "PENDING")
}

func TestInteractiveFeatureListNavigationDownKey(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	initialIdx := ifl.List.Index()

	// Navigate down with arrow key
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, initialIdx+1, ifl.List.Index())
}

func TestInteractiveFeatureListNavigationUpKey(t *testing.T) {
	ifl := NewInteractiveFeatureList(80, 20)
	p := &prd.PRD{
		Name: "Test PRD",
		Features: []prd.Feature{
			{ID: "feat-001", Description: "First", Passes: false},
			{ID: "feat-002", Description: "Second", Passes: false},
		},
	}
	ifl.SetPRD(p, "")

	// Move down first
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Navigate up with arrow key
	ifl, _ = ifl.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, ifl.List.Index())
}
