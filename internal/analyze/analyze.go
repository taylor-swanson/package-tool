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

// Package analyze provides a framework for analyzing and modifying packages.
package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"
)

type Analyzer struct {
	Name        string
	Description string
	CanFix      bool
	Run         func(ctx *Context) error
}

type Context struct {
	Package *fleetpkg.Integration
	Fix     bool
	Args    []string

	Result Result
}

type Result struct {
	Findings []Finding
	Fixes    []Fix
}

type Finding struct {
	Pos      Pos       `json:"pos"`
	Category string    `json:"category"`
	Message  string    `json:"message"`
	Related  []Related `json:"related,omitempty"`
}

type Fix struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Related struct {
	Pos     Pos    `json:"pos"`
	Message string `json:"message"`
}

type AST struct {
	File     *ast.File
	Modified bool
}

// LoadAST loads an AST from filename.
func LoadAST(filename string) (AST, error) {
	f, err := parser.ParseFile(filename, parser.ParseComments)
	if err != nil {
		return AST{}, fmt.Errorf("failed to parse AST for file %q: %w", filename, err)
	}

	return AST{File: f}, nil
}

// WriteFile writes the AST to filename.
func (a AST) WriteFile(filename string) error {
	p := printer.Printer{}
	d := p.PrintNode(a.File.Docs[0])

	if err := os.WriteFile(filename, d, 0o644); err != nil {
		return fmt.Errorf("failed to write AST to file %q: %w", filename, err)
	}

	return nil
}

type Pos struct {
	File   string
	Path   string
	Line   int
	Column int
}

func (p Pos) String() string {
	if p.Line == 0 {
		return p.File
	}
	if p.Column == 0 {
		return p.File + ":" + strconv.Itoa(p.Line)
	}

	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

func (p Pos) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// NewPosFromFileMetadata creates a new Pos from the given file metadata.
func NewPosFromFileMetadata(meta fleetpkg.FileMetadata) Pos {
	return Pos{
		File:   meta.Path(),
		Line:   meta.Line(),
		Column: meta.Column(),
	}
}

// NewPosFromPath creates a new Pos from the node at path in the given file.
func NewPosFromPath(f *ast.File, yamlPath string) (Pos, error) {
	p, err := yaml.PathString(yamlPath)
	if err != nil {
		return Pos{}, err
	}

	n, err := p.FilterFile(f)
	if err != nil {
		return Pos{}, err
	}

	t := n.GetToken().Position

	return Pos{
		File:   f.Name,
		Path:   yamlPath,
		Line:   t.Line,
		Column: t.Column,
	}, nil
}
