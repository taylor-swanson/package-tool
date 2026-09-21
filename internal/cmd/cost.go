package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/taylor-swanson/package-tool/internal/cost"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

func newCmdCost() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cost [DIR ...]",
		Short:   "Estimate the cost/complexity of data stream pipelines",
		Aliases: []string{"c"},
		RunE:    doCost,
	}

	cmd.Flags().StringSlice("filter-data-streams", nil, "filter by comma-separated list of data stream names")
	cmd.Flags().StringP("output", "o", analyzeFormatTextColor, "output format (choose from: "+strings.Join(analyzeFormats, ", ")+")")

	return cmd
}

func doCost(cmd *cobra.Command, args []string) error {
	filterDataStreams, _ := cmd.Flags().GetStringSlice("filter-data-streams")

	pkgDirs, err := filterPackages(args, cmd.Flags())
	if err != nil {
		return err
	}

	report := cost.Report{
		Timestamp: time.Now(),
		Reports:   map[string]cost.PackageReport{},
	}

	for _, pkgDir := range pkgDirs {
		pkg, err := fleetpkg.Load(pkgDir, fleetpkg.LoadModePipelineOnly)
		if err != nil {
			return err
		}

		pkgReport := cost.EstimatePackage(pkg, filterDataStreams...)

		report.Reports[pkg.Manifest.Name] = pkgReport
	}

	output, _ := cmd.Flags().GetString("output")
	switch output {
	case analyzeFormatJSON:
		err = cost.PrintJSON(os.Stdout, &report)
	case analyzeFormatTextColor:
		err = cost.PrintText(os.Stdout, &report, true)
	case analyzeFormatText:
		fallthrough
	default:
		err = cost.PrintText(os.Stdout, &report, false)
	}

	return err
}
