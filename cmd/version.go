package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mpjhorner/superralph/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Show the current version, build time, and git commit of SuperRalph.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(version.Info())

	// Check for updates
	release, err := version.CheckForUpdate()
	if err == nil && version.IsNewer(release) {
		fmt.Printf("\n%s New version available: %s%s\n",
			warnStyle.Render("!"),
			successStyle.Render(release.TagName),
			dimStyle.Render(" (run 'superralph update' to upgrade)"))
	}
}
