package cmdutil

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// parentPID returns pid's parent from the kern.proc.pid sysctl.
func parentPID(pid int) (int, error) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, fmt.Errorf("reading the parent of process %d: %w", pid, err)
	}
	return int(kp.Eproc.Ppid), nil
}

// processArgs reads pid's kern.procargs2 sysctl (see parseProcArgs2).
func processArgs(pid int) (procArgs, error) {
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return procArgs{}, fmt.Errorf("reading the arguments of process %d: %w", pid, err)
	}
	return parseProcArgs2(data)
}

// processEnviron returns the environment pid started with.
func processEnviron(pid int) ([]string, error) {
	p, err := processArgs(pid)
	return p.env, err
}

// processProgram returns the executable pid runs and its argv.
func processProgram(pid int) (string, []string, error) {
	p, err := processArgs(pid)
	return p.exe, p.argv, err
}
