package testutil

import (
	"testing"
	"time"
)

// Growth-check parameters for AssertLinearTime. Linear work grows by about
// linearScale between the two input sizes and quadratic work by its square
// (64), so linearMaxRatio sits near their geometric mean: far enough from
// both that timer noise and uneven contention cannot flip the verdict.
const (
	linearScale    = 8
	linearMaxRatio = 24
	linearSamples  = 3
	// linearFloor is the time under which the large input passes without a
	// ratio check: work this fast is not a denial-of-service risk, and its
	// ratio is dominated by timer resolution and scheduler jitter.
	linearFloor = 50 * time.Millisecond
)

// AssertLinearTime fails t when run's cost grows faster than linearly in its
// input size. It times run(n/8) and run(n), taking the best of three
// interleaved samples of each, and requires the ratio to stay well under
// quadratic growth.
//
// Comparing two sizes on the same machine at the same moment measures the
// algorithm rather than the machine. An absolute budget does not: the same
// linear code that takes 0.7s alone takes several seconds under -race while
// the rest of ./... runs in parallel, so a wall-clock limit fails on a busy
// developer box or CI runner without any change to the code.
func AssertLinearTime(t testing.TB, name string, n int, run func(n int)) {
	t.Helper()
	small := max(n/linearScale, 1)
	bestSmall, bestLarge := time.Duration(1<<63-1), time.Duration(1<<63-1)
	for range linearSamples {
		bestSmall = min(bestSmall, timeRun(run, small))
		bestLarge = min(bestLarge, timeRun(run, n))
	}
	if bestLarge < linearFloor {
		return
	}
	ratio := float64(bestLarge) / float64(max(bestSmall, time.Microsecond))
	if ratio > linearMaxRatio {
		t.Errorf("%s: size %d took %v but size %d took %v (%.1fx for %dx the input, want under %dx): growth is superlinear",
			name, small, bestSmall, n, bestLarge, ratio, n/small, linearMaxRatio)
	}
}

func timeRun(run func(n int), n int) time.Duration {
	start := time.Now()
	run(n)
	return time.Since(start)
}
