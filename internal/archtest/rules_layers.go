package archtest

import (
	"slices"
	"strings"
)

// Rules returns every architecture rule checked against baseline.txt.
func Rules() []Rule {
	return slices.Concat(layerRules(), symbolRules(), ownershipRules())
}

// layerRule forbids import edges from packages under From (minus Except)
// to packages under Deny. Patterns are module-relative directories
// ("internal/catalog") or external import paths ("net/http"), each matching
// itself and everything below it; Allow carves targets back out of Deny.
// Transitive rules follow the importer's module-internal dependency closure,
// so a denied package reached through a helper still counts. An empty From
// means every package. Test files are exempt.
type layerRule struct {
	ID         string
	From       []string
	Except     []string
	Deny       []string
	Allow      []string
	Transitive bool
}

// foundationPkgs may import only stdlib, pkg/branding and each other, so the
// hook runtime can use them without the catalog, TUI or network stacks
// (architecture §3.1).
var foundationPkgs = []string{
	"internal/projectctx", "internal/finding", "internal/procexec", "internal/nixexpr",
	"internal/pathmatch", "internal/secrets", "internal/claudesettings", "internal/mcpconfig",
	"pkg/fileutil",
}

// layerTable is the §3.1 layer matrix. Import cycles need no rule: the
// compiler rejects them.
var layerTable = []layerRule{
	// procexec is the sole owner of os/exec (exec-command), so a pkg package
	// that starts a process can only satisfy both rules by importing it.
	{ID: "pkg-public-leaf", From: []string{"pkg"}, Deny: []string{"internal", "addons", "instance"}, Allow: []string{"internal/procexec"}},
	{ID: "internal-no-adapters", From: []string{"internal"}, Except: []string{"internal/app"}, Deny: []string{"addons", "instance"}},
	{ID: "internal-no-app", From: []string{"internal"}, Except: []string{"internal/app"}, Deny: []string{"internal/app"}},
	{ID: "app-no-adapters", From: []string{"internal/app"}, Deny: []string{"addons", "instance"}},
	{
		ID: "foundation-leaf", From: foundationPkgs, Transitive: true,
		Deny: []string{"internal/catalog", "internal/toolreg", "internal/posture", "github.com/charmbracelet", "net/http"},
	},
	{
		ID: "hookrt-lean", From: []string{"internal/hookrt"}, Transitive: true,
		Deny: []string{"github.com/charmbracelet", "internal/posture", "internal/mcpregistry", "internal/selfupdate"},
	},
	// secretstest assembles credential-shaped samples; a production import
	// would ship them in the binary, so only _test.go files may import it.
	{ID: "secretstest-test-only", Except: []string{secretstestPkg}, Deny: []string{secretstestPkg}},
}

const secretstestPkg = "internal/secrets/secretstest"

func layerRules() []Rule {
	rules := make([]Rule, 0, len(layerTable))
	for _, lr := range layerTable {
		rules = append(rules, Rule{ID: lr.ID, Check: lr.check})
	}
	return rules
}

func (lr layerRule) check(repo *Repo) []Violation {
	graph := importGraph(repo)
	var out []Violation
	for _, pkg := range sortedPkgs(graph) {
		if (len(lr.From) > 0 && !underAny(pkg, lr.From)) || underAny(pkg, lr.Except) {
			continue
		}
		targets := graph[pkg]
		if lr.Transitive {
			targets = closureImports(graph, pkg)
		}
		for _, to := range targets {
			if underAny(to, lr.Deny) && !underAny(to, lr.Allow) {
				out = append(out, Violation{Rule: lr.ID, Subject: pkg + " -> " + to, Count: 1})
			}
		}
	}
	return out
}

// importGraph maps each package with non-test files to its sorted, distinct
// non-test imports. Module imports appear as module-relative directories.
func importGraph(repo *Repo) map[string][]string {
	graph := map[string][]string{}
	for _, f := range repo.Files {
		if f.IsTest {
			continue
		}
		for _, imp := range f.ImportPaths() {
			if rel, ok := repo.Rel(imp); ok {
				imp = rel
			}
			graph[f.Pkg] = append(graph[f.Pkg], imp)
		}
		if _, ok := graph[f.Pkg]; !ok {
			graph[f.Pkg] = nil
		}
	}
	for pkg, imps := range graph {
		slices.Sort(imps)
		graph[pkg] = slices.Compact(imps)
	}
	return graph
}

// closureImports returns the sorted, distinct imports of pkg and of every
// module package it reaches.
func closureImports(graph map[string][]string, pkg string) []string {
	seen := map[string]bool{pkg: true}
	queue := []string{pkg}
	var imports []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, imp := range graph[cur] {
			imports = append(imports, imp)
			if _, internal := graph[imp]; internal && !seen[imp] {
				seen[imp] = true
				queue = append(queue, imp)
			}
		}
	}
	slices.Sort(imports)
	return slices.Compact(imports)
}

func underAny(p string, patterns []string) bool {
	return slices.ContainsFunc(patterns, func(pat string) bool {
		return p == pat || strings.HasPrefix(p, pat+"/")
	})
}

func sortedPkgs(graph map[string][]string) []string {
	pkgs := make([]string, 0, len(graph))
	for p := range graph {
		pkgs = append(pkgs, p)
	}
	slices.Sort(pkgs)
	return pkgs
}
