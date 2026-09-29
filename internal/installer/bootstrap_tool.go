package installer

import (
	"errors"
	"fmt"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// Supported bootstrap_tools install methods.
const (
	// BootstrapInstallNpmGlobal installs the pinned release with an
	// age-gated `npm install -g`.
	BootstrapInstallNpmGlobal = "npm-global"
	// BootstrapInstallNixProfile installs the package from a flake pinned to
	// a commit with `nix profile install`.
	BootstrapInstallNixProfile = "nix-profile"
)

// ErrBootstrapPin marks a missing or unusable catalog bootstrap_tools pin.
// Callers refuse to install rather than fall back to whatever release the
// registry serves.
var ErrBootstrapPin = errors.New("invalid bootstrap tool pin")

// BootstrapToolInstallCmd returns the command that installs the release the
// catalog's bootstrap_tools.<name> entry pins. An npm-global entry installs
// that exact release with an age-gated `npm install -g`, with lifecycle
// scripts disabled unless the entry allows them; a nix-profile entry
// installs its attribute from a flake pinned to a commit with `nix profile
// install`, ignoring the flake's nixConfig. A missing, unpinned or
// unsupported entry is refused with [ErrBootstrapPin].
func BootstrapToolInstallCmd(cat *catalog.Catalog, name string, now time.Time) ([]string, error) {
	def, ok := cat.BootstrapTool(name)
	if !ok {
		return nil, fmt.Errorf("%w: catalog has no bootstrap_tools.%s entry", ErrBootstrapPin, name)
	}
	var (
		cmd []string
		err error
	)
	switch def.InstallMethod {
	case BootstrapInstallNpmGlobal:
		cmd, err = NpmGlobalInstallCmd(NpmPackage{
			Name:              def.PackageName,
			Version:           def.Version,
			RunInstallScripts: def.AllowInstallScripts,
		}, now)
	case BootstrapInstallNixProfile:
		cmd, err = NixProfileInstallCmd(NixPackage{Flake: def.Flake, Attribute: def.PackageName})
	default:
		return nil, fmt.Errorf("%w: bootstrap_tools.%s.install_method %q is not supported (want %s or %s)",
			ErrBootstrapPin, name, def.InstallMethod, BootstrapInstallNpmGlobal, BootstrapInstallNixProfile)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: bootstrap_tools.%s: %w", ErrBootstrapPin, name, err)
	}
	return cmd, nil
}
