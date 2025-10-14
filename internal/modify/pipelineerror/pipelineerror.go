package pipelineerror

import (
	"fmt"
	"os"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"

	"package-tool/internal/modify"
	"package-tool/internal/yamledit"
)

const Name = "pipeline-error"

var Modifier = &modify.Modifier{
	Name:        Name,
	Description: "Fix ingest pipeline error handler issues",
	Run:         run,
}

func run(pkg *fleetpkg.Integration) error {
	for _, ds := range pkg.DataStreams {
		for _, pipeline := range ds.Pipelines {
			f, err := parser.ParseFile(pipeline.Path(), parser.ParseComments)
			if err != nil {
				return fmt.Errorf("failed to parse pipeline %q: %w", pipeline.Path(), err)
			}

			modified := false
			onFailureNode := getOnFailureNode(f)
			if onFailureNode == nil {
				n := f.Docs[0].Body.(*ast.MappingNode)
				newNode := newPipelineOnFailureNode()
				newNode.AddColumn(n.GetToken().Position.IndentNum)

				n.Values = append(n.Values, newNode)

				modified = true
			} else {
				// 1. Check event.kind: pipeline_error
				setEventKindIndex := findPipelineOnFailureSetEventKind(&pipeline)
				if yamledit.AppendOrReplaceNode(onFailureNode, setEventKindIndex, newPipelineOnFailureSetEventKindNode()) {
					fmt.Println("  Modified event.kind")
					modified = true
				}

				// 2. Check error.message
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

func getOnFailureNode(f *ast.File) *ast.SequenceNode {
	p, err := yaml.PathString("$.on_failure")
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

func newPipelineOnFailureNode() *ast.MappingValueNode {
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
