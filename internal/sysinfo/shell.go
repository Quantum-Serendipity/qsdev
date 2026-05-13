package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
)

// detectShell populates the shell-related fields of the given OSInfo.
func detectShell(info *OSInfo) {
	name, path := detectCurrentShell()
	info.Shell = name
	info.ShellPath = path
	info.ShellRCFile = resolveShellRCFile(name)
}

// isKnownShell reports whether name is a recognized shell.
func isKnownShell(name string) bool {
	switch name {
	case "bash", "zsh", "fish", "pwsh", "powershell", "nu", "sh", "dash", "ksh", "tcsh", "csh":
		return true
	}
	return false
}

// resolveShellRCFile returns the path to the shell's interactive RC file.
func resolveShellRCFile(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	case "pwsh":
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	case "powershell":
		return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	case "nu":
		return filepath.Join(home, ".config", "nushell", "config.nu")
	case "cmd":
		return ""
	default:
		return ""
	}
}
