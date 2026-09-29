package cigeneration

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

// catalogVarNames returns the package-level variables in this package's
// non-test sources that are initialised with a typeName composite literal,
// e.g. `ImageSnyk = ImageRef{...}`. The coverage tests compare it against the
// hand-written verification sets, so a pin added to the catalog but not to
// those sets fails offline instead of silently going unverified upstream.
func catalogVarNames(t *testing.T, typeName string) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listing package sources: %v", err)
	}

	fset := token.NewFileSet()
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		names = append(names, fileVarNames(file, typeName)...)
	}
	slices.Sort(names)
	return names
}

// fileVarNames returns the package-level variables in file initialised with a
// typeName composite literal.
func fileVarNames(file *ast.File, typeName string) []string {
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, v := range vs.Values {
				lit, ok := v.(*ast.CompositeLit)
				if !ok {
					continue
				}
				if id, ok := lit.Type.(*ast.Ident); ok && id.Name == typeName {
					names = append(names, vs.Names[i].Name)
				}
			}
		}
	}
	return names
}

// assertCoversCatalog fails when the verification set and the catalog declared
// in source differ in either direction.
func assertCoversCatalog[V any](t *testing.T, typeName string, verified map[string]V) {
	t.Helper()

	declared := catalogVarNames(t, typeName)
	if len(declared) == 0 {
		t.Fatalf("found no %s variables in the package sources; the source scan is broken", typeName)
	}
	for _, name := range declared {
		if _, ok := verified[name]; !ok {
			t.Errorf("%s %s is declared but missing from the verification set, so it is never checked upstream", typeName, name)
		}
	}
	for name := range verified {
		if !slices.Contains(declared, name) {
			t.Errorf("verification set names %s, which is not a %s declared in the catalog", name, typeName)
		}
	}
}
