package toolcheck

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

const (
	// probeTimeout bounds how long a version probe may run.
	probeTimeout = 5 * time.Second
	// probeWaitDelay bounds how long Detect waits, after the probe is killed
	// on timeout, for its output pipes to close. Without it a grandchild that
	// inherited the pipes (e.g. a wrapper script that does not exec) keeps
	// Detect blocked until the grandchild exits, defeating the timeout.
	probeWaitDelay = 500 * time.Millisecond
	// maxProbeOutput caps the version output retained per stream.
	maxProbeOutput = 64 << 10
)

// Info holds the result of detecting a tool on the system.
type Info struct {
	Found bool
	Path  string
	// Version is the first non-empty line of the version output.
	Version string
	// Output is the full (bounded) version output: stdout, or stderr when the
	// tool printed nothing to stdout. Parsers for tools whose version is not
	// on the first line (or is printed as JSON) should use it.
	Output string
}

// Detect checks whether the named tool exists on PATH and, if so,
// runs it with versionArg to capture its version output.
func Detect(ctx context.Context, name, versionArg string) Info {
	path, err := exec.LookPath(name)
	if err != nil {
		return Info{}
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, versionArg)
	cmd.WaitDelay = probeWaitDelay
	stdout := &limitedBuffer{max: maxProbeOutput}
	stderr := &limitedBuffer{max: maxProbeOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return Info{Found: true, Path: path}
	}

	// Keep stdout and stderr apart so a warning printed on stderr cannot
	// displace the version; fall back to stderr for tools that print their
	// version there.
	output := stdout.String()
	if strings.TrimSpace(output) == "" {
		output = stderr.String()
	}
	return Info{Found: true, Path: path, Version: FirstLine(output), Output: output}
}

// FirstLine returns the first non-empty line of s, trimmed of surrounding
// whitespace.
func FirstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// limitedBuffer retains at most max bytes and silently discards the rest, so
// a misbehaving tool cannot make Detect buffer unbounded output. Writes always
// report success so the child is never stopped by a short write.
type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.max - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}
