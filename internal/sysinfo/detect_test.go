package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

// createMockBinaries writes minimal executable scripts into dir for each
// given tool name and returns the directory (which should be set as PATH).
func createMockBinaries(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		p := filepath.Join(dir, name)
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
	if !containsStr(info.AltPkgManagers, "paru") {
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
	if !containsStr(info.AltPkgManagers, "scoop") {
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
	if !containsStr(info.AltPkgManagers, "nix") {
		t.Errorf("AltPkgManagers = %v, want it to include %q", info.AltPkgManagers, "nix")
	}
}

// containsStr reports whether slice contains s.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
