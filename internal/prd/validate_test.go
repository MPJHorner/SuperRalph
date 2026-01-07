package prd

import (
	"testing"
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

			if result.Valid != tt.wantValid {
				t.Errorf("Validate().Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("Validate() returned %d errors, want %d", len(result.Errors), tt.wantErrors)
				for _, err := range result.Errors {
					t.Logf("  - %s", err.Error())
				}
			}
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
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
