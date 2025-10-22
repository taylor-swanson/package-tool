package pipelinetag

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"

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

type ProcessorInfo struct {
	Processor *fleetpkg.Processor
	Parent    *ProcessorInfo
	Path      string
	Index     int
}

func (p *ProcessorInfo) ParentProcessor() *fleetpkg.Processor {
	if p.Parent != nil {
		return p.Parent.Processor
	}

	return nil
}

type Change struct {
	Path            string
	Pos             analyze.Pos
	Processor       *fleetpkg.Processor
	ParentProcessor *fleetpkg.Processor
}

type ChangeSet struct {
	Pipeline   *fleetpkg.IngestPipeline
	DataStream *fleetpkg.DataStream
	Changes    []Change
}

func run(ctx *analyze.Context) (analyze.Result, error) {
	var result analyze.Result

	changes, err := doCheck(ctx, &result)
	if err != nil {
		return result, err
	}

	if ctx.Fix && len(changes) > 0 {
		if err := doFix(ctx, changes, &result); err != nil {
			return result, err
		}
	}

	return result, nil
}

func doCheck(ctx *analyze.Context, result *analyze.Result) ([]ChangeSet, error) {
	var changeSets []ChangeSet

	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {

			var changeSet ChangeSet
			seen := map[string]*ProcessorInfo{}

			for i, proc := range pipeline.Processors {
				procInfo := &ProcessorInfo{
					Processor: proc,
					Parent:    nil,
					Path:      fmt.Sprintf("$.processors[%d].%s", i, proc.Type),
					Index:     i,
				}

				if err := checkProcessorTag(procInfo, result, &changeSet, seen); err != nil {
					return nil, err
				}
			}

			if ctx.Fix && len(changeSet.Changes) > 0 {
				changeSet.DataStream = ds
				changeSet.Pipeline = &pipeline
				changeSets = append(changeSets, changeSet)
			}
		}
	}

	return changeSets, nil
}

func checkProcessorTag(proc *ProcessorInfo, result *analyze.Result, changeSet *ChangeSet, seen map[string]*ProcessorInfo) error {
	var invalid bool

	tag, ok := proc.Processor.Attributes["tag"].(string)
	if ok {
		if tag == "" {
			result.Findings = append(result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(proc.Processor.FileMetadata),
				Category: Name,
				Message:  fmt.Sprintf("Empty tag on %s processor at index %d", proc.Processor.Type, proc.Index),
			})
			invalid = true
		} else if _, dup := seen[tag]; dup {
			result.Findings = append(result.Findings, analyze.Finding{
				Pos:      analyze.NewPosFromFileMetadata(proc.Processor.FileMetadata),
				Category: Name,
				Message:  fmt.Sprintf("Duplicated tag on %s processor at index %d", proc.Processor.Type, proc.Index),
			})
			invalid = true

		}
	} else {
		result.Findings = append(result.Findings, analyze.Finding{
			Pos:      analyze.NewPosFromFileMetadata(proc.Processor.FileMetadata),
			Category: Name,
			Message:  fmt.Sprintf("Missing or invalid tag on %s processor at index %d", proc.Processor.Type, proc.Index),
		})
		invalid = true
	}

	if invalid {
		changeSet.Changes = append(changeSet.Changes, Change{
			Path:            proc.Path + ".tag",
			Pos:             analyze.NewPosFromFileMetadata(proc.Processor.FileMetadata),
			Processor:       proc.Processor,
			ParentProcessor: proc.ParentProcessor(),
		})
	} else {
		seen[tag] = proc
	}

	for i, onFailProc := range proc.Processor.OnFailure {
		onFailProcInfo := &ProcessorInfo{
			Processor: onFailProc,
			Parent:    proc,
			Path:      fmt.Sprintf("%s.on_failure[%d].%s", proc.Path, i, onFailProc.Type),
			Index:     i,
		}
		if err := checkProcessorTag(onFailProcInfo, result, changeSet, seen); err != nil {
			return err
		}
	}

	return nil
}

func doFix(ctx *analyze.Context, changeSets []ChangeSet, result *analyze.Result) error {
	for _, changeSet := range changeSets {
		f, err := parser.ParseFile(changeSet.Pipeline.Path(), parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse pipeline %q: %w", changeSet.Pipeline.Path(), err)
		}

		for _, change := range changeSet.Changes {
			tag, err := generateTag(change.Processor, change.ParentProcessor)
			if err != nil {
				return err // TODO: What about this error?
			}

			p, err := yaml.PathString(change.Path)
			if err != nil {
				return err // TODO: What about this error?
			}

			if err = yamledit.SetString(f, p, tag, true); err != nil {
				return err // TODO: What about this error?
			}

			result.Fixes = append(result.Fixes, analyze.Fix{
				Category: Name,
				Message:  fmt.Sprintf("Generated tag %q", tag), // TODO: Need more context.
			})
		}

		p := printer.Printer{}
		d := p.PrintNode(f.Docs[0])

		if err = os.WriteFile(changeSet.Pipeline.Path(), d, 0o644); err != nil {
			return fmt.Errorf("failed to write pipeline %q: %w", changeSet.Pipeline.Path(), err)
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
