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

// Package validations that finds usages of package validations in packages.
package validations

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

const Name = "validations"

var excludeCheckDesc = map[string]string{
	"JSE00001": "Rename message to event.original",
	"PSR00001": "Package with GA version is using an unreleased version of the spec",
	"PSR00002": "Package with non-stable semantic version and active beta features can't be released as stable version",
	"SVR00001": "Saved query, but not filter",
	"SVR00002": "Mandatory filters in dashboard",
	"SVR00003": "Dangling object IDs",
	"SVR00004": "References in dashboard",
	"SVR00005": "Kibana version for saved tags",
	"SVR00006": "Processor tag missing",
}

func fmtExcludeCheckDesc(id string) string {
	if desc, ok := excludeCheckDesc[id]; ok {
		return fmt.Sprintf("%s: %s", id, desc)
	}

	return id
}

var Analyzer = &analyze.Analyzer{
	Name: Name,
	Doc:  "Find usages of validations in packages",
	Run:  run,
}

var filters []string

func init() {
	Analyzer.Flags.StringSliceVar(&filters, "filters", nil, "comma separated list of validation codes")
}

func run(pass *analyze.Pass) error {
	var validation fleetpkg.Validation
	_, err := yamledit.ParseDocumentFile(filepath.Join(pass.Package.Path(), "validation.yml"), &validation)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	// Exclude checks
	for i, v := range validation.Errors.ExcludeChecks {
		if len(filters) > 0 {
			if !slices.Contains(filters, v) {
				continue
			}
		}
		pos, err := analyze.NewYAMLPathPosition(validation.Doc.AST(), fmt.Sprintf("$.errors.exclude_checks[%d]", i))
		if err != nil {
			return err
		}
		pass.Issues = append(pass.Issues, analyze.Issue{
			Analyzer: Name,
			Message:  fmtExcludeCheckDesc(v),
			Severity: analyze.SeverityWarn,
			Pos:      pos,
		})
	}

	var showDocsEnforced bool
	if len(filters) > 0 {
		for _, a := range filters {
			if strings.HasPrefix(a, "DOCS") {
				showDocsEnforced = true
				break
			}
		}
	} else {
		showDocsEnforced = true
	}

	// Docs structure
	if showDocsEnforced {
		if !validation.DocsStructureEnforced.Enabled {
			var pos token.Position
			if pos, err = analyze.NewYAMLPathPosition(validation.Doc.AST(), "$.docs_structure_enforced.enabled"); err != nil {
				pos = token.Position{Filename: validation.Doc.Filename()}
			}

			pass.Issues = append(pass.Issues, analyze.Issue{
				Analyzer: Name,
				Message:  "Docs structure is not enforced",
				Severity: analyze.SeverityWarn,
				Pos:      pos,
			})
		} else {
			for i, skip := range validation.DocsStructureEnforced.Skip {
				if len(filters) > 0 {
					if !slices.Contains(filters, skip.Title) {
						continue
					}
				}

				pos, err := analyze.NewYAMLPathPosition(validation.Doc.AST(), fmt.Sprintf("$.docs_structure_enforced.skip[%d]", i))
				if err != nil {
					return err
				}
				pass.Issues = append(pass.Issues, analyze.Issue{
					Analyzer: Name,
					Message:  "Docs structure skip: " + skip.Title,
					Severity: analyze.SeverityWarn,
					Pos:      pos,
				})
			}
		}
	}

	return nil
}
