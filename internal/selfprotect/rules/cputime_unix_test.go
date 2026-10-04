//go:build unix

package rules

import (
	"syscall"
	"testing"
	"time"
)

// cpuClockResolution is the granularity of processCPUTime: getrusage reports
// microseconds.
const cpuClockResolution = time.Microsecond

// processCPUTime returns the user plus system CPU time this process has used.
func processCPUTime(t *testing.T) time.Duration {
	t.Helper()
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		t.Fatalf("getrusage: %v", err)
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
