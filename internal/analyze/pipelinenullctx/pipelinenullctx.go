// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

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
	Name:   Name,
	Doc:    "Find and fix ingest pipeline ctx null-safe deference issues",
	CanFix: true,
	Run:    run,
}

var ctxMatchRe = regexp.MustCompile(`(^|[^.])(ctx)\?`)

func run(pass *analyze.Pass) error {
	for _, ds := range pass.Package.DataStreams {
		for _, pipeline := range ds.Pipelines {
			for _, proc := range pipeline.Processors {
				conditional, _ := proc.GetAttributeString("if")
				if strings.Contains(conditional, "ctx?.") {
					pos, err := analyze.NewYAMLPathPosition(pipeline.Doc.AST(), fmt.Sprintf("%s.%s.if", proc.Node.GetPath(), proc.Type))
					if err != nil {
						return err
					}
					pass.Issues = append(pass.Issues, analyze.Issue{
						Analyzer: Name,
						Message:  "Processor conditional uses unnecessary ctx null-safe deference",
						Severity: analyze.SeverityWarn,
						Pos:      pos,
					})
					if pass.Fix {
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
					}
				}
				if proc.Type == "script" {
					src, _ := proc.GetAttributeString("source")
					if strings.Contains(src, "ctx?.") {
						pos, err := analyze.NewYAMLPathPosition(pipeline.Doc.AST(), proc.Node.GetPath()+".script.source")
						if err != nil {
							return err
						}
						pass.Issues = append(pass.Issues, analyze.Issue{
							Analyzer: Name,
							Message:  "Painless script uses unnecessary ctx null-safe deference",
							Severity: analyze.SeverityWarn,
							Pos:      pos,
						})
						if pass.Fix {
							n, err := pipeline.Doc.GetNode(proc.Node.GetPath() + ".script.source")
							if err != nil {
								return err
							}
							switch sn := n.(type) {
							case *ast.StringNode:
								sn.Value = ctxMatchRe.ReplaceAllString(sn.Value, "$1$2")
							case *ast.LiteralNode:
								s := n.String()
								s = ctxMatchRe.ReplaceAllString(s, "$1$2")
								if err = setLiteralNodeString(pipeline.Doc.AST(), proc.Node.GetPath()+".script.source", s); err != nil {
									return err
								}
							default:
								return fmt.Errorf("unexpected node type: %T", n)
							}
						}
					}
				}
			}

			if pass.Fix && pipeline.Doc.Modified() {
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
