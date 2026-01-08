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

func TestDependenciesMet(t *testing.T) {
	tests := []struct {
		name     string
		features []Feature
		checkID  string
		want     bool
	}{
		{
			name: "no dependencies",
			features: []Feature{
				{ID: "feat-001", Passes: false},
			},
			checkID: "feat-001",
			want:    true,
		},
		{
			name: "dependency met",
			features: []Feature{
				{ID: "feat-001", Passes: true},
				{ID: "feat-002", Passes: false, DependsOn: []string{"feat-001"}},
			},
			checkID: "feat-002",
			want:    true,
		},
		{
			name: "dependency not met",
			features: []Feature{
				{ID: "feat-001", Passes: false},
				{ID: "feat-002", Passes: false, DependsOn: []string{"feat-001"}},
			},
			checkID: "feat-002",
			want:    false,
		},
		{
			name: "multiple dependencies all met",
			features: []Feature{
				{ID: "feat-001", Passes: true},
				{ID: "feat-002", Passes: true},
				{ID: "feat-003", Passes: false, DependsOn: []string{"feat-001", "feat-002"}},
			},
			checkID: "feat-003",
			want:    true,
		},
		{
			name: "multiple dependencies one not met",
			features: []Feature{
				{ID: "feat-001", Passes: true},
				{ID: "feat-002", Passes: false},
				{ID: "feat-003", Passes: false, DependsOn: []string{"feat-001", "feat-002"}},
			},
			checkID: "feat-003",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PRD{Features: tt.features}
			var feature *Feature
			for i := range p.Features {
				if p.Features[i].ID == tt.checkID {
					feature = &p.Features[i]
					break
				}
			}
			if feature == nil {
				t.Fatalf("Feature %s not found", tt.checkID)
			}
			if got := p.DependenciesMet(feature); got != tt.want {
				t.Errorf("DependenciesMet(%s) = %v, want %v", tt.checkID, got, tt.want)
			}
		})
	}
}

func TestNextFeatureWithDependencies(t *testing.T) {
	tests := []struct {
		name     string
		features []Feature
		wantID   string
	}{
		{
			name: "skip blocked feature pick lower priority",
			features: []Feature{
				{ID: "feat-001", Priority: PriorityHigh, Passes: false},
				{ID: "feat-002", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-001"}},
				{ID: "feat-003", Priority: PriorityMedium, Passes: false},
			},
			wantID: "feat-001", // feat-001 first because it's high priority and has no deps
		},
		{
			name: "blocked high priority skipped for unblocked high priority",
			features: []Feature{
				{ID: "feat-001", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-003"}},
				{ID: "feat-002", Priority: PriorityHigh, Passes: false},
				{ID: "feat-003", Priority: PriorityLow, Passes: false},
			},
			wantID: "feat-002", // feat-001 blocked by feat-003, so pick feat-002
		},
		{
			name: "all high priority blocked pick medium",
			features: []Feature{
				{ID: "feat-001", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-003"}},
				{ID: "feat-002", Priority: PriorityMedium, Passes: false},
				{ID: "feat-003", Priority: PriorityLow, Passes: false},
			},
			wantID: "feat-002", // feat-001 blocked, so pick feat-002 (medium)
		},
		{
			name: "dependency met allows selection",
			features: []Feature{
				{ID: "feat-001", Priority: PriorityLow, Passes: true},
				{ID: "feat-002", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-001"}},
			},
			wantID: "feat-002", // dependency met, so feat-002 is selectable
		},
		{
			name: "chain of dependencies",
			features: []Feature{
				{ID: "feat-001", Priority: PriorityMedium, Passes: true},
				{ID: "feat-002", Priority: PriorityMedium, Passes: false, DependsOn: []string{"feat-001"}},
				{ID: "feat-003", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-002"}},
			},
			wantID: "feat-002", // feat-003 is high but blocked by feat-002 which isn't done
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PRD{Features: tt.features}
			next := p.NextFeature()
			if next == nil {
				t.Fatal("NextFeature() returned nil")
			}
			if next.ID != tt.wantID {
				t.Errorf("NextFeature().ID = %q, want %q", next.ID, tt.wantID)
			}
		})
	}
}

func TestNextFeatureWithReason(t *testing.T) {
	p := &PRD{
		Features: []Feature{
			{ID: "feat-001", Priority: PriorityHigh, Passes: false, DependsOn: []string{"feat-003"}},
			{ID: "feat-002", Priority: PriorityMedium, Passes: false},
			{ID: "feat-003", Priority: PriorityLow, Passes: false},
		},
	}

	next, reason := p.NextFeatureWithReason()
	if next == nil {
		t.Fatal("NextFeatureWithReason() returned nil feature")
	}
	if next.ID != "feat-002" {
		t.Errorf("NextFeatureWithReason().ID = %q, want %q", next.ID, "feat-002")
	}
	if reason == "" {
		t.Error("NextFeatureWithReason() returned empty reason")
	}
	// Should mention that feat-001 is blocked
	if !contains(reason, "feat-001") || !contains(reason, "blocked") {
		t.Errorf("Reason should mention blocked feature: %q", reason)
	}
}

func TestGetBlockedFeatures(t *testing.T) {
	p := &PRD{
		Features: []Feature{
			{ID: "feat-001", Passes: false},
			{ID: "feat-002", Passes: false, DependsOn: []string{"feat-001"}},
			{ID: "feat-003", Passes: true},
			{ID: "feat-004", Passes: false, DependsOn: []string{"feat-003"}}, // not blocked, dep is met
		},
	}

	blocked := p.GetBlockedFeatures()
	if len(blocked) != 1 {
		t.Errorf("GetBlockedFeatures() returned %d features, want 1", len(blocked))
	}
	if len(blocked) > 0 && blocked[0].ID != "feat-002" {
		t.Errorf("GetBlockedFeatures()[0].ID = %q, want %q", blocked[0].ID, "feat-002")
	}
}

func TestGetUnmetDependencies(t *testing.T) {
	p := &PRD{
		Features: []Feature{
			{ID: "feat-001", Passes: true},
			{ID: "feat-002", Passes: false},
			{ID: "feat-003", Passes: false, DependsOn: []string{"feat-001", "feat-002"}},
		},
	}

	var feat003 *Feature
	for i := range p.Features {
		if p.Features[i].ID == "feat-003" {
			feat003 = &p.Features[i]
			break
		}
	}

	unmet := p.GetUnmetDependencies(feat003)
	if len(unmet) != 1 {
		t.Errorf("GetUnmetDependencies() returned %d, want 1", len(unmet))
	}
	if len(unmet) > 0 && unmet[0] != "feat-002" {
		t.Errorf("GetUnmetDependencies()[0] = %q, want %q", unmet[0], "feat-002")
	}
}

func TestFeatureDependsOnField(t *testing.T) {
	// Test that DependsOn field is properly initialized
	f := Feature{
		ID:        "feat-001",
		DependsOn: []string{"feat-000"},
	}
	if len(f.DependsOn) != 1 {
		t.Errorf("DependsOn length = %d, want 1", len(f.DependsOn))
	}
	if f.DependsOn[0] != "feat-000" {
		t.Errorf("DependsOn[0] = %q, want %q", f.DependsOn[0], "feat-000")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
