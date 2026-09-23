package devinit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/backendselect"
)

// noSandboxProbe reports a host with no sandbox tooling, forcing the
// unsandboxed backend so exec tests do not depend on the host's kernel.
func noSandboxProbe(context.Context) *sandbox.SystemCapabilities {
	return &sandbox.SystemCapabilities{}
}

// runSandboxExec runs `sandbox exec` with the given stdin and arguments and
// returns stdout, stderr and the command error.
func runSandboxExec(t *testing.T, probe capabilityProbe, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newSandboxExecCmd(probe)
	var stdout, stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func requireExitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var ec *exitcode.Error
	if !errors.As(err, &ec) {
		t.Fatalf("expected an exitcode.Error, got %T: %v", err, err)
	}
	return ec.Code
}

// TestSandboxExec_ForwardsStdinAndProjectDir is the regression test for the
// wrapped-hook fail-open: Claude Code delivers the tool call on stdin, and the
// hook must run in (and see) the project directory.
func TestSandboxExec_ForwardsStdinAndProjectDir(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "marker"), []byte("in-project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PROJECT_DIR", project)

	payload := `{"tool_name":"Bash","tool_input":{"command":"ls"}}`
	stdout, _, err := runSandboxExec(t, noSandboxProbe, payload, "--", "sh", "-c", "cat; echo; cat marker")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if want := payload + "\nin-project\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestSandboxExec_ExitCodes(t *testing.T) {
	tests := []struct {
		name       string
		projectDir func(t *testing.T) string
		args       []string
		wantCode   int
	}{
		{
			name:       "hook exit status is propagated",
			projectDir: func(t *testing.T) string { t.Helper(); return t.TempDir() },
			args:       []string{"--", "sh", "-c", "exit 3"},
			wantCode:   3,
		},
		{
			name:       "relative CLAUDE_PROJECT_DIR fails closed",
			projectDir: func(*testing.T) string { return "relative/dir" },
			args:       []string{"--", "sh", "-c", "exit 0"},
			wantCode:   hookBlockExitCode,
		},
		{
			name: "uncompilable policy fails closed instead of using defaults",
			projectDir: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				policyDir := filepath.Join(dir, ".qsdev")
				if err := os.MkdirAll(policyDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(policyDir, "policy.nix"), []byte("{ this is not nix"), 0o644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			args:     []string{"--", "sh", "-c", "exit 0"},
			wantCode: hookBlockExitCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLAUDE_PROJECT_DIR", tt.projectDir(t))
			_, _, err := runSandboxExec(t, noSandboxProbe, "", tt.args...)
			if got := requireExitCode(t, err); got != tt.wantCode {
				t.Errorf("exit code = %d, want %d (err: %v)", got, tt.wantCode, err)
			}
		})
	}
}

// TestSandboxExec_ReportsDegradationOnStderr verifies an unsandboxed run is
// announced on stderr rather than only in the (hidden by default) log file.
func TestSandboxExec_ReportsDegradationOnStderr(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", t.TempDir())
	_, stderr, err := runSandboxExec(t, noSandboxProbe, "", "--", "sh", "-c", "exit 0")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if !strings.Contains(stderr, "degraded isolation") || !strings.Contains(stderr, "unsandboxed") {
		t.Errorf("stderr does not report the unsandboxed tier: %q", stderr)
	}
}

// TestSandboxExec_BubblewrapEndToEnd runs a real bwrap sandbox when the host
// provides one and its coreutils live in the Nix store (the only host tree the
// sandbox mounts besides the project).
func TestSandboxExec_BubblewrapEndToEnd(t *testing.T) {
	caps := sandbox.ProbeCapabilitiesDefault(context.Background())
	if backend, _ := backendselect.ResolveBackend(*caps); backend.Name() != "bubblewrap" {
		t.Skip("bubblewrap backend not available on this host")
	}
	cat, err := hostExecutable("cat")
	if err != nil {
		t.Skipf("cat not found: %v", err)
	}
	if real, err := filepath.EvalSymlinks(cat); err != nil || !pathWithin(real, sandboxStoreDir) {
		t.Skip("host coreutils are not in the Nix store")
	}

	project := t.TempDir()
	marker := filepath.Join(project, "marker")
	if err := os.WriteFile(marker, []byte("in-project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PROJECT_DIR", project)
	realProbe := func(context.Context) *sandbox.SystemCapabilities { return caps }

	payload := `{"tool_name":"Bash"}`
	stdout, stderr, err := runSandboxExec(t, realProbe, payload, "--", "cat", "-", marker)
	if err != nil {
		t.Fatalf("exec failed: %v\nstderr: %s", err, stderr)
	}
	if want := payload + "in-project\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// storeLayout builds a fake host: store/ stands in for /nix/store (mounted into
// the sandbox) and host/bin/ for a PATH directory that is not mounted.
type storeLayout struct {
	project, store, hostBin string
}

func newStoreLayout(t *testing.T) storeLayout {
	t.Helper()
	root := t.TempDir()
	l := storeLayout{
		project: filepath.Join(root, "project"),
		store:   filepath.Join(root, "store"),
		hostBin: filepath.Join(root, "host", "bin"),
	}
	for _, d := range []string{l.project, filepath.Join(l.store, "bin"), l.hostBin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExec := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil { //nolint:gosec // test executables
			t.Fatal(err)
		}
	}
	link := func(target, path string) {
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(filepath.Join(l.store, "bin", "multi"), "\x7fELF")
	link("multi", filepath.Join(l.store, "bin", "cat"))
	link(filepath.Join(l.store, "bin", "cat"), filepath.Join(l.hostBin, "cat"))
	writeExec(filepath.Join(l.store, "bin", "interp"), "\x7fELF")
	link(filepath.Join(l.store, "bin", "interp"), filepath.Join(l.hostBin, "interp"))
	writeExec(filepath.Join(l.hostBin, "hostonly"), "\x7fELF")
	writeExec(filepath.Join(l.project, "tool"), "\x7fELF")
	writeExec(filepath.Join(l.project, "env-hook.py"), "#!/usr/bin/env interp\nprint()\n")
	writeExec(filepath.Join(l.project, "arg-hook.sh"), "#!"+filepath.Join(l.hostBin, "interp")+" -e\n")
	writeExec(filepath.Join(l.project, "store-hook.sh"), "#!"+filepath.Join(l.store, "bin", "interp")+"\n")
	writeExec(filepath.Join(l.project, "hostonly-hook.sh"), "#!"+filepath.Join(l.hostBin, "hostonly")+"\n")
	link(filepath.Join(l.hostBin, "interp"), filepath.Join(l.project, "venv-interp"))
	link(filepath.Join(l.hostBin, "hostonly"), filepath.Join(l.project, "escape"))
	return l
}

func TestNamespaceHookCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bubblewrap is Linux-only; the fixture relies on symlinks and extensionless executables")
	}
	l := newStoreLayout(t)
	t.Setenv("PATH", l.hostBin)
	storeBin := filepath.Join(l.store, "bin")

	tests := []struct {
		name    string
		command []string
		want    []string
		wantErr bool
	}{
		{
			name:    "binary inside the project is unchanged",
			command: []string{filepath.Join(l.project, "tool"), "a"},
			want:    []string{filepath.Join(l.project, "tool"), "a"},
		},
		{
			name:    "PATH lookup follows symlinks only until visible, keeping the multi-call name",
			command: []string{"cat", "-"},
			want:    []string{filepath.Join(storeBin, "cat"), "-"},
		},
		{
			name:    "env shebang is resolved on the host",
			command: []string{filepath.Join(l.project, "env-hook.py"), "x"},
			want:    []string{filepath.Join(storeBin, "interp"), filepath.Join(l.project, "env-hook.py"), "x"},
		},
		{
			name:    "hidden interpreter with an argument is made explicit",
			command: []string{filepath.Join(l.project, "arg-hook.sh")},
			want:    []string{filepath.Join(storeBin, "interp"), "-e", filepath.Join(l.project, "arg-hook.sh")},
		},
		{
			name:    "visible interpreter is left to the kernel",
			command: []string{filepath.Join(l.project, "store-hook.sh")},
			want:    []string{filepath.Join(l.project, "store-hook.sh")},
		},
		{
			name:    "project symlink linking out of the sandbox is followed to a visible target",
			command: []string{filepath.Join(l.project, "venv-interp")},
			want:    []string{filepath.Join(storeBin, "interp")},
		},
		{
			name:    "project symlink to a host-only executable is rejected",
			command: []string{filepath.Join(l.project, "escape")},
			wantErr: true,
		},
		{
			name:    "executable outside the sandbox is rejected",
			command: []string{"hostonly"},
			wantErr: true,
		},
		{
			name:    "interpreter outside the sandbox is rejected",
			command: []string{filepath.Join(l.project, "hostonly-hook.sh")},
			wantErr: true,
		},
		{
			name:    "missing command is rejected",
			command: []string{"does-not-exist"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &sandbox.SandboxConfig{
				ProjectDir:  l.project,
				HookCommand: tt.command,
				Mounts:      []sandbox.MountSpec{{Source: l.store, Target: l.store, ReadOnly: true}},
			}
			got, err := namespaceHookCommand(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSandboxVisibility_HidesPolicyDenyEntries verifies a path under a
// policy deny entry (masked inside the sandbox) is not treated as visible,
// even when it lies inside the project directory.
func TestSandboxVisibility_HidesPolicyDenyEntries(t *testing.T) {
	t.Parallel()
	cfg := &sandbox.SandboxConfig{ProjectDir: "/proj", Deny: []string{"/proj/secrets"}}
	visible := sandboxVisibility(cfg)
	tests := map[string]bool{
		"/proj/hooks/x.sh":     true,
		"/proj/secrets":        false,
		"/proj/secrets/key.sh": false,
		"/elsewhere/bin/tool":  false,
	}
	for p, want := range tests {
		if got := visible(p); got != want {
			t.Errorf("visible(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestDefaultHookName(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"/p/.claude/hooks/package-guard.py": "package-guard",
		"/p/.claude/hooks/audit-log.sh":     "audit-log",
		"qsdev":                             "qsdev",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got := defaultHookName(in); got != want {
				t.Errorf("defaultHookName(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

// TestSandboxStatus_ReportsExecBackend verifies status names the backend that
// exec resolves for the same capabilities, in both output formats.
func TestSandboxStatus_ReportsExecBackend(t *testing.T) {
	t.Parallel()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	caps := &sandbox.SystemCapabilities{HasSystemdRun: true, SystemdRunPath: self}
	probe := func(context.Context) *sandbox.SystemCapabilities { return caps }
	want, _ := backendselect.ResolveBackend(*caps)

	t.Run("text", func(t *testing.T) {
		t.Parallel()
		cmd := newSandboxStatusCmd(probe)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(nil)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Backend:") || !strings.Contains(out.String(), want.Name()) {
			t.Errorf("status text does not name backend %q:\n%s", want.Name(), out.String())
		}
	})
	t.Run("json", func(t *testing.T) {
		t.Parallel()
		cmd := newSandboxStatusCmd(probe)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		var status sandboxStatusJSON
		if err := json.Unmarshal(out.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if status.Backend != want.Name() {
			t.Errorf("backend = %q, want %q", status.Backend, want.Name())
		}
	})
}

// TestHookStdio pins that `sandbox exec` hands the hook the command's own
// streams. Claude Code delivers the tool-call payload on stdin; a hook that
// sees /dev/null instead either blocks everything or lets everything through.
func TestHookStdio(t *testing.T) {
	t.Parallel()

	cmd := newSandboxExecCmd(noSandboxProbe)
	in := strings.NewReader(`{"tool_name":"Bash"}`)
	var out, errOut bytes.Buffer
	cmd.SetIn(in)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	opts := hookStdio(cmd)

	if opts.Stdin != in {
		t.Errorf("Stdin = %v, want the command's input reader", opts.Stdin)
	}
	if opts.Stdout != &out {
		t.Errorf("Stdout = %v, want the command's output writer", opts.Stdout)
	}
	if opts.Stderr != &errOut {
		t.Errorf("Stderr = %v, want the command's error writer", opts.Stderr)
	}
}

// TestSandboxExec_BrokenPolicyFailsClosed pins that a policy file which exists
// but cannot be compiled stops `sandbox exec` instead of silently running the
// hook under the default policy.
func TestSandboxExec_BrokenPolicyFailsClosed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.nix")
	if err := os.WriteFile(policyPath, []byte("{ this is not nix"), 0o600); err != nil {
		t.Fatalf("writing policy: %v", err)
	}
	marker := filepath.Join(dir, "hook-ran")

	cmd := newSandboxExecCmd(noSandboxProbe)
	cmd.SetArgs([]string{"--policy", policyPath, "--", "touch", marker})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("sandbox exec with an uncompilable policy succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "sandbox policy") {
		t.Errorf("error = %v, want a sandbox policy error", err)
	}
	// Exit 1 is a non-blocking hook error in Claude Code, which would let the
	// tool call through with the guard hook never having run.
	var coded *exitcode.Error
	if !errors.As(err, &coded) || coded.Code != hookBlockExitCode {
		t.Errorf("error = %#v, want exit code %d (blocking)", err, hookBlockExitCode)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("hook ran despite the uncompilable policy")
	}
}

// TestRunSandboxed_UnavailablePolicyBackendIsSetupFailure pins that a backend
// the policy requires but the host cannot provide is reported as a sandbox
// setup failure (which `sandbox exec` turns into a blocking exit), never as a
// hook result, and that the hook does not run under another backend.
func TestRunSandboxed_UnavailablePolicyBackendIsSetupFailure(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), "hook-ran")
	cfg := &sandbox.SandboxConfig{
		HookCommand: []string{"touch", marker},
		Backend:     "bubblewrap",
	}
	// No bwrap was probed, so the required backend is not a candidate.
	_, err := runSandboxed(context.Background(), cfg, &sandbox.SystemCapabilities{}, io.Discard)
	if !errors.Is(err, sandbox.ErrSetupFailed) {
		t.Fatalf("runSandboxed error = %v, want one wrapping sandbox.ErrSetupFailed", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("hook ran under a backend the policy did not ask for")
	}
}
