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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrRootNotFound is an error indicating the package root could not be found.
var ErrRootNotFound = errors.New("package root not found")

// LocateRoot attempts to find the root of the package in the given directory.
// If the directory is not an absolute path, it will be converted to one using
// filepath.Abs. The root directory, as an absolute path, is returned,
// ErrRootNotFound if dir is not within the directory structure of a package,
// or some other error if an error was encountered.
func LocateRoot(dir string) (string, error) {
	var matches []string
	var err error

	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("unable to create absolute path: %w", err)
		}
	}

	for {
		if _, err = os.Stat(filepath.Join(dir, "manifest.yml")); err == nil {
			matches = append(matches, dir)
		}
		if _, err = os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	if len(matches) == 0 {
		return "", ErrRootNotFound
	}

	return matches[len(matches)-1], nil
}
