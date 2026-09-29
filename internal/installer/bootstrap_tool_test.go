package installer_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

// TestBootstrapToolInstallCmd_MissingEntry covers F110: a tool the catalog
// does not pin is refused rather than installed at whatever release the
// registry serves.
func TestBootstrapToolInstallCmd_MissingEntry(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	_, err = installer.BootstrapToolInstallCmd(cat, "no-such-tool", time.Now())
	if !errors.Is(err, installer.ErrBootstrapPin) {
		t.Fatalf("error = %v, want %v", err, installer.ErrBootstrapPin)
	}
}

// TestBootstrapToolInstallCmd_NixProfile covers F289: the embedded devenv
// and direnv pins install from nixpkgs pinned to a commit.
func TestBootstrapToolInstallCmd_NixProfile(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	for _, name := range []string{catalog.BootstrapToolDevenv, catalog.BootstrapToolDirenv} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			def, ok := cat.BootstrapTool(name)
			if !ok {
				t.Fatalf("embedded catalog has no bootstrap_tools.%s entry", name)
			}
			if def.InstallMethod != installer.BootstrapInstallNixProfile {
				t.Fatalf("install_method = %q, want %q", def.InstallMethod, installer.BootstrapInstallNixProfile)
			}
			cmd, err := installer.BootstrapToolInstallCmd(cat, name, time.Now())
			if err != nil {
				t.Fatalf("BootstrapToolInstallCmd: %v", err)
			}
			want, err := installer.NixProfileInstallCmd(installer.NixPackage{Flake: def.Flake, Attribute: def.PackageName})
			if err != nil {
				t.Fatalf("NixProfileInstallCmd: %v", err)
			}
			if !slices.Equal(cmd, want) {
				t.Errorf("BootstrapToolInstallCmd = %q, want %q", cmd, want)
			}
		})
	}
}
