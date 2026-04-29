// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"fmt"
	"path/filepath"

	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

// LoadMode defines a method by which a package is loaded.
type LoadMode int

const (
	// LoadModeFull loads all files in a package.
	LoadModeFull LoadMode = iota
	// LoadModePipelineOnly loads only pipeline and manifest files from a package.
	LoadModePipelineOnly
	// LoadModeFieldsOnly loads only field and manifest files from a package.
	LoadModeFieldsOnly
	// LoadModeManifestOnly loads only the manifest files from a package.
	LoadModeManifestOnly
)

// Load will load a package from the given directory.
func Load(dir string, mode LoadMode) (*Package, error) {
	pkg := Package{
		sourceDir: dir,
	}

	// -------------------------------------------------------------------------
	// manifest.yml

	if _, err := yamledit.ParseDocumentFile(filepath.Join(dir, "manifest.yml"), &pkg.Manifest); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// Build manifest (_dev/build/build.yml)

	var buildManifest BuildManifest
	if _, err := yamledit.ParseDocumentFile(filepath.Join(dir, "_dev", "build", "build.yml"), &buildManifest); err != nil {
		return nil, err
	}
	pkg.BuildManifest = &buildManifest

	// -------------------------------------------------------------------------
	// Data Streams

	var dataStreamManifests []string
	if pkg.Manifest.Type == "input" {
		dataStreamManifests = []string{filepath.Join(dir, "manifest.yml")}
	} else {
		var err error
		pkg.DataStreams = map[string]*DataStream{}
		if dataStreamManifests, err = filepath.Glob(filepath.Join(dir, "data_stream/*/manifest.yml")); err != nil {
			return nil, err
		}
	}
	for _, manifestPath := range dataStreamManifests {
		ds := &DataStream{
			sourceDir: filepath.Dir(manifestPath),
		}
		if pkg.Manifest.Type == "input" {
			pkg.Input = ds
		} else {
			pkg.DataStreams[filepath.Base(ds.sourceDir)] = ds

			if _, err := yamledit.ParseDocumentFile(manifestPath, &ds.Manifest); err != nil {
				return nil, err
			}

			// -----------------------------------------------------------------
			// Pipelines

			if mode == LoadModeFull || mode == LoadModePipelineOnly {
				pipelines, err := filepath.Glob(filepath.Join(ds.sourceDir, "elasticsearch/ingest_pipeline/*.yml"))
				if err != nil {
					return nil, err
				}

				if len(pipelines) > 0 {
					ds.Pipelines = map[string]*Pipeline{}
				}
				for _, pipelinePath := range pipelines {
					var pipeline Pipeline
					if _, err = yamledit.ParseDocumentFile(pipelinePath, &pipeline); err != nil {
						return nil, err
					}
					ds.Pipelines[filepath.Base(pipelinePath)] = &pipeline
				}
			}
		}
	}

	return &pkg, nil
}

// LoadManifest loads a manifest from a package directory.
func LoadManifest(dir string) (*Manifest, error) {
	var manifest Manifest

	if _, err := yamledit.ParseDocumentFile(filepath.Join(dir, "manifest.yml"), &manifest); err != nil {
		return nil, fmt.Errorf("unable to read manifest: %w", err)
	}

	return &manifest, nil
}
