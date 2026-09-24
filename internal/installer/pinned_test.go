package installer_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

func TestIsExactSemver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"2.1.273", true},
		{"v1.0.0", true},
		{"1.2.3-rc.1", true},
		{"1.2.3+build.5", true},
		{"", false},
		{"latest", false},
		{"stable", false},
		{"^2.1.0", false},
		{"~2.1.0", false},
		{"2.1", false},
		{"2.x", false},
		{">=2.0.0", false},
		{"2.1.273 || 2.1.274", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			if got := installer.IsExactSemver(tt.version); got != tt.want {
				t.Errorf("IsExactSemver(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestReleaseCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 7, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	if got, want := installer.ReleaseCutoff(now, installer.NpmMinReleaseAge), "2026-09-21T11:30:00Z"; got != want {
		t.Errorf("ReleaseCutoff = %q, want %q", got, want)
	}
}

// TestNpmGlobalInstallCmd covers F110: qsdev's own global npm installs name
// an exact release, are age-gated with --before, and run no lifecycle
// scripts unless the package is explicitly allowed them.
func TestNpmGlobalInstallCmd(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	const cutoff = "--before=2026-09-21T12:00:00Z"

	tests := []struct {
		name    string
		pkg     installer.NpmPackage
		want    []string
		wantErr error
	}{
		{
			name: "scripts disabled by default",
			pkg:  installer.NpmPackage{Name: "@scope/tool", Version: "1.2.3"},
			want: []string{"npm", "install", "-g", "--ignore-scripts", cutoff, "@scope/tool@1.2.3"},
		},
		{
			name: "scripts allowed",
			pkg:  installer.NpmPackage{Name: "@scope/tool", Version: "1.2.3", RunInstallScripts: true},
			want: []string{"npm", "install", "-g", cutoff, "@scope/tool@1.2.3"},
		},
		{
			name:    "dist-tag refused",
			pkg:     installer.NpmPackage{Name: "tool", Version: "latest"},
			wantErr: installer.ErrUnpinned,
		},
		{
			name:    "range refused",
			pkg:     installer.NpmPackage{Name: "tool", Version: "^1.2.3"},
			wantErr: installer.ErrUnpinned,
		},
		{
			name:    "missing version refused",
			pkg:     installer.NpmPackage{Name: "tool"},
			wantErr: installer.ErrUnpinned,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := installer.NpmGlobalInstallCmd(tt.pkg, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NpmGlobalInstallCmd: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("command = %q, want %q", got, tt.want)
			}
		})
	}

	for _, name := range []string{"", "--registry=https://evil.example/x", "-g"} {
		t.Run("bad name refused/"+name, func(t *testing.T) {
			t.Parallel()
			if _, err := installer.NpmGlobalInstallCmd(installer.NpmPackage{Name: name, Version: "1.2.3"}, now); err == nil {
				t.Errorf("NpmGlobalInstallCmd with name %q succeeded, want an error", name)
			}
		})
	}
}

// testRev is a full nixpkgs commit hash used by the flake-pin tests.
const testRev = "d233902339c02a9c334e7e593de68855ad26c4cb"

// TestIsPinnedFlakeRef covers F289: only a flake reference naming one commit
// counts as pinned; registry names, branches and tags follow whatever they
// point at when the install runs.
func TestIsPinnedFlakeRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{"github rev path", "github:NixOS/nixpkgs/" + testRev, true},
		{"gitlab rev path", "gitlab:owner/repo/" + testRev, true},
		{"github rev query", "github:NixOS/nixpkgs?rev=" + testRev, true},
		{"git url rev query", "git+https://example.com/nixpkgs.git?ref=main&rev=" + testRev, true},
		{"registry name", "nixpkgs", false},
		{"indirect registry ref", "flake:nixpkgs", false},
		{"branch", "github:NixOS/nixpkgs/nixpkgs-unstable", false},
		{"tag", "github:NixOS/nixpkgs/26.05", false},
		{"no ref", "github:NixOS/nixpkgs", false},
		{"short rev", "github:NixOS/nixpkgs/d2339023", false},
		{"uppercase rev", "github:NixOS/nixpkgs/D233902339C02A9C334E7E593DE68855AD26C4CB", false},
		{"extra path segment", "github:NixOS/nixpkgs/" + testRev + "/x", false},
		{"empty owner", "github:/nixpkgs/" + testRev, false},
		{"short rev query", "git+https://example.com/x.git?rev=d2339023", false},
		{"attribute embedded", "github:NixOS/nixpkgs/" + testRev + "#devenv", false},
		{"whitespace", "github:NixOS/nixpkgs/" + testRev + " nixpkgs", false},
		{"empty", "", false},
		// A rev query only pins a fetcher that checks the commit out.
		{"indirect registry rev query", "nixpkgs?rev=" + testRev, false},
		{"indirect flake: rev query", "flake:nixpkgs?rev=" + testRev, false},
		{"path rev query", "path:/tmp/nixpkgs?rev=" + testRev, false},
		{"tarball rev query", "tarball+https://example.com/nixpkgs.tar.gz?rev=" + testRev, false},
		{"https tarball rev query", "https://example.com/nixpkgs.tar.gz?rev=" + testRev, false},
		{"file rev query", "file+https://example.com/nixpkgs.tar.gz?rev=" + testRev, false},
		{"github branch path with rev query", "github:NixOS/nixpkgs/nixos-26.05/x?rev=" + testRev, false},
		{"git without rev", "git+https://example.com/nixpkgs.git?ref=main", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := installer.IsPinnedFlakeRef(tt.ref); got != tt.want {
				t.Errorf("IsPinnedFlakeRef(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

// TestNixProfileInstallCmd covers F289: the command installs from the pinned
// flake, overrides accept-flake-config to false, never passes
// --accept-flake-config, and refuses unpinned or malformed input.
func TestNixProfileInstallCmd(t *testing.T) {
	t.Parallel()

	pinned := "github:NixOS/nixpkgs/" + testRev
	tests := []struct {
		name     string
		pkg      installer.NixPackage
		want     []string
		wantErr  error // nil with wantFail: any error
		wantFail bool
	}{
		{
			name: "pinned",
			pkg:  installer.NixPackage{Flake: pinned, Attribute: "devenv"},
			want: []string{"nix", "profile", "install", "--option", "accept-flake-config", "false", pinned + "#devenv"},
		},
		{
			name: "nested attribute",
			pkg:  installer.NixPackage{Flake: pinned, Attribute: "python3Packages.black"},
			want: []string{"nix", "profile", "install", "--option", "accept-flake-config", "false", pinned + "#python3Packages.black"},
		},
		{name: "registry name", pkg: installer.NixPackage{Flake: "nixpkgs", Attribute: "devenv"}, wantErr: installer.ErrUnpinned, wantFail: true},
		{name: "branch", pkg: installer.NixPackage{Flake: "github:NixOS/nixpkgs/nixos-26.05", Attribute: "devenv"}, wantErr: installer.ErrUnpinned, wantFail: true},
		{name: "option-like flake", pkg: installer.NixPackage{Flake: "--accept-flake-config", Attribute: "devenv"}, wantFail: true},
		{name: "empty attribute", pkg: installer.NixPackage{Flake: pinned}, wantFail: true},
		{name: "attribute with selector", pkg: installer.NixPackage{Flake: pinned, Attribute: "devenv^out"}, wantFail: true},
		{name: "attribute with space", pkg: installer.NixPackage{Flake: pinned, Attribute: "devenv --impure"}, wantFail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := installer.NixProfileInstallCmd(tt.pkg)
			if tt.wantFail {
				if err == nil {
					t.Fatalf("NixProfileInstallCmd(%+v) = %q, want an error", tt.pkg, got)
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NixProfileInstallCmd: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("NixProfileInstallCmd = %q, want %q", got, tt.want)
			}
			if slices.Contains(got, "--accept-flake-config") {
				t.Errorf("command %q accepts the flake's nixConfig", got)
			}
		})
	}
}
