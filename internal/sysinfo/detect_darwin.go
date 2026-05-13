//go:build darwin

package sysinfo

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// detectPlatform populates macOS-specific fields in info using sw_vers,
// uname, and probing for developer tools and Homebrew.
func detectPlatform(info *OSInfo) {
	info.Family = "macos"
	info.Distro = "macos"

	version := runTrimmed("sw_vers", "-productVersion")
	productName := runTrimmed("sw_vers", "-productName")

	info.Version = version
	if productName != "" && version != "" {
		info.PrettyName = productName + " " + version
	} else if productName != "" {
		info.PrettyName = productName
	}

	info.Kernel = runTrimmed("uname", "-r")

	detectRosetta(info)
	detectXcodeCLT(info)
	detectHomebrew(info)
}

// detectRosetta checks whether the current process is running under Rosetta
// translation. This is only meaningful when the compiled binary is amd64
// running on Apple Silicon hardware.
func detectRosetta(info *OSInfo) {
	if runtime.GOARCH != "amd64" {
		return
	}
	out := runTrimmed("sysctl", "-n", "sysctl.proc_translated")
	if out == "1" {
		info.IsRosetta = true
	}
}

// detectXcodeCLT checks whether the Xcode Command Line Tools are installed.
func detectXcodeCLT(info *OSInfo) {
	cmd := exec.Command("xcode-select", "-p")
	if err := cmd.Run(); err == nil {
		info.XcodeCLT = true
	}
}

// detectHomebrew probes for Homebrew and determines its prefix.
func detectHomebrew(info *OSInfo) {
	// Check PATH first.
	if _, err := exec.LookPath("brew"); err == nil {
		info.HasHomebrew = true
	}

	// Check well-known install locations.
	switch {
	case fileExistsSimple("/opt/homebrew/bin/brew"):
		info.HasHomebrew = true
		info.HomebrewPrefix = "/opt/homebrew"
	case fileExistsSimple("/usr/local/bin/brew"):
		info.HasHomebrew = true
		info.HomebrewPrefix = "/usr/local"
	}
}

// runTrimmed executes a command and returns its stdout with leading/trailing
// whitespace removed. On any error it returns an empty string.
func runTrimmed(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// fileExistsSimple is a minimal existence check that avoids importing fileutil
// on darwin (where only this file is compiled).
func fileExistsSimple(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}
