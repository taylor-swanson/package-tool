// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package analyze provides a framework for analyzing and modifying packages.
package analyze

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

var PathCleaner = strings.NewReplacer(".", "_", " ", "_", "@", "")

type Analyzer struct {
	Name     string
	Doc      string
	CanFix   bool
	Flags    pflag.FlagSet
	Run      func(pass *Pass) error
	LoadMode fleetpkg.LoadMode
}

type Pass struct {
	Package *fleetpkg.Package
	Fix     bool
	Issues  []Issue
}
