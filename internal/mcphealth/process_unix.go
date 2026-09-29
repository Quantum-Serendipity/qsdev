//go:build unix

package mcphealth

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroup starts cmd as the leader of a new process group, so a
// launcher and every child it spawns share one group id.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs every process in the group led by proc. The
// negative PID targets the group; once every member has exited it fails with
// ESRCH, which is ignored.
func killProcessGroup(proc *os.Process) {
	_ = syscall.Kill(-proc.Pid, syscall.SIGKILL)
	_ = proc.Kill()
}
