package validation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckClaudePermissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		perms types.ClaudePermissionsConfig
		want  []string // "field[index]" of each error, in order
	}{
		{name: "empty"},
		{name: "valid", perms: types.ClaudePermissionsConfig{
			Allow: []string{"Bash(make *)", "Read", "mcp__context7__*", "WebFetch(domain:example.com)", "Bash(echo (x))"},
			Deny:  []string{"Read(/secrets/**)", "Bash(terraform apply *)"},
		}},
		{
			name: "malformed entries",
			perms: types.ClaudePermissionsConfig{
				Allow: []string{"Bash(make *", "Bash()", "(make)", "", "Bash make"},
				Deny:  []string{"Bash(x))y", "Read)", "Bash(a\nb)", "Bash(rm\u200b *)"},
			},
			want: []string{
				"claude_code.permissions.allow[0]", "claude_code.permissions.allow[1]",
				"claude_code.permissions.allow[2]", "claude_code.permissions.allow[3]",
				"claude_code.permissions.allow[4]",
				"claude_code.permissions.deny[0]", "claude_code.permissions.deny[1]",
				"claude_code.permissions.deny[2]", "claude_code.permissions.deny[3]",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := CheckClaudePermissions(tt.perms)
			if len(errs) != len(tt.want) {
				t.Fatalf("CheckClaudePermissions = %v, want %d errors %v", errs, len(tt.want), tt.want)
			}
			for i, e := range errs {
				if got := fmt.Sprintf("%s[%d]", e.Field, e.Index); got != tt.want[i] {
					t.Errorf("error %d at %s, want %s", i, got, tt.want[i])
				}
				if !errors.Is(e, ErrPermissionRule) {
					t.Errorf("error %d = %v, want ErrPermissionRule", i, e)
				}
			}
		})
	}
}
