package installer_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/installer"
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
	tmp := t.TempDir()
	// Manager exists and succeeds, but the tool itself is not on PATH
	// before install. The fake manager puts the tool on PATH; chmod is
	// resolved first because PATH holds only tmp while it runs.
	chmod, err := exec.LookPath("chmod")
	if err != nil {
		t.Skipf("chmod not found: %v", err)
	}
	tool := filepath.Join(tmp, "fake-tool-binary")
	writeScript(t, tmp, "fake-mgr", fmt.Sprintf("printf '#!/bin/sh\\necho v9.9.9\\n' > %q\n%q +x %q\n", tool, chmod, tool))
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
	if !strings.Contains(out, "installed successfully") || !strings.Contains(out, "v9.9.9") {
		t.Errorf("expected success message with the detected version in output, got: %s", out)
	}
}

// TestInstall_ManagerSucceedsBinaryMissing covers F289: an install command
// that exits 0 but leaves the binary unresolvable on PATH (e.g. the profile
// bin directory is not on PATH) is an error, not a success.
func TestInstall_ManagerSucceedsBinaryMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
	tmp := t.TempDir()
	writeScript(t, tmp, "fake-mgr", `exit 0`)
	withPATH(t, tmp)

	spec := testSpec()
	var installErr error
	out := captureStdout(t, func() {
		installErr = installer.Install(context.Background(), spec)
	})

	if !errors.Is(installErr, installer.ErrNotFoundAfterInstall) {
		t.Fatalf("error = %v, want %v", installErr, installer.ErrNotFoundAfterInstall)
	}
	if strings.Contains(out, "installed successfully") {
		t.Errorf("reported success although the binary is missing: %s", out)
	}
}

func TestInstall_ManagerFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
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

// floorSpec is testSpec with a version floor and a parser for the fake
// tool's "fake-tool X.Y.Z" output.
func floorSpec() installer.ToolSpec {
	spec := testSpec()
	spec.MinVersion = "2.1"
	spec.ParseVersion = func(raw string) string {
		fields := strings.Fields(raw)
		if len(fields) < 2 {
			return ""
		}
		return fields[1]
	}
	return spec
}

// writeUpgradingManager writes a fake manager that records it ran (in
// marker) and replaces the fake tool with one printing newVersion.
func writeUpgradingManager(t *testing.T, dir, marker, newVersion string) {
	t.Helper()
	chmod, err := exec.LookPath("chmod")
	if err != nil {
		t.Skipf("chmod not found: %v", err)
	}
	tool := filepath.Join(dir, "fake-tool-binary")
	writeScript(t, dir, "fake-mgr", fmt.Sprintf(
		"echo ran > %q\nprintf '#!/bin/sh\\necho fake-tool %s\\n' > %q\n%q +x %q\n",
		marker, newVersion, tool, chmod, tool))
}

// TestInstall_UpgradesOutdated covers U13-09: a found binary below the
// floor is not "already installed"; the install command runs and the new
// version is verified. With no floor, a found binary is left alone.
func TestInstall_UpgradesOutdated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
	tests := []struct {
		name        string
		spec        installer.ToolSpec
		wantInstall bool
	}{
		{"below floor upgrades", floorSpec(), true},
		{"no floor keeps found binary", testSpec(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			marker := filepath.Join(t.TempDir(), "ran")
			writeScript(t, tmp, "fake-tool-binary", `echo "fake-tool 1.4.1"`)
			writeUpgradingManager(t, tmp, marker, "2.1.2")
			withPATH(t, tmp)

			var installErr error
			out := captureStdout(t, func() {
				installErr = installer.Install(context.Background(), tt.spec)
			})
			if installErr != nil {
				t.Fatalf("Install: %v\n%s", installErr, out)
			}
			_, statErr := os.Stat(marker)
			if ran := statErr == nil; ran != tt.wantInstall {
				t.Fatalf("install command ran = %v, want %v\n%s", ran, tt.wantInstall, out)
			}
			if tt.wantInstall && !strings.Contains(out, "below the minimum 2.1") {
				t.Errorf("output does not explain the upgrade:\n%s", out)
			}
			if !tt.wantInstall && !strings.Contains(out, "already installed") {
				t.Errorf("output = %q, want 'already installed'", out)
			}
		})
	}
}

// TestInstall_OutdatedAfterInstall covers U13-09: an install that exits 0
// but leaves a version below the floor first on PATH fails with
// ErrOutdatedAfterInstall instead of reporting success.
func TestInstall_OutdatedAfterInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
	tmp := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	writeScript(t, tmp, "fake-tool-binary", `echo "fake-tool 1.4.1"`)
	writeUpgradingManager(t, tmp, marker, "2.0.9")
	withPATH(t, tmp)

	var installErr error
	out := captureStdout(t, func() {
		installErr = installer.Install(context.Background(), floorSpec())
	})
	if !errors.Is(installErr, installer.ErrOutdatedAfterInstall) {
		t.Fatalf("error = %v, want %v", installErr, installer.ErrOutdatedAfterInstall)
	}
	if !strings.Contains(installErr.Error(), "2.0.9") {
		t.Errorf("error %q does not name the version found", installErr)
	}
	if strings.Contains(out, "installed successfully") {
		t.Errorf("reported success although the version is below the floor: %s", out)
	}
}

// TestSimulate_Outdated covers U13-09: a dry run reports that it would
// upgrade a binary below the floor.
func TestSimulate_Outdated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer tests use Unix shell scripts")
	}
	tmp := t.TempDir()
	writeScript(t, tmp, "fake-tool-binary", `echo "fake-tool 1.4.1"`)
	writeScript(t, tmp, "fake-mgr", `exit 0`)
	withPATH(t, tmp)

	out := captureStdout(t, func() {
		if err := installer.Simulate(context.Background(), floorSpec()); err != nil {
			t.Fatalf("Simulate: %v", err)
		}
	})
	if !strings.Contains(out, "Would upgrade fake-tool from 1.4.1 to >= 2.1") || !strings.Contains(out, "fake-mgr install fake-tool") {
		t.Errorf("output = %q, want the upgrade and its command", out)
	}
}
