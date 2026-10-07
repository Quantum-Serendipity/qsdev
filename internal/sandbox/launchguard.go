package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// launchGuardWait bounds Started's read. The ready byte is written before
// the child exits, so it is already buffered when Started runs; the wait only
// matters when a lingering process still holds a copy of the write end.
const launchGuardWait = 250 * time.Millisecond

// stderrHeadLimit bounds how much of the child's stderr a guard retains for
// its error message. A sandbox reports its setup failure before the hook
// would start, so the diagnostic is at the start of the stream.
const stderrHeadLimit = 4096

// firstExtraFD is the fd number of a command's first ExtraFiles entry.
const firstExtraFD = 3

// LaunchGuard tells a sandbox's own setup failure apart from the hook's exit
// code. The backend passes ChildFile to the child as the shim's ready fd
// (internal/sandbox/shim); the shim writes one byte to it just before it
// execs the hook. After the child exits, Started reports whether that byte
// arrived: if not, the sandbox never reached the hook and its exit code
// belongs to the sandbox, which is ErrSetupFailed. A guard that cannot be
// passed (no ExtraFiles support, as on Windows) fails the start, which is
// also a setup failure: it fails closed.
type LaunchGuard struct {
	r, w *os.File
	head headWriter
	// pending is the read Started began on a pipe without deadline support;
	// later calls wait on it instead of starting another.
	pending *readyRead
}

// readyRead is a read of the ready byte running in its own goroutine. got is
// written before done is closed and read only after it is.
type readyRead struct {
	done chan struct{}
	got  bool
}

// NewLaunchGuard creates the guard's pipe. Close it when done.
func NewLaunchGuard() (*LaunchGuard, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("creating sandbox launch guard: %w", err)
	}
	return newLaunchGuard(r, w), nil
}

// newLaunchGuard builds a guard over an existing pipe.
func newLaunchGuard(r, w *os.File) *LaunchGuard {
	return &LaunchGuard{r: r, w: w, head: headWriter{limit: stderrHeadLimit}}
}

// ChildFile is the write end to pass to the child in cmd.ExtraFiles.
func (g *LaunchGuard) ChildFile() *os.File {
	return g.w
}

// AppendChildFile appends ChildFile to a command's ExtraFiles and returns
// them with the fd number the child sees it at, which follows from the files
// before it, so callers never hard-code it.
func (g *LaunchGuard) AppendChildFile(extra []*os.File) ([]*os.File, int) {
	return append(extra, g.w), firstExtraFD + len(extra)
}

// StderrTap is a writer to tee the child's stderr into (RunCommand's
// stderrTaps); the guard keeps its head for Err's message.
func (g *LaunchGuard) StderrTap() io.Writer {
	return &g.head
}

// StderrHead returns the start of the child's stderr seen through StderrTap.
func (g *LaunchGuard) StderrHead() []byte {
	return g.head.buf
}

// Err reports a run whose child exited with exitCode: nil when the ready byte
// arrived (the exit code is the hook's own), otherwise an error wrapping
// ErrSetupFailed with the first stderr line and, when non-empty, hint, which
// is computed only on that path.
func (g *LaunchGuard) Err(exitCode int, hint func() string) error {
	if g.Started() {
		return nil
	}
	msg := FirstLine(g.head.buf)
	if h := hint(); h != "" {
		msg += "; " + h
	}
	return fmt.Errorf("%w: sandbox did not start the hook (exit %d): %s", ErrSetupFailed, exitCode, msg)
}

// Started closes the parent's write end and reports whether the ready byte
// arrived. Call it after the child has exited. It returns within
// launchGuardWait even when another process still holds the write end, and
// returns false whenever it cannot tell.
//
// The bound is a read deadline where the pipe supports one. An os.Pipe on
// Windows is a synchronous anonymous pipe, which never does
// (os.ErrNoDeadline); there the read runs in a goroutine that Started waits
// on for launchGuardWait, and Close unblocks it if it is still pending.
func (g *LaunchGuard) Started() bool {
	if g.w != nil {
		_ = g.w.Close()
		g.w = nil
	}
	if g.pending != nil {
		return g.pending.wait()
	}
	err := g.r.SetReadDeadline(time.Now().Add(launchGuardWait))
	if errors.Is(err, os.ErrNoDeadline) {
		g.pending = startReadyRead(g.r)
		return g.pending.wait()
	}
	if err != nil {
		return false
	}
	var b [1]byte
	n, _ := g.r.Read(b[:])
	return n == 1
}

// startReadyRead reads the ready byte from r in a goroutine.
func startReadyRead(r *os.File) *readyRead {
	rr := &readyRead{done: make(chan struct{})}
	go func() {
		defer close(rr.done)
		var b [1]byte
		n, _ := r.Read(b[:])
		rr.got = n == 1
	}()
	return rr
}

// wait reports the read's result, or false when it has not finished within
// launchGuardWait.
func (rr *readyRead) wait() bool {
	timer := time.NewTimer(launchGuardWait)
	defer timer.Stop()
	select {
	case <-rr.done:
		return rr.got
	case <-timer.C:
		return false
	}
}

// Close releases both ends of the pipe.
func (g *LaunchGuard) Close() error {
	var errs []error
	if g.w != nil {
		errs = append(errs, g.w.Close())
		g.w = nil
	}
	errs = append(errs, g.r.Close())
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("closing sandbox launch guard: %w", err)
	}
	return nil
}

// headWriter retains the first limit bytes written to it and discards the rest.
// It never fails, so it can sit behind an io.MultiWriter without cutting off
// the primary stream.
type headWriter struct {
	buf   []byte
	limit int
}

func (h *headWriter) Write(p []byte) (int, error) {
	if room := h.limit - len(h.buf); room > 0 {
		h.buf = append(h.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

// FirstLine returns the first line of b, for error messages.
func FirstLine(b []byte) string {
	line, _, _ := bytes.Cut(b, []byte("\n"))
	return string(line)
}
