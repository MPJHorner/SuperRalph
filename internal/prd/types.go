package prd

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// PRD represents a Product Requirements Document
type PRD struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TestCommand string    `json:"testCommand"`
	Features    []Feature `json:"features"`
}

// Feature represents a single feature in the PRD
type Feature struct {
	ID          string   `json:"id"`
	Category    Category `json:"category"`
	Priority    Priority `json:"priority"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Passes      bool     `json:"passes"`
	DependsOn   []string `json:"depends_on,omitempty"` // Optional list of feature IDs that must pass first
}

// Category represents the type of feature
type Category string

const (
	CategoryFunctional  Category = "functional"
	CategoryUI          Category = "ui"
	CategoryIntegration Category = "integration"
	CategoryPerformance Category = "performance"
	CategorySecurity    Category = "security"
)

// ValidCategories returns all valid category values
func ValidCategories() []Category {
	return []Category{
		CategoryFunctional,
		CategoryUI,
		CategoryIntegration,
		CategoryPerformance,
		CategorySecurity,
	}
}

// IsValid checks if the category is valid
func (c Category) IsValid() bool {
	return lo.Contains(ValidCategories(), c)
}

// Priority represents the priority level of a feature
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// ValidPriorities returns all valid priority values
func ValidPriorities() []Priority {
	return []Priority{
		PriorityHigh,
		PriorityMedium,
		PriorityLow,
	}
}

// IsValid checks if the priority is valid
func (p Priority) IsValid() bool {
	return lo.Contains(ValidPriorities(), p)
}

// Stats returns statistics about the PRD
func (p *PRD) Stats() PRDStats {
	stats := PRDStats{
		TotalFeatures:   len(p.Features),
		PassingFeatures: lo.CountBy(p.Features, func(f Feature) bool { return f.Passes }),
		ByCategory:      make(map[Category]CategoryStats),
		ByPriority:      make(map[Priority]PriorityStats),
	}

	// Initialize all categories and priorities
	for _, cat := range ValidCategories() {
		stats.ByCategory[cat] = CategoryStats{}
	}
	for _, pri := range ValidPriorities() {
		stats.ByPriority[pri] = PriorityStats{}
	}

	// Count by category and priority
	for _, f := range p.Features {
		cs := stats.ByCategory[f.Category]
		cs.Total++
		if f.Passes {
			cs.Passing++
		}
		stats.ByCategory[f.Category] = cs

		ps := stats.ByPriority[f.Priority]
		ps.Total++
		if f.Passes {
			ps.Passing++
		}
		stats.ByPriority[f.Priority] = ps
	}

	return stats
}

// PRDStats holds statistics about a PRD
type PRDStats struct {
	TotalFeatures   int
	PassingFeatures int
	ByCategory      map[Category]CategoryStats
	ByPriority      map[Priority]PriorityStats
}

// CategoryStats holds statistics for a category
type CategoryStats struct {
	Total   int
	Passing int
}

// PriorityStats holds statistics for a priority
type PriorityStats struct {
	Total   int
	Passing int
}

// PercentComplete returns the percentage of features that pass
func (s PRDStats) PercentComplete() float64 {
	if s.TotalFeatures == 0 {
		return 0
	}
	return float64(s.PassingFeatures) / float64(s.TotalFeatures) * 100
}

// NextFeature returns the next feature to work on based on:
// 1. Skip features with passes: true
// 2. Skip features blocked by unmet dependencies
// 3. Highest priority first (high > medium > low)
// 4. ID order within same priority
func (p *PRD) NextFeature() *Feature {
	// Priority order: high > medium > low
	priorities := []Priority{PriorityHigh, PriorityMedium, PriorityLow}

	for _, priority := range priorities {
		for i := range p.Features {
			f := &p.Features[i]
			if !f.Passes && f.Priority == priority && p.DependenciesMet(f) {
				return f
			}
		}
	}
	return nil
}

// NextFeatureWithReason returns the next feature and a human-readable reason for why it was selected
func (p *PRD) NextFeatureWithReason() (*Feature, string) {
	next := p.NextFeature()
	if next == nil {
		return nil, "all features are complete"
	}

	var reasons []string
	reasons = append(reasons, "not yet passing")

	if len(next.DependsOn) > 0 {
		reasons = append(reasons, "all dependencies met")
	}

	reasons = append(reasons, fmt.Sprintf("%s priority", next.Priority))

	// Check if there are any higher-priority features that are blocked
	blockedHigherPriority := p.getBlockedHigherPriorityFeatures(next.Priority)
	if len(blockedHigherPriority) > 0 {
		reasons = append(reasons, fmt.Sprintf("features %v blocked by unmet dependencies", blockedHigherPriority))
	}

	return next, fmt.Sprintf("Selected %s: %s", next.ID, strings.Join(reasons, ", "))
}

// getBlockedHigherPriorityFeatures returns IDs of features with higher priority than the given one that are blocked
func (p *PRD) getBlockedHigherPriorityFeatures(selectedPriority Priority) []string {
	priorityOrder := map[Priority]int{
		PriorityHigh:   0,
		PriorityMedium: 1,
		PriorityLow:    2,
	}
	selectedOrder := priorityOrder[selectedPriority]

	blocked := lo.FilterMap(p.Features, func(f Feature, _ int) (string, bool) {
		fOrder := priorityOrder[f.Priority]
		// Only consider features with higher priority (lower order number)
		if !f.Passes && fOrder < selectedOrder && !p.DependenciesMet(&f) {
			return f.ID, true
		}
		return "", false
	})
	return blocked
}

// DependenciesMet returns true if all dependencies of the feature have passes: true
func (p *PRD) DependenciesMet(f *Feature) bool {
	if len(f.DependsOn) == 0 {
		return true
	}

	passingIDs := p.getPassingFeatureIDs()
	return lo.EveryBy(f.DependsOn, func(depID string) bool {
		return passingIDs[depID]
	})
}

// getPassingFeatureIDs returns a set of feature IDs that have passes: true
func (p *PRD) getPassingFeatureIDs() map[string]bool {
	passingFeatures := lo.Filter(p.Features, func(f Feature, _ int) bool {
		return f.Passes
	})
	return lo.SliceToMap(passingFeatures, func(f Feature) (string, bool) {
		return f.ID, true
	})
}

// GetBlockedFeatures returns features that are blocked by unmet dependencies
func (p *PRD) GetBlockedFeatures() []Feature {
	return lo.Filter(p.Features, func(f Feature, _ int) bool {
		return !f.Passes && !p.DependenciesMet(&f)
	})
}

// GetUnmetDependencies returns the IDs of dependencies that are not yet passing for a feature
func (p *PRD) GetUnmetDependencies(f *Feature) []string {
	if len(f.DependsOn) == 0 {
		return nil
	}

	passingIDs := p.getPassingFeatureIDs()
	return lo.Filter(f.DependsOn, func(depID string, _ int) bool {
		return !passingIDs[depID]
	})
}

// IsComplete returns true if all features pass
func (p *PRD) IsComplete() bool {
	return len(p.Features) > 0 && lo.EveryBy(p.Features, func(f Feature) bool {
		return f.Passes
	})
}
