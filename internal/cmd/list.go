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
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taylor-swanson/package-tool/internal/pkg"
)

const (
	listFormatPlain             = "plain"
	listFormatMarkdownList      = "md-list"
	listFormatMarkdownChecklist = "md-checklist"
	listFormatComma             = "comma"
	listFormatCommaLines        = "comma-lines"
	listFormatCustom            = "custom"
)

var listFormats = []string{
	listFormatPlain,
	listFormatMarkdownList,
	listFormatMarkdownChecklist,
	listFormatComma,
	listFormatCustom,
}

func newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List package contents",
		Aliases: []string{"l"},
		RunE:    doList,
	}

	cmd.Flags().String("custom-prefix", "", "Prefix to add for custom output")
	cmd.Flags().String("custom-suffix", "", "Suffix to add for custom output")
	cmd.Flags().Bool("custom-newline", false, "Add new lines for custom output")
	cmd.Flags().StringP("output", "o", listFormatPlain, "Output format (choose from: "+strings.Join(listFormats, ", ")+")")

	return cmd
}

func doList(cmd *cobra.Command, args []string) error {
	pkgDirs, err := filterPackages(args, cmd.Flags())
	if err != nil {
		return err
	}

	var packages []string
	for _, pkgDir := range pkgDirs {
		manifest, err := pkg.ReadManifest(pkgDir)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}

		packages = append(packages, manifest.Name)
	}

	output, _ := cmd.Flags().GetString("output")
	for i, p := range packages {
		switch output {
		case listFormatMarkdownList:
			_, _ = fmt.Println("- " + p)
		case listFormatMarkdownChecklist:
			_, _ = fmt.Println("- [ ] " + p)
		case listFormatComma:
			_, _ = fmt.Print(p)
			if i < len(packages)-1 {
				_, _ = fmt.Print(", ")
			} else {
				_, _ = fmt.Println("")
			}
		case listFormatCommaLines:
			_, _ = fmt.Println(p + ",")
		case listFormatCustom:
			customPrefix, _ := cmd.Flags().GetString("custom-prefix")
			customSuffix, _ := cmd.Flags().GetString("custom-suffix")
			customNewline, _ := cmd.Flags().GetBool("custom-newline")

			_, _ = fmt.Print(customPrefix)
			_, _ = fmt.Print(p)
			_, _ = fmt.Print(customSuffix)
			if customNewline {
				_, _ = fmt.Println()
			}
		case listFormatPlain:
			fallthrough
		default:
			_, _ = fmt.Println(p)
		}
	}

	return nil
}
