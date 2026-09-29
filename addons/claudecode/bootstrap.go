package claudecode

import (
	"fmt"
	"time"

	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

// StepNameInstallClaude is the name of the bootstrap step that installs Claude Code.
const StepNameInstallClaude = "Install Claude Code"

// claudeSpec returns the install spec for the Claude Code release cat pins
// (bootstrap_tools.claude-code): an age-gated `npm install -g` of that exact
// release, with lifecycle scripts disabled unless the pin allows them.
func claudeSpec(cat *catalog.Catalog, now time.Time) (installer.ToolSpec, error) {
	cmd, err := installer.BootstrapToolInstallCmd(cat, catalog.BootstrapToolClaudeCode, now)
	if err != nil {
		return installer.ToolSpec{}, err
	}
	return installer.ToolSpec{
		DisplayName:   "Claude Code",
		Binary:        "claude",
		VersionFlag:   "--version",
		InstallCmd:    cmd,
		ManagerBinary: "npm",
		ManagerName:   "npm",
		FallbackURL:   "https://nodejs.org/",
		DirectURL:     "https://docs.anthropic.com/en/docs/claude-code/overview",
	}, nil
}

// defaultClaudeSpec returns the install spec from the loaded catalog (built-in
// defaults plus the user's defaults overlay), with the release-age cutoff
// computed now.
func defaultClaudeSpec() (installer.ToolSpec, error) {
	cat, err := catalog.Default()
	if err != nil {
		return installer.ToolSpec{}, fmt.Errorf("loading catalog: %w", err)
	}
	return claudeSpec(cat, time.Now())
}

// InstallClaudeStep returns a bootstrap step that ensures Claude Code is
// installed, installing the catalog's pinned release when it is missing.
func InstallClaudeStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallClaude,
		func(ctx *bootstrap.Context) error {
			spec, err := defaultClaudeSpec()
			if err != nil {
				return err
			}
			return installer.Install(ctx, spec)
		},
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error {
			spec, err := defaultClaudeSpec()
			if err != nil {
				return err
			}
			return installer.Simulate(ctx, spec)
		}),
		bootstrap.SkipInContainer(),
	)
}
