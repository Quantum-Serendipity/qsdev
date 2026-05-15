package pkgmanager

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// DetectPackageManager returns the appropriate PackageManager for the given OS info.
// It uses DefaultRunner() for command execution.
func DetectPackageManager(osInfo *sysinfo.OSInfo) PackageManager {
	return DetectPackageManagerWithRunner(osInfo, nil)
}

// DetectPackageManagerWithRunner returns the appropriate PackageManager for the
// given OS info, using the provided runner for command execution.
// If runner is nil, DefaultRunner() is used.
func DetectPackageManagerWithRunner(osInfo *sysinfo.OSInfo, runner CommandRunner) PackageManager {
	if osInfo == nil {
		return NewApt(runner) // safe fallback
	}

	// Prefer Nix if available.
	if osInfo.HasNix {
		isNixOS := osInfo.Distro == "nixos" || osInfo.Family == "nixos"
		return NewNix(runner, isNixOS)
	}

	// Prefer Homebrew on macOS, or if explicitly detected on Linux.
	if osInfo.HasHomebrew && (osInfo.OS == "darwin" || osInfo.Family == "macos") {
		return NewBrew(runner)
	}

	// Switch on package manager name if explicitly set.
	if osInfo.PackageManager != "" {
		return managerByName(osInfo.PackageManager, runner)
	}

	// Switch on family.
	switch osInfo.Family {
	case "debian":
		return NewApt(runner)
	case "rhel", "fedora":
		return NewDnf(runner)
	case "arch":
		return NewPacman(runner)
	case "suse":
		return NewZypper(runner)
	case "alpine":
		return NewApk(runner)
	case "void":
		return NewXbps(runner)
	case "gentoo":
		return NewEmerge(runner)
	case "macos":
		if osInfo.HasHomebrew {
			return NewBrew(runner)
		}
		return NewBrew(runner) // Homebrew is the standard macOS PM
	case "windows":
		return NewWinget(runner)
	default:
		// Try Homebrew on Linux as fallback if available.
		if osInfo.HasHomebrew {
			return NewBrew(runner)
		}
		return NewApt(runner) // safe default
	}
}

// managerByName returns a PackageManager given its name string.
func managerByName(name string, runner CommandRunner) PackageManager {
	switch name {
	case "apt", "apt-get":
		return NewApt(runner)
	case "dnf", "yum":
		return NewDnf(runner)
	case "pacman":
		return NewPacman(runner)
	case "zypper":
		return NewZypper(runner)
	case "apk":
		return NewApk(runner)
	case "xbps", "xbps-install":
		return NewXbps(runner)
	case "emerge", "portage":
		return NewEmerge(runner)
	case "brew", "homebrew":
		return NewBrew(runner)
	case "nix":
		return NewNix(runner, false)
	case "winget":
		return NewWinget(runner)
	case "scoop":
		return NewScoop(runner)
	case "choco", "chocolatey":
		return NewChoco(runner)
	default:
		return NewApt(runner)
	}
}
