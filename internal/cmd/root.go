package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func Execute() error {
	cmd := cobra.Command{
		Use:   "package-tool",
		Short: "Tools for Elastic integration packages",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo

			logDebug, _ := cmd.Flags().GetBool("debug")
			if logDebug {
				level = slog.LevelDebug
			}

			logJSON, _ := cmd.Flags().GetBool("json")
			if logJSON {
				slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			} else {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	cmd.PersistentFlags().Bool("json", false, "enable JSON logging")

	cmd.PersistentFlags().Bool("deprecated", false, "Include deprecated packages")
	cmd.PersistentFlags().String("filter-owner", "", "Filter by owner of the package")
	cmd.PersistentFlags().String("filter-owner-type", "", "Filter by owner type of the package")
	cmd.PersistentFlags().String("filter-type", "", "Filter by type of the package")

	cmd.AddCommand(
		newCmdList(),
		newCmdReport(),
		newCmdVersion(),
	)

	return cmd.Execute()
}
