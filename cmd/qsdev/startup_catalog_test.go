package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

const (
	startupHelperEnv = "QSDEV_STARTUP_CATALOG_HELPER"
	treeHelperEnv    = "QSDEV_TREE_CATALOG_HELPER"
)

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

// TestCommandTreeCatalogHelper is not a real test. TestCommandTreeDoesNotLoadCatalog
// runs it in a fresh process, where TestMain has wired the runtime and
// initialized the addons exactly as Main does; it builds the root command and
// checks the catalog is still unloaded: setting a new project root succeeds
// only before the first load.
func TestCommandTreeCatalogHelper(t *testing.T) {
	if os.Getenv(treeHelperEnv) != "1" {
		t.Skip("helper process for TestCommandTreeDoesNotLoadCatalog")
	}
	instance.NewRootCommand()
	if err := catalog.SetProjectRoot(t.TempDir()); err != nil {
		t.Fatalf("building the command tree loaded the catalog: SetProjectRoot = %v", err)
	}
}

// TestCommandTreeDoesNotLoadCatalog guards the ordering the project defaults
// layer depends on: nothing may load the catalog before the command line is
// parsed and the command's project root is resolved, or the catalog would be
// frozen with some other root's project defaults.
func TestCommandTreeDoesNotLoadCatalog(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestCommandTreeCatalogHelper$", "-test.v")
	cmd.Env = append(os.Environ(), treeHelperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--- PASS: TestCommandTreeCatalogHelper") {
		t.Fatalf("helper did not run:\n%s", out)
	}
}
