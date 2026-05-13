//go:build windows

package sysinfo

import (
	"os"
	"os/exec"
)

// detectCurrentShell detects the current shell on Windows.
// It returns the shell name and full path.
func detectCurrentShell() (name, path string) {
	// 1. Check for PowerShell (PSModulePath is set in any PowerShell session).
	if os.Getenv("PSModulePath") != "" {
		// Prefer PowerShell Core (pwsh) over Windows PowerShell.
		if p, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh", p
		}
		if p, err := exec.LookPath("powershell"); err == nil {
			return "powershell", p
		}
	}

	// 2. Check for Git Bash / MSYS2.
	if os.Getenv("MSYSTEM") != "" {
		p, _ := exec.LookPath("bash")
		return "bash", p
	}

	// 3. Default to cmd.
	p, _ := exec.LookPath("cmd")
	return "cmd", p
}
