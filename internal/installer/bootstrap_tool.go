package installer

import (
	"errors"
	"fmt"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// BootstrapInstallNpmGlobal is the bootstrap_tools install method that
// installs the pinned release with an age-gated `npm install -g`. It is the
// only method supported.
const BootstrapInstallNpmGlobal = "npm-global"

// ErrBootstrapPin marks a missing or unusable catalog bootstrap_tools pin.
// Callers refuse to install rather than fall back to whatever release the
// registry serves.
var ErrBootstrapPin = errors.New("invalid bootstrap tool pin")

// BootstrapToolInstallCmd returns the command that installs the release the
// catalog's bootstrap_tools.<name> entry pins: an age-gated `npm install -g`
// of that exact release, with lifecycle scripts disabled unless the entry
// allows them. A missing, unpinned or unsupported entry is refused with
// [ErrBootstrapPin].
func BootstrapToolInstallCmd(cat *catalog.Catalog, name string, now time.Time) ([]string, error) {
	def, ok := cat.BootstrapTool(name)
	if !ok {
		return nil, fmt.Errorf("%w: catalog has no bootstrap_tools.%s entry", ErrBootstrapPin, name)
	}
	if def.InstallMethod != BootstrapInstallNpmGlobal {
		return nil, fmt.Errorf("%w: bootstrap_tools.%s.install_method %q is not supported (want %s)",
			ErrBootstrapPin, name, def.InstallMethod, BootstrapInstallNpmGlobal)
	}
	cmd, err := NpmGlobalInstallCmd(NpmPackage{
		Name:              def.PackageName,
		Version:           def.Version,
		RunInstallScripts: def.AllowInstallScripts,
	}, now)
	if err != nil {
		return nil, fmt.Errorf("%w: bootstrap_tools.%s: %w", ErrBootstrapPin, name, err)
	}
	return cmd, nil
}
