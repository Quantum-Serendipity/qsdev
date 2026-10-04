package archtest

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Repo is every Go source file of one module, parsed.
type Repo struct {
	// Module is the module path from go.mod.
	Module string
	// Files is sorted by Path.
	Files []*File
}

// File is one parsed Go source file.
type File struct {
	// Pkg is the package directory relative to the module root, slash
	// separated ("." for the root).
	Pkg string
	// Path is the file path relative to the module root, slash separated.
	Path   string
	IsTest bool
	AST    *ast.File
	// Imports maps each name an import is referenced by (its alias, or the
	// default package name) to the import path. Blank, dot and cgo imports
	// are listed in AST only.
	Imports map[string]string
}

// ErrNoModule reports a root without a readable module directive.
var ErrNoModule = errors.New("no module directive in go.mod")

// Load parses every .go file under root, which must hold a go.mod. It skips
// vendor and testdata trees, hidden and underscore directories, and nested
// modules, exactly as the go tool does, but ignores build constraints.
func Load(root string) (*Repo, error) {
	module, err := readModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	repo := &Repo{Module: module}
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDir(root, p, d.Name())
		}
		if !isGoSource(d.Name()) {
			return nil
		}
		f, err := parseFile(fset, root, p)
		if err != nil {
			return err
		}
		repo.Files = append(repo.Files, f)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", root, err)
	}
	sort.Slice(repo.Files, func(i, j int) bool { return repo.Files[i].Path < repo.Files[j].Path })
	return repo, nil
}

// Rel converts an import path inside the module to its slash-separated
// directory relative to the module root. ok is false for other modules.
func (r *Repo) Rel(importPath string) (rel string, ok bool) {
	if importPath == r.Module {
		return ".", true
	}
	if rest, found := strings.CutPrefix(importPath, r.Module+"/"); found {
		return rest, true
	}
	return "", false
}

func skipDir(root, p, name string) error {
	if p == root {
		return nil
	}
	if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return filepath.SkipDir
	}
	if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
		return filepath.SkipDir
	}
	return nil
}

func isGoSource(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "_")
}

func parseFile(fset *token.FileSet, root, p string) (*File, error) {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return nil, fmt.Errorf("relativising %s: %w", p, err)
	}
	rel = filepath.ToSlash(rel)
	src, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", rel, err)
	}
	f := &File{
		Pkg:     path.Dir(rel),
		Path:    rel,
		IsTest:  strings.HasSuffix(rel, "_test.go"),
		AST:     src,
		Imports: make(map[string]string, len(src.Imports)),
	}
	for _, spec := range src.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("%s: import %s: %w", rel, spec.Path.Value, err)
		}
		name := defaultImportName(imp)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name != "_" && name != "." && imp != "C" {
			f.Imports[name] = imp
		}
	}
	return f, nil
}

// ImportPaths returns the paths f imports, in source order.
func (f *File) ImportPaths() []string {
	out := make([]string, 0, len(f.AST.Imports))
	for _, spec := range f.AST.Imports {
		if imp, err := strconv.Unquote(spec.Path.Value); err == nil {
			out = append(out, imp)
		}
	}
	return out
}

var (
	majorVersion = regexp.MustCompile(`^v[0-9]+$`)
	gopkgVersion = regexp.MustCompile(`\.v[0-9]+$`)
)

// defaultImportName approximates the package name of an unaliased import
// from its path: the last element, skipping a /vN major-version suffix and
// a gopkg.in .vN suffix. That is exact for the standard library.
func defaultImportName(importPath string) string {
	elems := strings.Split(importPath, "/")
	name := elems[len(elems)-1]
	if len(elems) > 1 && majorVersion.MatchString(name) {
		name = elems[len(elems)-2]
	}
	return gopkgVersion.ReplaceAllString(name, "")
}

func readModulePath(gomod string) (string, error) {
	f, err := os.Open(gomod)
	if err != nil {
		return "", fmt.Errorf("reading module path: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", gomod, err)
	}
	return "", fmt.Errorf("%s: %w", gomod, ErrNoModule)
}
