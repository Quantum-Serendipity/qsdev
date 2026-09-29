//go:build !windows

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestEnvList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want []string
	}{
		// A nil exec.Cmd.Env inherits the whole parent environment, so an empty
		// map must produce an empty, non-nil list.
		{name: "empty map stays empty", env: map[string]string{}, want: []string{}},
		{name: "nil map stays empty", env: nil, want: []string{}},
		{name: "sorted pairs", env: map[string]string{"B": "2", "A": "1"}, want: []string{"A=1", "B=2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EnvList(tt.env)
			if got == nil {
				t.Fatal("EnvList returned nil")
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("EnvList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	exitErr := exec.Command("sh", "-c", "exit 7").Run()
	if exitErr == nil {
		t.Fatal("expected sh -c 'exit 7' to fail")
	}

	tests := []struct {
		name     string
		err      error
		wantCode int
		wantErr  bool
	}{
		{name: "success", err: nil, wantCode: 0},
		{name: "exit status", err: exitErr, wantCode: 7},
		// errors.As, not a type assertion: a wrapped exit status is still the
		// command's own exit code.
		{name: "wrapped exit status", err: fmt.Errorf("running: %w", exitErr), wantCode: 7},
		{name: "start failure", err: errors.New("exec: not found"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, err := ExitCode(tt.err)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExitCode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("ExitCode() = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

// TestUnsandboxedBackend_ForwardsStdin pins that the hook receives its stdin:
// Claude Code delivers the tool-call payload there.
func TestUnsandboxedBackend_ForwardsStdin(t *testing.T) {
	t.Parallel()

	payload := `{"tool_name":"Write","tool_input":{"file_path":"x"}}`
	cfg := &SandboxConfig{
		HookCommand: []string{"cat"},
		ExecOpts:    ExecOpts{Stdin: strings.NewReader(payload)},
	}

	res, err := (&UnsandboxedBackend{}).RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if got := string(res.Stdout); got != payload {
		t.Errorf("stdout = %q, want %q", got, payload)
	}
}

// TestUnsandboxedBackend_StreamsOutput pins that output goes to the configured
// writers as the hook runs instead of only being buffered into the result.
func TestUnsandboxedBackend_StreamsOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cfg := &SandboxConfig{
		HookCommand: []string{"sh", "-c", "printf out; printf err >&2"},
		ExecOpts:    ExecOpts{Stdout: &stdout, Stderr: &stderr},
	}

	if _, err := (&UnsandboxedBackend{}).RunHook(context.Background(), cfg); err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Errorf("streamed stdout=%q stderr=%q, want out/err", stdout.String(), stderr.String())
	}
}

// TestUnsandboxedBackend_EmptyEnvironmentDoesNotInherit is the fail-open
// regression: an empty (non-nil) Environment must give the hook an empty
// environment, not the parent's. It uses t.Setenv, so it is not parallel.
func TestUnsandboxedBackend_EmptyEnvironmentDoesNotInherit(t *testing.T) {
	t.Setenv("QSDEV_TEST_CANARY_SECRET", "leaked")
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}

	cfg := &SandboxConfig{
		HookCommand: []string{sh, "-c", `printf '%s' "${QSDEV_TEST_CANARY_SECRET:-ABSENT}"`},
		Environment: map[string]string{},
	}

	res, err := (&UnsandboxedBackend{}).RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if got := string(res.Stdout); got != "ABSENT" {
		t.Errorf("hook saw the parent environment: stdout=%q", got)
	}
}
