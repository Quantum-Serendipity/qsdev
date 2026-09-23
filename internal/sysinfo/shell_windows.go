//go:build windows

package sysinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// detectCurrentShell detects the current shell on Windows.
// It returns the shell name and full path.
func detectCurrentShell() (name, path string) {
	name = shellFromImage(parentProcessImage())
	if name == "" {
		name = shellFromEnv(os.Getenv)
	}
	return name, lookupShell(name)
}

// shellFromImage maps the parent process's executable name to a shell name,
// or "" when the parent is not a recognized shell (e.g. a terminal, an IDE or
// a build tool launched qsdev directly).
func shellFromImage(image string) string {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(image), filepath.Ext(image)))
	if base == "cmd" || isKnownShell(base) {
		return base
	}
	return ""
}

// shellFromEnv infers the shell from environment variables when the parent
// process is not a shell. PSModulePath is NOT a PowerShell signal on its own:
// Windows defines it machine-wide, so cmd.exe and Git Bash inherit it too.
// PowerShell prepends the per-user module directory (Documents\PowerShell or
// Documents\WindowsPowerShell) at startup, which only a PowerShell session (or
// a process it launched) carries.
func shellFromEnv(getenv func(string) string) string {
	// Git Bash / MSYS2 / Cygwin sessions.
	if getenv("MSYSTEM") != "" {
		return "bash"
	}
	if shell := getenv("SHELL"); shell != "" {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(filepath.FromSlash(shell)), ".exe"))
		if isKnownShell(base) {
			return base
		}
	}

	modulePath := strings.ToLower(getenv("PSModulePath"))
	switch {
	case strings.Contains(modulePath, `\documents\powershell\modules`):
		return "pwsh"
	case strings.Contains(modulePath, `\documents\windowspowershell\modules`):
		return "powershell"
	}
	return "cmd"
}

// lookupShell resolves a shell name to its executable path, preferring
// PowerShell Core when the session is PowerShell but only Windows PowerShell
// is installed (or vice versa).
func lookupShell(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	switch name {
	case "pwsh":
		p, _ := exec.LookPath("powershell")
		return p
	case "powershell":
		p, _ := exec.LookPath("pwsh")
		return p
	}
	return ""
}

// parentProcessImage returns the executable file name of this process's
// parent, or "" if it cannot be determined.
func parentProcessImage() string {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()

	self := uint32(os.Getpid())
	var parent uint32
	images := make(map[uint32]string)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		images[entry.ProcessID] = windows.UTF16ToString(entry.ExeFile[:])
		if entry.ProcessID == self {
			parent = entry.ParentProcessID
		}
	}
	if parent == 0 {
		return ""
	}
	return images[parent]
}
