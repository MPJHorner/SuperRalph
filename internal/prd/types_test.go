package prd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, tt.category.IsValid())
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
			assert.Equal(t, tt.want, tt.priority.IsValid())
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

	assert.Equal(t, 4, stats.TotalFeatures)
	assert.Equal(t, 2, stats.PassingFeatures)

	// Check category stats
	funcStats := stats.ByCategory[CategoryFunctional]
	assert.Equal(t, 2, funcStats.Total)
	assert.Equal(t, 1, funcStats.Passing)

	uiStats := stats.ByCategory[CategoryUI]
	assert.Equal(t, 1, uiStats.Total)
	assert.Equal(t, 1, uiStats.Passing)

	// Check priority stats
	highStats := stats.ByPriority[PriorityHigh]
	assert.Equal(t, 2, highStats.Total)
	assert.Equal(t, 1, highStats.Passing)
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
	require.NotNil(t, next)

	// Should return highest priority non-passing feature
	assert.Equal(t, "feat-004", next.ID)
}

func TestPRDNextFeatureAllPassing(t *testing.T) {
	prd := &PRD{
		Features: []Feature{
			{ID: "feat-001", Priority: PriorityHigh, Passes: true},
			{ID: "feat-002", Priority: PriorityMedium, Passes: true},
		},
	}

	next := prd.NextFeature()
	assert.Nil(t, next)
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
			assert.Equal(t, tt.want, prd.IsComplete())
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
			assert.Equal(t, tt.want, stats.PercentComplete())
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
			require.NotNil(t, feature, "Feature %s not found", tt.checkID)
			assert.Equal(t, tt.want, p.DependenciesMet(feature))
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
			require.NotNil(t, next)
			assert.Equal(t, tt.wantID, next.ID)
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
	require.NotNil(t, next)
	assert.Equal(t, "feat-002", next.ID)
	assert.NotEmpty(t, reason)
	// Should mention that feat-001 is blocked
	assert.Contains(t, reason, "feat-001")
	assert.Contains(t, reason, "blocked")
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
	require.Len(t, blocked, 1)
	assert.Equal(t, "feat-002", blocked[0].ID)
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
	require.Len(t, unmet, 1)
	assert.Equal(t, "feat-002", unmet[0])
}

func TestFeatureDependsOnField(t *testing.T) {
	// Test that DependsOn field is properly initialized
	f := Feature{
		ID:        "feat-001",
		DependsOn: []string{"feat-000"},
	}
	require.Len(t, f.DependsOn, 1)
	assert.Equal(t, "feat-000", f.DependsOn[0])
}
