package status

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

// TestStatusTier1ReturnsCachedWhenUnchanged verifies that the first auto call
// runs the thorough tier (there is no verified result to serve before it), and
// that a later auto call with nothing changed returns the cached Tier 1 result.
func TestStatusTier1ReturnsCachedWhenUnchanged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)

	checker := newStatusChecker(dir)
	first := call(t, checker.handle, map[string]any{}).Structured.(map[string]any)
	if first["tier"] != 2 {
		t.Errorf("first auto call tier = %v, want 2 (no cached result yet)", first["tier"])
	}

	m := call(t, checker.handle, map[string]any{}).Structured.(map[string]any)
	if m["tier"] != 1 {
		t.Errorf("tier = %v, want 1 (cached)", m["tier"])
	}
	if m["cached"] != true {
		t.Errorf("cached = %v, want true", m["cached"])
	}
}

// writeLedger writes a state ledger tracking rel (with its current content hash
// and mode, under the overwrite strategy) so drift detection can verify it.
func writeLedger(t *testing.T, dir, rel string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	hash, err := state.ComputeFileHash(full)
	if err != nil {
		t.Fatalf("hash %s: %v", rel, err)
	}
	info, err := os.Stat(full)
	if err != nil {
		t.Fatalf("stat %s: %v", rel, err)
	}
	st := types.GeneratedState{Files: map[string]types.FileState{
		rel: {Hash: hash, Strategy: types.Overwrite, Mode: info.Mode()},
	}}
	statePath := filepath.Join(dir, state.StateFilePaths()[0])
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := state.SaveStateToFile(statePath, st); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

// hasDrift reports whether a status payload lists a drift item of the type.
func hasDrift(m map[string]any, typ string) bool {
	items, _ := m["drift"].([]driftItem)
	for _, d := range items {
		if d.Type == typ {
			return true
		}
	}
	return false
}

// TestStatusReportsTamperedGeneratedFile is the regression test for status
// reporting zero drift after a machine-owned generated file was tampered with:
// the auto tier must notice the change and report file-modification drift from
// the state ledger, and keep reporting it on later calls (drift is not consumed
// by being reported once).
func TestStatusReportsTamperedGeneratedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rel := filepath.Join(".claude", "settings.json")
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(`{"permissions":{"deny":["Bash(curl:*)"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLedger(t, dir, rel)

	checker := newStatusChecker(dir)
	clean := call(t, checker.handle, map[string]any{}).Structured.(map[string]any)
	if hasDrift(clean, "file_modification") {
		t.Fatalf("untampered project reported file drift: %v", clean["drift"])
	}

	// Tamper: strip the deny rules, and move the mtime so the fast path notices.
	path := filepath.Join(dir, rel)
	if err := os.WriteFile(path, []byte(`{"permissions":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}

	for i, tier := range []string{"auto", "auto", "2"} {
		m := call(t, checker.handle, map[string]any{"tier": tier}).Structured.(map[string]any)
		if !hasDrift(m, "file_modification") {
			t.Errorf("call %d (tier %s): tampered generated file not reported; drift=%v", i, tier, m["drift"])
		}
	}
}

// TestStatusAutoFallsBackOnConfigChange is the regression test for auto mode
// serving the cached result after the project config changed: an edited config
// must force Tier 2 and be reported as config drift until regeneration.
func TestStatusAutoFallsBackOnConfigChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)
	cfg := filepath.Join(dir, ".qsdev.yaml")
	if err := os.WriteFile(cfg, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checker := newStatusChecker(dir)
	call(t, checker.handle, map[string]any{"tier": "2"})

	if err := os.WriteFile(cfg, []byte("version: 1\ntools:\n  disabled: [x]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		m := call(t, checker.handle, map[string]any{}).Structured.(map[string]any)
		if !hasDrift(m, "config_changed") {
			t.Errorf("call %d: config change not reported; tier=%v drift=%v", i, m["tier"], m["drift"])
		}
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

// TestSummarizeMCP verifies how the static MCP findings map onto the doctor's
// check: a broken .mcp.json fails, misconfigured or degraded servers and
// section warnings (such as an unloadable catalog, F279) warn, and every
// result points at `qsdev mcp status` for liveness.
func TestSummarizeMCP(t *testing.T) {
	t.Parallel()

	ok := doctor.MCPServerInfo{Name: "good", Status: doctor.MCPStatusOK}
	broken := doctor.MCPServerInfo{Name: "broken", Status: doctor.MCPStatusMisconfigured, Issues: []doctor.MCPIssue{
		{Severity: mcphealth.SeverityError, Message: "command \"missing-mcp\" not found on PATH"},
		{Severity: mcphealth.SeverityWarning, Message: "second issue"},
	}}
	noenv := doctor.MCPServerInfo{Name: "noenv", Status: doctor.MCPStatusDegraded, Issues: []doctor.MCPIssue{
		{Severity: mcphealth.SeverityWarning, Message: "required environment variable \"TOKEN\" is not set"},
	}}
	escape := doctor.MCPServerInfo{Name: "evil\u001b[2K", Status: doctor.MCPStatusMisconfigured, Issues: []doctor.MCPIssue{
		{Severity: mcphealth.SeverityError, Message: "bad"},
	}}

	tests := []struct {
		name       string
		ms         *doctor.MCPSection
		err        error
		wantStatus string
		want       []string
		notWant    []string
	}{
		{name: "unreadable .mcp.json", err: errors.New("parsing .mcp.json: bad"), wantStatus: checkFail,
			want: []string{"parsing .mcp.json: bad", "fix the JSON in .mcp.json"}},
		{name: "nothing configured", wantStatus: checkPass, want: []string{"no MCP servers configured"}},
		{name: "no servers", ms: &doctor.MCPSection{Detected: true}, wantStatus: checkPass, want: []string{"no MCP servers configured"}},
		{name: "all valid", ms: &doctor.MCPSection{Servers: []doctor.MCPServerInfo{ok, ok}}, wantStatus: checkPass,
			want: []string{"2 configured; 0 misconfigured, 0 degraded", "liveness: run `qsdev mcp status`"}},
		{name: "problems", ms: &doctor.MCPSection{Servers: []doctor.MCPServerInfo{ok, broken, noenv}}, wantStatus: checkWarn,
			want: []string{
				"3 configured; 1 misconfigured, 1 degraded",
				`broken (command "missing-mcp" not found on PATH)`,
				`noenv (required environment variable "TOKEN" is not set)`,
				"liveness: run `qsdev mcp status` (starts trusted definitions only)",
			},
			notWant: []string{"good", "second issue"}},
		{name: "catalog warning", ms: &doctor.MCPSection{Servers: []doctor.MCPServerInfo{ok}, Warnings: []string{"catalog: bad yaml"}},
			wantStatus: checkWarn, want: []string{"1 configured; 0 misconfigured, 0 degraded", "catalog: bad yaml"}},
		{name: "name escaped", ms: &doctor.MCPSection{Servers: []doctor.MCPServerInfo{escape}}, wantStatus: checkWarn,
			want: []string{`"evil\x1b[2K" (bad)`}, notWant: []string{"\u001b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := summarizeMCP(tt.ms, tt.err)
			if res.Name != "mcp" || res.Status != tt.wantStatus {
				t.Errorf("summarizeMCP() = %+v, want status %q", res, tt.wantStatus)
			}
			got := res.Detail + "\n" + res.Remediation
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("result %q does not contain %q", got, w)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("result %q contains %q", got, w)
				}
			}
		})
	}
}

// TestStatusTier2DoesNotHoldLockDuringDetect (R21) verifies the checker mutex is
// not held while detection runs. The injected detect tries TryLock: it succeeds
// only if the lock is free, proving the fast path is not blocked by detection.
func TestStatusTier2DoesNotHoldLockDuringDetect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)
	checker := newStatusChecker(dir)

	lockFree := make(chan bool, 1)
	checker.detectFn = func(ctx context.Context, root string) types.DetectedProject {
		ok := checker.mu.TryLock()
		if ok {
			checker.mu.Unlock()
		}
		lockFree <- ok
		return detect.Detect(ctx, root)
	}

	call(t, checker.handle, map[string]any{"tier": "2"})
	if got := <-lockFree; !got {
		t.Error("checker mutex was held during detection; concurrent fast path would block")
	}
}

// TestStatusTier2DetectRunsConcurrently (R21) verifies two concurrent Tier 2
// calls run their detection concurrently rather than serializing on the mutex.
// Both injected detects must enter before either is released; if detection were
// serialized under the lock the second call could never enter (caught by the
// rendezvous timeout below).
func TestStatusTier2DetectRunsConcurrently(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeState(t, dir)
	checker := newStatusChecker(dir)

	const n = 2
	entered := make(chan struct{}, n)
	release := make(chan struct{})
	checker.detectFn = func(ctx context.Context, root string) types.DetectedProject {
		entered <- struct{}{}
		<-release
		return detect.Detect(ctx, root)
	}

	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			// Call handle directly; t.Fatal must not run off the test goroutine.
			_, _ = checker.handle(context.Background(), &spi.ToolCallContext{},
				&spi.ToolRequest{Arguments: map[string]any{"tier": "2"}})
			done <- struct{}{}
		}()
	}

	for i := 0; i < n; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("detections did not run concurrently; mutex serialized detect")
		}
	}
	close(release)
	for i := 0; i < n; i++ {
		<-done
	}
}
