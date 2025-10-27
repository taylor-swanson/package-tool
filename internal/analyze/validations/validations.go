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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/internal/pkg"
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
	"SVR00006": "Processor tag missgin",
	"SVR00007": "Processor tag duplicated",
}

func fmtExcludeCheckDesc(id string) string {
	if desc, ok := excludeCheckDesc[id]; ok {
		return fmt.Sprintf("%s: %s", id, desc)
	}

	return id
}

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find usages of validations in packages",
	Run:         run,
}

func run(ctx *analyze.Context) error {
	var validation pkg.Validation
	err := pkg.ReadYAML(filepath.Join(ctx.Package.Path(), "validation.yml"), &validation, true)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	// Exclude checks
	for _, v := range validation.Errors.ExcludeChecks {
		if len(ctx.Args) > 0 {
			if !slices.Contains(ctx.Args, v) {
				continue
			}
		}
		ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
			Category: Name,
			Message:  fmtExcludeCheckDesc(v),
		})
	}

	var showDocsEnforced bool
	if len(ctx.Args) > 0 {
		for _, a := range ctx.Args {
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
			ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
				Category: Name,
				Message:  "Docs structure is not enforced",
			})
		} else {
			for _, skip := range validation.DocsStructureEnforced.Skip {
				if len(ctx.Args) > 0 {
					if !slices.Contains(ctx.Args, skip.Title) {
						continue
					}
				}

				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Category: Name,
					Message:  "Docs structure skip: " + skip.Title,
				})
			}
		}
	}

	return nil
}
