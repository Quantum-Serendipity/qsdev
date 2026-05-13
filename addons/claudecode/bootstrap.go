package claudecode

import (
	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/installer"
)

// StepNameInstallClaude is the name of the bootstrap step that installs Claude Code.
const StepNameInstallClaude = "Install Claude Code"

var claudeSpec = installer.ToolSpec{
	DisplayName:   "Claude Code",
	Binary:        "claude",
	VersionFlag:   "--version",
	InstallCmd:    []string{"npm", "install", "-g", "@anthropic-ai/claude-code"},
	ManagerBinary: "npm",
	ManagerName:   "npm",
	FallbackURL:   "https://nodejs.org/",
	DirectURL:     "https://docs.anthropic.com/en/docs/claude-code/overview",
}

// InstallClaudeStep returns a bootstrap step that ensures Claude Code is installed.
func InstallClaudeStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallClaude,
		func(ctx *bootstrap.Context) error { return installer.Install(ctx, claudeSpec) },
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error { return installer.Simulate(ctx, claudeSpec) }),
		bootstrap.SkipInContainer(),
	)
}
