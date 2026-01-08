package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/mpjhorner/superralph/internal/prd"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the prd.json file in the current directory",
	Long: `Validate checks that the prd.json file exists and has the correct structure.

It verifies:
  - All required fields are present
  - Categories are valid (functional, ui, integration, performance, security)
  - Priorities are valid (high, medium, low)
  - Feature IDs are unique
  - All features have at least one step`,
	Run: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

func runValidate(cmd *cobra.Command, args []string) {
	// Check if prd.json exists
	if !prd.ExistsInCurrentDir() {
		fmt.Println(errorStyle.Render("✗") + " prd.json not found in current directory")
		fmt.Println(dimStyle.Render("  Run 'superralph plan' to create one"))
		os.Exit(1)
	}

	// Load the PRD
	p, err := prd.LoadFromCurrentDir()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to load prd.json")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	// Validate the PRD
	result := prd.Validate(p)

	if !result.Valid {
		fmt.Println(errorStyle.Render("✗") + " prd.json has validation errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  %s %s\n", errorStyle.Render("•"), e.Error())
		}
		fmt.Println()
		os.Exit(1)
	}

	// Success - show summary
	fmt.Println(successStyle.Render("✓") + " prd.json is valid\n")

	stats := p.Stats()

	fmt.Printf("  %s %s\n", boldStyle.Render("Project:"), p.Name)
	fmt.Printf("  %s %s\n", boldStyle.Render("Test Command:"), p.TestCommand)
	fmt.Printf("  %s %d features (%d passing, %d remaining)\n\n",
		boldStyle.Render("Features:"),
		stats.TotalFeatures,
		stats.PassingFeatures,
		stats.TotalFeatures-stats.PassingFeatures,
	)

	// Show breakdown by category
	fmt.Println(dimStyle.Render("  By Category:"))
	for _, cat := range prd.ValidCategories() {
		cs := stats.ByCategory[cat]
		if cs.Total > 0 {
			fmt.Printf("    %-12s %d/%d\n", cat, cs.Passing, cs.Total)
		}
	}

	fmt.Println()

	// Show breakdown by priority
	fmt.Println(dimStyle.Render("  By Priority:"))
	for _, pri := range prd.ValidPriorities() {
		ps := stats.ByPriority[pri]
		if ps.Total > 0 {
			fmt.Printf("    %-12s %d/%d\n", pri, ps.Passing, ps.Total)
		}
	}

	fmt.Println()

	// Show next feature to work on
	if next := p.NextFeature(); next != nil {
		fmt.Printf("  %s %s \"%s\"\n", boldStyle.Render("Next:"), next.ID, next.Description)
	} else if p.IsComplete() {
		fmt.Println(successStyle.Render("  All features complete!"))
	}
}
