package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setLLRestrictPath overrides the ldflags-injected helper path for one test.
// It is package state, so callers must not run in parallel.
func setLLRestrictPath(t *testing.T, p string) {
	t.Helper()
	prev := llRestrictPath
	llRestrictPath = p
	t.Cleanup(func() { llRestrictPath = prev })
}

// fakeLLRestrictOnPath puts an `ll-restrict` on PATH that claims Landlock ABI
// 4, the shape of a helper an attacker or a stray install could provide.
func fakeLLRestrictOnPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake helper is a shell script")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "ll-restrict")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\necho landlock-abi:4\n"), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return helper
}

// TestLLRestrictBin_NoPathFallback verifies the helper is only ever the
// build-time Nix store path: a helper found on PATH, or an injected path that
// is missing or outside the store, is not used (U19-02).
func TestLLRestrictBin_NoPathFallback(t *testing.T) {
	helper := fakeLLRestrictOnPath(t)

	tests := []struct {
		name     string
		injected string
	}{
		{name: "no injected path ignores PATH", injected: ""},
		{name: "missing store path", injected: NixStoreDir + "/00000000000000000000000000000000-missing/bin/ll-restrict"},
		{name: "existing path outside the store", injected: helper},
		{name: "store prefix escaped with dot-dot", injected: NixStoreDir + "/../.." + helper},
		{name: "relative path", injected: "ll-restrict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setLLRestrictPath(t, tt.injected)
			if got := LLRestrictBin(); got != "" {
				t.Errorf("LLRestrictBin() = %q, want \"\"", got)
			}
		})
	}
}

// TestLLRestrictBin_ExistingStorePath verifies an injected path that exists
// in the Nix store is returned unchanged.
func TestLLRestrictBin_ExistingStorePath(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not found: %v", err)
	}
	real, err := filepath.EvalSymlinks(sh)
	if err != nil || !strings.HasPrefix(real, NixStoreDir+"/") {
		t.Skip("no Nix store file on this host")
	}
	setLLRestrictPath(t, real)
	if got := LLRestrictBin(); got != real {
		t.Errorf("LLRestrictBin() = %q, want %q", got, real)
	}
}

// TestDetermineTier_FakeHelperOnPathIsBwrapOnly verifies the real prober does
// not report Landlock from a helper that is merely on PATH, so the tier never
// claims a layer the backend would then fail to apply.
func TestDetermineTier_FakeHelperOnPathIsBwrapOnly(t *testing.T) {
	fakeLLRestrictOnPath(t)
	setLLRestrictPath(t, "")

	caps := ProbeCapabilities(context.Background(), &ExecSandboxProber{})
	if caps.LandlockABI != 0 {
		t.Errorf("LandlockABI = %d, want 0", caps.LandlockABI)
	}
	if tier := DetermineTier(caps); TierClaimsLandlock(tier) {
		t.Errorf("tier %v claims Landlock from a helper on PATH", tier)
	}
}
