package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// createMockBinaries writes minimal executable scripts into dir for each
// given tool name and returns the directory (which should be set as PATH).
// On Windows, it creates .exe files so exec.LookPath can find them.
func createMockBinaries(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		filename := name
		if runtime.GOOS == "windows" {
			filename = name + ".exe"
		}
		p := filepath.Join(dir, filename)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("creating mock binary %s: %v", name, err)
		}
	}
}

func TestDetectPackageManagers_Debian(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "apt-get")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "debian"}
	detectPackageManagers(info)

	if info.PackageManager != "apt" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "apt")
	}
}

func TestDetectPackageManagers_RHEL_Dnf(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "dnf")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "rhel"}
	detectPackageManagers(info)

	if info.PackageManager != "dnf" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "dnf")
	}
}

func TestDetectPackageManagers_RHEL_Yum(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "yum")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "rhel"}
	detectPackageManagers(info)

	if info.PackageManager != "yum" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "yum")
	}
}

func TestDetectPackageManagers_Arch(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "pacman", "paru")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "arch"}
	detectPackageManagers(info)

	if info.PackageManager != "pacman" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "pacman")
	}
	if !slices.Contains(info.AltPkgManagers, "paru") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "paru")
	}
}

func TestDetectPackageManagers_NixOS(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "nixos"}
	detectPackageManagers(info)

	if info.PackageManager != "nix" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "nix")
	}
	if !info.HasNix {
		t.Error("HasNix = false, want true")
	}
}

func TestDetectPackageManagers_MacOS(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "brew")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "macos"}
	detectPackageManagers(info)

	if info.PackageManager != "brew" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "brew")
	}
}

func TestDetectPackageManagers_Windows(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "winget", "scoop")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "windows"}
	detectPackageManagers(info)

	if info.PackageManager != "winget" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "winget")
	}
	if !slices.Contains(info.AltPkgManagers, "scoop") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "scoop")
	}
}

func TestDetectPackageManagers_NixOnNonNixOS(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "apt-get", "nix")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "debian"}
	detectPackageManagers(info)

	if info.PackageManager != "apt" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "apt")
	}
	if !info.HasNix {
		t.Error("HasNix = false, want true")
	}
	if !slices.Contains(info.AltPkgManagers, "nix") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "nix")
	}
}

func TestDetectPackageManagers_Suse(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "suse"}
	detectPackageManagers(info)

	if info.PackageManager != "zypper" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "zypper")
	}
}

func TestDetectPackageManagers_Alpine(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "alpine"}
	detectPackageManagers(info)

	if info.PackageManager != "apk" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "apk")
	}
}

func TestDetectPackageManagers_Void(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "void"}
	detectPackageManagers(info)

	if info.PackageManager != "xbps" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "xbps")
	}
}

func TestDetectPackageManagers_Gentoo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "gentoo"}
	detectPackageManagers(info)

	if info.PackageManager != "emerge" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "emerge")
	}
}

func TestDetectPackageManagers_BrewOnNonMacOS(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "brew")
	t.Setenv("PATH", tmp)

	// Use a Linux family so the "non-macOS brew" detection fires.
	info := &OSInfo{Family: "debian"}
	detectPackageManagers(info)

	if !info.HasHomebrew {
		t.Error("HasHomebrew = false, want true for brew on non-macOS")
	}
	if !slices.Contains(info.AltPkgManagers, "brew") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "brew")
	}
}

func TestDetectPackageManagers_Windows_ScoopOnly(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "scoop")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "windows"}
	detectPackageManagers(info)

	if info.PackageManager != "scoop" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "scoop")
	}
}

func TestDetectPackageManagers_Windows_ChocoOnly(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "choco")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "windows"}
	detectPackageManagers(info)

	if info.PackageManager != "choco" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "choco")
	}
}

func TestDetectPackageManagers_Windows_AllThree(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "winget", "scoop", "choco")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "windows"}
	detectPackageManagers(info)

	if info.PackageManager != "winget" {
		t.Errorf("PackageManager = %q, want %q (winget should win)", info.PackageManager, "winget")
	}
	if !slices.Contains(info.AltPkgManagers, "scoop") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "scoop")
	}
	if !slices.Contains(info.AltPkgManagers, "choco") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "choco")
	}
}

func TestDetectPackageManagers_Windows_NoneAvailable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "windows"}
	detectPackageManagers(info)

	if info.PackageManager != "" {
		t.Errorf("PackageManager = %q, want empty when nothing is installed", info.PackageManager)
	}
}

func TestDetectPackageManagers_MacOS_NoBrew(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "macos"}
	detectPackageManagers(info)

	if info.PackageManager != "" {
		t.Errorf("PackageManager = %q, want empty when brew is not installed", info.PackageManager)
	}
}

func TestDetectPackageManagers_MacOS_WithPort(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "brew", "port")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "macos"}
	detectPackageManagers(info)

	if info.PackageManager != "brew" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "brew")
	}
	if !slices.Contains(info.AltPkgManagers, "port") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "port")
	}
}

func TestDetectPackageManagers_Arch_WithYay(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "pacman", "yay")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "arch"}
	detectPackageManagers(info)

	if info.PackageManager != "pacman" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "pacman")
	}
	if !slices.Contains(info.AltPkgManagers, "yay") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "yay")
	}
}

func TestDetectPackageManagers_Arch_BothHelpers(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "pacman", "paru", "yay")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "arch"}
	detectPackageManagers(info)

	if !slices.Contains(info.AltPkgManagers, "paru") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "paru")
	}
	if !slices.Contains(info.AltPkgManagers, "yay") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "yay")
	}
}

func TestDetectPackageManagers_Debian_WithSnap(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "apt-get", "snap")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "debian"}
	detectPackageManagers(info)

	if info.PackageManager != "apt" {
		t.Errorf("PackageManager = %q, want %q", info.PackageManager, "apt")
	}
	if !slices.Contains(info.AltPkgManagers, "snap") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "snap")
	}
}

func TestDetectPackageManagers_Debian_WithFlatpak(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "apt-get", "flatpak")
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "debian"}
	detectPackageManagers(info)

	if !slices.Contains(info.AltPkgManagers, "flatpak") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "flatpak")
	}
}

func TestDetectPackageManagers_UnknownFamily(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: "unknown"}
	detectPackageManagers(info)

	if info.PackageManager != "" {
		t.Errorf("PackageManager = %q, want empty for unknown family", info.PackageManager)
	}
}

func TestDetectPackageManagers_EmptyFamily(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{Family: ""}
	detectPackageManagers(info)

	if info.PackageManager != "" {
		t.Errorf("PackageManager = %q, want empty for empty family", info.PackageManager)
	}
}

func TestLookPathAppend_NoneFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	info := &OSInfo{}
	lookPathAppend(info, "nonexistent1", "nonexistent2")

	if len(info.AltPkgManagers) != 0 {
		t.Errorf("AltPkgManagers = %v, want empty", info.AltPkgManagers)
	}
}

func TestLookPathAppend_SomeFound(t *testing.T) {
	tmp := t.TempDir()
	createMockBinaries(t, tmp, "exists1")
	t.Setenv("PATH", tmp)

	info := &OSInfo{}
	lookPathAppend(info, "exists1", "nonexistent")

	if len(info.AltPkgManagers) != 1 {
		t.Errorf("AltPkgManagers = %v, want exactly 1 element", info.AltPkgManagers)
	}
	if !slices.Contains(info.AltPkgManagers, "exists1") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "exists1")
	}
}
