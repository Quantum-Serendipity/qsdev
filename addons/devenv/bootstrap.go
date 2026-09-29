package devenv

import (
	"fmt"
	"time"

	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

const (
	// StepNameInstallDevenv is the name of the bootstrap step that installs devenv.
	StepNameInstallDevenv = "Install devenv"
	// StepNameInstallDirenv is the name of the bootstrap step that installs direnv.
	StepNameInstallDirenv = "Install direnv"
)

// nixToolSpec describes a tool the bootstrap installs with Nix. Its install
// command comes from the catalog's bootstrap_tools pin, not from here.
type nixToolSpec struct {
	catalogName string // bootstrap_tools entry, e.g. catalog.BootstrapToolDevenv
	displayName string
	binary      string
	directURL   string
}

var (
	devenvTool = nixToolSpec{
		catalogName: catalog.BootstrapToolDevenv,
		displayName: "devenv",
		binary:      "devenv",
		directURL:   "https://devenv.sh/getting-started/",
	}
	direnvTool = nixToolSpec{
		catalogName: catalog.BootstrapToolDirenv,
		displayName: "direnv",
		binary:      "direnv",
		directURL:   "https://direnv.net/docs/installation.html",
	}
)

// toolSpec returns the install spec for t: `nix profile install` of the
// attribute the catalog pins (bootstrap_tools.<name>) from a flake pinned to
// a commit, with the flake's nixConfig ignored.
func (t nixToolSpec) toolSpec(cat *catalog.Catalog) (installer.ToolSpec, error) {
	cmd, err := installer.BootstrapToolInstallCmd(cat, t.catalogName, time.Now())
	if err != nil {
		return installer.ToolSpec{}, err
	}
	return installer.ToolSpec{
		DisplayName:   t.displayName,
		Binary:        t.binary,
		VersionFlag:   "--version",
		InstallCmd:    cmd,
		ManagerBinary: "nix",
		ManagerName:   "Nix",
		FallbackURL:   "https://nixos.org/download",
		DirectURL:     t.directURL,
	}, nil
}

// defaultToolSpec returns t's install spec from the loaded catalog (built-in
// defaults plus the user's defaults overlay).
func (t nixToolSpec) defaultToolSpec() (installer.ToolSpec, error) {
	cat, err := catalog.Default()
	if err != nil {
		return installer.ToolSpec{}, fmt.Errorf("loading catalog: %w", err)
	}
	return t.toolSpec(cat)
}

// installStep returns a bootstrap step that ensures t is installed,
// installing the catalog's pinned package when it is missing.
func (t nixToolSpec) installStep(name string) *bootstrap.Step {
	return bootstrap.NewStep(
		name,
		func(ctx *bootstrap.Context) error {
			spec, err := t.defaultToolSpec()
			if err != nil {
				return err
			}
			return installer.Install(ctx, spec)
		},
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error {
			spec, err := t.defaultToolSpec()
			if err != nil {
				return err
			}
			return installer.Simulate(ctx, spec)
		}),
		bootstrap.SkipInContainer(),
	)
}

// InstallDevenvStep returns a bootstrap step that ensures devenv is installed.
func InstallDevenvStep() *bootstrap.Step {
	return devenvTool.installStep(StepNameInstallDevenv)
}

// InstallDirenvStep returns a bootstrap step that ensures direnv is installed.
func InstallDirenvStep() *bootstrap.Step {
	return direnvTool.installStep(StepNameInstallDirenv)
}
