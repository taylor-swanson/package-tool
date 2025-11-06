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
	"strconv"
	"strings"

	"github.com/goccy/go-yaml/ast"

	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
	"github.com/taylor-swanson/package-tool/pkg/yamledit"
)

var PathCleaner = strings.NewReplacer(".", "_", " ", "_", "@", "")

type Analyzer struct {
	Name        string
	Description string
	CanFix      bool
	Run         func(ctx *Context) error
}

type Context struct {
	Package *fleetpkg.Package
	Fix     bool
	Args    []string

	Result Result
}

type Result struct {
	Findings []Finding `json:"findings,omitempty"`
	Fixes    []Fix     `json:"fixes,omitempty"`
}

type Finding struct {
	Pos      *Pos      `json:"pos,omitempty"`
	Category string    `json:"category"`
	Message  string    `json:"message"`
	Related  []Related `json:"related,omitempty"`
}

type Fix struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Related struct {
	Pos     *Pos   `json:"pos,omitempty"`
	Message string `json:"message"`
}

type Pos struct {
	File   string
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

func NewPos(node ast.Node, filename string) *Pos {
	p := node.GetToken().Position
	return &Pos{
		File:   filename,
		Line:   p.Line,
		Column: p.Column,
	}
}

func NewPosFromPath(doc *yamledit.Document, path string) (*Pos, error) {
	n, err := doc.GetNode(path)
	if err != nil {
		return nil, err
	}

	p := n.GetToken().Position

	return &Pos{
		File:   doc.Filename(),
		Line:   p.Line,
		Column: p.Column,
	}, nil
}

func MustPosFromPath(doc *yamledit.Document, path string) *Pos {
	pos, err := NewPosFromPath(doc, path)
	if err != nil {
		panic(err)
	}

	return pos
}
