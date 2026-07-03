package mcpserve

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// writeDisablingConfig writes a .qsdev.yaml into dir that disables toolName and
// returns dir. It is the minimal fixture proving a project-config deny reaches
// the enforcing Guardrail.
func writeDisablingConfig(t *testing.T, dir, toolName string) {
	t.Helper()
	body := "version: 1\ntools:\n  disabled:\n    - " + toolName + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// callThroughChain drives a single tool call through chain and reports whether
// the final handler ran and the result it produced. A Guardrail deny short-
// circuits before the final handler, so ran stays false and res.IsError is true.
func callThroughChain(t *testing.T, chain *spi.Chain, tool, category string) (ran bool, res *spi.ToolResult) {
	t.Helper()
	cc := &spi.ToolCallContext{AgentID: "test-agent", ToolName: tool, Category: category}
	final := func(context.Context, *spi.ToolCallContext, *spi.ToolRequest) (*spi.ToolResult, error) {
		ran = true
		return &spi.ToolResult{Text: "handler ran"}, nil
	}
	var err error
	res, err = chain.Execute(context.Background(), cc, &spi.ToolRequest{Name: tool}, final)
	if err != nil {
		t.Fatalf("chain.Execute returned Go error: %v", err)
	}
	return ran, res
}

// TestChainForModeEnforcesDisabledTool is the BL-P1-3 regression: a tool disabled
// in .qsdev.yaml (tools.disabled) must be BLOCKED by the enforcing Guardrail on an
// MCP call — not merely reported denied by qsdev_policy_check. It exercises the
// real server-construction seam (projectPolicy -> chainForMode) for BOTH native
// and gateway modes.
//
// The "no policy" arm proves the pre-fix behavior (chainForMode built the chain
// with no policy, so Guardrail ran permissive-by-default and ADMITTED the disabled
// tool) is what the fix closes: with the derived policy fed in, the same call is
// denied. That arm is RED before the wiring existed and GREEN after.
func TestChainForModeEnforcesDisabledTool(t *testing.T) {
	const disabledTool = "qsdev_security_scan"
	const allowedTool = "qsdev_status"

	dir := t.TempDir()
	writeDisablingConfig(t, dir, disabledTool)

	policy, err := projectPolicy(dir)
	if err != nil {
		t.Fatalf("projectPolicy returned error for a valid config: %v", err)
	}
	if policy == nil {
		t.Fatal("projectPolicy returned nil for a config with tools.disabled; the deny would never be enforced")
	}

	for _, mode := range []container.DeployMode{container.DeployNative, container.DeployGateway} {
		t.Run(string(mode), func(t *testing.T) {
			enforced := chainForMode(mode, policy)

			// The disabled tool is denied: Guardrail short-circuits (IsError, final
			// handler never runs).
			ran, res := callThroughChain(t, enforced, disabledTool, middleware.CategorySecurity)
			if ran {
				t.Errorf("disabled tool %q ran despite tools.disabled — Guardrail not enforcing", disabledTool)
			}
			if res == nil || !res.IsError {
				t.Fatalf("disabled tool result = %+v, want IsError (denied by policy)", res)
			}

			// A tool NOT in tools.disabled still runs — deny rules only subtract.
			ran, res = callThroughChain(t, enforced, allowedTool, middleware.CategoryStatus)
			if !ran || res == nil || res.IsError {
				t.Errorf("allowed tool %q was blocked; ran=%v res=%+v", allowedTool, ran, res)
			}

			// Pre-fix contrast: with NO policy fed (nil), the same disabled tool is
			// ADMITTED — the exact BL-P1-3 gap the derived policy closes.
			permissive := chainForMode(mode, nil)
			ranPermissive, _ := callThroughChain(t, permissive, disabledTool, middleware.CategorySecurity)
			if !ranPermissive {
				t.Errorf("sanity: without a policy the disabled tool should be admitted (permissive-by-default); "+
					"got blocked, so the test no longer demonstrates the %s gap", mode)
			}
		})
	}
}

// TestProjectPolicyFailsClosedOnUnparseableConfig is the M5 regression: a PRESENT
// but unparseable .qsdev.yaml (an unknown/typo'd key the strict decoder rejects)
// must fail closed — projectPolicy returns an error so the server refuses to
// start un-narrowed — rather than a nil permissive policy that silently drops
// every tools.disabled deny. An ABSENT config stays benign (nil policy, no error).
func TestProjectPolicyFailsClosedOnUnparseableConfig(t *testing.T) {
	// Absent config: benign — no error, permissive (nil) policy.
	empty := t.TempDir()
	if policy, err := projectPolicy(empty); err != nil || policy != nil {
		t.Fatalf("absent config: got (policy=%v, err=%v), want (nil, nil)", policy, err)
	}

	// Present but unparseable: an unknown top-level key the strict decoder rejects
	// alongside a real tools.disabled deny.
	bad := t.TempDir()
	body := "version: 1\ntools:\n  disabled:\n    - qsdev_security_scan\nbogus_unknown_key: true\n"
	if err := os.WriteFile(filepath.Join(bad, ".qsdev.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := projectPolicy(bad)
	if err == nil {
		t.Fatal("unparseable config: projectPolicy returned nil error — the server would run " +
			"permissively, silently dropping every tools.disabled deny (fail open)")
	}
	if policy != nil {
		t.Errorf("unparseable config: policy = %v, want nil (must not hand back a usable permissive policy)", policy)
	}
}
