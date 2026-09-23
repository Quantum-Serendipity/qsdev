package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const startupHelperEnv = "QSDEV_STARTUP_CATALOG_HELPER"

// TestStartupCatalogHelper is not a real test. TestStartupDoesNotLoadCatalog
// re-runs this test binary with a malformed org config; the binary gets this
// far only if initializing every package it links did not load the catalog.
func TestStartupCatalogHelper(t *testing.T) {
	if os.Getenv(startupHelperEnv) != "1" {
		t.Skip("helper process for TestStartupDoesNotLoadCatalog")
	}
}

// TestStartupDoesNotLoadCatalog guards against loading the catalog during
// package initialization. An init-time load panics on a malformed user org
// config before main runs, crashing every command — including the hooks and
// the `defaults` commands that exist to repair the file — and freezes the
// catalog before main applies branding.
func TestStartupDoesNotLoadCatalog(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(bad, []byte("tools: [\n"), 0o600); err != nil {
		t.Fatalf("writing malformed org config: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestStartupCatalogHelper$", "-test.v")
	cmd.Env = append(os.Environ(), startupHelperEnv+"=1", "QSDEV_ORG_CONFIG="+bad)
	out, err := cmd.CombinedOutput()
	if err != nil || strings.Contains(string(out), "panic:") {
		t.Fatalf("package initialization failed with a malformed org config: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--- PASS: TestStartupCatalogHelper") {
		t.Fatalf("helper did not run:\n%s", out)
	}
}
