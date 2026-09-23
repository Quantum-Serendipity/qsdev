//go:build !unix

package devenv

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// runProcessGroup is the non-unix fallback. Process groups and group-targeted
// signals are not portable, so it relies on exec.CommandContext to kill the
// launched process when the timeout fires. Child processes are not guaranteed to
// be reaped on these platforms, which is acceptable because nix_run targets
// unix-like development hosts. The process runs in dir (the server's working
// directory when dir is empty).
func runProcessGroup(ctx context.Context, dir, name string, argv []string, stdin string, timeout time.Duration) procResult {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, argv...) //nolint:gosec // argv is an explicit array; no shell interpolation
	cmd.Dir = dir

	outBuf, errBuf := newCappedBuffer(maxProcOutputBytes), newCappedBuffer(maxProcOutputBytes)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	cmd.WaitDelay = procWaitDelay
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	start := time.Now()
	runErr := cmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded

	exitCode := 0
	switch {
	case cmd.ProcessState != nil:
		exitCode = cmd.ProcessState.ExitCode()
	case runErr != nil:
		exitCode = -1
	}

	return procResult{
		stdout:          outBuf.String(),
		stderr:          errBuf.String(),
		stdoutTruncated: outBuf.truncated,
		stderrTruncated: errBuf.truncated,
		exitCode:        exitCode,
		timedOut:        timedOut,
		duration:        time.Since(start),
	}
}
