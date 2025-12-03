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

// Package duplicateprocessor provides an analyzer that finds processors
// duplicated in ingest pipelines.
package duplicateprocessor

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

const Name = "duplicate-processor"

var Analyzer = &analyze.Analyzer{
	Name: Name,
	Doc:  "Find processors duplicated in ingest pipelines",
	Run:  run,
}

type processorNode struct {
	Hash       uint64
	Processor  *fleetpkg.Processor
	DataStream string
	Pipeline   string
	Index      int
}

func run(pass *analyze.Pass) error {
	for dsName, ds := range pass.Package.DataStreams {
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
				procIssues, err := checkProcessor(pipeline, &node, seen)
				pass.Issues = append(pass.Issues, procIssues...)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkProcessor(pipeline *fleetpkg.Pipeline, node *processorNode, seen map[uint64]*processorNode) ([]analyze.Issue, error) {
	var issues []analyze.Issue
	var err error

	if node.Hash, err = generateProcessorHash(node.Processor); err != nil {
		return nil, fmt.Errorf("unable to hash processor at index %d in pipeline %q: %w", node.Index, node.Pipeline, err)
	}

	if other, dup := seen[node.Hash]; dup {
		issues = append(issues, analyze.Issue{
			Analyzer: Name,
			Message:  fmt.Sprintf("Processor %s in %s at %s already exists in %s at %s in data stream %s", node.Processor.Type, node.Pipeline, node.Processor.Node.GetPath(), other.Pipeline, other.Processor.Node.GetPath(), node.DataStream),
			Severity: analyze.SeverityWarn,
			Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), node.Processor.Node),
			Related: []analyze.Related{{
				Pos:     analyze.NewYAMLNodePosition(pipeline.Doc.AST(), other.Processor.Node),
				Message: fmt.Sprintf("Processor first seen in %s at index %d", other.Pipeline, other.Index),
			}},
		})
	} else {
		seen[node.Hash] = node
	}

	return issues, nil
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
