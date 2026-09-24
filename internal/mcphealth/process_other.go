//go:build !unix

package mcphealth

import (
	"os"
	"os/exec"
)

// setProcessGroup is a no-op where process groups are not portable.
func setProcessGroup(*exec.Cmd) {}

// killProcessGroup kills the server process. Children it spawned are not
// guaranteed to be killed on these platforms; cmd.WaitDelay still keeps Close
// bounded.
func killProcessGroup(proc *os.Process) {
	_ = proc.Kill()
}
