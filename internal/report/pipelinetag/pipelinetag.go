package pipelinetag

import (
	"fmt"

	"github.com/andrewkroh/go-fleetpkg"

	"package-tool/internal/report"
)

const Name = "pipeline-tag"

var Reporter = &report.Reporter{
	Name:        Name,
	Description: "Check for ingest pipeline tag issues",
	Run:         run,
}

func run(pkg *fleetpkg.Integration) (report.Result, error) {
	var result report.Result

	for _, ds := range pkg.DataStreams {
		for _, pipeline := range ds.Pipelines {
			type ProcessorInfo struct {
				Index     int
				Processor *fleetpkg.Processor
			}

			seen := map[string]ProcessorInfo{}

			for i, proc := range pipeline.Processors {
				raw, ok := proc.Attributes["tag"]
				if !ok {
					result.Issues = append(result.Issues, report.Diagnostic{
						Pos:      report.NewPos(proc.FileMetadata),
						Category: Name,
						Message:  fmt.Sprintf("Missing tag on %s processor at index %d", proc.Type, i),
					})
					continue
				}
				tag, ok := raw.(string)
				if !ok {
					result.Issues = append(result.Issues, report.Diagnostic{
						Pos:      report.NewPos(proc.FileMetadata),
						Category: Name,
						Message:  fmt.Sprintf("Invalid tag type (got: %T want: string) on %s processor at index %d", raw, proc.Type, i),
					})
					continue
				}

				if other, dup := seen[tag]; dup {
					result.Issues = append(result.Issues, report.Diagnostic{
						Pos:      report.NewPos(proc.FileMetadata),
						Category: Name,
						Message:  fmt.Sprintf("Duplicate tag %q on %s processor index %d", tag, proc.Type, i),
						Related: []report.Related{{
							Pos:     report.NewPos(other.Processor.FileMetadata),
							Message: fmt.Sprintf("Processor first seen at index %d", other.Index),
						}},
					})
				}

				seen[tag] = ProcessorInfo{
					Index:     i,
					Processor: proc,
				}
			}
		}
	}

	return result, nil
}
