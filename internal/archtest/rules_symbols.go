package archtest

import (
	"go/ast"
)

// symbolBan forbids references to Pkg.Name outside the Owners directories.
// Pkg is a module-relative directory ("internal/catalog") or a standard
// import path ("os/exec"); references resolve through each file's own import
// names, so aliases are caught and same-named locals and methods are not.
// Only non-test files are checked unless Tests is set, which checks only
// test files.
type symbolBan struct {
	ID     string
	Pkg    string
	Name   string
	Owners []string
	Tests  bool
}

var (
	cmdInstance     = []string{"cmd", "instance"}
	projectctxOwner = []string{"internal/projectctx"}
	singletonOwners = []string{"instance", "internal/projectctx"}
	// mcpregistryOwner is the only package that may start or dial a
	// configured MCP server: ProbeAll probes only what PlanProbes trusts.
	mcpregistryOwner = []string{"internal/mcpregistry"}
)

// symbolTable holds the "one way to do X" bans of architecture §5.
var symbolTable = []symbolBan{
	{ID: "exec-command", Pkg: "os/exec", Name: "Command", Owners: []string{"internal/procexec"}},
	{ID: "exec-commandcontext", Pkg: "os/exec", Name: "CommandContext", Owners: []string{"internal/procexec"}},
	{ID: "exec-lookpath", Pkg: "os/exec", Name: "LookPath", Owners: []string{"internal/procexec"}},
	{ID: "os-writefile", Pkg: "os", Name: "WriteFile"},
	{ID: "context-background", Pkg: "context", Name: "Background", Owners: cmdInstance},
	{ID: "context-todo", Pkg: "context", Name: "TODO", Owners: cmdInstance},
	{ID: "os-getwd", Pkg: "os", Name: "Getwd", Owners: projectctxOwner},
	{ID: "os-userhomedir", Pkg: "os", Name: "UserHomeDir", Owners: projectctxOwner},
	{ID: "catalog-mustdefault", Pkg: "internal/catalog", Name: "MustDefault", Owners: singletonOwners},
	{ID: "toolreg-defaultregistry", Pkg: "internal/toolreg", Name: "DefaultRegistry", Owners: singletonOwners},
	{ID: "ecosystem-defaultregistry", Pkg: "pkg/ecosystem", Name: "DefaultRegistry", Owners: singletonOwners},
	{ID: "config-parseqsdevconfig", Pkg: "internal/config", Name: "ParseQsdevConfig", Owners: []string{"internal/projectmodel"}},
	{ID: "os-stderr", Pkg: "os", Name: "Stderr", Owners: []string{"cmd", "instance", "internal/procexec"}},
	{ID: "test-os-chdir", Pkg: "os", Name: "Chdir", Tests: true},
	{ID: "test-os-unsetenv", Pkg: "os", Name: "Unsetenv", Tests: true},
	{ID: "mcphealth-checkall", Pkg: "internal/mcphealth", Name: "CheckAll", Owners: mcpregistryOwner},
	{ID: "mcphealth-checkserver", Pkg: "internal/mcphealth", Name: "CheckServer", Owners: mcpregistryOwner},
	{ID: "mcphealth-probetarget", Pkg: "internal/mcphealth", Name: "ProbeTarget", Owners: mcpregistryOwner},
}

// panicOwners may panic: procexec's forbid-exec guard is a test-only trap
// for a programmer error that an error return could let callers swallow.
var panicOwners = []string{"internal/procexec"}

// initOwners may self-register from init(): the ecosystem SPI modules and
// the extlog providers. Everywhere else registration is explicit.
var initOwners = []string{"pkg/ecosystem/modules", "internal/extlog/providers"}

func symbolRules() []Rule {
	rules := make([]Rule, 0, len(symbolTable)+2)
	for _, b := range symbolTable {
		rules = append(rules, Rule{ID: b.ID, Check: b.check})
	}
	return append(rules,
		Rule{ID: "panic", Check: checkPanic},
		Rule{ID: "init-non-module", Check: checkInit},
	)
}

func (b symbolBan) check(repo *Repo) []Violation {
	return countPerPkg(repo, b.ID, b.Owners, b.Tests, func(f *File, n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != b.Name {
			return false
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return false
		}
		imp, ok := f.Imports[x.Name]
		if !ok {
			return false
		}
		if rel, inModule := repo.Rel(imp); inModule {
			imp = rel
		}
		return imp == b.Pkg
	})
}

// checkPanic counts calls of the panic builtin in library code outside
// panicOwners.
func checkPanic(repo *Repo) []Violation {
	return countPerPkg(repo, "panic", panicOwners, false, func(_ *File, n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return false
		}
		id, ok := call.Fun.(*ast.Ident)
		return ok && id.Name == "panic"
	})
}

// checkInit counts package initializers outside initOwners, standing in for
// gochecknoinits scoped to non-module packages.
func checkInit(repo *Repo) []Violation {
	return countPerPkg(repo, "init-non-module", initOwners, false, func(_ *File, n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		return ok && fn.Recv == nil && fn.Name.Name == "init"
	})
}

// countPerPkg counts the nodes matching match in the files of each package
// outside owners, restricted to test files when tests is set and to
// non-test files otherwise.
func countPerPkg(repo *Repo, rule string, owners []string, tests bool, match func(*File, ast.Node) bool) []Violation {
	counts := map[string]int{}
	for _, f := range repo.Files {
		if f.IsTest != tests || underAny(f.Pkg, owners) {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if n != nil && match(f, n) {
				counts[f.Pkg]++
			}
			return true
		})
	}
	out := make([]Violation, 0, len(counts))
	for pkg, n := range counts {
		out = append(out, Violation{Rule: rule, Subject: pkg, Count: n})
	}
	return out
}
