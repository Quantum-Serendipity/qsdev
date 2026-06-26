package devenv

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
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

// TestEnvInfoReturnsPathAndFiltersSecrets proves env_info reports PATH entries
// and never emits the value of a sensitive environment variable.
func TestEnvInfoReturnsPathAndFiltersSecrets(t *testing.T) {
	// Build a sentinel value at runtime (not a contiguous secret-shaped literal)
	// whose presence in the output would prove a leak.
	secretValue := "sentinel-" + "leak-" + "marker-42"
	t.Setenv("MY_SECRET_TOKEN", secretValue)

	env := newEnvInfo(t.TempDir())
	res := call(t, env.handle, map[string]any{"probe": "all"})

	// The whole serialized result must not contain the sensitive value.
	blob, err := json.Marshal(res.Structured)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(blob), secretValue) {
		t.Fatal("env_info leaked the value of a sensitive variable")
	}

	structured := res.Structured.(map[string]any)

	// PATH probe returned entries.
	pathProbe := structured["path"].(map[string]any)
	if pathProbe["count"].(int) == 0 {
		t.Error("expected at least one PATH entry")
	}

	// The sensitive var name is recorded in filtered_names (name only, no value).
	envProbe := structured["env"].(map[string]any)
	filtered, _ := envProbe["filtered_names"].([]string)
	found := false
	for _, n := range filtered {
		if n == "MY_SECRET_TOKEN" {
			found = true
		}
	}
	if !found {
		t.Errorf("MY_SECRET_TOKEN not recorded as filtered; got %v", filtered)
	}
}

func TestEnvInfoUnknownProbe(t *testing.T) {
	t.Parallel()
	env := newEnvInfo(t.TempDir())
	res := call(t, env.handle, map[string]any{"probe": "bogus"})
	if !res.IsError {
		t.Fatal("expected IsError for unknown probe")
	}
}

func TestNixRunMissingCommand(t *testing.T) {
	t.Parallel()
	nix := newNixRunner()
	res := call(t, nix.handle, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when command is missing")
	}
	if res.Structured.(map[string]any)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", res.Structured.(map[string]any)["status"])
	}
}

// TestRunProcessGroupCapturesOutput exercises the execution engine that backs
// nix_run: it captures stdout and stderr separately and reports the exit code.
// It uses a POSIX shell rather than nix so it is deterministic and offline.
func TestRunProcessGroupCapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell semantics; not run on windows")
	}
	res := runProcessGroup(context.Background(), "sh",
		[]string{"-c", "printf out; printf err 1>&2; exit 3"}, "", 5*time.Second)
	if res.startErr != nil {
		t.Fatalf("start error: %v", res.startErr)
	}
	if res.stdout != "out" {
		t.Errorf("stdout = %q, want %q", res.stdout, "out")
	}
	if res.stderr != "err" {
		t.Errorf("stderr = %q, want %q", res.stderr, "err")
	}
	if res.exitCode != 3 {
		t.Errorf("exit code = %d, want 3", res.exitCode)
	}
	if res.timedOut {
		t.Error("did not expect timeout")
	}
}

// TestRunProcessGroupTimeoutKillsGroup proves a timeout terminates the whole
// process group promptly: the launched shell backgrounds a 5s sleep, so if the
// group were not killed the call would block ~5s. A prompt return (well under
// the child's lifetime) with timed_out set proves the group was signaled.
func TestRunProcessGroupTimeoutKillsGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group kill semantics are unix-specific")
	}
	start := time.Now()
	res := runProcessGroup(context.Background(), "sh",
		[]string{"-c", "sleep 5 & wait"}, "", 200*time.Millisecond)
	elapsed := time.Since(start)

	if !res.timedOut {
		t.Error("expected timed_out=true")
	}
	if elapsed > 3*time.Second {
		t.Errorf("call took %v; process group was not killed promptly", elapsed)
	}
}

// TestInstallableRejection proves the nix_run installable policy: remote flake
// references (URLs and schemed flakerefs) are rejected, while local references
// and scheme-less registry aliases are allowed.
func TestInstallableRejection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ref      string
		rejected bool
	}{
		{"github remote", "github:owner/repo#pkg", true},
		{"gitlab remote", "gitlab:owner/repo", true},
		{"sourcehut remote", "sourcehut:~user/repo", true},
		{"git plus ssh url", "git+ssh://host/repo", true},
		{"http url", "http://example.com/x.tar.gz", true},
		{"https url", "https://example.com/x.tar.gz", true},
		{"tarball url", "tarball+https://example.com/x.tar.gz", true},
		{"file plus http url", "file+http://example.com/x", true},
		{"flake plus scheme", "flake+github:owner/repo", true},
		{"bare attr", "hello", false},
		{"project flake attr", ".#hello", false},
		{"project flake root", ".", false},
		{"local relative path", "./flake#pkg", false},
		{"local absolute path", "/srv/flake#pkg", false},
		{"local path scheme", "path:./flake#pkg", false},
		{"registry alias", "nixpkgs#hello", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, rejected := installableRejection(tt.ref)
			if rejected != tt.rejected {
				t.Errorf("installableRejection(%q) rejected=%v, want %v", tt.ref, rejected, tt.rejected)
			}
		})
	}
}

// TestNixRunRejectsRemoteInstallable proves the handler turns a remote ref into
// a tool-level error (IsError) that names the rejected ref, without needing nix
// to be installed and without ever executing it.
func TestNixRunRejectsRemoteInstallable(t *testing.T) {
	t.Parallel()
	const ref = "github:owner/repo#pkg"
	nix := newNixRunner()
	res := call(t, nix.handle, map[string]any{"command": ref})
	if !res.IsError {
		t.Fatal("expected IsError for a remote installable")
	}
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured is %T", res.Structured)
	}
	if structured["status"] != "error" {
		t.Errorf("status = %v, want error", structured["status"])
	}
	if structured["command"] != ref {
		t.Errorf("rejection did not name the ref; got %v", structured["command"])
	}
}

// TestRunProcessGroupTimeoutVsCancellation proves the unix process-group path
// distinguishes a deadline timeout (timed_out=true) from a caller cancellation
// (timed_out=false), matching the non-unix path.
func TestRunProcessGroupTimeoutVsCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group semantics are unix-specific")
	}
	t.Run("cancellation reports timed_out=false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel before running
		res := runProcessGroup(ctx, "sh", []string{"-c", "sleep 5 & wait"}, "", 5*time.Second)
		if res.timedOut {
			t.Error("cancellation must not be reported as timed_out")
		}
	})
	t.Run("deadline reports timed_out=true", func(t *testing.T) {
		res := runProcessGroup(context.Background(), "sh", []string{"-c", "sleep 5 & wait"}, "", 150*time.Millisecond)
		if !res.timedOut {
			t.Error("deadline must be reported as timed_out")
		}
	})
}

// TestNixRunExecutes runs the full handler against a real nix when present. It is
// skipped when nix is unavailable. A live `nix run` may require network/flake
// evaluation, so the test asserts only that the handler executes nix and returns
// a structured result with an exit code (it does not require a successful run).
func TestNixRunExecutes(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not installed; skipping live nix_run execution test")
	}
	nix := newNixRunner()
	res := call(t, nix.handle, map[string]any{
		"command": "nixpkgs#hello",
		"args":    []any{"--version"},
		"timeout": "20s",
	})
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured is %T", res.Structured)
	}
	if _, ok := structured["exit_code"]; !ok {
		t.Errorf("expected an exit_code in the result; got %v", structured)
	}
}
