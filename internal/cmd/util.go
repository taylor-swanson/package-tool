package cmd

import (
	"log/slog"
	"strings"

	"github.com/spf13/pflag"

	"github.com/taylor-swanson/package-tool/internal/pkg"
)

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
		root, err := pkg.LocateRoot(dir)
		if err != nil {
			slog.Error("Failed to locate package root directory", slog.String("path", dir), slog.String("error", err.Error()))
			continue
		}

		manifest, err := pkg.ReadManifest(root)
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
