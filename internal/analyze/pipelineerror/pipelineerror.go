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

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/internal/yamledit"
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
			var pipelineAST analyze.AST
			var err error

			if ctx.Fix {
				if pipelineAST, err = analyze.LoadAST(pipeline.Path()); err != nil {
					return err
				}
			}

			if len(pipeline.OnFailure) == 0 {
				onFailurePos := analyze.Pos{File: pipeline.Path()}
				ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline must include on_failure handler",
				})

				if ctx.Fix {
					n := pipelineAST.File.Docs[0].Body.(*ast.MappingNode)
					newNode := newOnFailure()
					newNode.AddColumn(n.GetToken().Position.IndentNum)

					n.Values = append(n.Values, newNode)
					ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
						Category: Name,
						Message:  "Added pipeline on_failure handler",
					})

					pipelineAST.Modified = true
				}
			} else {
				// Check set event.kind
				setEventKindIndex := findSetEventKind(&pipeline)
				if setEventKindIndex != -1 {
					if s, ok := pipeline.OnFailure[setEventKindIndex].Attributes["value"].(string); ok && s != "pipeline_error" {
						ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
							Pos:      analyze.Pos{},
							Category: Name,
							Message:  "Pipeline on_failure must set event.kind to 'pipeline_error'",
						})
					}
				} else {
					ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
						Pos:      analyze.Pos{},
						Category: Name,
						Message:  "Pipeline on_failure must set event.kind to 'pipeline_error'",
					})
				}
				if ctx.Fix {
					onFailureNode := yamledit.GetSequenceNode(pipelineAST.File, "$.on_failure")
					if yamledit.AppendOrReplaceNode(onFailureNode, setEventKindIndex, newSetEventKind()) {
						ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
							Category: Name,
							Message:  "Modified set event.kind",
						})
						pipelineAST.Modified = true
					}
				}

				// Check set error.message
				setErrorMessageIndex, setErrorMessageType := findSetErrorMessage(&pipeline)
				if setErrorMessageIndex != -1 {
					s, ok := pipeline.OnFailure[setErrorMessageIndex].Attributes["value"].(string)
					if !ok {
						ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
							Pos:      analyze.Pos{},
							Category: Name,
							Message:  "Pipeline on_failure error.message must be set to a string value",
						})
					} else {
						if !strings.Contains(s, "_ingest.on_failure_processor_tag") {
							ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
								Pos:      analyze.Pos{},
								Category: Name,
								Message:  "Pipeline on_failure error.message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
						if !strings.Contains(s, "_ingest.pipeline") {
							ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
								Pos:      analyze.Pos{},
								Category: Name,
								Message:  "Pipeline on_failure error.message must include the pipeline name ('_ingest.pipeline')",
							})
						}
					}
				} else {
					setErrorMessageType = "set"
					ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
						Pos:      analyze.Pos{},
						Category: Name,
						Message:  "Pipeline on_failure must set error.message",
					})
				}
				if ctx.Fix {
					onFailureNode := yamledit.GetSequenceNode(pipelineAST.File, "$.on_failure")
					if yamledit.AppendOrReplaceNode(onFailureNode, setErrorMessageIndex, newSetErrorMessage(setErrorMessageType)) {
						ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
							Category: Name,
							Message:  fmt.Sprintf("Modified error.message (%s)", setErrorMessageType),
						})
						pipelineAST.Modified = true
					}
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

func findSetEventKind(pipeline *fleetpkg.IngestPipeline) int {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" {
			if s, ok := proc.Attributes["field"].(string); ok && s == "event.kind" {
				return i
			}
		}
	}

	return -1
}

func findSetErrorMessage(pipeline *fleetpkg.IngestPipeline) (int, string) {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" || proc.Type == "append" {
			if s, ok := proc.Attributes["field"].(string); ok && s == "error.message" {
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
