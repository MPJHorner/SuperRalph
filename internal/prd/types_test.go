package prd

import (
	"testing"
)

func TestCategoryIsValid(t *testing.T) {
	tests := []struct {
		category Category
		want     bool
	}{
		{CategoryFunctional, true},
		{CategoryUI, true},
		{CategoryIntegration, true},
		{CategoryPerformance, true},
		{CategorySecurity, true},
		{Category("invalid"), false},
		{Category(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if got := tt.category.IsValid(); got != tt.want {
				t.Errorf("Category(%q).IsValid() = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestPriorityIsValid(t *testing.T) {
	tests := []struct {
		priority Priority
		want     bool
	}{
		{PriorityHigh, true},
		{PriorityMedium, true},
		{PriorityLow, true},
		{Priority("invalid"), false},
		{Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.want {
				t.Errorf("Priority(%q).IsValid() = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestPRDStats(t *testing.T) {
	prd := &PRD{
		Name:        "Test Project",
		Description: "Test description",
		TestCommand: "go test ./...",
		Features: []Feature{
			{ID: "feat-001", Category: CategoryFunctional, Priority: PriorityHigh, Passes: true},
			{ID: "feat-002", Category: CategoryFunctional, Priority: PriorityHigh, Passes: false},
			{ID: "feat-003", Category: CategoryUI, Priority: PriorityMedium, Passes: true},
			{ID: "feat-004", Category: CategorySecurity, Priority: PriorityLow, Passes: false},
		},
	}

	stats := prd.Stats()

	if stats.TotalFeatures != 4 {
		t.Errorf("TotalFeatures = %d, want 4", stats.TotalFeatures)
	}

	if stats.PassingFeatures != 2 {
		t.Errorf("PassingFeatures = %d, want 2", stats.PassingFeatures)
	}

	// Check category stats
	funcStats := stats.ByCategory[CategoryFunctional]
	if funcStats.Total != 2 || funcStats.Passing != 1 {
		t.Errorf("Functional stats = %+v, want {Total:2, Passing:1}", funcStats)
	}

	uiStats := stats.ByCategory[CategoryUI]
	if uiStats.Total != 1 || uiStats.Passing != 1 {
		t.Errorf("UI stats = %+v, want {Total:1, Passing:1}", uiStats)
	}

	// Check priority stats
	highStats := stats.ByPriority[PriorityHigh]
	if highStats.Total != 2 || highStats.Passing != 1 {
		t.Errorf("High priority stats = %+v, want {Total:2, Passing:1}", highStats)
	}
}

func TestPRDNextFeature(t *testing.T) {
	prd := &PRD{
		Features: []Feature{
			{ID: "feat-001", Priority: PriorityLow, Passes: false},
			{ID: "feat-002", Priority: PriorityHigh, Passes: true},
			{ID: "feat-003", Priority: PriorityMedium, Passes: false},
			{ID: "feat-004", Priority: PriorityHigh, Passes: false},
		},
	}

	next := prd.NextFeature()
	if next == nil {
		t.Fatal("NextFeature() returned nil")
	}

	// Should return highest priority non-passing feature
	if next.ID != "feat-004" {
		t.Errorf("NextFeature().ID = %q, want %q", next.ID, "feat-004")
	}
}

func TestPRDNextFeatureAllPassing(t *testing.T) {
	prd := &PRD{
		Features: []Feature{
			{ID: "feat-001", Priority: PriorityHigh, Passes: true},
			{ID: "feat-002", Priority: PriorityMedium, Passes: true},
		},
	}

	next := prd.NextFeature()
	if next != nil {
		t.Errorf("NextFeature() = %v, want nil", next)
	}
}

func TestPRDIsComplete(t *testing.T) {
	tests := []struct {
		name     string
		features []Feature
		want     bool
	}{
		{
			name:     "empty",
			features: []Feature{},
			want:     false,
		},
		{
			name: "all passing",
			features: []Feature{
				{Passes: true},
				{Passes: true},
			},
			want: true,
		},
		{
			name: "some failing",
			features: []Feature{
				{Passes: true},
				{Passes: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prd := &PRD{Features: tt.features}
			if got := prd.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRDStatsPercentComplete(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		passing int
		want    float64
	}{
		{"empty", 0, 0, 0},
		{"none passing", 10, 0, 0},
		{"all passing", 10, 10, 100},
		{"half passing", 10, 5, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := PRDStats{
				TotalFeatures:   tt.total,
				PassingFeatures: tt.passing,
			}
			if got := stats.PercentComplete(); got != tt.want {
				t.Errorf("PercentComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
