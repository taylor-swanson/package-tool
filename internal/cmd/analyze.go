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

	"package-tool/internal/analyze"
	"package-tool/internal/analyze/duplicateprocessor"
	"package-tool/internal/analyze/pipelineerror"
	"package-tool/internal/analyze/pipelineeventoriginal"
	"package-tool/internal/analyze/pipelinetag"
	"package-tool/internal/analyze/processorfield"
	"package-tool/internal/analyze/validations"
)

const (
	analyzeFormatTextColor = "text-color"
	analyzeFormatText      = "text"
	analyzeFormatJSON      = "json"
)

var analyzeFormats = []string{
	analyzeFormatTextColor,
	analyzeFormatText,
	analyzeFormatJSON,
}

var analyzers = []*analyze.Analyzer{
	duplicateprocessor.Analyzer,
	pipelineerror.Analyzer,
	pipelineeventoriginal.Analyzer,
	pipelinetag.Analyzer,
	processorfield.Analyzer,
	validations.Analyzer,
}

func getAnalyzer(name string) (*analyze.Analyzer, bool) {
	for _, a := range analyzers {
		if a.Name == name {
			return a, true
		}
	}

	return nil, false
}

func newCmdAnalyze() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analyze ANALYZE [DIR ...]",
		Short:   "Analyze and fix package issues",
		Aliases: []string{"a"},
		RunE:    doAnalyze,
	}

	cmd.Flags().Bool("fix", false, "apply fixes suggested by analyzer (if applicable)")
	cmd.Flags().StringP("output", "o", analyzeFormatTextColor, "output format (choose from: "+strings.Join(analyzeFormats, ", ")+")")
	cmd.Flags().StringSliceP("args", "a", nil, "extra arguments to pass to analyzer")

	return cmd
}

func printAnalyzers() {
	_, _ = fmt.Fprintln(os.Stderr, "Please provide one of the following analyzers:")
	_, _ = fmt.Fprintln(os.Stderr, "")
	tw := tabwriter.NewWriter(os.Stderr, 0, 2, 3, ' ', 0)
	for _, a := range analyzers {
		var fixStr string
		if a.CanFix {
			fixStr = "(FIX)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", a.Name, a.Description, fixStr)
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintln(os.Stderr, "")
}

func doAnalyze(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		printAnalyzers()
		return errors.New("no analyzer specified")
	}

	analyzer, ok := getAnalyzer(args[0])
	if !ok {
		printAnalyzers()
		return fmt.Errorf("invalid analyzer: %s", args[0])
	}

	pkgDirs, err := filterPackages(args[1:], cmd.Flags())
	if err != nil {
		return err
	}

	fix, _ := cmd.Flags().GetBool("fix")
	args, _ = cmd.Flags().GetStringSlice("args")

	results := map[string]analyze.Result{}
	for _, pkgDir := range pkgDirs {
		pkg, err := fleetpkg.Read(pkgDir)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}

		ctx := analyze.Context{
			Package: pkg,
			Fix:     fix,
			Args:    args,
		}

		if err = analyzer.Run(&ctx); err != nil {
			slog.Error("Failed to run analyzer", slog.String("package", pkg.Manifest.Name), slog.String("analyzer", analyzer.Name), slog.String("error", err.Error()))
			continue
		}
		results[pkg.Manifest.Name] = ctx.Result
	}

	output, _ := cmd.Flags().GetString("output")
	switch output {
	case analyzeFormatJSON:
		err = analyze.JSON(os.Stdout, results)
	case analyzeFormatText:
		err = analyze.Text(os.Stdout, results)
	case analyzeFormatTextColor:
		fallthrough
	default:
		err = analyze.ColorText(os.Stdout, results)
	}

	if err != nil {
		return err
	}

	return nil
}
