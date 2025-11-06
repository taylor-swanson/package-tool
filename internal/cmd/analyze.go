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
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

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
		Use:     "analyze ANALYZER [DIR ...]",
		Short:   "Analyze and fix package issues",
		Aliases: []string{"a"},
		RunE:    doAnalyze,
	}

	cmd.Flags().Bool("fix", false, "apply fixes suggested by analyzer (if applicable)")
	cmd.Flags().Bool("list", false, "list available analyzers")
	cmd.Flags().Bool("exclude-no-findings", false, "exclude packages with no findings from output")
	cmd.Flags().StringP("output", "o", analyzeFormatTextColor, "output format (choose from: "+strings.Join(analyzeFormats, ", ")+")")
	cmd.Flags().StringSliceP("args", "a", nil, "extra arguments to pass to analyzer")

	return cmd
}

func printAnalyzers() {
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
	if list, _ := cmd.Flags().GetBool("list"); list {
		printAnalyzers()
		return nil
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Please provide one of the following analyzers:")
		_, _ = fmt.Fprintln(os.Stderr, "")
		printAnalyzers()
		return errors.New("no analyzer specified")
	}

	analyzer, ok := getAnalyzer(args[0])
	if !ok {
		_, _ = fmt.Fprintln(os.Stderr, "Please provide one of the following analyzers:")
		_, _ = fmt.Fprintln(os.Stderr, "")
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
		pkg, err := fleetpkg.Load(pkgDir)
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

	if excludeEmpty, _ := cmd.Flags().GetBool("exclude-no-findings"); excludeEmpty {
		for k, v := range results {
			if len(v.Findings) == 0 && len(v.Fixes) == 0 {
				delete(results, k)
			}
		}
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
