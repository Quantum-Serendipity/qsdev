package validation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckHookPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		policy types.HooksConfig
		want   []string // "field[index]" of each error, in order
	}{
		{name: "empty"},
		{name: "valid", policy: types.HooksConfig{
			FileBoundary: types.FileBoundaryConfig{ExtraReadPaths: []string{"/opt/sdk"}},
			ToolGates:    types.ToolGatesConfig{Allowed: []string{"Read", "mcp__context7__*"}, Denied: []string{"WebFetch"}},
		}},
		{
			name: "invalid entries in every list",
			policy: types.HooksConfig{
				FileBoundary: types.FileBoundaryConfig{ExtraReadPaths: []string{"/"}},
				ToolGates:    types.ToolGatesConfig{Allowed: []string{"Read", "Bash,Read"}, Denied: []string{""}},
			},
			want: []string{"hooks.file_boundary.extra_read_paths[0]", "hooks.tool_gates.allowed[1]", "hooks.tool_gates.denied[0]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := CheckHookPolicy(tt.policy)
			if len(errs) != len(tt.want) {
				t.Fatalf("CheckHookPolicy = %v, want %d errors %v", errs, len(tt.want), tt.want)
			}
			for i, e := range errs {
				if got := fmt.Sprintf("%s[%d]", e.Field, e.Index); got != tt.want[i] {
					t.Errorf("error %d at %s, want %s", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestPolicyEntryError_UnwrapsToolNamePattern(t *testing.T) {
	t.Parallel()
	errs := CheckHookPolicy(types.HooksConfig{ToolGates: types.ToolGatesConfig{Denied: []string{"a b"}}})
	if len(errs) != 1 || !errors.Is(errs[0], ErrToolNamePattern) {
		t.Fatalf("CheckHookPolicy = %v, want one ErrToolNamePattern", errs)
	}
	if want := `hooks.tool_gates.denied entry "a b": ` + ErrToolNamePattern.Error(); errs[0].Error() != want {
		t.Errorf("Error() = %q, want %q", errs[0].Error(), want)
	}
}

// TestCheckHookPolicy_PackageGuard covers hooks.package_guard: list entries
// are package names, optionally qualified by an ecosystem package-guard.py
// knows; min_age_days is at least one day; new_dep_gate is an enumerated
// value. Scalar errors carry Index -1.
func TestCheckHookPolicy_PackageGuard(t *testing.T) {
	t.Parallel()
	zero, seven := 0, 7
	tests := []struct {
		name  string
		guard types.PackageGuardConfig
		want  []string // "field[index]" of each error, in order
	}{
		{name: "empty"},
		{name: "valid entries", guard: types.PackageGuardConfig{
			Allowlist:  []string{"@scope/pkg", "pypi:Flask_Cors", "github.com/a/b", "jsr:@s/n", "NPM:left-pad", "lodash.merge"},
			Denylist:   []string{"npm:evil", "golang:github.com/x/y", "crates.io:serde_json"},
			MinAgeDays: &seven,
			NewDepGate: types.NewDepGateAsk,
		}},
		{name: "min_age_days 0", guard: types.PackageGuardConfig{MinAgeDays: &zero}, want: []string{"hooks.package_guard.min_age_days[-1]"}},
		{name: "new_dep_gate maybe", guard: types.PackageGuardConfig{NewDepGate: "maybe"}, want: []string{"hooks.package_guard.new_dep_gate[-1]"}},
		{name: "comma", guard: types.PackageGuardConfig{Denylist: []string{"ok", "a,b"}}, want: []string{"hooks.package_guard.denylist[1]"}},
		{name: "unknown qualifier", guard: types.PackageGuardConfig{Allowlist: []string{"npmm:x"}}, want: []string{"hooks.package_guard.allowlist[0]"}},
		{name: "whitespace", guard: types.PackageGuardConfig{Allowlist: []string{"left pad", "tab\tname", " lead"}}, want: []string{
			"hooks.package_guard.allowlist[0]", "hooks.package_guard.allowlist[1]", "hooks.package_guard.allowlist[2]",
		}},
		{name: "empty, empty name and control", guard: types.PackageGuardConfig{Denylist: []string{"", "npm:", "a" + string(rune(0x200b)) + "b", "x\x00"}}, want: []string{
			"hooks.package_guard.denylist[0]", "hooks.package_guard.denylist[1]", "hooks.package_guard.denylist[2]", "hooks.package_guard.denylist[3]",
		}},
		{name: "all fields in order", guard: types.PackageGuardConfig{
			Allowlist: []string{"a b"}, Denylist: []string{"a,b"}, MinAgeDays: &zero, NewDepGate: "maybe",
		}, want: []string{
			"hooks.package_guard.allowlist[0]", "hooks.package_guard.denylist[0]",
			"hooks.package_guard.min_age_days[-1]", "hooks.package_guard.new_dep_gate[-1]",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := CheckHookPolicy(types.HooksConfig{PackageGuard: tt.guard})
			if len(errs) != len(tt.want) {
				t.Fatalf("CheckHookPolicy = %v, want %d errors %v", errs, len(tt.want), tt.want)
			}
			for i, e := range errs {
				if got := fmt.Sprintf("%s[%d]", e.Field, e.Index); got != tt.want[i] {
					t.Errorf("error %d at %s, want %s", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestPolicyEntryError_ScalarRendersWithoutEntry(t *testing.T) {
	t.Parallel()
	zero := 0
	errs := CheckHookPolicy(types.HooksConfig{PackageGuard: types.PackageGuardConfig{MinAgeDays: &zero}})
	if len(errs) != 1 || !errors.Is(errs[0], ErrPackageGuardMinAgeDays) {
		t.Fatalf("CheckHookPolicy = %v, want one ErrPackageGuardMinAgeDays", errs)
	}
	if want := `hooks.package_guard.min_age_days "0": ` + ErrPackageGuardMinAgeDays.Error(); errs[0].Error() != want {
		t.Errorf("Error() = %q, want %q", errs[0].Error(), want)
	}
}
