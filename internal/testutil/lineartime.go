package testutil

import (
	"testing"
	"time"
)

// Growth-check parameters for AssertLinearTime. The small input runs
// linearScale times per sample, so linear work costs about the same in both
// timed windows (ratio near 1) and quadratic work costs linearScale times more
// in the large one (ratio near 8). linearMaxRatio sits near their geometric
// mean: far enough from both that timer noise cannot flip the verdict.
const (
	linearScale    = 8
	linearMaxRatio = 3
	linearSamples  = 3
	// linearFloor is the time under which the large input passes without a
	// ratio check: work this fast is not a denial-of-service risk, and its
	// ratio is dominated by timer resolution and scheduler jitter.
	linearFloor = 50 * time.Millisecond
)

// AssertLinearTime fails t when run's cost grows faster than linearly in its
// input size. Each sample times linearScale back-to-back runs of size n/8
// against one run of size n, taking the best of three interleaved samples of
// each, and requires the ratio to stay well under quadratic growth.
//
// Comparing two sizes on the same machine at the same moment measures the
// algorithm rather than the machine. An absolute budget does not: the same
// linear code that takes 0.7s alone takes several seconds under -race while
// the rest of ./... runs in parallel, so a wall-clock limit fails on a busy
// developer box or CI runner without any change to the code. Giving both
// sizes timing windows of the same length matters for the same reason: a
// short window often escapes the preemption, GC and contention that a window
// eight times longer absorbs, which inflates the ratio of linear code.
func AssertLinearTime(t testing.TB, name string, n int, run func(n int)) {
	t.Helper()
	small := max(n/linearScale, 1)
	reps := n / small
	bestSmall, bestLarge := time.Duration(1<<63-1), time.Duration(1<<63-1)
	for range linearSamples {
		bestSmall = min(bestSmall, timeRuns(run, small, reps))
		bestLarge = min(bestLarge, timeRuns(run, n, 1))
	}
	if bestLarge < linearFloor {
		return
	}
	ratio := float64(bestLarge) / float64(max(bestSmall, time.Microsecond))
	if ratio > linearMaxRatio {
		t.Errorf("%s: %d runs of size %d took %v but one run of size %d took %v (%.1fx, want under %dx): growth is superlinear",
			name, reps, small, bestSmall, n, bestLarge, ratio, linearMaxRatio)
	}
}

// timeRuns returns how long reps back-to-back calls of run(n) take.
func timeRuns(run func(n int), n, reps int) time.Duration {
	start := time.Now()
	for range reps {
		run(n)
	}
	return time.Since(start)
}
