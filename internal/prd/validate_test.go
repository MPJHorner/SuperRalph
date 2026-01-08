package prd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		prd        *PRD
		wantValid  bool
		wantErrors int
	}{
		{
			name: "valid PRD",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{"Step 1", "Step 2"},
						Passes:      false,
					},
				},
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "missing name",
			prd: &PRD{
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "missing test command",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "no features",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features:    []Feature{},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "invalid category",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    Category("invalid"),
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "invalid priority",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    Priority("critical"),
						Description: "Test feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "duplicate feature IDs",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature 1",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
					{
						ID:          "feat-001",
						Category:    CategoryUI,
						Priority:    PriorityMedium,
						Description: "Test feature 2",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "feature missing steps",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "feature with empty step",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Test feature",
						Steps:       []string{"Step 1", "", "Step 3"},
						Passes:      false,
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "multiple errors",
			prd: &PRD{
				Name:        "",
				Description: "",
				TestCommand: "",
				Features:    []Feature{},
			},
			wantValid:  false,
			wantErrors: 4, // name, description, testCommand, features
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.prd)

			assert.Equal(t, tt.wantValid, result.Valid)
			assert.Len(t, result.Errors, tt.wantErrors)
		})
	}
}

func TestValidationErrorString(t *testing.T) {
	tests := []struct {
		err  ValidationError
		want string
	}{
		{
			err:  ValidationError{Field: "name", Message: "is required"},
			want: "name: is required",
		},
		{
			err:  ValidationError{Field: "", Message: "general error"},
			want: "general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestValidateDependsOn(t *testing.T) {
	tests := []struct {
		name       string
		prd        *PRD
		wantValid  bool
		wantErrors int
	}{
		{
			name: "valid depends_on",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "First feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
					{
						ID:          "feat-002",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Second feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{"feat-001"},
					},
				},
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "depends_on references unknown feature",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "First feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{"feat-999"}, // doesn't exist
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "depends_on self reference",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Self-referencing feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{"feat-001"}, // depends on itself
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "depends_on empty string",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Feature with empty dep",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{""},
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "multiple depends_on errors",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Feature with multiple bad deps",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{"feat-001", "feat-999", ""}, // self, unknown, empty
					},
				},
			},
			wantValid:  false,
			wantErrors: 3,
		},
		{
			name: "valid multiple dependencies",
			prd: &PRD{
				Name:        "Test Project",
				Description: "Test description",
				TestCommand: "go test ./...",
				Features: []Feature{
					{
						ID:          "feat-001",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "First feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
					{
						ID:          "feat-002",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Second feature",
						Steps:       []string{"Step 1"},
						Passes:      false,
					},
					{
						ID:          "feat-003",
						Category:    CategoryFunctional,
						Priority:    PriorityHigh,
						Description: "Third feature depends on 1 and 2",
						Steps:       []string{"Step 1"},
						Passes:      false,
						DependsOn:   []string{"feat-001", "feat-002"},
					},
				},
			},
			wantValid:  true,
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.prd)

			assert.Equal(t, tt.wantValid, result.Valid)
			assert.Len(t, result.Errors, tt.wantErrors)
		})
	}
}
