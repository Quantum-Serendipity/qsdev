package hookio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
)

// EvalDeadline bounds a self-protection hook's whole run, reading stdin
// included. Claude Code treats a hook that outlives its registered timeout as
// non-blocking and lets the call through, so the hook must deny on its own
// first: the deadline sits at most 80% of, and at least 2s under, the
// registered timeout.
const EvalDeadline = 7 * time.Second

// RunWithDeadline runs eval until it returns or ctx is done, whichever comes
// first. eval writes into a private buffer that is copied to w only when eval
// finishes first, so an evaluation abandoned at the deadline can never write
// to w afterwards or interleave with the timeout deny.
//
// When ctx ends first, RunWithDeadline writes an SP-TIMEOUT deny to w and
// returns timedOut=true; the abandoned goroutine is left to the process exit.
// Otherwise it returns eval's error, with a panic in eval recovered into an
// error.
func RunWithDeadline(ctx context.Context, eval func(ctx context.Context, w io.Writer) error, w io.Writer) (timedOut bool, err error) {
	type result struct {
		out *bytes.Buffer
		err error
	}
	done := make(chan result, 1)
	go func() {
		var out bytes.Buffer
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("evaluation panicked: %v", r)
				}
			}()
			return eval(ctx, &out)
		}()
		done <- result{&out, err}
	}()

	select {
	case <-ctx.Done():
		WriteDeny(w, "SP-TIMEOUT", fmt.Sprintf("evaluation exceeded the %v deadline; denying (fail closed)", EvalDeadline))
		return true, nil
	case res := <-done:
		if _, werr := res.out.WriteTo(w); werr != nil && res.err == nil {
			return false, fmt.Errorf("writing hook output: %w", werr)
		}
		return false, res.err
	}
}
