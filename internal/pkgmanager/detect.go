package pkgmanager

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// familyManagers maps an OS family to its native package manager. It drives
// both detection (DetectPackageManager) and name resolution (PackageFor), so
// the family-specific package names are only applied when the detected
// manager is the one those names belong to.
var familyManagers = map[string]string{
	"debian":  "apt",
	"rhel":    "dnf",
	"fedora":  "dnf",
	"arch":    "pacman",
	"suse":    "zypper",
	"alpine":  "apk",
	"void":    "xbps",
	"gentoo":  "emerge",
	"macos":   "brew", // Homebrew is the standard macOS PM
	"windows": "winget",
}

// managerAliases maps alternate package manager names (as reported by
// sysinfo or users) to the canonical Name() of the implementation.
var managerAliases = map[string]string{
	"apt-get":      "apt",
	"yum":          "dnf",
	"xbps-install": "xbps",
	"portage":      "emerge",
	"homebrew":     "brew",
	"chocolatey":   "choco",
}

// canonicalManager returns the canonical name for a package manager name.
func canonicalManager(name string) string {
	if c, ok := managerAliases[name]; ok {
		return c
	}
	return name
}

// DetectPackageManager returns the appropriate PackageManager for the given OS info.
// It uses DefaultRunner() for command execution.
func DetectPackageManager(osInfo *sysinfo.OSInfo) PackageManager {
	return DetectPackageManagerWithRunner(osInfo, nil)
}

// DetectPackageManagerWithRunner returns the appropriate PackageManager for the
// given OS info, using the provided runner for command execution.
// If runner is nil, DefaultRunner() is used.
//
// The returned manager may differ from osInfo.PackageManager (Nix is preferred
// whenever it is installed), so callers must resolve package names and build
// install commands from the returned manager (PackageFor, InstallCommand),
// never from osInfo.PackageManager.
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

	if name, ok := familyManagers[osInfo.Family]; ok {
		return managerByName(name, runner)
	}
	// Try Homebrew on Linux as fallback if available.
	if osInfo.HasHomebrew {
		return NewBrew(runner)
	}
	return NewApt(runner) // safe default
}

// managerByName returns a PackageManager given its name string.
func managerByName(name string, runner CommandRunner) PackageManager {
	switch canonicalManager(name) {
	case "apt":
		return NewApt(runner)
	case "dnf":
		return NewDnf(runner)
	case "pacman":
		return NewPacman(runner)
	case "zypper":
		return NewZypper(runner)
	case "apk":
		return NewApk(runner)
	case "xbps":
		return NewXbps(runner)
	case "emerge":
		return NewEmerge(runner)
	case "brew":
		return NewBrew(runner)
	case "nix":
		return NewNix(runner, false)
	case "winget":
		return NewWinget(runner)
	case "scoop":
		return NewScoop(runner)
	case "choco":
		return NewChoco(runner)
	default:
		return NewApt(runner)
	}
}
