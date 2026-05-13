package installer_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/installer"
)

// testSpec returns a ToolSpec that references fake binaries so tests
// never invoke real package managers.
func testSpec() installer.ToolSpec {
	return installer.ToolSpec{
		DisplayName:   "fake-tool",
		Binary:        "fake-tool-binary",
		VersionFlag:   "--version",
		InstallCmd:    []string{"fake-mgr", "install", "fake-tool"},
		ManagerBinary: "fake-mgr",
		ManagerName:   "FakeMgr",
		FallbackURL:   "https://example.com/install-mgr",
		DirectURL:     "https://example.com/install-tool",
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured output: %v", err)
	}
	return buf.String()
}

// writeScript creates an executable shell script in dir.
func writeScript(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("writing script %s: %v", name, err)
	}
}

// withPATH temporarily replaces PATH so that only dir is searched.
func withPATH(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("PATH")
	t.Setenv("PATH", dir)
	// t.Setenv restores on cleanup automatically
	_ = orig
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	tmp := t.TempDir()
	writeScript(t, tmp, "fake-tool-binary", `echo "v1.2.3"`)
	withPATH(t, tmp)

	spec := testSpec()
	out := captureStdout(t, func() {
		err := installer.Install(context.Background(), spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "already installed") {
		t.Errorf("expected 'already installed' in output, got: %s", out)
	}
	if !strings.Contains(out, "v1.2.3") {
		t.Errorf("expected version in output, got: %s", out)
	}
}

func TestInstall_ManagerNotAvailable(t *testing.T) {
	// PATH is empty -- neither the tool nor the manager exist.
	tmp := t.TempDir()
	withPATH(t, tmp)

	spec := testSpec()
	var installErr error
	out := captureStdout(t, func() {
		installErr = installer.Install(context.Background(), spec)
	})

	if installErr == nil {
		t.Fatal("expected error when neither tool nor manager is available")
	}
	if !strings.Contains(installErr.Error(), "manual installation required") {
		t.Errorf("unexpected error message: %v", installErr)
	}
	if !strings.Contains(out, "Install options") {
		t.Errorf("expected fallback instructions in output, got: %s", out)
	}
	if !strings.Contains(out, spec.FallbackURL) {
		t.Errorf("expected fallback URL in output, got: %s", out)
	}
	if !strings.Contains(out, spec.DirectURL) {
		t.Errorf("expected direct URL in output, got: %s", out)
	}
}

func TestInstall_ManagerSucceeds(t *testing.T) {
	tmp := t.TempDir()
	// Manager exists and succeeds, but the tool itself is not on PATH
	// before install. The fake manager just exits 0.
	writeScript(t, tmp, "fake-mgr", `exit 0`)
	withPATH(t, tmp)

	spec := testSpec()
	out := captureStdout(t, func() {
		err := installer.Install(context.Background(), spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Installing fake-tool") {
		t.Errorf("expected install message in output, got: %s", out)
	}
	if !strings.Contains(out, "installed successfully") {
		t.Errorf("expected success message in output, got: %s", out)
	}
}

func TestInstall_ManagerFails(t *testing.T) {
	tmp := t.TempDir()
	writeScript(t, tmp, "fake-mgr", `exit 1`)
	withPATH(t, tmp)

	spec := testSpec()
	var installErr error
	captureStdout(t, func() {
		installErr = installer.Install(context.Background(), spec)
	})

	if installErr == nil {
		t.Fatal("expected error when manager command fails")
	}
	if !strings.Contains(installErr.Error(), "installing fake-tool") {
		t.Errorf("unexpected error message: %v", installErr)
	}
}

func TestSimulate_AlreadyInstalled(t *testing.T) {
	tmp := t.TempDir()
	writeScript(t, tmp, "fake-tool-binary", `echo "v1.2.3"`)
	withPATH(t, tmp)

	spec := testSpec()
	out := captureStdout(t, func() {
		err := installer.Simulate(context.Background(), spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "already installed") {
		t.Errorf("expected 'already installed' in output, got: %s", out)
	}
}

func TestSimulate_ManagerAvailable(t *testing.T) {
	tmp := t.TempDir()
	// Manager exists but tool does not.
	writeScript(t, tmp, "fake-mgr", `exit 0`)
	withPATH(t, tmp)

	spec := testSpec()
	out := captureStdout(t, func() {
		err := installer.Simulate(context.Background(), spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Would run") {
		t.Errorf("expected 'Would run' in output, got: %s", out)
	}
	if !strings.Contains(out, "fake-mgr install fake-tool") {
		t.Errorf("expected install command in output, got: %s", out)
	}
}

func TestSimulate_NoManager(t *testing.T) {
	tmp := t.TempDir()
	withPATH(t, tmp)

	spec := testSpec()
	out := captureStdout(t, func() {
		err := installer.Simulate(context.Background(), spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Would need manual") {
		t.Errorf("expected manual install message in output, got: %s", out)
	}
	if !strings.Contains(out, "FakeMgr not available") {
		t.Errorf("expected manager name in output, got: %s", out)
	}
}
