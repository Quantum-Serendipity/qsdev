//go:build linux

package sysinfo

import (
	"testing"
	"time"
)

func TestDetectOS_Linux(t *testing.T) {
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
	if elapsed > 50*time.Millisecond {
		t.Errorf("DetectOS() took %v, want <50ms", elapsed)
	}
}

func TestDetectOS_FamilyIsValid(t *testing.T) {
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
	info := DetectOS()
	if info.Shell == "" {
		t.Error("DetectOS().Shell is empty")
	}
	if info.ShellRCFile == "" {
		t.Error("DetectOS().ShellRCFile is empty")
	}
}

func TestDetectOS_KernelIsSet(t *testing.T) {
	info := DetectOS()
	if info.Kernel == "" {
		t.Error("DetectOS().Kernel is empty on Linux")
	}
}
