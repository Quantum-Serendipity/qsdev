package archtest

import (
	"go/ast"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// literalRule forbids string literals naming an owned value outside Owners.
// A literal names a value when one of its path elements (see pathElements)
// equals it, so bare names, paths and path formats are caught while prose
// such as error messages, comments, struct tags and import paths are not. Values derives the list, from the
// repository when a canonical table exists, so the rule never keeps a second
// copy of it. A rule whose Values come back empty reports noValuesSubject
// instead of silently passing.
//
// New ownership canons (credential patterns, ecosystem ids) are new rows
// here or in ownershipRules; the engine needs no change.
type literalRule struct {
	ID     string
	Owners []string
	Values func(*Repo) []string
}

// flagRule forbids registering a command-line flag called Name outside
// internal/cmdutil, whose shared helpers give it one meaning everywhere.
type flagRule struct {
	ID   string
	Name string
}

const (
	lockfileSourcePkg = "pkg/ecosystem"
	lockfileSourceVar = "LockFilesByEcosystem"
	// noValuesSubject marks a literal rule that found nothing to own, which
	// means its source table moved and the rule must be repointed.
	noValuesSubject = "(no owned literals found: repoint the rule's source)"
)

// archtestPkg owns every literal: its rule table has to name them.
const archtestPkg = "internal/archtest"

var projectctxMarkerOwners = []string{"internal/projectctx", "pkg/branding"}

// literalTable is the literal-ownership half of XS-WS13 M1.
var literalTable = []literalRule{
	{ID: "literal-settings-json", Owners: []string{"internal/claudesettings"}, Values: fixedValues("settings.json")},
	{ID: "literal-marker", Owners: projectctxMarkerOwners, Values: fixedValues(".qsdev.yaml", ".devinit", ".qsdev")},
	lockfileRule(),
}

var flagTable = []flagRule{
	{ID: "flag-raw-json", Name: "json"},
	{ID: "flag-raw-force", Name: "force"},
}

// ownershipTable holds the symbol bans that guard a single owner.
var ownershipTable = []symbolBan{
	{ID: "logging-walkup", Pkg: "internal/logging", Name: "WalkUp", Owners: projectctxOwner},
}

// flagNameArg is the index of the name argument of each pflag registration
// method that can declare a "json" or "force" flag.
var flagNameArg = map[string]int{
	"Bool": 0, "BoolP": 0, "BoolVar": 1, "BoolVarP": 1,
	"String": 0, "StringP": 0, "StringVar": 1, "StringVarP": 1,
}

func ownershipRules() []Rule {
	rules := []Rule{
		{ID: "severity-type", Check: checkSeverityTypes},
		{ID: "sarif-writer", Check: checkSARIFWriters},
	}
	for _, lr := range literalTable {
		rules = append(rules, Rule{ID: lr.ID, Check: lr.check})
	}
	for _, fr := range flagTable {
		rules = append(rules, Rule{ID: fr.ID, Check: fr.check})
	}
	for _, b := range ownershipTable {
		rules = append(rules, Rule{ID: b.ID, Check: b.check})
	}
	return rules
}

// lockfileRule owns the lockfile names listed by pkg/ecosystem.
func lockfileRule() literalRule {
	return literalRule{ID: "literal-lockfile", Owners: []string{lockfileSourcePkg}, Values: lockfileNames}
}

func fixedValues(values ...string) func(*Repo) []string {
	return func(*Repo) []string { return values }
}

// lockfileNames reads the string values of pkg/ecosystem's
// LockFilesByEcosystem from source; its keys are ecosystem ids, not names.
func lockfileNames(repo *Repo) []string {
	var names []string
	for _, f := range repo.Files {
		if f.Pkg != lockfileSourcePkg || f.IsTest {
			continue
		}
		for _, v := range packageVarValues(f, lockfileSourceVar) {
			lit, ok := v.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, elt := range lit.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					elt = kv.Value
				}
				names = append(names, stringLits(elt)...)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// packageVarValues returns the initialisers of the package-level var name.
func packageVarValues(f *File, name string) []ast.Expr {
	for _, decl := range f.AST.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, id := range vs.Names {
				if id.Name == name && i < len(vs.Values) {
					return vs.Values[i : i+1]
				}
			}
		}
	}
	return nil
}

func stringLits(n ast.Node) []string {
	var out []string
	ast.Inspect(n, func(n ast.Node) bool {
		if s, ok := stringLit(n); ok {
			out = append(out, s)
		}
		return true
	})
	return out
}

func stringLit(n ast.Node) (string, bool) {
	lit, ok := n.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

func (lr literalRule) check(repo *Repo) []Violation {
	values := lr.Values(repo)
	if len(values) == 0 {
		return []Violation{{Rule: lr.ID, Subject: noValuesSubject, Count: 1}}
	}
	skip := nonValueLiterals(repo)
	return countPerPkg(repo, lr.ID, append(slices.Clip(lr.Owners), archtestPkg), false, func(_ *File, n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); !ok || skip[lit] {
			return false
		}
		s, ok := stringLit(n)
		return ok && slices.ContainsFunc(pathElements(s), func(tok string) bool { return slices.Contains(values, tok) })
	})
}

// nonValueLiterals collects the string literals that are metadata rather
// than values: struct field tags and import paths.
func nonValueLiterals(repo *Repo) map[*ast.BasicLit]bool {
	skip := map[*ast.BasicLit]bool{}
	for _, f := range repo.Files {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Field:
				if n.Tag != nil {
					skip[n.Tag] = true
				}
			case *ast.ImportSpec:
				skip[n.Path] = true
			}
			return true
		})
	}
	return skip
}

// pathElements splits s on both path separators, so "~/.qsdev/bin" and
// `%s\.qsdev` both yield ".qsdev" on every GOOS.
func pathElements(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '\\' })
}

// checkSeverityTypes counts exported defined non-struct types named
// *Severity outside internal/finding, which owns the one Severity enum.
func checkSeverityTypes(repo *Repo) []Violation {
	return countPerPkg(repo, "severity-type", []string{"internal/finding"}, false, func(_ *File, n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Assign.IsValid() || !ts.Name.IsExported() || !strings.HasSuffix(ts.Name.Name, "Severity") {
			return false
		}
		_, isStruct := ts.Type.(*ast.StructType)
		return !isStruct
	})
}

// checkSARIFWriters counts SARIF log document types outside
// internal/finding/render: structs with both "$schema" and "runs" JSON
// fields.
func checkSARIFWriters(repo *Repo) []Violation {
	return countPerPkg(repo, "sarif-writer", []string{"internal/finding/render"}, false, func(_ *File, n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok {
			return false
		}
		names := jsonFieldNames(st)
		return slices.Contains(names, "$schema") && slices.Contains(names, "runs")
	})
}

func jsonFieldNames(st *ast.StructType) []string {
	var names []string
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag, ok := stringLit(field.Tag)
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
		names = append(names, name)
	}
	return names
}

func (fr flagRule) check(repo *Repo) []Violation {
	return countPerPkg(repo, fr.ID, []string{"internal/cmdutil"}, false, func(_ *File, n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		i, ok := flagNameArg[sel.Sel.Name]
		if !ok || i >= len(call.Args) {
			return false
		}
		name, ok := stringLit(call.Args[i])
		return ok && name == fr.Name
	})
}
