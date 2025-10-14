package duplicateprocessor

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"

	"package-tool/internal/report"
)

const Name = "duplicate-processor"

var Reporter = &report.Reporter{
	Name:        Name,
	Description: "Check for duplicated processors in pipelines",
	Run:         run,
}

func run(pkg *fleetpkg.Integration) (report.Result, error) {
	var result report.Result

	type ProcessorInfo struct {
		Hash      uint64
		Processor *fleetpkg.Processor
		Pipeline  string
		Index     int
	}

	for dsName, ds := range pkg.DataStreams {
		seen := map[uint64]ProcessorInfo{}

		var pipelines []fleetpkg.IngestPipeline
		pipelines = append(pipelines, ds.Pipelines["default.yml"])
		for filename, pipeline := range ds.Pipelines {
			if filename == "default.yml" {
				continue
			}
			pipelines = append(pipelines, pipeline)
		}

		for _, pipeline := range pipelines {
			for i, proc := range pipeline.Processors {

				hash, err := generateProcessorHash(proc)
				if err != nil {
					return result, fmt.Errorf("unable to hash processor at index %d in pipeline %q: %w", i, pipeline.Path(), err)
				}

				if other, dup := seen[hash]; dup {
					result.Issues = append(result.Issues, report.Diagnostic{
						Pos:      report.NewPos(proc.FileMetadata),
						Category: Name,
						Message:  fmt.Sprintf("Processor %s in %s at index %d already exists in %s at index %d in data stream %s", proc.Type, filepath.Base(pipeline.Path()), i, other.Pipeline, other.Index, dsName),
						Related: []report.Related{{
							Pos:     report.NewPos(other.Processor.FileMetadata),
							Message: fmt.Sprintf("Processor first seen in %s at index %d", other.Pipeline, other.Index),
						}},
					})
					continue
				}

				seen[hash] = ProcessorInfo{
					Hash:      hash,
					Processor: proc,
					Pipeline:  filepath.Base(pipeline.Path()),
					Index:     i,
				}
			}
		}
	}

	return result, nil
}

func generateProcessorHash(proc *fleetpkg.Processor) (uint64, error) {
	b, err := json.Marshal(proc)
	if err != nil {
		return 0, err
	}

	h := fnv.New64a()
	_, _ = h.Write(b)

	return h.Sum64(), nil
}
