//go:build linux

package sysinfo

import (
	"testing"
	"time"
)

func TestDetectOS_Linux(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.OS != "linux" {
		t.Errorf("DetectOS().OS = %q, want \"linux\"", info.OS)
	}
	if info.Arch == "" {
		t.Error("DetectOS().Arch is empty")
	}
	if info.Family == "" {
		t.Error("DetectOS().Family is empty")
	}
	if info.Shell == "" {
		t.Error("DetectOS().Shell is empty")
	}
}

func TestDetectOS_Performance(t *testing.T) {
	start := time.Now()
	_ = DetectOS()
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("DetectOS() took %v, want <100ms", elapsed)
	}
}

func TestDetectOS_FamilyIsValid(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	validFamilies := map[string]bool{
		"debian":  true,
		"rhel":    true,
		"arch":    true,
		"suse":    true,
		"alpine":  true,
		"void":    true,
		"gentoo":  true,
		"nixos":   true,
		"unknown": true,
	}
	if !validFamilies[info.Family] {
		t.Errorf("DetectOS().Family = %q, want one of debian/rhel/arch/suse/alpine/void/gentoo/nixos/unknown", info.Family)
	}
}

func TestDetectOS_ShellIsDetected(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.Shell == "" {
		t.Error("DetectOS().Shell is empty")
	}
	if info.ShellRCFile == "" {
		t.Error("DetectOS().ShellRCFile is empty")
	}
}

func TestDetectOS_KernelIsSet(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.Kernel == "" {
		t.Error("DetectOS().Kernel is empty on Linux")
	}
}

func TestDetectOS_DistroIsSet(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.Distro == "" {
		t.Error("DetectOS().Distro is empty on Linux")
	}
}

func TestDetectOS_PrettyNameIsSet(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.PrettyName == "" {
		t.Error("DetectOS().PrettyName is empty on Linux")
	}
}

func TestDetectOS_PackageManagerIsSet(t *testing.T) {
	t.Parallel()
	info := DetectOS()
	if info.PackageManager == "" {
		t.Error("DetectOS().PackageManager is empty on Linux")
	}
}

func TestDetectOS_Idempotent(t *testing.T) {
	t.Parallel()
	info1 := DetectOS()
	info2 := DetectOS()

	if info1.OS != info2.OS {
		t.Errorf("OS mismatch: %q vs %q", info1.OS, info2.OS)
	}
	if info1.Arch != info2.Arch {
		t.Errorf("Arch mismatch: %q vs %q", info1.Arch, info2.Arch)
	}
	if info1.Family != info2.Family {
		t.Errorf("Family mismatch: %q vs %q", info1.Family, info2.Family)
	}
	if info1.Distro != info2.Distro {
		t.Errorf("Distro mismatch: %q vs %q", info1.Distro, info2.Distro)
	}
	if info1.PackageManager != info2.PackageManager {
		t.Errorf("PackageManager mismatch: %q vs %q", info1.PackageManager, info2.PackageManager)
	}
}
