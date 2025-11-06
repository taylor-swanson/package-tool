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

// Package pipelineerror provides an analyzer that finds and fixes ingest
// pipeline error handler issues.
package pipelineerror

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
	"github.com/taylor-swanson/package-tool/pkg/yamledit"

	"github.com/taylor-swanson/package-tool/internal/analyze"
)

const Name = "pipeline-error"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline error handler issues",
	CanFix:      true,
	Run:         run,
}

func run(ctx *analyze.Context) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			if len(pipeline.OnFailure) == 0 {
				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Pos:      &analyze.Pos{File: pipeline.Path()},
					Category: Name,
					Message:  "Pipeline must include on_failure handler",
				})

				if ctx.Fix {
					_, err := pipeline.Doc.SetKeyNode("$", "on_failure", newOnFailure(), yamledit.IndexAppend)
					if err != nil {
						return err
					}
					ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
						Category: Name,
						Message:  "Added pipeline on_failure handler",
					})
				}
			} else {
				// Check set event.kind
				setEventKindIndex := findSetEventKind(pipeline)
				if setEventKindIndex != -1 {
					if s, ok := pipeline.OnFailure[setEventKindIndex].GetAttributeString("value"); !ok || s != "pipeline_error" {
						ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
							Pos:      analyze.NewPos(pipeline.OnFailure[setEventKindIndex].Node, pipeline.Path()),
							Category: Name,
							Message:  "Pipeline on_failure must set event.kind to 'pipeline_error'",
						})
					}
				} else {
					ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
						Pos:      analyze.MustPosFromPath(pipeline.Doc, "$.on_failure"),
						Category: Name,
						Message:  "Pipeline on_failure must set event.kind to 'pipeline_error'",
					})
				}
				if ctx.Fix {
					modified, err := pipeline.Doc.AddNode("$.on_failure", newSetEventKind(), setEventKindIndex, true)
					if err != nil {
						return err
					}
					if modified {
						ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
							Category: Name,
							Message:  "Modified set event.kind",
						})
					}
				}

				// Check set error.message
				setErrorMessageIndex, setErrorMessageType := findSetErrorMessage(pipeline)
				if setErrorMessageIndex != -1 {
					s, ok := pipeline.OnFailure[setErrorMessageIndex].GetAttributeString("value")
					if !ok {
						ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
							Pos:      analyze.NewPos(pipeline.OnFailure[setErrorMessageIndex].Node, pipeline.Path()),
							Category: Name,
							Message:  "Pipeline on_failure error.message must be set to a string value",
						})
					} else {
						if !strings.Contains(s, "_ingest.on_failure_processor_tag") {
							ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
								Pos:      analyze.NewPos(pipeline.OnFailure[setErrorMessageIndex].Node, pipeline.Path()),
								Category: Name,
								Message:  "Pipeline on_failure error.message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
						if !strings.Contains(s, "_ingest.pipeline") {
							ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
								Pos:      analyze.NewPos(pipeline.OnFailure[setErrorMessageIndex].Node, pipeline.Path()),
								Category: Name,
								Message:  "Pipeline on_failure error.message must include the pipeline name ('_ingest.pipeline')",
							})
						}
					}
				} else {
					setErrorMessageType = "set"
					ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
						Pos:      analyze.MustPosFromPath(pipeline.Doc, "$.on_failure"),
						Category: Name,
						Message:  "Pipeline on_failure must set error.message",
					})
				}
				if ctx.Fix {
					modified, err := pipeline.Doc.AddNode("$.on_failure", newSetErrorMessage(setErrorMessageType), setErrorMessageIndex, true)
					if err != nil {
						return err
					}
					if modified {
						ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
							Category: Name,
							Message:  fmt.Sprintf("Modified error.message (%s)", setErrorMessageType),
						})
					}
				}
			}

			if ctx.Fix && pipeline.Doc.Modified() {
				if err := pipeline.Doc.WriteFile(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func findSetEventKind(pipeline *fleetpkg.Pipeline) int {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" {
			if s, ok := proc.GetAttributeString("field"); ok && s == "event.kind" {
				return i
			}
		}
	}

	return -1
}

func findSetErrorMessage(pipeline *fleetpkg.Pipeline) (int, string) {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" || proc.Type == "append" {
			if s, ok := proc.GetAttributeString("field"); ok && s == "error.message" {
				return i, proc.Type
			}
		}
	}

	return -1, ""
}

func newOnFailure() *ast.MappingValueNode {
	f, err := parser.ParseBytes([]byte(`
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
  - set:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        {{#_ingest.on_failure_processor_tag}}with tag '{{{ _ingest.on_failure_processor_tag }}}'
        {{/_ingest.on_failure_processor_tag}}in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
`), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body.(*ast.MappingNode).Values[0]
}

func newSetErrorMessage(procType string) ast.Node {
	f, err := parser.ParseBytes([]byte(fmt.Sprintf(`
%s:
  field: error.message
  value: >-
    Processor '{{{ _ingest.on_failure_processor_type }}}'
    {{#_ingest.on_failure_processor_tag}}with tag '{{{ _ingest.on_failure_processor_tag }}}'
    {{/_ingest.on_failure_processor_tag}}in pipeline '{{{ _ingest.pipeline }}}'
    failed with message '{{{ _ingest.on_failure_message }}}'
`, procType)), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body
}

func newSetEventKind() ast.Node {
	f, err := parser.ParseBytes([]byte(`
set:
  field: event.kind
  value: pipeline_error
`), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body
}
