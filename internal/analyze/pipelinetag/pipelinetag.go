package pipelinetag

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"

	"package-tool/internal/analyze"
	"package-tool/internal/yamledit"
)

const Name = "pipeline-tag"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline tag issues",
	CanFix:      true,
	Run:         run,
}

type processorNode struct {
	Processor *fleetpkg.Processor
	Parent    *processorNode
	Path      string
	Index     int
}

func (p *processorNode) ParentProcessor() *fleetpkg.Processor {
	if p.Parent != nil {
		return p.Parent.Processor
	}

	return nil
}

func run(ctx *analyze.Context) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			seen := map[string]*processorNode{}

			var pipelineAST analyze.AST
			var err error

			if ctx.Fix {
				pipelineAST, err = analyze.LoadAST(pipeline.Path())
				if err != nil {
					return err
				}
			}

			for i, proc := range pipeline.Processors {
				node := processorNode{
					Processor: proc,
					Path:      fmt.Sprintf("$.processors[%d].%s", i, proc.Type),
					Index:     i,
				}

				if err = processTag(ctx, &pipelineAST, &node, seen); err != nil {
					return err
				}
			}

			if ctx.Fix && pipelineAST.Modified {
				if err = pipelineAST.WriteFile(pipeline.Path()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func processTag(ctx *analyze.Context, pipelineAST *analyze.AST, node *processorNode, seen map[string]*processorNode) error {
	var invalid bool
	var err error

	tag, ok := node.Processor.Attributes["tag"].(string)
	if ok {
		if tag == "" {
			ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(node.Processor.FileMetadata), // TODO: This shifts during fixes. Maybe this should be deferred to final report.
				Category: Name,
				Message:  fmt.Sprintf("Empty tag on %s processor at index %d", node.Processor.Type, node.Index),
			})
			invalid = true
		} else if _, dup := seen[tag]; dup {
			ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(node.Processor.FileMetadata), // TODO: This shifts during fixes. Maybe this should be deferred to final report.
				Category: Name,
				Message:  fmt.Sprintf("Duplicated tag on %s processor at index %d", node.Processor.Type, node.Index),
			})
			invalid = true

		}
	} else {
		ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
			Pos:      analyze.NewPosFromFileMetadata(node.Processor.FileMetadata), // TODO: This shifts during fixes. Maybe this should be deferred to final report.
			Category: Name,
			Message:  fmt.Sprintf("Missing or invalid tag on %s processor at index %d", node.Processor.Type, node.Index),
		})
		invalid = true
	}

	if invalid && ctx.Fix {
		if tag, err = generateTag(node.Processor, node.ParentProcessor()); err != nil {
			return err
		}

		p, err := yaml.PathString(node.Path + ".tag")
		if err != nil {
			return err // TODO: What about this error?
		}

		if err = yamledit.SetString(pipelineAST.File, p, tag, true); err != nil {
			return err // TODO: What about this error?
		}

		ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
			Category: Name,
			Message:  fmt.Sprintf("Generated tag %q for %s processor at index %d", tag, node.Processor.Type, node.Index),
		})

		pipelineAST.Modified = true
	}

	seen[tag] = node

	for i, onFailProc := range node.Processor.OnFailure {
		onFailProcNode := &processorNode{
			Processor: onFailProc,
			Parent:    node,
			Path:      fmt.Sprintf("%s.on_failure[%d].%s", node.Path, i, onFailProc.Type),
			Index:     i,
		}
		if err = processTag(ctx, pipelineAST, onFailProcNode, seen); err != nil {
			return err
		}
	}

	return nil
}

func generateTag(proc *fleetpkg.Processor, parent *fleetpkg.Processor) (string, error) {
	hash, err := generateProcessorHash(proc, parent)
	if err != nil {
		return "", err
	}

	field, ok := proc.Attributes["field"].(string)
	if !ok || field == "" {
		return proc.Type + "_" + hash, nil
	}
	field = yamledit.PathCleaner.Replace(field)

	targetField, ok := proc.Attributes["target_field"].(string)
	if !ok || targetField == "" {
		return fmt.Sprintf("%s_%s_%s", proc.Type, field, hash), nil
	}
	targetField = yamledit.PathCleaner.Replace(targetField)

	return fmt.Sprintf("%s_%s_to_%s_%s", proc.Type, field, targetField, hash), nil
}

func generateProcessorHash(proc *fleetpkg.Processor, parent *fleetpkg.Processor) (string, error) {
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
