package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/taylor-swanson/package-tool/internal/version"
)

func newCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Report version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("package-tool version %s [build %v] [commit %v]\n", version.Version, version.Build, version.Commit)
		},
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	return cmd
}
