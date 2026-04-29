// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package fleetpkg provides Go type for fleet packages.
package fleetpkg

import (
	"encoding/json"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

// Package is a fleet package.
type Package struct {
	Manifest      Manifest
	BuildManifest *BuildManifest
	Input         *DataStream
	DataStreams   map[string]*DataStream

	sourceDir string
}

// Path is the path to the root of the package.
func (i *Package) Path() string {
	return i.sourceDir
}

// Manifest is the package manifest.
type Manifest struct {
	Name          string `yaml:"name"`
	Title         string `yaml:"title"`
	Version       string `yaml:"version"`
	Description   string `yaml:"description"`
	Type          string `yaml:"type"`
	FormatVersion string `yaml:"format_version"`
	Owner         struct {
		Github string `yaml:"github"`
		Type   string `yaml:"type"`
	} `yaml:"owner"`

	Doc *yamledit.Document `yaml:"-"`
}

// Path is the path to the manifest file.
func (m *Manifest) Path() string { return m.Doc.Filename() }

// BuildManifest is the package build manifest.
type BuildManifest struct {
	Dependencies struct {
		ECS struct {
			Reference      string `yaml:"reference"`
			ImportMappings *bool  `yaml:"import_mappings,omitempty"`
		} `yaml:"ecs"`
	} `yaml:"dependencies"`

	Doc *yamledit.Document `yaml:"-"`
}

// Path is the path to the build manifest file.
func (m *BuildManifest) Path() string { return m.Doc.Filename() }

// DataStreamManifest is the data stream manifest file.
type DataStreamManifest struct {
	Title string `yaml:"title"`
	Type  string `yaml:"type"`

	Doc *yamledit.Document `yaml:"-"`
}

// Path is the path to the manifest file.
func (m *DataStreamManifest) Path() string { return m.Doc.Filename() }

// DataStream is a data stream within the package.
type DataStream struct {
	Manifest  DataStreamManifest
	Pipelines map[string]*Pipeline

	sourceDir string
}

// Path is the path to the data stream.
func (d *DataStream) Path() string {
	return d.sourceDir
}

// Pipeline is an ingest pipeline.
type Pipeline struct {
	Description string       `yaml:"description"`
	Processors  []*Processor `yaml:"processors,omitempty"`
	OnFailure   []*Processor `yaml:"on_failure,omitempty"`

	Doc *yamledit.Document `yaml:"-"`
}

// Path is the path to the pipeline.
func (p *Pipeline) Path() string { return p.Doc.Filename() }

// Processor is an ingest pipeline processor.
type Processor struct {
	Type       string
	Attributes map[string]any
	OnFailure  []*Processor

	Node ast.Node
}

// GetAttribute gets an attribute of the processor.
func (p *Processor) GetAttribute(key string) (any, bool) {
	v, ok := p.Attributes[key]
	if !ok {
		return nil, false
	}

	return v, true
}

// GetAttributeString gets a string attribute of the processor.
func (p *Processor) GetAttributeString(key string) (string, bool) {
	v, ok := p.Attributes[key].(string)
	if !ok {
		return "", false
	}

	return v, true
}

// GetAttributeFloat gets a float attribute of the processor.
func (p *Processor) GetAttributeFloat(key string) (float64, bool) {
	v, ok := p.Attributes[key].(float64)
	if !ok {
		return 0, false
	}

	return v, true
}

// GetAttributeInt gets an int attribute of the processor.
func (p *Processor) GetAttributeInt(key string) (int, bool) {
	v, ok := p.Attributes[key].(int)
	if !ok {
		return 0, false
	}

	return v, true
}

// GetAttributeBool gets a bool attribute of the processor.
func (p *Processor) GetAttributeBool(key string) (bool, bool) {
	v, ok := p.Attributes[key].(bool)
	if !ok {
		return false, false
	}

	return v, true
}

// UnmarshalYAML implements a YAML unmarshaler.
func (p *Processor) UnmarshalYAML(node ast.Node) error {
	var procMap map[string]struct {
		Attributes map[string]any `yaml:",inline"`
		OnFailure  []*Processor   `yaml:"on_failure"`
	}
	if err := yaml.NodeToValue(node, &procMap); err != nil {
		return err
	}

	// The struct representation used here is much more convenient
	// to work with than the original map of map format.
	for k, v := range procMap {
		p.Type = k
		p.Attributes = v.Attributes
		p.OnFailure = v.OnFailure

		delete(p.Attributes, "on_failure")

		break
	}

	p.Node = node

	return nil
}

// MarshalJSON implements a JSON marshaler.
func (p *Processor) MarshalJSON() ([]byte, error) {
	properties := make(map[string]any, len(p.Attributes)+1)
	for k, v := range p.Attributes {
		properties[k] = v
	}
	if len(p.OnFailure) > 0 {
		properties["on_failure"] = p.OnFailure
	}
	return json.Marshal(map[string]any{
		p.Type: properties,
	})
}

// Validation is the validation.yml file of a package.
type Validation struct {
	Errors struct {
		ExcludeChecks []string `yaml:"exclude_checks,omitempty"`
	} `yaml:"errors,omitempty"`

	DocsStructureEnforced struct {
		Enabled bool `yaml:"enabled"`
		Version int  `yaml:"version"`
		Skip    []struct {
			Title  string `yaml:"title"`
			Reason string `yaml:"reason"`
		} `yaml:"skip,omitempty"`
	} `yaml:"docs_structure_enforced"`

	Doc *yamledit.Document `yaml:"-"`
}
