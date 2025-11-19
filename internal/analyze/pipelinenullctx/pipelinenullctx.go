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

// Package pipelinenullctx provides an analyzer that finds and fixes issues with
// unnecessary null-safe operators with ctx in ingest pipelines.
package pipelinenullctx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/taylor-swanson/package-tool/internal/analyze"
)

const Name = "pipeline-null-ctx"

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find and fix ingest pipeline ctx null-safe deference issues",
	CanFix:      true,
	Run:         run,
}

var ctxMatchRe = regexp.MustCompile(`(^|[^.])(ctx)\?`)

func run(ctx *analyze.Context) error {
	for _, ds := range ctx.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				conditional, _ := proc.GetAttributeString("if")
				if strings.Contains(conditional, "ctx?.") {
					ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
						Pos:      analyze.MustPosFromPath(pipeline.Doc, fmt.Sprintf("%s.%s.if", proc.Node.GetPath(), proc.Type)),
						Category: Name,
						Message:  "Processor conditional uses unnecessary ctx null-safe deference",
					})
					if ctx.Fix {
						n, err := pipeline.Doc.GetNode(fmt.Sprintf("%s.%s.if", proc.Node.GetPath(), proc.Type))
						if err != nil {
							return err
						}
						switch sn := n.(type) {
						case *ast.StringNode:
							sn.Value = ctxMatchRe.ReplaceAllString(sn.Value, "$1$2")
						case *ast.LiteralNode:
							s := n.String()
							s = ctxMatchRe.ReplaceAllString(s, "$1$2")
							if err = setLiteralNodeString(pipeline.Doc.AST(), n.GetPath(), s); err != nil {
								return err
							}
						default:
							return fmt.Errorf("unexpected node type: %T", n)
						}
						ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
							Category: Name,
							Message:  "Removed unnecessary ctx null-safe deference from processor conditional",
						})
					}
				}
				if proc.Type == "script" {
					src, _ := proc.GetAttributeString("source")
					if strings.Contains(src, "ctx?.") {
						ctx.Result.Findings = append(ctx.Result.Findings, analyze.Finding{
							Pos:      analyze.MustPosFromPath(pipeline.Doc, proc.Node.GetPath()+".script.source"),
							Category: Name,
							Message:  "Painless script uses unnecessary ctx null-safe deference",
						})
						if ctx.Fix {
							n, err := pipeline.Doc.GetNode(proc.Node.GetPath() + ".script.source")
							if err != nil {
								return err
							}
							switch sn := n.(type) {
							case *ast.StringNode:
								sn.Value = ctxMatchRe.ReplaceAllString(sn.Value, "$1$2")
								ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
									Category: Name,
									Message:  "Removed unnecessary ctx null-safe deference from painless script",
								})
							case *ast.LiteralNode:
								s := n.String()
								s = ctxMatchRe.ReplaceAllString(s, "$1$2")
								if err = setLiteralNodeString(pipeline.Doc.AST(), proc.Node.GetPath()+".script.source", s); err != nil {
									return err
								}
								ctx.Result.Fixes = append(ctx.Result.Fixes, analyze.Fix{
									Category: Name,
									Message:  "Removed unnecessary ctx null-safe deference from painless script",
								})
							default:
								return fmt.Errorf("unexpected node type: %T", n)
							}
						}
					}
				}
			}

			if ctx.Fix && pipeline.Doc.Modified() {
				if err := pipeline.Doc.WriteFile(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func setLiteralNodeString(f *ast.File, path, value string) error {
	replacement, err := parser.ParseBytes([]byte("tmp: "+value), parser.ParseComments)
	if err != nil {
		return err
	}
	replacementNode := replacement.Docs[0].Body.(*ast.MappingNode).Values[0].Value

	p, err := yaml.PathString(path)
	if err != nil {
		return err
	}

	return p.ReplaceWithNode(f, replacementNode)
}
