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
	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

const Name = "processor-field"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find usages of fields in ingest pipeline processors",
	Run:         run,
}

var findKeys = []string{
	"field",
	"target_field",
	"copy_from",
}

func run(ctx *analyze.Context) error {
	if len(ctx.Args) == 0 {
		return nil
	}
	field := ctx.Args[0]

	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				findInProcessor(ctx, pipeline, proc, field)
			}
			for _, proc := range pipeline.OnFailure {
				findInProcessor(ctx, pipeline, proc, field)
			}
		}
	}

	return nil
}

func findInProcessor(ctx *analyze.Context, pipeline *fleetpkg.Pipeline, proc *fleetpkg.Processor, field string) {
	for _, k := range findKeys {
		if s, ok := proc.Attributes[k]; ok && field == s {
			ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
				Pos:      analyze.NewPos(proc.Node, pipeline.Path()),
				Category: Name,
				Message:  "Found usage",
			})
			break
		}
	}
}
