package pkgmanager

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sysinfo"
)

func TestDetectPackageManagerWithRunner(t *testing.T) {
	mock := NewMockRunner()

	tests := []struct {
		name     string
		osInfo   *sysinfo.OSInfo
		wantName string
	}{
		{
			name:     "nil osInfo defaults to apt",
			osInfo:   nil,
			wantName: "apt",
		},
		{
			name:     "debian family",
			osInfo:   &sysinfo.OSInfo{Family: "debian", Distro: "ubuntu"},
			wantName: "apt",
		},
		{
			name:     "rhel family",
			osInfo:   &sysinfo.OSInfo{Family: "rhel", Distro: "centos"},
			wantName: "dnf",
		},
		{
			name:     "fedora family",
			osInfo:   &sysinfo.OSInfo{Family: "fedora", Distro: "fedora"},
			wantName: "dnf",
		},
		{
			name:     "arch family",
			osInfo:   &sysinfo.OSInfo{Family: "arch", Distro: "arch"},
			wantName: "pacman",
		},
		{
			name:     "suse family",
			osInfo:   &sysinfo.OSInfo{Family: "suse", Distro: "opensuse"},
			wantName: "zypper",
		},
		{
			name:     "alpine family",
			osInfo:   &sysinfo.OSInfo{Family: "alpine", Distro: "alpine"},
			wantName: "apk",
		},
		{
			name:     "void family",
			osInfo:   &sysinfo.OSInfo{Family: "void", Distro: "void"},
			wantName: "xbps",
		},
		{
			name:     "gentoo family",
			osInfo:   &sysinfo.OSInfo{Family: "gentoo", Distro: "gentoo"},
			wantName: "emerge",
		},
		{
			name:     "macos family with homebrew",
			osInfo:   &sysinfo.OSInfo{Family: "macos", OS: "darwin", HasHomebrew: true},
			wantName: "brew",
		},
		{
			name:     "macos family without explicit homebrew",
			osInfo:   &sysinfo.OSInfo{Family: "macos", OS: "darwin"},
			wantName: "brew",
		},
		{
			name:     "windows family",
			osInfo:   &sysinfo.OSInfo{Family: "windows", OS: "windows"},
			wantName: "winget",
		},
		{
			name:     "nix preferred when available",
			osInfo:   &sysinfo.OSInfo{Family: "debian", HasNix: true},
			wantName: "nix",
		},
		{
			name:     "homebrew on darwin preferred",
			osInfo:   &sysinfo.OSInfo{Family: "macos", OS: "darwin", HasHomebrew: true, HasNix: false},
			wantName: "brew",
		},
		{
			name:     "explicit package manager name apt-get",
			osInfo:   &sysinfo.OSInfo{PackageManager: "apt-get"},
			wantName: "apt",
		},
		{
			name:     "explicit package manager name yum",
			osInfo:   &sysinfo.OSInfo{PackageManager: "yum"},
			wantName: "dnf",
		},
		{
			name:     "explicit package manager name homebrew",
			osInfo:   &sysinfo.OSInfo{PackageManager: "homebrew"},
			wantName: "brew",
		},
		{
			name:     "explicit package manager name chocolatey",
			osInfo:   &sysinfo.OSInfo{PackageManager: "chocolatey"},
			wantName: "choco",
		},
		{
			name:     "unknown family with homebrew",
			osInfo:   &sysinfo.OSInfo{Family: "unknown", HasHomebrew: true},
			wantName: "brew",
		},
		{
			name:     "unknown family without homebrew",
			osInfo:   &sysinfo.OSInfo{Family: "unknown"},
			wantName: "apt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := DetectPackageManagerWithRunner(tt.osInfo, mock)
			if pm.Name() != tt.wantName {
				t.Errorf("DetectPackageManagerWithRunner(%+v) = %q, want %q", tt.osInfo, pm.Name(), tt.wantName)
			}
		})
	}
}

func TestDetectPackageManagerNixOSMode(t *testing.T) {
	mock := NewMockRunner()
	osInfo := &sysinfo.OSInfo{
		Family: "nixos",
		Distro: "nixos",
		HasNix: true,
	}

	pm := DetectPackageManagerWithRunner(osInfo, mock)
	if pm.Name() != "nix" {
		t.Fatalf("expected nix, got %s", pm.Name())
	}

	// The NixOS Nix implementation should return an error on Install.
	err := pm.Install(context.TODO(), "git")
	if err == nil {
		t.Fatal("expected error for NixOS install")
	}
}

func TestDetectPackageManagerNonNil(t *testing.T) {
	// DetectPackageManager (without runner) should not return nil.
	pm := DetectPackageManager(&sysinfo.OSInfo{Family: "debian"})
	if pm == nil {
		t.Fatal("DetectPackageManager should not return nil")
	}
}
