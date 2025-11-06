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

package fleetpkg

import (
	"fmt"
	"path/filepath"

	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

func Load(dir string) (*Package, error) {
	pkg := Package{
		sourceDir: dir,
	}

	// -------------------------------------------------------------------------
	// manifest.yml

	if _, err := yamledit.ParseDocumentFile(filepath.Join(dir, "manifest.yml"), &pkg.Manifest); err != nil {
		return nil, err
	}

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

	return &pkg, nil
}

func LoadManifest(dir string) (*Manifest, error) {
	var manifest Manifest

	if _, err := yamledit.ParseDocumentFile(filepath.Join(dir, "manifest.yml"), &manifest); err != nil {
		return nil, fmt.Errorf("unable to read manifest: %w", err)
	}

	return &manifest, nil
}
