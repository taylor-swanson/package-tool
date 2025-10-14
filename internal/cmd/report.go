package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/spf13/cobra"

	"package-tool/internal/report"
	"package-tool/internal/report/duplicateprocessor"
	"package-tool/internal/report/pipelineerror"
	"package-tool/internal/report/pipelinetag"
)

const (
	reportFormatTextColor = "text-color"
	reportFormatText      = "text"
	reportFormatJSON      = "json"
)

var reportFormats = []string{
	reportFormatTextColor,
	reportFormatText,
	reportFormatJSON,
}

var reporters = []*report.Reporter{
	duplicateprocessor.Reporter,
	pipelineerror.Reporter,
	pipelinetag.Reporter,
}

func getReporter(name string) (*report.Reporter, bool) {
	for _, r := range reporters {
		if r.Name == name {
			return r, true
		}
	}

	return nil, false
}

func newCmdReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report REPORT [DIR ...]",
		Short: "Report on package issues",
		RunE:  doReport,
	}

	cmd.Flags().StringP("output", "o", reportFormatTextColor, "Output format (choose from: "+strings.Join(reportFormats, ", ")+")")

	return cmd
}

func printReporters() {
	_, _ = fmt.Fprintln(os.Stderr, "Please provide one of the following reporters:")
	_, _ = fmt.Fprintln(os.Stderr, "")
	tw := tabwriter.NewWriter(os.Stderr, 0, 2, 3, ' ', 0)
	for _, r := range reporters {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", r.Name, r.Description)
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintln(os.Stderr, "")
}

func doReport(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		printReporters()
		return errors.New("no reporter specified")
	}

	reporter, ok := getReporter(args[0])
	if !ok {
		printReporters()
		return fmt.Errorf("invalid reporter: %s", args[0])
	}

	pkgDirs, err := filterPackages(args[1:], cmd.Flags())
	if err != nil {
		return err
	}

	results := map[string]report.Result{}
	for _, pkgDir := range pkgDirs {
		pkg, err := fleetpkg.Read(pkgDir)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}

		result, err := reporter.Run(pkg)
		if err != nil {
			slog.Error("Failed to run reporter", slog.String("package", pkg.Manifest.Name), slog.String("reporter", reporter.Name), slog.String("error", err.Error()))
			continue
		}
		results[pkg.Manifest.Name] = result
	}

	output, _ := cmd.Flags().GetString("output")
	switch output {
	case reportFormatJSON:
		err = report.JSON(os.Stdout, results)
	case reportFormatText:
		err = report.Text(os.Stdout, results)
	case reportFormatTextColor:
		fallthrough
	default:
		err = report.ColorText(os.Stdout, results)
	}

	if err != nil {
		return err
	}

	return nil
}
