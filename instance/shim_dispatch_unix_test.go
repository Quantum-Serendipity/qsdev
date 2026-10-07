//go:build !windows

package instance

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestMain_DispatchesShim: Main hands a shim argv to the shim before any
// runtime, addon or config work, so the hook's exit code and the ready byte
// come straight back. Without the dispatch the argv would reach the command
// tree instead.
func TestMain_DispatchesShim(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	argv := append(shim.Argv(exe, 3), "sh", "-c", "exit 7")
	cmd := exec.Command(exe, argv[1:]...) //nolint:gosec // the test binary itself
	home := t.TempDir()
	cmd.Env = append(os.Environ(),
		branding.Get().EnvNoUpdate+"=1",
		"HOME="+home, "XDG_STATE_HOME="+home, "XDG_CONFIG_HOME="+home,
		"XDG_CACHE_HOME="+home, "XDG_DATA_HOME="+home)
	cmd.ExtraFiles = []*os.File{w}
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("shim run = %v, want exit 7", err)
	}
	_ = w.Close()
	ready, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 {
		t.Errorf("ready fd received %q, want exactly one byte", ready)
	}
}
