// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package pipelineeventoriginal provides an analyzer that finds and fixes
// ingest pipeline event.original issues.
package pipelineeventoriginal

import (
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

const Name = "pipeline-event-original"

var Analyzer = &analyze.Analyzer{
	Name:   Name,
	Doc:    "Find and fix ingest pipeline event.original issues",
	CanFix: true,
	Run:    run,
}

func run(pass *analyze.Pass) error {
	for _, ds := range pass.Package.DataStreams {
		for pipelineName, pipeline := range ds.Pipelines {
			isDefault := pipelineName == "default.yml"

			// 1. No remove event.original
			removeEventOriginalIndex := findRemoveEventOriginal(pipeline)
			if removeEventOriginalIndex != -1 {
				pass.Issues = append(pass.Issues, analyze.Issue{
					Analyzer: Name,
					Message:  "Pipeline must not remove event.original",
					Severity: analyze.SeverityWarn,
					Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), pipeline.Processors[removeEventOriginalIndex].Node),
				})
				if pass.Fix {
					if _, err := pipeline.Doc.DeleteNode(pipeline.Processors[removeEventOriginalIndex].Node.GetPath()); err != nil {
						return err
					}
				}
			}

			// 2. Add preserve_original_event to tags if error.message set (only in default pipeline)
			if isDefault {
				appendPreserveTagOnErrorIndex := findAppendPreserveTagOnError(pipeline)
				if appendPreserveTagOnErrorIndex != -1 {
					pass.Issues = append(pass.Issues, analyze.Issue{
						Analyzer: Name,
						Message:  "Default pipeline must append 'preserve_original_event' to tags when error.message is set",
						Severity: analyze.SeverityWarn,
						Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), pipeline.Processors[appendPreserveTagOnErrorIndex].Node),
					})
				}
				if pass.Fix {
					if _, err := pipeline.Doc.AddNode("$.processors", newAppendPreserveTagOnError(), appendPreserveTagOnErrorIndex, true); err != nil {
						return err
					}
				}
			}

			// 3. Add preserve_original_event to tags in on_failure
			appendPreserveTagOnFailureIndex := findAppendPreserveTagOnFailure(pipeline)
			if appendPreserveTagOnFailureIndex != -1 {
				pass.Issues = append(pass.Issues, analyze.Issue{
					Analyzer: Name,
					Message:  "Default pipeline must append 'preserve_original_event' to tags when error.message is set",
					Severity: analyze.SeverityWarn,
					Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), pipeline.Processors[appendPreserveTagOnFailureIndex].Node),
				})
			}
			if pass.Fix {
				if _, err := pipeline.Doc.AddNode("$.on_failure", newAppendPreserveTagOnFailure(), yamledit.IndexAppend, true); err != nil {
					return err
				}
			}

			if pass.Fix && pipeline.Doc.Modified() {
				if err := pipeline.Doc.WriteFile(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func findRemoveEventOriginal(p *fleetpkg.Pipeline) int {
	for i, proc := range p.Processors {
		if proc.Type != "remove" {
			continue
		}
		if s, ok := proc.GetAttributeString("field"); ok && s == "event.original" {
			return i
		}
	}

	return -1
}

func findAppendPreserveTagOnError(p *fleetpkg.Pipeline) int {
	for i, proc := range p.Processors {
		if proc.Type != "append" {
			continue
		}
		if s, ok := proc.GetAttributeString("field"); ok && s == "tags" {
			if s, ok := proc.GetAttributeString("value"); ok && s == "preserve_original_event" {
				return i
			}
		}
	}

	return -1
}

func findAppendPreserveTagOnFailure(p *fleetpkg.Pipeline) int {
	for i, proc := range p.OnFailure {
		if proc.Type != "append" {
			continue
		}
		if s, ok := proc.GetAttributeString("field"); ok && s == "tags" {
			if s, ok := proc.GetAttributeString("value"); ok && s == "preserve_original_event" {
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
