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

func run(ctx *analyze.Context) error {
	if len(ctx.Args) == 0 {
		return nil
	}
	field := ctx.Args[0]

	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				findInProcessor(ctx, proc, field)
			}
			for _, proc := range pipeline.OnFailure {
				findInProcessor(ctx, proc, field)
			}
		}
	}

	return nil
}

func findInProcessor(ctx *analyze.Context, proc *fleetpkg.Processor, field string) {
	for _, k := range findKeys {
		if s, ok := proc.Attributes[k]; ok && field == s {
			ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(proc.FileMetadata),
				Category: Name,
				Message:  "Found usage",
			})
			break
		}
	}
}
