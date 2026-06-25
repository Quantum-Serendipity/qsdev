package middleware

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestPolicyDecideCascade exercises the by_agent > by_user > by_tool_type >
// default resolution order directly on Policy.Decide.
func TestPolicyDecideCascade(t *testing.T) {
	t.Parallel()

	policy := &Policy{
		ByAgent: map[string]Rule{
			"bot-a": {DenyCategories: []string{CategoryCredential}},
			"bot-b": {AllowCategories: []string{CategoryNetwork}},
		},
		ByUser: map[string]Rule{
			"alice": {DenyTools: []string{"nix_run"}},
		},
		ByToolType: map[string]Verdict{
			CategoryNetwork: VerdictDeny,
		},
		Default: VerdictAllow,
	}

	cases := []struct {
		name           string
		agent, user    string
		category, tool string
		want           Verdict
	}{
		{"by_agent deny wins over default", "bot-a", "", CategoryCredential, "credential_vend", VerdictDeny},
		{"by_agent allow overrides tool_type deny", "bot-b", "", CategoryNetwork, "fetch", VerdictAllow},
		{"by_user deny matches tool name", "bot-c", "alice", CategoryProcess, "nix_run", VerdictDeny},
		{"by_tool_type deny when no agent/user opinion", "bot-c", "bob", CategoryNetwork, "fetch", VerdictDeny},
		{"default allow when nothing matches", "bot-c", "bob", CategorySearch, "grep", VerdictAllow},
		{"agent rule present but irrelevant falls through", "bot-a", "", CategorySearch, "grep", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := policy.Decide(tc.agent, tc.user, tc.category, tc.tool); got != tc.want {
				t.Errorf("Decide(%q,%q,%q,%q) = %v, want %v",
					tc.agent, tc.user, tc.category, tc.tool, got, tc.want)
			}
		})
	}
}

// TestPolicyNilAllows confirms a nil policy is permissive (defensive).
func TestPolicyNilAllows(t *testing.T) {
	t.Parallel()
	var p *Policy
	if got := p.Decide("a", "", CategoryCredential, "x"); got != VerdictAllow {
		t.Errorf("nil policy Decide = %v, want allow", got)
	}
	if got := DefaultPolicy().Decide("a", "", CategoryCredential, "x"); got != VerdictAllow {
		t.Errorf("DefaultPolicy Decide = %v, want allow (permissive-by-default)", got)
	}
}

// TestGuardrailShortCircuits proves a denied call returns an IsError result
// WITHOUT invoking the inner chain, and an allowed call proceeds.
func TestGuardrailShortCircuits(t *testing.T) {
	t.Parallel()

	policy := &Policy{
		ByToolType: map[string]Verdict{CategoryCredential: VerdictDeny},
		Default:    VerdictAllow,
	}
	g := Guardrail{policy: policy}

	t.Run("deny short-circuits, inner not run", func(t *testing.T) {
		t.Parallel()
		var ran bool
		cc := &spi.ToolCallContext{AgentID: "bot", ToolName: "credential_vend", Category: CategoryCredential}
		res, err := g.Handle(context.Background(), cc, &spi.ToolRequest{Name: "credential_vend"}, okHandler(&ran, "secret"))
		if err != nil {
			t.Fatalf("unexpected Go error: %v", err)
		}
		if ran {
			t.Error("inner handler ran despite deny")
		}
		if res == nil || !res.IsError {
			t.Fatalf("deny result = %+v, want IsError", res)
		}
	})

	t.Run("allow proceeds to inner", func(t *testing.T) {
		t.Parallel()
		var ran bool
		cc := &spi.ToolCallContext{AgentID: "bot", ToolName: "grep", Category: CategorySearch}
		res, err := g.Handle(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, okHandler(&ran, "hit"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ran {
			t.Error("inner handler did not run on allow")
		}
		if res == nil || res.IsError || res.Text != "hit" {
			t.Fatalf("allow result = %+v, want Text=hit", res)
		}
	})
}
