package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// GenerateGdevReference produces a .claude/gdev-reference.md file — a comprehensive
// reference document loaded via @-import from CLAUDE.md. This keeps the main
// CLAUDE.md lean while providing full CLI and workflow documentation.
func GenerateGdevReference(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("# gdev Reference\n\n")
	b.WriteString("## CLI Commands\n\n")

	commands := []struct{ Cmd, Desc string }{
		{"gdev init", "Initialize project with detection, wizard, and generation"},
		{"gdev init --update", "Update generated files to latest templates"},
		{"gdev init --mode join", "Join an existing gdev-managed project"},
		{"gdev devenv doctor", "Check system prerequisites and project health"},
		{"gdev devenv setup", "Install missing prerequisites"},
		{"gdev enable <tool>", "Enable a tool (updates configs, adds to shared files)"},
		{"gdev disable <tool>", "Disable a tool (cleans up all owned files)"},
		{"gdev status", "Show enabled/disabled tools and configuration state"},
		{"gdev list", "Show all available tools with categories"},
		{"gdev check", "Validate configuration for CI enforcement"},
		{"gdev check --format json", "Machine-readable check output"},
		{"gdev check --audit-level medium", "Set minimum severity that fails CI"},
		{"gdev config migrate", "Migrate .gdev.yaml to latest schema version"},
	}
	for _, c := range commands {
		fmt.Fprintf(&b, "### `%s`\n%s\n\n", c.Cmd, c.Desc)
	}

	b.WriteString("## Security Policy\n\n")
	b.WriteString("- Package installations are blocked by deny rules in `.claude/settings.json`\n")
	b.WriteString("- Use `gdev enable` to add tools, never configure manually\n")
	b.WriteString("- Run `gdev devenv doctor` after configuration changes\n")
	b.WriteString("- All security settings can only be strengthened, never weakened, via `.gdev.local.yaml`\n\n")

	b.WriteString("## Troubleshooting\n\n")
	b.WriteString("### gdev commands not found\nInstall gdev: see project README for installation instructions.\n\n")
	b.WriteString("### devenv not activated\nRun `direnv allow` in the project root, then `devenv shell`.\n\n")
	b.WriteString("### Permission denied on tool operations\nCheck `.claude/settings.json` deny rules. Use `gdev check --deny-rules` to validate.\n\n")

	return &types.GeneratedFile{
		Path:     ".claude/gdev-reference.md",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.LibraryManaged,
	}, nil
}
