package prd

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
	for _, valid := range ValidCategories() {
		if c == valid {
			return true
		}
	}
	return false
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
	for _, valid := range ValidPriorities() {
		if p == valid {
			return true
		}
	}
	return false
}

// Stats returns statistics about the PRD
func (p *PRD) Stats() PRDStats {
	stats := PRDStats{
		TotalFeatures: len(p.Features),
		ByCategory:    make(map[Category]CategoryStats),
		ByPriority:    make(map[Priority]PriorityStats),
	}

	for _, cat := range ValidCategories() {
		stats.ByCategory[cat] = CategoryStats{}
	}
	for _, pri := range ValidPriorities() {
		stats.ByPriority[pri] = PriorityStats{}
	}

	for _, f := range p.Features {
		if f.Passes {
			stats.PassingFeatures++
		}

		// Update category stats
		cs := stats.ByCategory[f.Category]
		cs.Total++
		if f.Passes {
			cs.Passing++
		}
		stats.ByCategory[f.Category] = cs

		// Update priority stats
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

// NextFeature returns the next feature to work on (highest priority, not passing)
func (p *PRD) NextFeature() *Feature {
	// Priority order: high > medium > low
	priorities := []Priority{PriorityHigh, PriorityMedium, PriorityLow}

	for _, priority := range priorities {
		for i := range p.Features {
			if !p.Features[i].Passes && p.Features[i].Priority == priority {
				return &p.Features[i]
			}
		}
	}
	return nil
}

// IsComplete returns true if all features pass
func (p *PRD) IsComplete() bool {
	for _, f := range p.Features {
		if !f.Passes {
			return false
		}
	}
	return len(p.Features) > 0
}
