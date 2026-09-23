package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// MaxCapturedOutput caps how many bytes of each stream (stdout, stderr) a hook
// may buffer. Output beyond the cap is discarded and a truncation notice is
// appended, so a runaway hook cannot exhaust memory.
const MaxCapturedOutput = 16 << 20

// commandWaitDelay bounds how long RunCommand waits for the hook's output pipes
// to close after the hook exits or its context ends. Without it, a descendant
// that inherited stdout (e.g. a backgrounded daemon) keeps the run blocked.
const commandWaitDelay = 5 * time.Second

// RunCommand runs cmd to completion and collects its result at the given tier.
// cmd must have been created with exec.CommandContext(ctx, ...). Stdout and
// stderr are captured, each capped at MaxCapturedOutput. A non-zero exit is
// reported through SandboxResult.ExitCode with a nil error. When ctx ends
// before the command completes, the error wraps ctx.Err() instead of reporting
// the kill as an exit code.
func RunCommand(ctx context.Context, cmd *exec.Cmd, tier DegradationTier) (*SandboxResult, error) {
	stdout := &cappedBuffer{limit: MaxCapturedOutput}
	stderr := &cappedBuffer{limit: MaxCapturedOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = commandWaitDelay
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil && ctx.Err() != nil {
		return nil, fmt.Errorf("running %s: %w", cmd.Path, ctx.Err())
	}

	exitCode := 0
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		exitCode = exitErr.ExitCode()
	case errors.Is(err, exec.ErrWaitDelay):
		// The hook exited successfully but a descendant still held its
		// output open; the pipes were closed after WaitDelay. The hook's own
		// status stands.
		exitCode = cmd.ProcessState.ExitCode()
	default:
		return nil, fmt.Errorf("running %s: %w", cmd.Path, err)
	}

	return &SandboxResult{
		ExitCode: exitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: duration,
		Tier:     tier,
	}, nil
}

// cappedBuffer is an io.Writer that keeps at most limit bytes and silently
// drops the rest, so the writing process never sees a short write or EPIPE.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated int
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	keep := min(len(p), max(c.limit-c.buf.Len(), 0))
	c.buf.Write(p[:keep])
	c.truncated += len(p) - keep
	return len(p), nil
}

// Bytes returns the captured output, followed by a notice when some of it was
// dropped.
func (c *cappedBuffer) Bytes() []byte {
	if c.truncated == 0 {
		return c.buf.Bytes()
	}
	notice := "\n[qsdev: output truncated, " + strconv.Itoa(c.truncated) + " bytes dropped]\n"
	return append(c.buf.Bytes(), notice...)
}
