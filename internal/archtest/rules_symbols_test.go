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
		{"instance may create root contexts", Key{"context-background", "instance"}, 0},
		{"instance may use singletons", Key{"catalog-mustdefault", "instance"}, 0},
		{"unqualified call in defining package ignored", Key{"catalog-mustdefault", "internal/catalog"}, 0},
	})
	// Comments, strings, test files and same-named methods on local
	// variables must add nothing beyond the entries above.
	if n := len(got) - countRule(got, "init-non-module") - countRule(got, "test-os-chdir"); n != 7 {
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

func countRule(s Set, rule string) int {
	n := 0
	for k := range s {
		if k.Rule == rule {
			n++
		}
	}
	return n
}
