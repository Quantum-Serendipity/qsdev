package sysinfo

import (
	"os/exec"
	"runtime"
)

// DetectOS returns a fully populated OSInfo for the current system.
func DetectOS() *OSInfo {
	info := &OSInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	detectPlatform(info)
	detectShell(info)
	detectPackageManagers(info)
	return info
}

// detectPackageManagers identifies the primary and alternate package managers
// available on the system based on the OS family and PATH.
func detectPackageManagers(info *OSInfo) {
	switch info.Family {
	case "debian":
		info.PackageManager = "apt"
		lookPathAppend(info, "snap", "flatpak")

	case "rhel":
		if lookPathExists("dnf") {
			info.PackageManager = "dnf"
		} else {
			info.PackageManager = "yum"
		}

	case "arch":
		info.PackageManager = "pacman"
		lookPathAppend(info, "paru", "yay")

	case "suse":
		info.PackageManager = "zypper"

	case "alpine":
		info.PackageManager = "apk"

	case "void":
		info.PackageManager = "xbps"

	case "gentoo":
		info.PackageManager = "emerge"

	case "nixos":
		info.PackageManager = "nix"
		info.HasNix = true

	case "macos":
		if lookPathExists("brew") {
			info.PackageManager = "brew"
		}
		lookPathAppend(info, "port")

	case "windows":
		// Priority order: winget > scoop > choco
		for _, mgr := range []string{"winget", "scoop", "choco"} {
			if lookPathExists(mgr) {
				if info.PackageManager == "" {
					info.PackageManager = mgr
				} else {
					info.AltPkgManagers = append(info.AltPkgManagers, mgr)
				}
			}
		}
	}

	// Universal: detect nix on non-NixOS systems.
	if info.Family != "nixos" && lookPathExists("nix") {
		info.HasNix = true
		info.AltPkgManagers = append(info.AltPkgManagers, "nix")
	}

	// Universal: detect homebrew on non-macOS systems.
	if info.Family != "macos" && lookPathExists("brew") {
		info.HasHomebrew = true
		info.AltPkgManagers = append(info.AltPkgManagers, "brew")
	}
}

// lookPathExists reports whether name is found on the system PATH.
func lookPathExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// lookPathAppend checks each name on PATH and appends any found to
// info.AltPkgManagers.
func lookPathAppend(info *OSInfo, names ...string) {
	for _, name := range names {
		if lookPathExists(name) {
			info.AltPkgManagers = append(info.AltPkgManagers, name)
		}
	}
}
