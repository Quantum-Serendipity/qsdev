package testutil

import (
	"testing"
	"time"
)

// errorRecorder captures Errorf calls so a test can check whether a helper
// would have failed its caller.
type errorRecorder struct {
	testing.TB
	failed bool
}

func (r *errorRecorder) Helper() {}

func (r *errorRecorder) Errorf(string, ...any) { r.failed = true }

func TestAssertLinearTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		n        int
		run      func(n int)
		wantFail bool
	}{
		{
			name: "linear",
			n:    8000,
			run:  func(n int) { time.Sleep(time.Duration(n) * 10 * time.Microsecond) },
		},
		{
			name:     "quadratic",
			n:        800,
			run:      func(n int) { time.Sleep(time.Duration(n*n) * 100 * time.Nanosecond) },
			wantFail: true,
		},
		{
			name: "quadratic but under the floor",
			n:    80,
			run:  func(n int) { time.Sleep(time.Duration(n*n) * 100 * time.Nanosecond) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &errorRecorder{TB: t}
			AssertLinearTime(rec, tt.name, tt.n, tt.run)
			if rec.failed != tt.wantFail {
				t.Errorf("AssertLinearTime failed = %v, want %v", rec.failed, tt.wantFail)
			}
		})
	}
}
