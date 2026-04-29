// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package analyze

import (
	"go/token"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// NewYAMLPathPosition creates a new position based on a YAML file and path.
func NewYAMLPathPosition(f *ast.File, path string) (token.Position, error) {
	p, err := yaml.PathString(path)
	if err != nil {
		return token.Position{}, err
	}

	n, err := p.FilterFile(f)
	if err != nil {
		return token.Position{}, err
	}
	if n == nil {
		return token.Position{}, yaml.ErrNotFoundNode
	}

	return NewYAMLNodePosition(f, n), nil
}

// NewYAMLNodePosition creates a new position based on a YAML file and node.
func NewYAMLNodePosition(f *ast.File, node ast.Node) token.Position {
	t := node.GetToken()

	return token.Position{
		Filename: f.Name,
		Line:     t.Position.Line,
		Column:   t.Position.Column,
	}
}
