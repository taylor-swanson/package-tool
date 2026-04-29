// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

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
