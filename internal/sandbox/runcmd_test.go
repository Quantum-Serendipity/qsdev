//go:build !windows

package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunCommand_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		script     string
		wantExit   int
		wantStdout string
	}{
		{"success", "printf ok", 0, "ok"},
		{"non-zero exit is not an error", "printf partial; exit 42", 42, "partial"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			res, err := RunCommand(ctx, exec.CommandContext(ctx, "sh", "-c", tt.script), TierUnsandboxed)
			if err != nil {
				t.Fatalf("RunCommand error: %v", err)
			}
			if res.ExitCode != tt.wantExit || string(res.Stdout) != tt.wantStdout {
				t.Errorf("got exit=%d stdout=%q, want exit=%d stdout=%q",
					res.ExitCode, res.Stdout, tt.wantExit, tt.wantStdout)
			}
			if res.Tier != TierUnsandboxed {
				t.Errorf("Tier = %v, want %v", res.Tier, TierUnsandboxed)
			}
		})
	}
}

// TestRunCommand_ContextDeadlineIsAnError is the regression for a timed-out
// hook being reported as ExitCode -1 with a nil error.
func TestRunCommand_ContextDeadlineIsAnError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	res, err := RunCommand(ctx, exec.CommandContext(ctx, "sh", "-c", "sleep 10"), TierUnsandboxed)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunCommand error = %v (result %+v), want context.DeadlineExceeded", err, res)
	}
}

// TestRunCommand_DescendantHoldingOutputDoesNotHang is the regression for a
// hook that backgrounds a process inheriting stdout: without a WaitDelay the
// run blocked until that descendant exited.
func TestRunCommand_DescendantHoldingOutputDoesNotHang(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 3 & printf done")
	cmd.WaitDelay = 200 * time.Millisecond

	start := time.Now()
	res, err := RunCommand(ctx, cmd, TierUnsandboxed)
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("RunCommand blocked for %v on a descendant holding stdout", elapsed)
	}
	if res.ExitCode != 0 || string(res.Stdout) != "done" {
		t.Errorf("got exit=%d stdout=%q, want exit=0 stdout=%q", res.ExitCode, res.Stdout, "done")
	}
}

func TestRunCommand_SetsDefaultWaitDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "true")
	if _, err := RunCommand(ctx, cmd, TierUnsandboxed); err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if cmd.WaitDelay != commandWaitDelay {
		t.Errorf("WaitDelay = %v, want %v", cmd.WaitDelay, commandWaitDelay)
	}
}

func TestCappedBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		writes     []string
		wantPrefix string
		wantNotice bool
	}{
		{"under the cap", []string{"abc", "de"}, "abcde", false},
		{"exactly the cap", []string{"abcdefgh"}, "abcdefgh", false},
		{"over the cap in one write", []string{"abcdefghij"}, "abcdefgh", true},
		{"over the cap across writes", []string{"abcdef", "ghij", "kl"}, "abcdefgh", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &cappedBuffer{limit: 8}
			for _, w := range tt.writes {
				if n, err := c.Write([]byte(w)); err != nil || n != len(w) {
					t.Fatalf("Write(%q) = %d, %v; want %d, nil", w, n, err, len(w))
				}
			}
			got := string(c.Bytes())
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("Bytes() = %q, want prefix %q", got, tt.wantPrefix)
			}
			if hasNotice := strings.Contains(got, "output truncated"); hasNotice != tt.wantNotice {
				t.Errorf("Bytes() = %q, truncation notice = %v, want %v", got, hasNotice, tt.wantNotice)
			}
			if !tt.wantNotice && got != tt.wantPrefix {
				t.Errorf("Bytes() = %q, want %q", got, tt.wantPrefix)
			}
		})
	}
}
