package processorfield

import (
	"package-tool/internal/analyze"

	"github.com/andrewkroh/go-fleetpkg"
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

func run(ctx *analyze.Context) (analyze.Result, error) {
	var result analyze.Result

	if len(ctx.Args) == 0 {
		return result, nil
	}
	field := ctx.Args[0]

	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				findInProcessor(proc, field, &result)
			}
			for _, proc := range pipeline.OnFailure {
				findInProcessor(proc, field, &result)
			}
		}
	}

	return result, nil
}

func findInProcessor(proc *fleetpkg.Processor, field string, result *analyze.Result) {
	for _, k := range findKeys {
		if s, ok := proc.Attributes[k]; ok && field == s {
			result.Findings = append(result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(proc.FileMetadata),
				Category: Name,
				Message:  "Found usage",
			})
			break
		}
	}
}
