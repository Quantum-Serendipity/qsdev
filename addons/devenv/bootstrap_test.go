package devenv_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

func TestInstallDevenvStep_NotNil(t *testing.T) {
	step := devenv.InstallDevenvStep()
	if step == nil {
		t.Fatal("InstallDevenvStep() returned nil")
	}
}

func TestInstallDirenvStep_NotNil(t *testing.T) {
	step := devenv.InstallDirenvStep()
	if step == nil {
		t.Fatal("InstallDirenvStep() returned nil")
	}
}

// TestNixToolSpec_EmbeddedPin covers F289: the bootstrap installs devenv and
// direnv from nixpkgs pinned to a commit, not through the mutable `nixpkgs`
// registry entry, and never with --accept-flake-config.
func TestNixToolSpec_EmbeddedPin(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	tests := []struct {
		name    string
		entry   string
		binary  string
		specFor func(*catalog.Catalog) (installer.ToolSpec, error)
	}{
		{"devenv", catalog.BootstrapToolDevenv, "devenv", devenv.ExportDevenvToolSpec},
		{"direnv", catalog.BootstrapToolDirenv, "direnv", devenv.ExportDirenvToolSpec},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			def, ok := cat.BootstrapTool(tt.entry)
			if !ok {
				t.Fatalf("embedded catalog has no bootstrap_tools.%s pin", tt.entry)
			}
			if !installer.IsPinnedFlakeRef(def.Flake) {
				t.Fatalf("bootstrap_tools.%s.flake = %q, want a flake pinned to a commit", tt.entry, def.Flake)
			}
			spec, err := tt.specFor(cat)
			if err != nil {
				t.Fatalf("toolSpec: %v", err)
			}
			want := []string{"nix", "profile", "install", "--option", "accept-flake-config", "false", def.Flake + "#" + tt.binary}
			if !slices.Equal(spec.InstallCmd, want) {
				t.Errorf("InstallCmd = %q, want %q", spec.InstallCmd, want)
			}
			if spec.Binary != tt.binary || spec.ManagerBinary != "nix" {
				t.Errorf("Binary, ManagerBinary = %q, %q; want %q, nix", spec.Binary, spec.ManagerBinary, tt.binary)
			}
		})
	}
}

// TestNixToolSpec_RefusesUnpinnedOverlay covers F289: a defaults overlay that
// points devenv at a mutable flake reference stops the install instead of
// installing whatever that reference resolves to.
func TestNixToolSpec_RefusesUnpinnedOverlay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flake string
	}{
		{"registry name", "nixpkgs"},
		{"indirect registry ref", "flake:nixpkgs"},
		{"branch", "github:NixOS/nixpkgs/nixpkgs-unstable"},
		{"no ref", "github:NixOS/nixpkgs"},
		{"short rev", "github:NixOS/nixpkgs/d2339023"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			overlay := filepath.Join(t.TempDir(), "defaults.yaml")
			body := "bootstrap_tools:\n    devenv:\n        flake: \"" + tt.flake + "\"\n"
			if err := os.WriteFile(overlay, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			cat, err := catalog.Load(catalog.WithOrgConfigFile(overlay))
			if err != nil {
				t.Fatalf("catalog.Load: %v", err)
			}
			_, err = devenv.ExportDevenvToolSpec(cat)
			if !errors.Is(err, installer.ErrBootstrapPin) || !errors.Is(err, installer.ErrUnpinned) {
				t.Errorf("toolSpec error = %v, want %v wrapping %v", err, installer.ErrBootstrapPin, installer.ErrUnpinned)
			}
		})
	}
}
