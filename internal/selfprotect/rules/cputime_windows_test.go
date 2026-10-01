//go:build windows

package rules

import (
	"syscall"
	"testing"
	"time"
)

// cpuClockResolution is the granularity of processCPUTime: GetProcessTimes
// advances once per clock interrupt, 15.625ms by default.
const cpuClockResolution = 15625 * time.Microsecond

// processCPUTime returns the user plus kernel CPU time this process has used.
func processCPUTime(t *testing.T) time.Duration {
	t.Helper()
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatalf("GetCurrentProcess: %v", err)
	}
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		t.Fatalf("GetProcessTimes: %v", err)
	}
	return filetimeDuration(kernel) + filetimeDuration(user)
}

// filetimeDuration converts a FILETIME interval, counted in 100ns units.
func filetimeDuration(ft syscall.Filetime) time.Duration {
	return time.Duration(int64(ft.HighDateTime)<<32|int64(ft.LowDateTime)) * 100
}
