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

func TestHookPolicyError_UnwrapsToolNamePattern(t *testing.T) {
	t.Parallel()
	errs := CheckHookPolicy(types.HooksConfig{ToolGates: types.ToolGatesConfig{Denied: []string{"a b"}}})
	if len(errs) != 1 || !errors.Is(errs[0], ErrToolNamePattern) {
		t.Fatalf("CheckHookPolicy = %v, want one ErrToolNamePattern", errs)
	}
	if want := `hooks.tool_gates.denied entry "a b": ` + ErrToolNamePattern.Error(); errs[0].Error() != want {
		t.Errorf("Error() = %q, want %q", errs[0].Error(), want)
	}
}
