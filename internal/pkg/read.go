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

package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"
	"gopkg.in/yaml.v3"
)

func ReadManifest(dir string) (*fleetpkg.Manifest, error) {
	var manifest fleetpkg.Manifest

	if err := ReadYAML(filepath.Join(dir, "manifest.yml"), &manifest, true); err != nil {
		return nil, fmt.Errorf("unable to read manifest: %w", err)
	}

	return &manifest, nil
}

func ReadYAML(path string, v any, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(strict)

	if err = dec.Decode(v); err != nil {
		return fmt.Errorf("failed decoding %s: %w", path, err)
	}

	return nil
}
