package status

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
)

func call(t *testing.T, h spi.ToolHandler, args map[string]any) *spi.ToolResult {
	t.Helper()
	res, err := h(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Arguments: args})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
	return res
}

// writeState creates the primary state file under dir so the status checker has
// a tracked mtime to compare against.
func writeState(t *testing.T, dir string) {
	t.Helper()
	rel := state.StateFilePaths()[0]
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(full, []byte("files: {}\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

// TestStatusTier1ReturnsCachedWhenUnchanged verifies that, when the state file is
// unchanged since construction, an auto-tier call resolves to the cached Tier 1
// result without re-running detection.
func TestStatusTier1ReturnsCachedWhenUnchanged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)

	checker := newStatusChecker(dir)                 // captures the state mtime
	res := call(t, checker.handle, map[string]any{}) // tier auto

	m := res.Structured.(map[string]any)
	if m["tier"] != 1 {
		t.Errorf("tier = %v, want 1 (cached)", m["tier"])
	}
	if m["cached"] != true {
		t.Errorf("cached = %v, want true", m["cached"])
	}
}

// TestStatusTier2Forced verifies the thorough tier runs detection and returns an
// uncached result with a drift list.
func TestStatusTier2Forced(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)

	checker := newStatusChecker(dir)
	res := call(t, checker.handle, map[string]any{"tier": "2"})

	m := res.Structured.(map[string]any)
	if m["tier"] != 2 {
		t.Errorf("tier = %v, want 2", m["tier"])
	}
	if m["cached"] != false {
		t.Errorf("cached = %v, want false", m["cached"])
	}
	if _, ok := m["drift"]; !ok {
		t.Error("expected a drift field in tier 2 status")
	}
}

// TestDevenvDoctorRunsAllChecks verifies the doctor runs all seven checks in
// parallel and returns structured per-check results plus an aggregate.
func TestDevenvDoctorRunsAllChecks(t *testing.T) {
	t.Parallel()
	doc := newDoctorChecker(t.TempDir())
	res := call(t, doc.handle, map[string]any{})

	m := res.Structured.(map[string]any)
	checks, ok := m["checks"].([]checkResult)
	if !ok {
		t.Fatalf("checks is %T, want []checkResult", m["checks"])
	}
	if len(checks) != 7 {
		t.Fatalf("ran %d checks, want 7", len(checks))
	}
	wantNames := map[string]bool{
		"config": false, "state": false, "tools": false, "nix": false,
		"mcp": false, "hooks": false, "permissions": false,
	}
	for _, c := range checks {
		if _, known := wantNames[c.Name]; !known {
			t.Errorf("unexpected check %q", c.Name)
		}
		wantNames[c.Name] = true
		switch c.Status {
		case checkPass, checkWarn, checkFail:
		default:
			t.Errorf("check %q has invalid status %q", c.Name, c.Status)
		}
	}
	for name, seen := range wantNames {
		if !seen {
			t.Errorf("missing check %q", name)
		}
	}
	if _, ok := m["overall"]; !ok {
		t.Error("expected an overall verdict")
	}
}

// TestDevenvDoctorSingleCheck verifies the check filter runs exactly one check.
func TestDevenvDoctorSingleCheck(t *testing.T) {
	t.Parallel()
	doc := newDoctorChecker(t.TempDir())
	res := call(t, doc.handle, map[string]any{"check": "config"})
	checks := res.Structured.(map[string]any)["checks"].([]checkResult)
	if len(checks) != 1 || checks[0].Name != "config" {
		t.Errorf("got %+v, want a single config check", checks)
	}
}
