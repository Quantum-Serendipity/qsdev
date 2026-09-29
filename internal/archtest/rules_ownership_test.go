package archtest

import (
	"slices"
	"sync"
	"testing"
)

const ownershipFixtureRoot = "testdata/ownership"

var (
	ownershipFixtureOnce sync.Once
	ownershipFixture     *Repo
	ownershipFixtureErr  error
)

// loadOwnershipFixture parses the ownership-rule fixture once; tests only
// read it.
func loadOwnershipFixture(t *testing.T) *Repo {
	t.Helper()
	ownershipFixtureOnce.Do(func() { ownershipFixture, ownershipFixtureErr = Load(ownershipFixtureRoot) })
	if ownershipFixtureErr != nil {
		t.Fatalf("Load(%s): %v", ownershipFixtureRoot, ownershipFixtureErr)
	}
	return ownershipFixture
}

// wantOnly checks the cases and that rule has exactly entries keys in got.
func wantOnly(t *testing.T, got Set, rule string, entries int, tests []violationCase) {
	t.Helper()
	wantViolations(t, got, tests)
	if n := countRule(got, rule); n != entries {
		t.Errorf("got %d %s entries, want %d: %v", n, rule, entries, got)
	}
}

func TestSeverityOwnership(t *testing.T) {
	t.Parallel()
	got := Collect(loadOwnershipFixture(t), ownershipRules())
	wantOnly(t, got, "severity-type", 1, []violationCase{
		{"exported defined non-struct type flagged", Key{"severity-type", "internal/consumer"}, 1},
		{"owner package allowed", Key{"severity-type", "internal/finding"}, 0},
	})
}

func TestLiteralOwnership(t *testing.T) {
	t.Parallel()
	got := Collect(loadOwnershipFixture(t), ownershipRules())
	wantViolations(t, got, []violationCase{
		// fixture.lock is only a lockfile name because the fixture's
		// LockFilesByEcosystem lists it; the map key "fixture" is not one.
		{"lockfile names derived from LockFilesByEcosystem", Key{"literal-lockfile", "internal/consumer"}, 2},
		{"lockfile owner allowed", Key{"literal-lockfile", "pkg/ecosystem"}, 0},
		{"lockfile owner subtree allowed", Key{"literal-lockfile", "pkg/ecosystem/modules/golang"}, 0},
		{"settings.json path element; prose, tag and comment ignored", Key{"literal-settings-json", "internal/consumer"}, 1},
		{"settings.json owner allowed", Key{"literal-settings-json", "internal/claudesettings"}, 0},
		{"marker tokens", Key{"literal-marker", "internal/consumer"}, 2},
		{"branding defines markers", Key{"literal-marker", "pkg/branding"}, 0},
		{"projectctx owns markers", Key{"literal-marker", "internal/projectctx"}, 0},
		{"private SARIF document", Key{"sarif-writer", "internal/report"}, 1},
		{"SARIF owner allowed", Key{"sarif-writer", "internal/finding/render"}, 0},
		{"WalkUp outside projectctx", Key{"logging-walkup", "internal/consumer"}, 1},
		{"projectctx may walk up", Key{"logging-walkup", "internal/projectctx"}, 0},
		{"defining package unqualified call ignored", Key{"logging-walkup", "internal/logging"}, 0},
	})
	for _, rule := range []string{"literal-lockfile", "literal-settings-json", "literal-marker", "sarif-writer", "logging-walkup"} {
		if n := countRule(got, rule); n != 1 {
			t.Errorf("got %d %s entries, want 1: %v", n, rule, got)
		}
	}
}

func TestLockfileNames(t *testing.T) {
	t.Parallel()
	got := lockfileNames(loadOwnershipFixture(t))
	slices.Sort(got)
	if want := []string{"fixture.lock", "go.sum"}; !slices.Equal(got, want) {
		t.Errorf("lockfileNames = %q, want %q", got, want)
	}
}

// TestLiteralOwnership_MissingSourceFailsClosed proves that losing the
// derived value list reports a violation instead of silently passing.
func TestLiteralOwnership_MissingSourceFailsClosed(t *testing.T) {
	t.Parallel()
	got := Collect(loadSymbolFixture(t), []Rule{{ID: "literal-lockfile", Check: lockfileRule().check}})
	if len(got) != 1 {
		t.Fatalf("got %v, want one missing-source violation", got)
	}
	for k := range got {
		if k.Subject != noValuesSubject {
			t.Errorf("subject = %q, want %q", k.Subject, noValuesSubject)
		}
	}
}

func TestFlagRegistration(t *testing.T) {
	t.Parallel()
	got := Collect(loadOwnershipFixture(t), ownershipRules())
	wantOnly(t, got, "flag-raw-json", 1, []violationCase{
		{"Bool and BoolVar json", Key{"flag-raw-json", "addons/foo"}, 2},
		{"cmdutil owns output flags", Key{"flag-raw-json", "internal/cmdutil"}, 0},
	})
	wantOnly(t, got, "flag-raw-force", 1, []violationCase{
		{"BoolP and StringVarP force", Key{"flag-raw-force", "addons/foo"}, 2},
	})
}

func TestPathElements(t *testing.T) {
	t.Parallel()
	tests := []struct {
		lit  string
		want []string
	}{
		{"settings.json", []string{"settings.json"}},
		{"%s/.claude/settings.json", []string{"%s", ".claude", "settings.json"}},
		{`C:\Users\x\.qsdev\bin`, []string{"C:", "Users", "x", ".qsdev", "bin"}},
		{"~/.qsdev//package-lock.json", []string{"~", ".qsdev", "package-lock.json"}},
		{"parsing settings.json: %w", []string{"parsing settings.json: %w"}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.lit, func(t *testing.T) {
			t.Parallel()
			if got := pathElements(tt.lit); !slices.Equal(got, tt.want) {
				t.Errorf("pathElements(%q) = %q, want %q", tt.lit, got, tt.want)
			}
		})
	}
}
