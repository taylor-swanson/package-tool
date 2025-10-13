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

	"package-tool/internal/modify"
	"package-tool/internal/yamledit"
)

const Name = "pipeline-tag"

var Modifier = &modify.Modifier{
	Name:        Name,
	Description: "Fix ingest pipeline tag issues",
	Run:         run,
}

func run(pkg *fleetpkg.Integration) error {
	for _, ds := range pkg.DataStreams {
		for _, pipeline := range ds.Pipelines {
			f, err := parser.ParseFile(pipeline.Path(), parser.ParseComments)
			if err != nil {
				return fmt.Errorf("failed to parse pipeline %q: %w", pipeline.Path(), err)
			}

			modified := false
			seen := map[string]struct{}{}

			for i, proc := range pipeline.Processors {
				tag, ok := proc.Attributes["tag"].(string)
				if ok && tag != "" {
					if _, dup := seen[tag]; !dup {
						seen[tag] = struct{}{}
						continue
					}
				}

				if tag, err = generateTag(proc, nil); err != nil {
					return fmt.Errorf("failed to create tag for processor at index %d in pipeline %q: %w", i, pipeline.Path(), err)
				}
				if _, dup := seen[tag]; dup { // TODO: Make error permissive, otherwise resolve duplicate
					return fmt.Errorf("generated duplicate tag for processor at index %d in pipeline %q: %s", i, pipeline.Path(), tag)
				}

				yamlPath, err := yaml.PathString(fmt.Sprintf("$.processors[%d].%s.tag", i, proc.Type))
				if err != nil {
					return fmt.Errorf("failed to create yaml path for processor at index %d in pipeline %q: %w", i, pipeline.Path(), err)
				}
				if err = yamledit.SetString(f, yamlPath, tag); err != nil {
					return fmt.Errorf("failed to set tag for processor at index %d in pipeline %q: %w", i, pipeline.Path(), err)
				}

				modified = true
				seen[tag] = struct{}{}

				// TODO: on_failure processors and foreach processors !!!
			}

			if !modified {
				continue
			}

			p := printer.Printer{}
			d := p.PrintNode(f.Docs[0])

			if err = os.WriteFile(pipeline.Path(), d, 0o644); err != nil {
				return fmt.Errorf("failed to write pipeline %q: %w", pipeline.Path(), err)
			}
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
