package status

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
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

// TestCheckMCPProbesConcurrently (R20) verifies the MCP probes run concurrently
// rather than serially. Each fake probe blocks until every probe has started; if
// checkMCP probed sequentially the first probe would wait forever for the others
// to start, deadlocking and tripping the test timeout. No timing assertion is
// made, so the test is deterministic.
func TestCheckMCPProbesConcurrently(t *testing.T) {
	t.Parallel()
	const n = 5 // below mcpProbeConcurrency so all probes may start at once
	servers := make([]mcphealth.ServerConfig, n)
	for i := range servers {
		servers[i] = mcphealth.ServerConfig{Name: fmt.Sprintf("srv-%02d", i)}
	}

	var entered sync.WaitGroup
	entered.Add(n)
	proceed := make(chan struct{})

	doc := newDoctorChecker(t.TempDir())
	doc.mcpServers = func() []mcphealth.ServerConfig { return servers }
	doc.probeMCP = func(ctx context.Context, _ mcphealth.ServerConfig) *mcphealth.ServerHealth {
		entered.Done()
		select {
		case <-proceed:
		case <-ctx.Done():
		}
		return &mcphealth.ServerHealth{Status: mcphealth.StatusHealthy}
	}

	go func() {
		entered.Wait() // unblocks only once every probe is running
		close(proceed)
	}()

	res := doc.checkMCP(context.Background())
	if res.Status != checkPass {
		t.Errorf("status = %q, want %q (all healthy)", res.Status, checkPass)
	}
	if !strings.Contains(res.Detail, fmt.Sprintf("%d healthy, 0 unhealthy of %d", n, n)) {
		t.Errorf("detail = %q, want %d healthy/0 unhealthy", res.Detail, n)
	}
}

// TestCheckMCPCountsHealth (R20) verifies the concurrent probe aggregation
// counts healthy vs. unhealthy correctly and warns when any server is unhealthy,
// regardless of the order probes complete.
func TestCheckMCPCountsHealth(t *testing.T) {
	t.Parallel()
	servers := []mcphealth.ServerConfig{
		{Name: "charlie"}, {Name: "alpha"}, {Name: "delta"}, {Name: "bravo"},
	}
	unhealthy := map[string]bool{"bravo": true, "delta": true}

	doc := newDoctorChecker(t.TempDir())
	doc.mcpServers = func() []mcphealth.ServerConfig { return servers }
	doc.probeMCP = func(_ context.Context, cfg mcphealth.ServerConfig) *mcphealth.ServerHealth {
		st := mcphealth.StatusHealthy
		if unhealthy[cfg.Name] {
			st = mcphealth.StatusUnreachable
		}
		return &mcphealth.ServerHealth{Status: st}
	}

	res := doc.checkMCP(context.Background())
	if res.Status != checkWarn {
		t.Errorf("status = %q, want %q", res.Status, checkWarn)
	}
	if !strings.Contains(res.Detail, "2 healthy, 2 unhealthy of 4") {
		t.Errorf("detail = %q, want 2 healthy/2 unhealthy/4", res.Detail)
	}
}

// TestCheckMCPNoServers verifies the empty-registry fast path stays a pass.
func TestCheckMCPNoServers(t *testing.T) {
	t.Parallel()
	doc := newDoctorChecker(t.TempDir())
	doc.mcpServers = func() []mcphealth.ServerConfig { return nil }
	res := doc.checkMCP(context.Background())
	if res.Status != checkPass {
		t.Errorf("status = %q, want %q for no servers", res.Status, checkPass)
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
	checker.detectFn = func(root string) types.DetectedProject {
		ok := checker.mu.TryLock()
		if ok {
			checker.mu.Unlock()
		}
		lockFree <- ok
		return detect.Detect(root)
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
	checker.detectFn = func(root string) types.DetectedProject {
		entered <- struct{}{}
		<-release
		return detect.Detect(root)
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
