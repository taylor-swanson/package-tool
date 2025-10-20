package pipelineeventoriginal

import (
	"fmt"
	"os"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"

	"package-tool/internal/analyze"
	"package-tool/internal/yamledit"
)

const Name = "pipeline-event-original"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline event.original issues",
	CanFix:      true,
	Run:         run,
}

func run(ctx *analyze.Context) (analyze.Result, error) {
	var result analyze.Result

	if err := doCheck(ctx, &result); err != nil {
		return result, err
	}

	if ctx.Fix && len(result.Findings) > 0 {
		if err := doFix(ctx, &result); err != nil {
			return result, err
		}
	}

	return result, nil
}

func doCheck(ctx *analyze.Context, result *analyze.Result) error {
	// TODO

	return nil
}

func doFix(ctx *analyze.Context, result *analyze.Result) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			f, err := parser.ParseFile(pipeline.Path(), parser.ParseComments)
			if err != nil {
				return fmt.Errorf("failed to parse pipeline %q: %w", pipeline.Path(), err)
			}

			processorsNode := getSequenceNode(f, "$.processors")
			if processorsNode == nil {
				return nil
			}
			onFailureNode := getSequenceNode(f, "$.on_failure")
			if onFailureNode == nil {
				return nil
			}

			modified := false

			// 1. Delete remove event.original if in processor list
			if removeEventOriginalIndex := findRemoveEventOriginal(&pipeline); removeEventOriginalIndex != -1 {
				if yamledit.RemoveNode(processorsNode, removeEventOriginalIndex) {
					modified = true
				}
			}
			// 2. Add append tags preserve_original_event to on_failure
			appendPreserveIndex := findAppendPreserveProcessor(&pipeline)
			if yamledit.AppendOrReplaceNode(processorsNode, appendPreserveIndex, newPipelineAppendPreserveOriginalEvent()) {
				modified = true
			}
			// 3. Add append tags preserve_original_event to processors when error.message is set
			appendPreserveOnFailureIndex := findAppendPreserveFailureProcessor(&pipeline)
			if yamledit.AppendOrReplaceNode(onFailureNode, appendPreserveOnFailureIndex, newPipelineOnFailureAppendPreserveOriginalEvent()) {
				modified = true
			}

			if !modified {
				continue
			}

			p := printer.Printer{}
			d := p.PrintNode(f.Docs[0])

			if err = os.WriteFile(pipeline.Path(), d, 0o644); err != nil {
				return fmt.Errorf("failed to write pipeline %q: %w", pipeline.Path(), err)
			}
		}
	}

	return nil
}

func getSequenceNode(f *ast.File, yamlPath string) *ast.SequenceNode {
	p, err := yaml.PathString(yamlPath)
	if err != nil {
		panic(err)
	}

	n, err := p.FilterFile(f)
	if yaml.IsNotFoundNodeError(err) {
		return nil
	}
	if err != nil {
		panic(err)
	}

	return n.(*ast.SequenceNode)
}

func findAppendPreserveProcessor(p *fleetpkg.IngestPipeline) int {
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

func findAppendPreserveFailureProcessor(p *fleetpkg.IngestPipeline) int {
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

func newPipelineOnFailureAppendPreserveOriginalEvent() ast.Node {
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

func newPipelineAppendPreserveOriginalEvent() ast.Node {
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
