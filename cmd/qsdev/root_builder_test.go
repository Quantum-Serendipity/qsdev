package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootBuildersAreSerial keeps every test that builds the real command
// tree out of the parallel phase. instance.NewRootCommand runs gdev's
// cmd.Root, which AddCommands the shared instance.Commands() onto a fresh
// root and so re-parents them: two parallel builders (or a builder and a
// walker following c.Root()) race on that shared state under -race.
func TestRootBuildersAreSerial(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("listing test files: %v", err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if callsSelector(fn.Body, "", "Parallel") && callsSelector(fn.Body, "instance", "NewRootCommand") {
				t.Errorf("%s: %s calls t.Parallel and builds the real command tree; build it from a serial test", name, fn.Name.Name)
			}
		}
	}
}

// callsSelector reports whether body calls recv.sel(), or any x.sel() when
// recv is empty.
func callsSelector(body ast.Node, recv, sel string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		se, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || se.Sel.Name != sel {
			return true
		}
		if recv == "" {
			found = true
		} else if id, ok := se.X.(*ast.Ident); ok && id.Name == recv {
			found = true
		}
		return !found
	})
	return found
}

func TestCallsSelector(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, src, recv, sel string
		want                 bool
	}{
		{"direct", "t.Parallel()", "", "Parallel", true},
		{"in subtest", `t.Run("x", func(t *testing.T) { t.Parallel() })`, "", "Parallel", true},
		{"receiver matches", "walk(instance.NewRootCommand())", "instance", "NewRootCommand", true},
		{"other receiver", "other.NewRootCommand()", "instance", "NewRootCommand", false},
		{"absent", "x := 1; _ = x", "", "Parallel", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := parser.ParseFile(token.NewFileSet(), "x.go", "package p\nfunc f() {\n"+tt.src+"\n}\n", 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			body := f.Decls[0].(*ast.FuncDecl).Body
			if got := callsSelector(body, tt.recv, tt.sel); got != tt.want {
				t.Errorf("callsSelector(%q, %q, %q) = %v, want %v", tt.src, tt.recv, tt.sel, got, tt.want)
			}
		})
	}
}
