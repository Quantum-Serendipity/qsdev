package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// writeConfig writes a .qsdev.yaml fixture into dir and returns its path.
func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, ".qsdev.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

// structuredMap unwraps a result's structured payload as a map.
func structuredMap(t *testing.T, res *spi.ToolResult) map[string]any {
	t.Helper()
	if res == nil {
		t.Fatal("nil result")
	}
	m, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured is %T, want map[string]any", res.Structured)
	}
	return m
}

func call(t *testing.T, h spi.ToolHandler, args map[string]any) *spi.ToolResult {
	t.Helper()
	res, err := h(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Arguments: args})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	return res
}

const policyFixture = `version: 1
claude_code:
  permission_level: strict
tools:
  enabled:
    - semgrep
  disabled:
    - dangerous_tool
`

func TestPolicyCheckEvaluatesDenyRule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir)

	t.Run("denied tool", func(t *testing.T) {
		// A denied verdict is a normal evaluation, not a tool error: IsError stays
		// false (only graceful-degradation results set it).
		res := call(t, pc.handle, map[string]any{"tool_name": "dangerous_tool"})
		if res.IsError {
			t.Errorf("a normal deny verdict must not set IsError")
		}
		eval := structuredMap(t, res)["evaluation"].(policyDecision)
		if eval.Decision != decisionDenied {
			t.Errorf("decision = %q, want denied", eval.Decision)
		}
		if eval.Source != sourceProject || eval.Rule != "tools.disabled" {
			t.Errorf("source/rule = %q/%q, want project/tools.disabled", eval.Source, eval.Rule)
		}
	})

	t.Run("allowed tool", func(t *testing.T) {
		eval := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "semgrep"}))["evaluation"].(policyDecision)
		if eval.Decision != decisionAllowed || eval.Source != sourceProject {
			t.Errorf("got %+v, want allowed/project", eval)
		}
	})

	t.Run("unknown tool falls through to default", func(t *testing.T) {
		eval := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "mystery"}))["evaluation"].(policyDecision)
		// permission_level: strict escalates the default to "ask".
		if eval.Decision != decisionAsk || eval.Source != sourceDefault {
			t.Errorf("got %+v, want ask/default", eval)
		}
	})
}

func TestPolicyCheckNotConfigured(t *testing.T) {
	t.Parallel()
	pc := newPolicyChecker(t.TempDir()) // no .qsdev.yaml present
	res := call(t, pc.handle, map[string]any{"tool_name": "anything"})
	if !res.IsError {
		t.Fatal("expected IsError for missing policy")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}

func TestPolicyCheckFastPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir)

	// Warm the cache so the measured call hits the cached, no-parse path.
	call(t, pc.handle, map[string]any{"tool_name": "semgrep"})

	start := time.Now()
	call(t, pc.handle, map[string]any{"tool_name": "semgrep"})
	elapsed := time.Since(start)
	if elapsed > 5*time.Millisecond {
		t.Errorf("cached policy_check took %v, want <=5ms", elapsed)
	}
}

func TestSecurityScanNotConfiguredWithoutLockFile(t *testing.T) {
	t.Parallel()
	scanner := newSecurityScanner(t.TempDir()) // empty dir, no lock file
	res := call(t, scanner.handle, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when no lock file found")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}

func TestCredentialVendNotConfigured(t *testing.T) {
	t.Parallel()
	cv := newCredentialVendor()

	t.Run("missing provider", func(t *testing.T) {
		res := call(t, cv.handle, map[string]any{})
		if !res.IsError || structuredMap(t, res)["status"] != "not_configured" {
			t.Errorf("got IsError=%t status=%v, want not_configured", res.IsError, structuredMap(t, res)["status"])
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		res := call(t, cv.handle, map[string]any{"provider": "digitalocean"})
		if !res.IsError || structuredMap(t, res)["status"] != "not_configured" {
			t.Errorf("got IsError=%t status=%v, want not_configured", res.IsError, structuredMap(t, res)["status"])
		}
	})
}

// TestCredentialVendAWSNoCredentials proves the AWS path degrades to a structured
// not_configured result (rather than crashing) when no credentials can be
// resolved. IMDS is disabled and every credential source is pointed at a
// nonexistent file so the test is fully offline and deterministic.
func TestCredentialVendAWSNoCredentials(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", missing)
	t.Setenv("AWS_CONFIG_FILE", missing)

	cv := newCredentialVendor()
	res := call(t, cv.handle, map[string]any{"provider": "aws"})
	if !res.IsError {
		t.Fatal("expected IsError when AWS has no credentials")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}
