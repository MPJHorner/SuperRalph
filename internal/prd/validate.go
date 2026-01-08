package prd

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidationResult holds the results of PRD validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validate validates the PRD and returns a ValidationResult
func Validate(p *PRD) ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate required fields
	if strings.TrimSpace(p.Name) == "" {
		result.addError("name", "is required")
	}

	if strings.TrimSpace(p.Description) == "" {
		result.addError("description", "is required")
	}

	if strings.TrimSpace(p.TestCommand) == "" {
		result.addError("testCommand", "is required")
	}

	if len(p.Features) == 0 {
		result.addError("features", "must have at least one feature")
	}

	// Validate each feature
	seenIDs := make(map[string]bool)
	for i, f := range p.Features {
		prefix := fmt.Sprintf("features[%d]", i)

		// Validate ID
		if strings.TrimSpace(f.ID) == "" {
			result.addError(prefix+".id", "is required")
		} else if seenIDs[f.ID] {
			result.addError(prefix+".id", fmt.Sprintf("duplicate id '%s'", f.ID))
		} else {
			seenIDs[f.ID] = true
		}

		// Validate category
		if !f.Category.IsValid() {
			result.addError(prefix+".category", fmt.Sprintf("invalid category '%s' (must be one of: %s)",
				f.Category, validCategoryList()))
		}

		// Validate priority
		if !f.Priority.IsValid() {
			result.addError(prefix+".priority", fmt.Sprintf("invalid priority '%s' (must be one of: %s)",
				f.Priority, validPriorityList()))
		}

		// Validate description
		if strings.TrimSpace(f.Description) == "" {
			result.addError(prefix+".description", "is required")
		}

		// Validate steps
		if len(f.Steps) == 0 {
			result.addError(prefix+".steps", "must have at least one step")
		} else {
			for j, step := range f.Steps {
				if strings.TrimSpace(step) == "" {
					result.addError(fmt.Sprintf("%s.steps[%d]", prefix, j), "cannot be empty")
				}
			}
		}
	}

	// Validate depends_on references (second pass, after all IDs are collected)
	for i, f := range p.Features {
		prefix := fmt.Sprintf("features[%d]", i)
		for j, depID := range f.DependsOn {
			if strings.TrimSpace(depID) == "" {
				result.addError(fmt.Sprintf("%s.depends_on[%d]", prefix, j), "cannot be empty")
			} else if !seenIDs[depID] {
				result.addError(fmt.Sprintf("%s.depends_on[%d]", prefix, j),
					fmt.Sprintf("references unknown feature '%s'", depID))
			} else if depID == f.ID {
				result.addError(fmt.Sprintf("%s.depends_on[%d]", prefix, j),
					"feature cannot depend on itself")
			}
		}
	}

	return result
}

func (r *ValidationResult) addError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: message})
}

func validCategoryList() string {
	cats := ValidCategories()
	strs := make([]string, len(cats))
	for i, c := range cats {
		strs[i] = string(c)
	}
	return strings.Join(strs, ", ")
}

func validPriorityList() string {
	pris := ValidPriorities()
	strs := make([]string, len(pris))
	for i, p := range pris {
		strs[i] = string(p)
	}
	return strings.Join(strs, ", ")
}
