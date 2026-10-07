//go:build !unix && !windows

package perfbudget

import (
	"errors"
	"time"
)

// spinCPU needs the process's own CPU time, which the spin helper reads
// through getrusage on unix and GetProcessTimes on Windows; this OS offers
// neither.
func spinCPU(time.Duration) error {
	return errors.New("spinning needs the process's own CPU time, which this OS does not report")
}
