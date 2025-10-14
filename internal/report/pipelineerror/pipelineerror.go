package pipelineerror

import (
	"strings"

	"github.com/andrewkroh/go-fleetpkg"

	"package-tool/internal/report"
)

const Name = "pipeline-error"

var Reporter = &report.Reporter{
	Name:        Name,
	Description: "Check for ingest pipeline error handler issues",
	Run:         run,
}

func run(pkg *fleetpkg.Integration) (report.Result, error) {
	var result report.Result

	for _, ds := range pkg.DataStreams {
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
							result.Issues = append(result.Issues, report.Diagnostic{
								Pos:      report.NewPos(proc.FileMetadata),
								Category: Name,
								Message:  "Pipeline on_failure error message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
					case "event.kind":
						hasEventKind = true
						value, _ := proc.Attributes["value"]
						if value != "pipeline_error" {
							result.Issues = append(result.Issues, report.Diagnostic{
								Pos:      report.NewPos(proc.FileMetadata),
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
							result.Issues = append(result.Issues, report.Diagnostic{
								Pos:      report.NewPos(proc.FileMetadata),
								Category: Name,
								Message:  "Pipeline on_failure error message must include the processor tag ('_ingest.on_failure_processor_tag')",
							})
						}
					}
				}
			}

			var onFailurePos report.Pos
			if len(pipeline.OnFailure) > 0 {
				onFailurePos = report.NewPos(pipeline.OnFailure[0].FileMetadata)
			} else {
				onFailurePos = report.Pos{File: pipeline.Path()}
				result.Issues = append(result.Issues, report.Diagnostic{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline must include on_failure handler",
				})
			}

			if !hasErrorMessage {
				result.Issues = append(result.Issues, report.Diagnostic{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline on_failure handler must set error.message",
				})
			}
			if !hasEventKind {
				result.Issues = append(result.Issues, report.Diagnostic{
					Pos:      onFailurePos,
					Category: Name,
					Message:  "Pipeline on_failure handler must set event.kind to 'pipeline_error'",
				})
			}
		}
	}

	return result, nil
}
