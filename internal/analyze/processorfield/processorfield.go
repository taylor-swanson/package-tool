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

// Package processorfield provides an analyzer that finds usages of fields in
// ingest pipeline processors.
package processorfield

import (
	"errors"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

const Name = "processor-field"

var Analyzer = &analyze.Analyzer{
	Name: Name,
	Doc:  "Find usages of fields in ingest pipeline processors",
	Run:  run,
}

var findKeys = []string{
	"field",
	"target_field",
	"copy_from",
}

var fieldName string

func init() {
	Analyzer.Flags.StringVar(&fieldName, "name", "", "the name of the field")
}

func run(pass *analyze.Pass) error {
	if fieldName == "" {
		return errors.New("field name is required")
	}

	for _, ds := range pass.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				pass.Issues = append(pass.Issues, findInProcessor(pipeline, proc)...)
			}
			for _, proc := range pipeline.OnFailure {
				pass.Issues = append(pass.Issues, findInProcessor(pipeline, proc)...)
			}
		}
	}

	return nil
}

func findInProcessor(pipeline *fleetpkg.Pipeline, proc *fleetpkg.Processor) []analyze.Issue {
	var issues []analyze.Issue

	for _, k := range findKeys {
		if s, ok := proc.Attributes[k]; ok && fieldName == s {
			issues = append(issues, analyze.Issue{
				Analyzer: Name,
				Message:  "Found usage",
				Severity: analyze.SeverityInfo,
				Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), proc.Node),
				Related:  nil,
			})
			break
		}
	}

	return issues
}
