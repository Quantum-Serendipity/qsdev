package types

import (
	"slices"
	"testing"
)

func TestMCPPolicy_Filter(t *testing.T) {
	t.Parallel()
	servers := []string{"context7", "github", "socket", "semble"}
	tests := []struct {
		name   string
		policy MCPPolicy
		want   []string
	}{
		{name: "no policy", policy: MCPPolicy{}, want: servers},
		{name: "allowed alone restricts nothing", policy: MCPPolicy{Allowed: []string{"github"}}, want: servers},
		{name: "named block", policy: MCPPolicy{Blocked: []string{"github", "semble"}}, want: []string{"context7", "socket"}},
		{name: "wildcard blocks all", policy: MCPPolicy{Blocked: []string{MCPWildcard}}, want: nil},
		{
			name:   "wildcard keeps allowed",
			policy: MCPPolicy{Blocked: []string{MCPWildcard}, Allowed: []string{"socket", "context7"}},
			want:   []string{"context7", "socket"},
		},
		{
			name:   "named block beats allow",
			policy: MCPPolicy{Blocked: []string{"github"}, Allowed: []string{"github"}},
			want:   []string{"context7", "socket", "semble"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.policy.Filter(slices.Clone(servers)); !slices.Equal(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
			for _, s := range servers {
				if got, want := tt.policy.Permits(s), slices.Contains(tt.want, s); got != want {
					t.Errorf("Permits(%q) = %v, want %v", s, got, want)
				}
			}
		})
	}
}
