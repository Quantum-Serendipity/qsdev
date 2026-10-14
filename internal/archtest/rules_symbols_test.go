package archtest

import (
	"sync"
	"testing"
)

const symbolFixtureRoot = "testdata/symbols"

var (
	symbolFixtureOnce sync.Once
	symbolFixture     *Repo
	symbolFixtureErr  error
)

// loadSymbolFixture parses the symbol-rule fixture once; tests only read it.
func loadSymbolFixture(t *testing.T) *Repo {
	t.Helper()
	symbolFixtureOnce.Do(func() { symbolFixture, symbolFixtureErr = Load(symbolFixtureRoot) })
	if symbolFixtureErr != nil {
		t.Fatalf("Load(%s): %v", symbolFixtureRoot, symbolFixtureErr)
	}
	return symbolFixture
}

// wantViolations checks got[key] against each case, where 0 means absent.
func wantViolations(t *testing.T, got Set, tests []violationCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if n := got[tt.key]; n != tt.count {
				t.Errorf("%v count = %d, want %d (all: %v)", tt.key, n, tt.count, got)
			}
		})
	}
}

type violationCase struct {
	name  string
	key   Key
	count int
}

func TestSymbolRules(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"aliased import caught", Key{"os-writefile", "internal/writer"}, 1},
		{"calls counted per package", Key{"exec-command", "internal/writer"}, 2},
		{"windows-only file scanned", Key{"exec-lookpath", "internal/writer"}, 1},
		{"context root outside cmd/instance", Key{"context-background", "internal/writer"}, 1},
		{"os.Stderr reference", Key{"os-stderr", "internal/writer"}, 1},
		{"panic builtin", Key{"panic", "internal/writer"}, 1},
		{"aliased module import caught", Key{"catalog-mustdefault", "internal/user"}, 1},
		{"owner package allowed", Key{"exec-command", "internal/procexec"}, 0},
		{"owner of os.Stderr allowed", Key{"os-stderr", "internal/procexec"}, 0},
		{"procexec guard may panic", Key{"panic", "internal/procexec"}, 0},
		{"instance may create root contexts", Key{"context-background", "instance"}, 0},
		{"instance may use singletons", Key{"catalog-mustdefault", "instance"}, 0},
		{"unqualified call in defining package ignored", Key{"catalog-mustdefault", "internal/catalog"}, 0},
	})
	// Comments, strings, test files and same-named methods on local
	// variables must add nothing beyond the entries above.
	n := len(got) - countRule(got, "init-non-module") - countRule(got, "test-os-chdir") - countRule(got, "test-os-unsetenv") -
		countRule(got, bashLiteralRuleID) - countRule(got, credentialStoreLiteralRuleID)
	for _, r := range mcphealthProbeRules {
		n -= countRule(got, r)
	}
	if n != 7 {
		t.Errorf("got %d symbol-ban entries, want 7: %v", n, got)
	}
}

func TestInitRule(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"addon init flagged", Key{"init-non-module", "addons/foo"}, 2},
		{"ecosystem module allowed", Key{"init-non-module", "pkg/ecosystem/modules/golang"}, 0},
		{"extlog provider allowed", Key{"init-non-module", "internal/extlog/providers/npm"}, 0},
		{"method named init ignored", Key{"init-non-module", "internal/thing"}, 0},
	})
	if n := countRule(got, "init-non-module"); n != 1 {
		t.Errorf("got %d init-non-module entries, want 1: %v", n, got)
	}
}

func TestChdirInTests(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"test file calls", Key{"test-os-chdir", "internal/writer"}, 2},
		{"no owner exemption", Key{"test-os-chdir", "internal/procexec"}, 1},
	})
	if n := countRule(got, "test-os-chdir"); n != 2 {
		t.Errorf("got %d test-os-chdir entries, want 2: %v", n, got)
	}
}

func TestUnsetenvInTests(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"test file calls", Key{"test-os-unsetenv", "internal/writer"}, 1},
		{"non-test file ignored", Key{"test-os-unsetenv", "internal/procexec"}, 0},
	})
	if n := countRule(got, "test-os-unsetenv"); n != 1 {
		t.Errorf("got %d test-os-unsetenv entries, want 1: %v", n, got)
	}
}

func TestBashLiteralRule(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"==, != and case clause counted", Key{bashLiteralRuleID, "internal/toolname"}, 3},
		{"cmdscan owns the shell tool names", Key{bashLiteralRuleID, "internal/selfprotect/cmdscan"}, 0},
		{"denyutil compares permission-rule prefixes", Key{bashLiteralRuleID, "pkg/denyutil"}, 0},
	})
	if n := countRule(got, bashLiteralRuleID); n != 1 {
		t.Errorf("got %d %s entries, want 1 (test files exempt): %v", n, bashLiteralRuleID, got)
	}
}

// TestNoLiteralBashToolComparison requires every hook and rule to ask
// cmdscan.IsShellTool whether a tool runs a shell command: a comparison with
// the literal "Bash" let PowerShell and Monitor bypass the check (U18-07). It
// runs the one rule on the module, which has no baseline entry.
func TestNoLiteralBashToolComparison(t *testing.T) {
	t.Parallel()
	repo, err := Load(moduleRoot)
	if err != nil {
		t.Fatalf("loading module: %v", err)
	}
	dropGitIgnored(t, repo)
	for k, n := range Collect(repo, []Rule{{ID: bashLiteralRuleID, Check: checkBashLiteral}}) {
		t.Errorf("%v: %d comparisons with the literal %q; use cmdscan.IsShellTool", k, n, bashToolName)
	}
}

func TestCredentialStoreLiteralRule(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		// A whole file store, a directory glob, a Bash rule, a raw literal,
		// a store followed by a space and one followed by a quote; the
		// sub-paths, project-relative names, $HOME script and longer
		// siblings in the same package add nothing.
		{"whole-store literals counted", Key{credentialStoreLiteralRuleID, "internal/creds"}, 6},
		{"canon owner allowed", Key{credentialStoreLiteralRuleID, "internal/secrets"}, 0},
	})
	if n := countRule(got, credentialStoreLiteralRuleID); n != 1 {
		t.Errorf("got %d %s entries, want 1 (test files exempt): %v", n, credentialStoreLiteralRuleID, got)
	}
}

// TestNoCredentialStoreLiteralOutsideCanon requires every whole-store
// "~/<rel>" spelling to come from the internal/secrets canon: a private copy
// drifts when the canon grows (XS-WS5 E6). It runs the one rule on the
// module, which has no baseline entry.
func TestNoCredentialStoreLiteralOutsideCanon(t *testing.T) {
	t.Parallel()
	repo, err := Load(moduleRoot)
	if err != nil {
		t.Fatalf("loading module: %v", err)
	}
	dropGitIgnored(t, repo)
	for k, n := range Collect(repo, []Rule{{ID: credentialStoreLiteralRuleID, Check: checkCredentialStoreLiteral}}) {
		t.Errorf("%v: %d whole credential-store literals; derive them from secrets.CredentialStores", k, n)
	}
}

func countRule(s Set, rule string) int {
	n := 0
	for k := range s {
		if k.Rule == rule {
			n++
		}
	}
	return n
}

var mcphealthProbeRules = []string{"mcphealth-checkall", "mcphealth-checkserver", "mcphealth-probetarget"}

// TestSymbolBan_McphealthProbeOnlyFromMcpregistry: only mcpregistry, behind
// PlanProbes' trust gate, may start or dial a configured MCP server.
func TestSymbolBan_McphealthProbeOnlyFromMcpregistry(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), symbolRules())
	wantViolations(t, got, []violationCase{
		{"planted CheckServer via alias and method value", Key{"mcphealth-checkserver", "internal/doctor"}, 2},
		{"planted CheckAll", Key{"mcphealth-checkall", "internal/doctor"}, 1},
		{"planted ProbeTarget", Key{"mcphealth-probetarget", "internal/doctor"}, 1},
		{"mcpregistry owns CheckAll", Key{"mcphealth-checkall", "internal/mcpregistry"}, 0},
		{"mcpregistry owns ProbeTarget", Key{"mcphealth-probetarget", "internal/mcpregistry"}, 0},
		{"unqualified call in mcphealth ignored", Key{"mcphealth-checkserver", "internal/mcphealth"}, 0},
	})
	total := 0
	for _, r := range mcphealthProbeRules {
		total += countRule(got, r)
	}
	if total != 3 {
		t.Errorf("got %d mcphealth probe entries, want 3 (test files exempt): %v", total, got)
	}
}
