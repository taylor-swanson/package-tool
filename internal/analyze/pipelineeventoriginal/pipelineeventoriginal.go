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

// Package pipelineeventoriginal provides an analyzer that finds and fixes
// ingest pipeline event.original issues.
package pipelineeventoriginal

import (
	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/internal/yamledit"
)

const Name = "pipeline-event-original"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline event.original issues",
	CanFix:      true,
	Run:         run,
}

func run(ctx *analyze.Context) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			var pipelineAST analyze.AST
			var err error

			if ctx.Fix {
				if pipelineAST, err = analyze.LoadAST(pipeline.Path()); err != nil {
					return err
				}
			}

			// 1. No remove event.original
			removeEventOriginalIndex := findRemoveEventOriginal(&pipeline)
			if removeEventOriginalIndex != -1 {
				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Pos:      analyze.Pos{}, // TODO: Pos
					Category: Name,
					Message:  "Pipeline must not remove event.original",
				})

				if ctx.Fix {
					processorsNode := yamledit.GetSequenceNode(pipelineAST.File, "$.processors")
					yamledit.RemoveNode(processorsNode, removeEventOriginalIndex)
					ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
						Category: Name,
						Message:  "Removed processor to remove event.original",
					})
					pipelineAST.Modified = true
				}
			}

			// 2. Add preserve_original_event to tags if error.message set
			appendPreserveTagOnErrorIndex := findAppendPreserveTagOnError(&pipeline)
			if appendPreserveTagOnErrorIndex != -1 {
				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Pos:      analyze.Pos{}, // TODO: Pos
					Category: Name,
					Message:  "Pipeline must append 'preserve_original_event' to tags when error.message is set",
				})
			}
			if ctx.Fix {
				processorsNode := yamledit.GetSequenceNode(pipelineAST.File, "$.processors")
				if yamledit.AppendOrReplaceNode(processorsNode, appendPreserveTagOnErrorIndex, newAppendPreserveTagOnError()) {
					ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
						Category: Name,
						Message:  "Modified or added append preserve_original_event to tags when error.message is set",
					})
					pipelineAST.Modified = true
				}
			}

			// 3. Add preserve_original_event to tags in on_failure
			appendPreserveTagOnFailureIndex := findAppendPreserveTagOnFailure(&pipeline)
			if appendPreserveTagOnFailureIndex != -1 {
				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Pos:      analyze.Pos{}, // TODO: Pos
					Category: Name,
					Message:  "Pipeline must append 'preserve_original_event' to tags when error.message is set",
				})
			}
			if ctx.Fix {
				onFailureNode := yamledit.GetSequenceNode(pipelineAST.File, "$.on_failure")
				if yamledit.AppendOrReplaceNode(onFailureNode, appendPreserveTagOnFailureIndex, newAppendPreserveTagOnFailure()) {
					ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
						Category: Name,
						Message:  "Modified or added append preserve_original_event to tags in pipeline on_failure",
					})
					pipelineAST.Modified = true
				}
			}

			if ctx.Fix && pipelineAST.Modified {
				if err = pipelineAST.WriteFile(pipeline.Path()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func findRemoveEventOriginal(p *fleetpkg.IngestPipeline) int {
	for i, proc := range p.Processors {
		if proc.Type != "remove" {
			continue
		}
		if s, ok := proc.Attributes["field"].(string); ok && s == "event.original" {
			return i
		}
	}

	return -1
}

func findAppendPreserveTagOnError(p *fleetpkg.IngestPipeline) int {
	for i, proc := range p.Processors {
		if proc.Type != "append" {
			continue
		}
		if s, ok := proc.Attributes["field"].(string); ok && s == "tags" {
			if s, ok := proc.Attributes["value"].(string); ok && s == "preserve_original_event" {
				return i
			}
		}
	}

	return -1
}

func findAppendPreserveTagOnFailure(p *fleetpkg.IngestPipeline) int {
	for i, proc := range p.OnFailure {
		if proc.Type != "append" {
			continue
		}
		if s, ok := proc.Attributes["field"].(string); ok && s == "tags" {
			if s, ok := proc.Attributes["value"].(string); ok && s == "preserve_original_event" {
				return i
			}
		}
	}

	return -1
}

func newAppendPreserveTagOnError() ast.Node {
	f, err := parser.ParseBytes([]byte(`
append:
  tag: append_preserve_original_event_on_error
  field: tags
  value: preserve_original_event
  allow_duplicates: false
  if: ctx.error?.message != null
`), parser.ParseComments)

	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body
}

func newAppendPreserveTagOnFailure() ast.Node {
	f, err := parser.ParseBytes([]byte(`
append:
  field: tags
  value: preserve_original_event
  allow_duplicates: false
`), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body
}
