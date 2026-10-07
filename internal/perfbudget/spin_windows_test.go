//go:build windows

package perfbudget

import (
	"fmt"
	"syscall"
	"time"
)

// spinCPU burns CPU until this process has used at least d of it, read
// from GetProcessTimes rather than the clock, so a busy machine makes it
// run longer rather than burn less.
func spinCPU(d time.Duration) error {
	proc, err := syscall.GetCurrentProcess()
	if err != nil {
		return fmt.Errorf("GetCurrentProcess: %w", err)
	}
	x := 0
	for {
		var creation, exit, kernel, user syscall.Filetime
		if err := syscall.GetProcessTimes(proc, &creation, &exit, &kernel, &user); err != nil {
			return fmt.Errorf("GetProcessTimes: %w", err)
		}
		if filetimeDuration(kernel)+filetimeDuration(user) >= d {
			return nil
		}
		for i := range 100_000 {
			x += i
		}
		_ = x
	}
}

// filetimeDuration reads a FILETIME holding a duration, in 100 ns units.
func filetimeDuration(ft syscall.Filetime) time.Duration {
	return time.Duration(int64(ft.HighDateTime)<<32|int64(ft.LowDateTime)) * 100
}
