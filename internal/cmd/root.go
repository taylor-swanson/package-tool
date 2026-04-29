// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package cmd provides commands for the package-tool CLI.
package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/taylor-swanson/package-tool/internal/version"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

func Execute() error {
	cmd := cobra.Command{
		Use:   version.Name,
		Short: "Tools for Elastic integration packages",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	cmd.PersistentFlags().Bool("json", false, "enable JSON logging")

	cmd.PersistentFlags().Bool("deprecated", false, "include deprecated packages")
	cmd.PersistentFlags().String("filter-owner", "", "filter by owner of the package")
	cmd.PersistentFlags().String("filter-owner-type", "", "filter by owner type of the package")
	cmd.PersistentFlags().String("filter-type", "", "filter by type of the package")

	cmd.AddCommand(
		newCmdAnalyze(),
		newCmdList(),
		newCmdVersion(),
	)

	return cmd.Execute()
}

func filterPackages(dirs []string, flags *pflag.FlagSet) ([]string, error) {
	var filtered []string

	includeDeprecated, _ := flags.GetBool("deprecated")
	filterOwner, _ := flags.GetString("filter-owner")
	filterOwnerType, _ := flags.GetString("filter-owner-type")
	filterType, _ := flags.GetString("filter-type")

	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	for _, dir := range dirs {
		root, err := fleetpkg.LocateRoot(dir)
		if err != nil {
			slog.Error("Failed to locate package root directory", slog.String("path", dir), slog.String("error", err.Error()))
			continue
		}

		manifest, err := fleetpkg.LoadManifest(root)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", root), slog.String("error", err.Error()))
			continue
		}

		if filterOwner != "" && filterOwner != manifest.Owner.Github {
			continue
		}
		if filterOwnerType != "" && filterOwnerType != manifest.Owner.Type {
			continue
		}
		if filterType != "" && filterType != manifest.Type {
			continue
		}
		if !includeDeprecated && strings.Contains(strings.ToLower(manifest.Title), "deprecated") {
			continue
		}

		filtered = append(filtered, root)
	}

	return filtered, nil
}
