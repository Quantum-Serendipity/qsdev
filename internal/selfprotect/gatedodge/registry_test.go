package gatedodge

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// TestGuardedFileRegistryConsistency pins the registry's invariants: each
// entry is named by a unique lower-cased base name, reports a rule ID and
// checks something; canon's guarded list (which the hook sandbox and the
// shell-rewrite deny read) names only files with a Change check; and every
// entry is found by its name in any case and under either separator.
func TestGuardedFileRegistryConsistency(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, g := range guardedFiles {
		if g.Name != strings.ToLower(g.Name) || g.Name != path.Base(g.Name) || strings.ContainsAny(g.Name, `/\`) {
			t.Errorf("entry %q: Name must be a lower-cased base name", g.Name)
		}
		if seen[g.Name] {
			t.Errorf("entry %q listed twice", g.Name)
		}
		seen[g.Name] = true
		if g.RuleID == "" {
			t.Errorf("entry %q has no RuleID", g.Name)
		}
		if g.Introduced == nil && g.Change == nil {
			t.Errorf("entry %q has neither an Introduced nor a Change check", g.Name)
		}
		for _, p := range []string{g.Name, strings.ToUpper(g.Name), "/x/" + g.Name, `C:\p\` + g.Name} {
			if got := Lookup(p); len(got) != 1 || got[0].Name != g.Name {
				t.Errorf("Lookup(%q) = %v, want the %q entry", p, names(got), g.Name)
			}
		}
	}

	changed := ChangeGuardedNames()
	if !slices.IsSorted(changed) {
		t.Errorf("ChangeGuardedNames() = %v, want sorted", changed)
	}
	for _, name := range canon.GuardedConfigFiles() {
		if !slices.Contains(changed, name) {
			t.Errorf("canon.GuardedConfigFiles() names %q, which has no Change check (ChangeGuardedNames() = %v)", name, changed)
		}
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"upper case", []string{".QSDEV.YAML"}, []string{"GD-001"}},
		{"mixed case under a unix dir", []string{"/x/.Npmrc"}, []string{"GD-004"}},
		{"windows path", []string{`C:\p\devenv.nix`}, []string{"GD-002"}},
		{"claude.md", []string{"/p/CLAUDE.md"}, []string{"GD-005"}},
		{"canonical before input", []string{"/a/.qsdev.yaml", "claude.md"}, []string{"GD-001", "GD-005"}},
		{"input then canonical keeps order", []string{"claude.md", "/a/.qsdev.yaml"}, []string{"GD-005", "GD-001"}},
		{"deduplicated", []string{"/a/.npmrc", ".NPMRC"}, []string{"GD-004"}},
		{"empty path skipped", []string{"", "/a/.npmrc"}, []string{"GD-004"}},
		{"guarded name linked elsewhere", []string{"/shared/npmrc-shared", "/p/.npmrc"}, []string{"GD-004"}},
		// Windows opens the file through its name aliases.
		{"trailing dot", []string{".qsdev.yaml."}, []string{"GD-001"}},
		{"trailing dots and spaces", []string{`C:\p\.npmrc. .`}, []string{"GD-004"}},
		{"data stream", []string{".qsdev.yaml::$DATA"}, []string{"GD-001"}},
		{"named stream under a dir", []string{"/p/devenv.nix:x"}, []string{"GD-002"}},
		{"suffix is not the name", []string{"/p/x.npmrc"}, nil},
		{"leading dot is kept", []string{"/p/npmrc"}, nil},
		{"unrelated", []string{"/p/main.go"}, nil},
		{"empty", []string{""}, nil},
		{"none", nil, nil},
		// Ported from TestResultRuleFor.
		{"bare .npmrc", []string{".npmrc"}, []string{"GD-004"}},
		{"/p/.NPMRC", []string{"/p/.NPMRC"}, []string{"GD-004"}},
		{"/p/.Pre-Commit-Config.yaml", []string{"/p/.Pre-Commit-Config.yaml"}, []string{"GD-003"}},
		{"/p/.pre-commit-config.yaml", []string{"/p/.pre-commit-config.yaml"}, []string{"GD-003"}},
		{"/p/pnpm-workspace.yaml", []string{"/p/pnpm-workspace.yaml"}, []string{"GD-004"}},
		{"/p/.yarnrc.yml", []string{"/p/.yarnrc.yml"}, []string{"GD-004"}},
		{"/p/.yarnrc", []string{"/p/.yarnrc"}, []string{"GD-004"}},
		{"/p/bunfig.toml", []string{"/p/bunfig.toml"}, []string{"GD-004"}},
		// .qsdev.yaml had no result rule before the registry folded GD-001 in.
		{"/p/.qsdev.yaml", []string{"/p/.qsdev.yaml"}, []string{"GD-001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, g := range Lookup(tt.paths...) {
				got = append(got, g.RuleID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Lookup(%q) rule IDs = %v, want %v", tt.paths, got, tt.want)
			}
		})
	}
}

func names(gs []GuardedFile) []string {
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, g.Name)
	}
	return out
}
