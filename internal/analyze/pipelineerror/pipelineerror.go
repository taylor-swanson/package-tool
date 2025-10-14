package pipelineerror

import (
	"fmt"
	"os"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"

	"package-tool/internal/analyze"
	"package-tool/internal/yamledit"
)

const Name = "pipeline-error"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline error handler issues",
	CanFix:      true,
	Run:         run,
}

func run(ctx *analyze.Context) (analyze.Result, error) {
	var result analyze.Result

	if err := doCheck(ctx, &result); err != nil {
		return result, err
	}

	if ctx.Fix && len(result.Issues) > 0 {
		if err := doFix(ctx, &result); err != nil {
			return result, err
		}
	}

	return result, nil
}

func doCheck(ctx *analyze.Context, result *analyze.Result) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			var hasEventKind bool
			var hasErrorMessage bool

			for _, proc := range pipeline.OnFailure {
				switch proc.Type {
				case "set":
					field, _ := proc.Attributes["field"]
					switch field {
					case "error.message":
						hasErrorMessage = true
						value, _ := proc.Attributes["value"].(string)
						if !strings.Contains(value, "_ingest.on_failure_processor_tag") {
							result.Issues = append(result.Issues, analyze.Issue{
								Pos:      analyze.NewPos(proc.FileMetadata),
								Category: Name,
								Message:  "Pipeline on_failure error message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
					case "event.kind":
						hasEventKind = true
						value, _ := proc.Attributes["value"]
						if value != "pipeline_error" {
							result.Issues = append(result.Issues, analyze.Issue{
								Pos:      analyze.NewPos(proc.FileMetadata),
								Category: Name,
								Message:  "Pipeline on_failure handler must set event.kind to 'pipeline_error'",
							})
						}
					}
				case "append":
					field, _ := proc.Attributes["field"]
					switch field {
					case "error.message":
						hasErrorMessage = true
						value, _ := proc.Attributes["value"].(string)
						if !strings.Contains(value, "_ingest.on_failure_processor_tag") {
							result.Issues = append(result.Issues, analyze.Issue{
								Pos:      analyze.NewPos(proc.FileMetadata),
								Category: Name,
								Message:  "Pipeline on_failure error message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
					}
				}
			}

			var onFailurePos analyze.Pos
			if len(pipeline.OnFailure) > 0 {
				onFailurePos = analyze.NewPos(pipeline.OnFailure[0].FileMetadata)
			} else {
				onFailurePos = analyze.Pos{File: pipeline.Path()}
				result.Issues = append(result.Issues, analyze.Issue{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline must include on_failure handler",
				})
			}

			if !hasErrorMessage {
				result.Issues = append(result.Issues, analyze.Issue{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline on_failure handler must set error.message",
				})
			}
			if !hasEventKind {
				result.Issues = append(result.Issues, analyze.Issue{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline on_failure handler must set event.kind to 'pipeline_error'",
				})
			}
		}
	}

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

			modified := false
			onFailureNode := getSequenceNode(f, "$.on_failure")
			if onFailureNode == nil {
				n := f.Docs[0].Body.(*ast.MappingNode)
				newNode := newPipelineOnFailureNode()
				newNode.AddColumn(n.GetToken().Position.IndentNum)

				n.Values = append(n.Values, newNode)

				modified = true
			} else {
				// 1. Add append tags preserve_original_event to processors when error.message is set
				appendPreserveOnFailureIndex := findAppendPreserveFailureProcessor(&pipeline)
				if yamledit.AppendOrReplaceNode(onFailureNode, appendPreserveOnFailureIndex, newPipelineOnFailureAppendPreserveOriginalEvent()) {
					modified = true
				}

				// 2. Check event.kind: pipeline_error
				setEventKindIndex := findPipelineOnFailureSetEventKind(&pipeline)
				if yamledit.AppendOrReplaceNode(onFailureNode, setEventKindIndex, newPipelineOnFailureSetEventKindNode()) {
					fmt.Println("  Modified event.kind")
					modified = true
				}

				// 3. Check error.message
				errorMessageIndex, errorMessageType := findPipelineOnFailureSetErrorMessage(&pipeline)
				if errorMessageType == "append" {
					if yamledit.AppendOrReplaceNode(onFailureNode, errorMessageIndex, newPipelineOnFailureSetErrorMessageNode("append")) {
						fmt.Println("  Modified error.message (append)")
						modified = true
					}
				} else {
					if yamledit.AppendOrReplaceNode(onFailureNode, errorMessageIndex, newPipelineOnFailureSetErrorMessageNode("set")) {
						fmt.Println("  Modified error.message (set)")
						modified = true
					}
				}

				// 4. Delete remove event.original if in processor list
				if removeEventOriginalIndex := findRemoveEventOriginal(&pipeline); removeEventOriginalIndex != -1 {
					if yamledit.RemoveNode(processorsNode, removeEventOriginalIndex) {
						modified = true
					}
				}

				// 5. Add append tags preserve_original_event to on_failure
				appendPreserveIndex := findAppendPreserveProcessor(&pipeline)
				if yamledit.AppendOrReplaceNode(processorsNode, appendPreserveIndex, newPipelineAppendPreserveOriginalEvent()) {
					modified = true
				}

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

func findPipelineOnFailureSetEventKind(pipeline *fleetpkg.IngestPipeline) int {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" {
			if s, ok := proc.Attributes["field"].(string); ok && s == "event.kind" {
				return i
			}
		}
	}

	return -1
}

func findPipelineOnFailureSetErrorMessage(pipeline *fleetpkg.IngestPipeline) (int, string) {
	for i, proc := range pipeline.OnFailure {
		if proc.Type == "set" || proc.Type == "append" {
			if s, ok := proc.Attributes["field"].(string); ok && s == "error.message" {
				return i, proc.Type
			}
		}
	}

	return -1, ""
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

func newPipelineOnFailureNode() *ast.MappingValueNode {
	f, err := parser.ParseBytes([]byte(`
on_failure:
  - append:
      field: tags
      value: preserve_original_event
      allow_duplicates: false
  - set:
      field: event.kind
      value: pipeline_error
  - set:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        {{#_ingest.on_failure_processor_tag}}with tag '{{{ _ingest.on_failure_processor_tag }}}'
        {{/_ingest.on_failure_processor_tag}}failed with message '{{{ _ingest.on_failure_message }}}'
`), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body.(*ast.MappingNode).Values[0]
}

func newPipelineOnFailureSetErrorMessageNode(procType string) ast.Node {
	f, err := parser.ParseBytes([]byte(fmt.Sprintf(`
%s:
  field: error.message
  value: >-
    Processor '{{{ _ingest.on_failure_processor_type }}}'
    {{#_ingest.on_failure_processor_tag}}with tag '{{{ _ingest.on_failure_processor_tag }}}'
    {{/_ingest.on_failure_processor_tag}}failed with message '{{{ _ingest.on_failure_message }}}'
`, procType)), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return f.Docs[0].Body
}

func newPipelineOnFailureSetEventKindNode() ast.Node {
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
