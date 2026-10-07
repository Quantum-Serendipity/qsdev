package perfbudget

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"
)

// Row identifies one measurement: the budget it is held to, its subject (a
// hook as the suites name it, such as "qsdev selfprotect" or
// "package-guard", or a command line) and the payload it ran on ("" for
// none). A budget's tracked skips select rows by subject and payload.
type Row struct {
	Budget  string
	Subject string
	Payload string
}

func (r Row) String() string {
	if r.Payload == "" {
		return fmt.Sprintf("%s (budget %s)", r.Subject, r.Budget)
	}
	return fmt.Sprintf("%s on payload %s (budget %s)", r.Subject, r.Payload, r.Budget)
}

// Dist summarises samples by nearest rank.
type Dist struct {
	P50, P95, Max time.Duration
}

// Stats is a measurement of N timed runs. CPU is what budgets gate; Wall is
// for display. ExitCode, Stdout and Stderr come from the last run.
type Stats struct {
	N        int
	CPU      Dist
	Wall     Dist
	ExitCode int
	Stdout   string
	Stderr   string
}

// newDist returns the nearest-rank p50, p95 and maximum of samples, which it
// leaves in their order.
func newDist(samples []time.Duration) Dist {
	if len(samples) == 0 {
		return Dist{}
	}
	sorted := slices.Sorted(slices.Values(samples))
	rank := func(p float64) time.Duration {
		return sorted[int(math.Ceil(p*float64(len(sorted))))-1]
	}
	return Dist{P50: rank(0.50), P95: rank(0.95), Max: sorted[len(sorted)-1]}
}

// Verdict is the outcome of holding a row to its budget.
type Verdict int

const (
	// VerdictPass: within budget, and no tracked skip covers the row.
	VerdictPass Verdict = iota
	// VerdictPassMayRemoveSkip: a tracked skip covers the row, but it is
	// within budget with the allowance to spare (CPU p95 × allowance <=
	// budget × factor, the margin a row needs to land without a skip), so
	// its owner may remove the skip. Check still skips the row.
	VerdictPassMayRemoveSkip
	// VerdictSkip: a tracked skip covers the row, which is over budget or
	// within it by less than the allowance, and under the skip's ceiling.
	VerdictSkip
	// VerdictFail: over budget with no tracked skip, over the ceiling of
	// the tracked skip covering it, or a row that cannot be judged.
	VerdictFail
)

func (v Verdict) String() string {
	switch v {
	case VerdictPass:
		return "pass"
	case VerdictPassMayRemoveSkip:
		return "pass-may-remove-skip"
	case VerdictSkip:
		return "skip"
	case VerdictFail:
		return "fail"
	}
	return fmt.Sprintf("Verdict(%d)", int(v))
}

// Result is a verdict with the owner of the tracked skip covering the row,
// if any, the scaled limit and skip ceiling, and a readable message. A
// skip's message starts with "<owner>: ", a pass that may remove its skip
// with "<owner> may remove its skip: ".
type Result struct {
	Verdict    Verdict
	Workstream string
	Limit      time.Duration
	Ceiling    time.Duration
	Message    string
}

// Judge holds s to row's CPU budget: its CPU p95 must not exceed budget ×
// factor × allowance, where factor is the calibration factor of the machine
// that measured s. When a tracked skip of the budget covers row (the first
// one that does applies), the row fails only over the skip's ceiling ×
// factor × allowance; it passes, saying its owner may remove the skip, when
// its CPU p95 × allowance is within budget × factor, and skips otherwise.
// A row whose budget is unknown or not a CPU budget, or that has no runs or
// no positive factor, fails.
func (b Budgets) Judge(row Row, s Stats, factor float64) Result {
	bud, ok := b.Lookup(row.Budget)
	switch {
	case !ok:
		return Result{Verdict: VerdictFail, Message: fmt.Sprintf("%s: budgets.yaml declares no budget %q", row, row.Budget)}
	case bud.CPUP95 == 0:
		return Result{Verdict: VerdictFail, Message: fmt.Sprintf("%s: budget %s is not a cpu_p95_ms budget; its owner measures it", row, bud.ID)}
	case factor <= 0:
		return Result{Verdict: VerdictFail, Message: fmt.Sprintf("%s: calibration factor %v must be positive", row, factor)}
	case s.N == 0:
		return Result{Verdict: VerdictFail, Message: fmt.Sprintf("%s: no runs measured", row)}
	}
	limit := b.scale(bud.CPUP95, factor)
	r := Result{Limit: limit}
	skip, tracked := bud.skipFor(row)
	if !tracked {
		if s.CPU.P95 > limit {
			r.Verdict = VerdictFail
			r.Message = fmt.Sprintf("%s is over budget: %s; %s", row, b.compare(s, "limit", "budget", bud.CPUP95, limit, factor), s.detail())
		} else {
			r.Verdict = VerdictPass
			r.Message = fmt.Sprintf("%s is within budget: %s", row, b.compare(s, "limit", "budget", bud.CPUP95, limit, factor))
		}
		return r
	}

	r.Workstream, r.Ceiling = skip.Workstream, b.scale(skip.Ceiling, factor)
	switch {
	case s.CPU.P95 > r.Ceiling:
		r.Verdict = VerdictFail
		r.Message = fmt.Sprintf("%s is over the ceiling of its %s skip, so it regressed beyond the budget %s owes: %s; %s",
			row, skip.Workstream, skip.Workstream, b.compare(s, "ceiling", "ceiling", skip.Ceiling, r.Ceiling, factor), s.detail())
	case s.CPU.P95 <= time.Duration(float64(limit)/(b.Allowance*b.Allowance)):
		// Within budget × factor even after the allowance's margin, as a
		// row must be to land without a skip, so one such pass is enough
		// to remove the skip without the gate flapping.
		r.Verdict = VerdictPassMayRemoveSkip
		r.Message = fmt.Sprintf("%s may remove its skip: %s is within budget with the allowance to spare: %s",
			skip.Workstream, row, b.compare(s, "limit", "budget", bud.CPUP95, limit, factor))
	default:
		r.Verdict = VerdictSkip
		r.Message = fmt.Sprintf("%s: %s is over budget, or within it by less than the allowance: %s; tracked skip: %s",
			skip.Workstream, row, b.compare(s, "limit", "budget", bud.CPUP95, limit, factor), skip.Reason)
	}
	return r
}

// scale returns the measured limit for a budget or ceiling of d on a
// machine with the given calibration factor: d × factor × allowance.
func (b Budgets) scale(d time.Duration, factor float64) time.Duration {
	return time.Duration(float64(d) * factor * b.Allowance)
}

// compare spells out how s's CPU p95 compares to the limit called name, the
// one budgets.yaml sets as source scaled by factor and the allowance.
func (b Budgets) compare(s Stats, name, source string, unscaled, scaled time.Duration, factor float64) string {
	op := "<="
	if s.CPU.P95 > scaled {
		op = ">"
	}
	return fmt.Sprintf("CPU p95 %s ms %s %s %s ms (%s %s ms × calibration %.2f × allowance %.2f)",
		ms(s.CPU.P95), op, name, ms(scaled), source, ms(unscaled), factor, b.Allowance)
}

// detail spells out every statistic of s.
func (s Stats) detail() string {
	return fmt.Sprintf("cpu p50/p95/max %s/%s/%s ms; wall p50/p95/max %s/%s/%s ms; %d runs; rc %d",
		ms(s.CPU.P50), ms(s.CPU.P95), ms(s.CPU.Max), ms(s.Wall.P50), ms(s.Wall.P95), ms(s.Wall.Max), s.N, s.ExitCode)
}

// Check judges row (see Judge), reports it (see Report) and applies the
// verdict to t: a fail is t.Errorf, and a row a tracked skip covers ends t
// with t.Skipf, whether it skips or may remove its skip; only the message
// tells them apart. So the set of skipped rows is the set of rows tracked
// skips cover, and measurement noise around the budget cannot change it
// from run to run. Run each row in its own subtest, so a skip ends only
// that row.
func (b Budgets) Check(t testing.TB, row Row, s Stats, factor float64) Result {
	t.Helper()
	r := b.Judge(row, s, factor)
	Report(t, row, s, r)
	switch r.Verdict {
	case VerdictFail:
		t.Errorf("%s", r.Message)
	case VerdictSkip, VerdictPassMayRemoveSkip:
		t.Skipf("%s", r.Message)
	case VerdictPass:
	}
	return r
}

// ms formats d as milliseconds with one decimal.
func ms(d time.Duration) string {
	return fmt.Sprintf("%.1f", float64(d)/float64(time.Millisecond))
}
