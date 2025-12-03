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
