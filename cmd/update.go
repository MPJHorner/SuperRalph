package cmd

import (
	"fmt"
	"os"

	"github.com/mpjhorner/superralph/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update SuperRalph to the latest version",
	Long: `Update downloads and installs the latest version of SuperRalph from GitHub.

If a newer version is available, it will:
  1. Download the appropriate binary for your OS/architecture
  2. Replace the current binary (may require sudo)

You can also reinstall using:
  curl -fsSL https://raw.githubusercontent.com/MPJHorner/SuperRalph/main/install.sh | sh`,
	Run: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Printf("Current version: %s\n", version.Short())
	fmt.Println("Checking for updates...")

	release, err := version.CheckForUpdate()
	if err != nil {
		fmt.Println(errorStyle.Render("✗") + " Failed to check for updates")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		os.Exit(1)
	}

	if !version.IsNewer(release) {
		fmt.Println(successStyle.Render("✓") + " Already at latest version")
		return
	}

	fmt.Printf("New version available: %s\n\n", successStyle.Render(release.TagName))

	if err := version.SelfUpdate(); err != nil {
		fmt.Println(errorStyle.Render("✗") + " Update failed")
		fmt.Println(dimStyle.Render("  " + err.Error()))
		fmt.Println()
		fmt.Println("You can manually update by running:")
		fmt.Println(dimStyle.Render("  curl -fsSL https://raw.githubusercontent.com/MPJHorner/SuperRalph/main/install.sh | sh"))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("✓") + " Update complete!")
}
