//go:build windows

package sysinfo

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// versionRe extracts the version number from the output of "cmd /c ver",
// e.g. "Microsoft Windows [Version 10.0.22631]" → "10.0.22631".
var versionRe = regexp.MustCompile(`\[Version\s+([^\]]+)\]`)

// detectPlatform populates Windows-specific fields in info.
func detectPlatform(info *OSInfo) {
	info.Family = "windows"
	info.Distro = "windows"

	out, err := exec.Command("cmd", "/c", "ver").Output()
	if err == nil {
		full := strings.TrimSpace(string(out))
		info.PrettyName = full
		if m := versionRe.FindStringSubmatch(full); len(m) > 1 {
			info.Version = m[1]
		}
	}

	// Detect Windows Terminal.
	if os.Getenv("WT_SESSION") != "" {
		info.WindowsTerminal = true
	}

	// Detect Git Bash / MSYS2 environment.
	if os.Getenv("MSYSTEM") != "" {
		info.GitBash = true
	}
}
