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

package duplicateprocessor

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/taylor-swanson/package-tool/internal/analyze"
)

const Name = "duplicate-processor"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find processors duplicated in ingest pipelines",
	Run:         run,
}

type processorNode struct {
	Hash       uint64
	Processor  *fleetpkg.Processor
	DataStream string
	Pipeline   string
	Index      int
}

func run(ctx *analyze.Context) error {
	for dsName, ds := range ctx.Package.DataStreams {
		seen := map[uint64]*processorNode{}

		var pipelineNames []string
		if _, ok := ds.Pipelines["default.yml"]; ok {
			pipelineNames = []string{"default.yml"}
		}
		for name := range ds.Pipelines {
			if name == "default.yml" {
				continue
			}
			pipelineNames = append(pipelineNames, name)
		}

		for _, name := range pipelineNames {
			pipeline := ds.Pipelines[name]
			for i, proc := range pipeline.Processors {
				node := processorNode{
					Processor:  proc,
					DataStream: dsName,
					Pipeline:   name,
					Index:      i,
				}
				if err := checkProcessor(ctx, &node, seen); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkProcessor(ctx *analyze.Context, node *processorNode, seen map[uint64]*processorNode) error {
	var err error

	if node.Hash, err = generateProcessorHash(node.Processor); err != nil {
		return fmt.Errorf("unable to hash processor at index %d in pipeline %q: %w", node.Index, node.Pipeline, err)
	}

	if other, dup := seen[node.Hash]; dup {
		ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
			Pos:      analyze.NewPosFromFileMetadata(node.Processor.FileMetadata),
			Category: Name,
			Message:  fmt.Sprintf("Processor %s in %s at index %d already exists in %s at index %d in data stream %s", node.Processor.Type, node.Pipeline, node.Index, other.Pipeline, other.Index, node.DataStream),
			Related: []analyze.Related{{
				Pos:     analyze.NewPosFromFileMetadata(other.Processor.FileMetadata),
				Message: fmt.Sprintf("Processor first seen in %s at index %d", other.Pipeline, other.Index),
			}},
		})
	} else {
		seen[node.Hash] = node
	}

	return nil
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
