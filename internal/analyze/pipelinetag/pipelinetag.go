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

// Package pipelinetag provides an analyzer that finds and fixes ingest pipeline
// tag issues.
package pipelinetag

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/taylor-swanson/package-tool/internal/analyze"
	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

const Name = "pipeline-tag"

var Analyzer = &analyze.Analyzer{
	Name:   Name,
	Doc:    "Find and fix ingest pipeline tag issues",
	CanFix: true,
	Run:    run,
}

type processorNode struct {
	Processor *fleetpkg.Processor
	Parent    *processorNode
}

func (p *processorNode) ParentProcessor() *fleetpkg.Processor {
	if p.Parent != nil {
		return p.Parent.Processor
	}

	return nil
}

func run(pass *analyze.Pass) error {
	for _, ds := range pass.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			seen := map[string]*processorNode{}

			for _, proc := range pipeline.Processors {
				node := processorNode{
					Processor: proc,
				}

				procIssues, err := processTag(pipeline, &node, seen, pass.Fix)
				pass.Issues = append(pass.Issues, procIssues...)
				if err != nil {
					return err
				}
			}

			if pass.Fix && pipeline.Doc.Modified() {
				if err := pipeline.Doc.WriteFile(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func processTag(pipeline *fleetpkg.Pipeline, node *processorNode, seen map[string]*processorNode, fix bool) ([]analyze.Issue, error) {
	var issues []analyze.Issue
	var invalid bool
	var err error

	procPath := node.Processor.Node.GetPath()
	tag, ok := node.Processor.Attributes["tag"].(string)
	if ok {
		if tag == "" {
			issues = append(issues, analyze.Issue{
				Analyzer: Name,
				Message:  fmt.Sprintf("Empty tag on %s processor at %s", node.Processor.Type, procPath),
				Severity: analyze.SeverityError,
				Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), node.Processor.Node),
			})
			invalid = true
		} else if _, dup := seen[tag]; dup {
			issues = append(issues, analyze.Issue{
				Analyzer: Name,
				Message:  fmt.Sprintf("Duplicated tag on %s processor at %s", node.Processor.Type, procPath),
				Severity: analyze.SeverityError,
				Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), node.Processor.Node),
			})
			invalid = true

		}
	} else {
		issues = append(issues, analyze.Issue{
			Analyzer: Name,
			Message:  fmt.Sprintf("Missing or invalid tag on %s processor at %s", node.Processor.Type, procPath),
			Severity: analyze.SeverityError,
			Pos:      analyze.NewYAMLNodePosition(pipeline.Doc.AST(), node.Processor.Node),
		})
		invalid = true
	}

	if invalid && fix {
		if tag, err = generateTag(node.Processor, node.ParentProcessor()); err != nil {
			return issues, err
		}
		if _, err := pipeline.Doc.SetKeyValue(fmt.Sprintf("%s.%s", node.Processor.Node.GetPath(), node.Processor.Type), "tag", tag, yamledit.IndexPrepend); err != nil {
			return issues, err
		}
	}

	seen[tag] = node

	for _, onFailProc := range node.Processor.OnFailure {
		onFailProcNode := &processorNode{
			Processor: onFailProc,
			Parent:    node,
		}
		procIssues, err := processTag(pipeline, onFailProcNode, seen, fix)
		issues = append(issues, procIssues...)
		if err != nil {
			return issues, err
		}
	}

	return issues, nil
}

func generateTag(proc, parent *fleetpkg.Processor) (string, error) {
	hash, err := generateProcessorHash(proc, parent)
	if err != nil {
		return "", err
	}

	field, ok := proc.Attributes["field"].(string)
	if !ok || field == "" {
		return proc.Type + "_" + hash, nil
	}
	field = analyze.PathCleaner.Replace(field)

	targetField, ok := proc.Attributes["target_field"].(string)
	if !ok || targetField == "" {
		return fmt.Sprintf("%s_%s_%s", proc.Type, field, hash), nil
	}
	targetField = analyze.PathCleaner.Replace(targetField)

	return fmt.Sprintf("%s_%s_to_%s_%s", proc.Type, field, targetField, hash), nil
}

func generateProcessorHash(proc, parent *fleetpkg.Processor) (string, error) {
	b, err := json.Marshal(proc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal processor for hashing: %w", err)
	}

	h := fnv.New32a()
	_, _ = h.Write(b)

	if parent != nil {
		b, err = json.Marshal(parent)
		if err != nil {
			return "", fmt.Errorf("failed to marshal parent processor for hashing: %w", err)
		}

		_, _ = h.Write(b)
	}

	return fmt.Sprintf("%08x", h.Sum32()), nil
}
