package cigeneration

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
)

// actionRefType is the composite-literal type SyncActionPins rewrites.
const actionRefType = "ActionRef"

// errMalformedRef reports an ActionRef literal SyncActionPins cannot rewrite.
var errMalformedRef = errors.New("malformed ActionRef literal")

// splice replaces src[start:end] with text.
type splice struct {
	start, end int
	text       string
}

// SyncActionPins rewrites the SHA and Tag of every ActionRef composite literal
// in src whose Owner/Repo appears in pins (keyed by action path, as returned by
// ParseWorkflowPins), and returns the gofmt-formatted result. Entries the
// workflows do not use are left byte-identical.
//
// Only the SHA and Tag string literals are replaced, by byte offset, so
// comments and layout survive. A literal whose Owner or Repo is not a string
// literal, or a used literal without string-literal SHA and Tag fields, is an
// error rather than a silent skip.
func SyncActionPins(src []byte, pins map[string]WorkflowPin) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing catalog source: %w", err)
	}

	var (
		edits   []splice
		walkErr error
	)
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || walkErr != nil {
			return walkErr == nil
		}
		if id, ok := lit.Type.(*ast.Ident); !ok || id.Name != actionRefType {
			return true
		}
		litEdits, err := actionRefEdits(fset, lit, pins)
		if err != nil {
			walkErr = fmt.Errorf("%s: %w", fset.Position(lit.Pos()), err)
			return false
		}
		edits = append(edits, litEdits...)
		return false
	})
	if walkErr != nil {
		return nil, walkErr
	}

	out, err := format.Source(applySplices(src, edits))
	if err != nil {
		return nil, fmt.Errorf("formatting synced catalog: %w", err)
	}
	return out, nil
}

// actionRefEdits returns the splices that bring one ActionRef literal in line
// with pins, or none when the workflows do not use it.
func actionRefEdits(fset *token.FileSet, lit *ast.CompositeLit, pins map[string]WorkflowPin) ([]splice, error) {
	fields := make(map[string]*ast.BasicLit, len(lit.Elts))
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("%w: positional fields", errMalformedRef)
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("%w: non-identifier key", errMalformedRef)
		}
		if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			fields[key.Name] = bl
		} else {
			fields[key.Name] = nil
		}
	}

	owner, err := stringField(fields, "Owner")
	if err != nil {
		return nil, err
	}
	repo, err := stringField(fields, "Repo")
	if err != nil {
		return nil, err
	}
	pin, used := pins[owner+"/"+repo]
	if !used {
		return nil, nil
	}

	var edits []splice
	for name, want := range map[string]string{"SHA": pin.SHA, "Tag": pin.Tag} {
		if _, err := stringField(fields, name); err != nil {
			return nil, err
		}
		bl := fields[name]
		edits = append(edits, splice{
			start: fset.Position(bl.Pos()).Offset,
			end:   fset.Position(bl.End()).Offset,
			text:  strconv.Quote(want),
		})
	}
	return edits, nil
}

// stringField returns the unquoted value of a string-literal field.
func stringField(fields map[string]*ast.BasicLit, name string) (string, error) {
	bl, ok := fields[name]
	if !ok {
		return "", fmt.Errorf("%w: no %s field", errMalformedRef, name)
	}
	if bl == nil {
		return "", fmt.Errorf("%w: %s is not a string literal", errMalformedRef, name)
	}
	v, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", fmt.Errorf("%w: unquoting %s: %w", errMalformedRef, name, err)
	}
	return v, nil
}

// applySplices applies non-overlapping edits to a copy of src.
func applySplices(src []byte, edits []splice) []byte {
	slices.SortFunc(edits, func(a, b splice) int { return a.start - b.start })
	out := make([]byte, 0, len(src))
	last := 0
	for _, e := range edits {
		out = append(out, src[last:e.start]...)
		out = append(out, e.text...)
		last = e.end
	}
	return append(out, src[last:]...)
}
