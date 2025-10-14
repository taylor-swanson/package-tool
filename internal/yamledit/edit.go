package yamledit

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

// DeleteNode deletes a node from a map.
func DeleteNode(f *ast.File, p *yaml.Path) error {
	n, err := p.FilterFile(f)
	if err != nil {
		return err
	}

	for _, d := range f.Docs {
		m := ast.Parent(d, n)
		if m == nil {
			continue
		}
		switch p := ast.Parent(d, m).(type) {
		case *ast.MappingNode:
			for i, e := range p.Values {
				if e == m {
					p.Values = append(p.Values[:i], p.Values[i+1:]...)
					break
				}
			}
		default:
			return fmt.Errorf("failed to get parent node: %w", err)
		}
	}
	return nil
}

func RemoveNode(seq *ast.SequenceNode, index int) bool {
	if index < 0 || index >= len(seq.Values) {
		seq.Values = append(seq.Values[:index], seq.Values[index+1:]...)
		seq.ValueHeadComments = append(seq.ValueHeadComments[:index], seq.ValueHeadComments[index+1:]...)

		return true
	}

	return false
}

func AppendOrReplaceNode(seq *ast.SequenceNode, index int, node ast.Node) bool {
	if index < 0 || index >= len(seq.Values) {
		// Append
		seq.Values = append(seq.Values, node)
		seq.ValueHeadComments = append(seq.ValueHeadComments, node.GetComment())
		return true
	}
	if !nodeEqual(seq.Values[index], node) {
		// Replace
		seq.Values[index] = node
		seq.ValueHeadComments[index] = node.GetComment()
		return true
	}

	return false
}

// SetString replaces the node at the specified path with a StringNode.
func SetString(f *ast.File, p *yaml.Path, value string) error {
	_, err := p.FilterFile(f)
	if err != nil {
		if yaml.IsNotFoundNodeError(err) {
			// If the key does not exist, then try to add it.
			parent, key, err := cutPath(p)
			if err != nil {
				return err
			}

			return appendMapNode(f, parent, key, value)
		}
		return err
	}

	replacement, err := yaml.ValueToNode(value)
	if err != nil {
		return err
	}

	return p.ReplaceWithNode(f, replacement)
}

// appendMapNode appends a new key/value to an existing map.
func appendMapNode(f *ast.File, p *yaml.Path, key string, value any) error {
	n, err := p.FilterFile(f)
	if err != nil {
		return err
	}

	// Build new mapping value.
	newNode, err := yaml.ValueToNode(map[string]any{
		key: value,
	})
	if err != nil {
		return err
	}
	newValue := newNode.(*ast.MappingNode).Values[0]

	// For maps with a single key. Relates https://github.com/goccy/go-yaml/issues/310.
	switch v := n.(type) {
	case *ast.MappingValueNode:
		n = ast.Mapping(
			token.New(":", ":", n.GetToken().Position),
			false,
			v)
	}

	switch n := n.(type) {
	case *ast.MappingNode:
		// Match indent.
		newValue.AddColumn(n.GetToken().Position.IndentNum)
		n.Values = append(n.Values, newValue)
	default:
		return fmt.Errorf("node found at path %s is not a map (found %T)", p.String(), n)
	}

	return nil
}

// cutPath slices the YAML path around the last dot.
func cutPath(p *yaml.Path) (before *yaml.Path, after string, err error) {
	pathStr := p.String()

	idx := strings.LastIndex(pathStr, ".")
	if idx < 0 {
		return nil, "", fmt.Errorf("cannot get parent path of %q", pathStr)
	}

	beforeStr := pathStr[:idx]
	after = pathStr[idx+1:]

	before, err = yaml.PathString(beforeStr)
	if err != nil {
		return nil, "", err
	}

	return before, after, nil
}

func nodeEqual(a, b ast.Node) bool {
	var x, y any
	if err := yaml.NodeToValue(a, &x); err != nil {
		panic(err)
	}
	if err := yaml.NodeToValue(b, &y); err != nil {
		panic(err)
	}

	return reflect.DeepEqual(x, y)
}
