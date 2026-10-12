//go:build unix

package perfbudget

import (
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

// spinCPU burns CPU until this process has used at least d of it. It reads
// the process's own CPU time, not the clock, so a busy machine makes it run
// longer rather than burn less.
func spinCPU(d time.Duration) error {
	x := 0
	for {
		var ru syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
			return fmt.Errorf("getrusage: %w", err)
		}
		if time.Duration(ru.Utime.Nano()+ru.Stime.Nano()) >= d {
			return nil
		}
		for i := range 100_000 {
			x += i
		}
		_ = x
	}
}

// TestSpinOverBudgetFails stands in for a regression: a helper that spins
// 50 ms of CPU past its scaled budget, and past the ceiling of a tracked
// skip, is measured and judged, and the verdict fails naming the subject
// and the payload. A tracked skip waives a budget only up to its ceiling,
// so a regression on a surface whose budget is still owed fails too. This
// is the plan's "injected 50 ms" criterion as CPU, which a sleep would not
// burn.
func TestSpinOverBudgetFails(t *testing.T) {
	summaryFile(t)
	b := runBudgets(t)
	capped, _ := b.Lookup("capped")
	factor := float64(testutil.RaceScale)
	spin := time.Duration(float64(capped.Skips[0].Ceiling)*factor) + 50*time.Millisecond
	s := b.Run(t, helperSpec(t, "spin:"+spin.String()), 1)
	if s.ExitCode != 0 {
		t.Fatalf("helper exited %d: %s", s.ExitCode, s.Stderr)
	}
	if s.CPU.P95 < spin {
		t.Fatalf("measured CPU p95 %v, want at least the %v the helper spun", s.CPU.P95, spin)
	}

	row := Row{Budget: "hook", Subject: "helper", Payload: "bash_ls"}
	r := b.Judge(row, s, factor)
	if r.Verdict != VerdictFail {
		t.Fatalf("Judge = %+v, want a fail", r)
	}
	for _, want := range []string{"helper", "bash_ls", "over budget"} {
		if !strings.Contains(r.Message, want) {
			t.Errorf("fail message %q does not mention %q", r.Message, want)
		}
	}
	rec := &recorder{TB: t}
	b.Check(rec, row, s, factor)
	if len(rec.errors) != 1 || rec.errors[0] != r.Message {
		t.Errorf("Check reported %q, want the fail %q", rec.errors, r.Message)
	}

	// Under a tracked skip whose ceiling it is past, it still fails.
	over := b.Judge(Row{Budget: "capped", Subject: "helper", Payload: "bash_ls"}, s, factor)
	if over.Verdict != VerdictFail || over.Workstream != "XD-WS3" || !strings.Contains(over.Message, "over the ceiling of its XD-WS3 skip") {
		t.Errorf("Judge past a tracked skip's ceiling = %+v, want a fail naming the XD-WS3 skip", over)
	}
	rec = &recorder{TB: t}
	b.Check(rec, Row{Budget: "capped", Subject: "helper", Payload: "bash_ls"}, s, factor)
	if len(rec.errors) != 1 || len(rec.skips) != 0 {
		t.Errorf("Check past a tracked skip's ceiling made errors %q and skips %q, want one error", rec.errors, rec.skips)
	}

	// Under a tracked skip whose ceiling it is within, it skips with the
	// skip's owner.
	skipped := b.Judge(Row{Budget: "tracked", Subject: "helper", Payload: "bash_ls"}, s, factor)
	if skipped.Verdict != VerdictSkip || !strings.HasPrefix(skipped.Message, "XD-WS3: ") {
		t.Errorf("Judge under a tracked skip = %+v, want a skip starting \"XD-WS3: \"", skipped)
	}
}
