// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/internal/analyze/duplicateprocessor"
	"github.com/taylor-swanson/package-tool/internal/analyze/pipelineerror"
	"github.com/taylor-swanson/package-tool/internal/analyze/pipelineeventoriginal"
	"github.com/taylor-swanson/package-tool/internal/analyze/pipelinenullctx"
	"github.com/taylor-swanson/package-tool/internal/analyze/pipelinetag"
	"github.com/taylor-swanson/package-tool/internal/analyze/processorfield"
	"github.com/taylor-swanson/package-tool/internal/analyze/validations"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
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
	pipelinenullctx.Analyzer,
	pipelinetag.Analyzer,
	processorfield.Analyzer,
	validations.Analyzer,
}

func getAnalyzer(name string) *analyze.Analyzer {
	for _, a := range analyzers {
		if a.Name == name {
			return a
		}
	}

	return nil
}

func newCmdAnalyze() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analyze ANALYZER[,ANALYZER...] [DIR ...]",
		Short:   "Analyze and fix package issues",
		Aliases: []string{"a"},
		RunE:    doAnalyze,
	}

	cmd.Flags().Bool("fix", false, "apply fixes suggested by the analyzers (if applicable)")
	cmd.Flags().Bool("list", false, "list available analyzers")
	cmd.Flags().StringP("output", "o", analyzeFormatTextColor, "output format (choose from: "+strings.Join(analyzeFormats, ", ")+")")

	for _, a := range analyzers {
		prefix := a.Name + "."
		a.Flags.VisitAll(func(f *pflag.Flag) {
			name := prefix + f.Name
			cmd.Flags().Var(f.Value, name, f.Usage)
		})
	}

	return cmd
}

func printAnalyzers(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 3, ' ', 0)
	for _, a := range analyzers {
		var fixStr string
		if a.CanFix {
			fixStr = "(FIX)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", a.Name, a.Doc, fixStr)
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintln(w, "")
}

func doAnalyze(cmd *cobra.Command, args []string) error {
	if list, _ := cmd.Flags().GetBool("list"); list {
		printAnalyzers(cmd.OutOrStdout())
		return nil
	}
	if len(args) == 0 {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Please provide one or more of the following analyzers:\n\n")
		printAnalyzers(cmd.OutOrStderr())
		return errors.New("no analyzer specified")
	}

	var analyzerNames []string
	var selected []*analyze.Analyzer
	for _, s := range strings.Split(args[0], ",") {
		if l := getAnalyzer(s); l != nil {
			selected = append(selected, l)
			analyzerNames = append(analyzerNames, l.Name)
		} else {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), "Please provide a valid analyzer:\n\n")
			printAnalyzers(cmd.OutOrStderr())
			return fmt.Errorf("invalid analyzer name: %s", s)
		}
	}

	pkgDirs, err := filterPackages(args[1:], cmd.Flags())
	if err != nil {
		return err
	}

	fix, _ := cmd.Flags().GetBool("fix")

	report := analyze.Report{
		Timestamp: time.Now(),
		Analyzers: analyzerNames,
		Reports:   map[string]analyze.PackageReport{},
	}

	for _, pkgDir := range pkgDirs {
		manifest, err := fleetpkg.LoadManifest(pkgDir)
		if err != nil {
			slog.Error("Failed to read package manifest", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}
		pkgReport := analyze.PackageReport{
			Analyzers: map[string]analyze.IssueReport{},
		}
		report.Packages = append(report.Packages, manifest.Name)

		for _, a := range selected {
			pkg, err := fleetpkg.Load(pkgDir, a.LoadMode)
			if err != nil {
				slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
				continue
			}

			ctx := analyze.Pass{
				Package: pkg,
				Fix:     fix,
			}
			if err = a.Run(&ctx); err != nil {
				slog.Error("Failed to run linter", slog.String("package", pkg.Manifest.Name), slog.String("linter", a.Name), slog.String("error", err.Error()))
				continue
			}

			pkgReport.Analyzers[a.Name] = analyze.IssueReport{
				Issues: ctx.Issues,
			}
		}

		report.Reports[manifest.Name] = pkgReport
	}

	output, _ := cmd.Flags().GetString("output")
	switch output {
	case analyzeFormatJSON:
		err = analyze.PrintJSON(os.Stdout, &report)
	case analyzeFormatTextColor:
		err = analyze.PrintText(os.Stdout, &report, true)
	case analyzeFormatText:
		fallthrough
	default:
		err = analyze.PrintText(os.Stdout, &report, false)
	}

	if err != nil {
		return err
	}

	return nil
}
