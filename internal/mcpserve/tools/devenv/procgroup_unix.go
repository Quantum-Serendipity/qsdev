//go:build unix

package devenv

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// runProcessGroup starts name with argv in a NEW process group (Setpgid) so the
// launcher and every child it spawns share one group id. On timeout or context
// cancellation it signals the whole group via a negative PID, killing orphaned
// children (e.g. a nix build) rather than leaking them, then reaps the launcher.
// stdout and stderr are captured separately.
func runProcessGroup(ctx context.Context, name string, argv []string, stdin string, timeout time.Duration) procResult {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(name, argv...) //nolint:gosec // argv is an explicit array; no shell interpolation
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return procResult{startErr: err, duration: time.Since(start)}
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timedOut := false
	var waitErr error
	select {
	case <-ctx.Done():
		// Negative PID targets the entire process group; SIGKILL guarantees the
		// group dies even if a child ignores softer signals.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done // reap so cmd.ProcessState is populated
		timedOut = true
	case waitErr = <-done:
	}

	return procResult{
		stdout:   outBuf.String(),
		stderr:   errBuf.String(),
		exitCode: processExitCode(cmd, waitErr),
		timedOut: timedOut,
		duration: time.Since(start),
	}
}

// processExitCode resolves the exit code from the finished command, returning -1
// when the process was killed or never produced a status.
func processExitCode(cmd *exec.Cmd, waitErr error) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if waitErr != nil {
		return -1
	}
	return 0
}
