package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateQsdevReference produces a .claude/qsdev-reference.md file — a comprehensive
// reference document loaded via @-import from CLAUDE.md. This keeps the main
// CLAUDE.md lean while providing full CLI and workflow documentation.
func GenerateQsdevReference(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("# qsdev Reference\n\n")
	b.WriteString("## CLI Commands\n\n")

	commands := []struct{ Cmd, Desc string }{
		{"qsdev init", "Initialize project with detection, wizard, and generation"},
		{"qsdev init --update", "Update generated files to latest templates"},
		{"qsdev init --mode join", "Join an existing qsdev-managed project"},
		{"qsdev devenv doctor", "Check system prerequisites and project health"},
		{"qsdev devenv setup", "Install missing prerequisites"},
		{"qsdev enable <tool>", "Enable a tool (updates configs, adds to shared files)"},
		{"qsdev disable <tool>", "Disable a tool (cleans up all owned files)"},
		{"qsdev status", "Show enabled/disabled tools and configuration state"},
		{"qsdev list", "Show all available tools with categories"},
		{"qsdev check", "Validate configuration for CI enforcement"},
		{"qsdev check --format json", "Machine-readable check output"},
		{"qsdev check --audit-level medium", "Set minimum severity that fails CI"},
		{"qsdev config migrate", "Migrate .qsdev.yaml to latest schema version"},
	}
	for _, c := range commands {
		fmt.Fprintf(&b, "### `%s`\n%s\n\n", c.Cmd, c.Desc)
	}

	b.WriteString("## Security Policy\n\n")
	b.WriteString("- Package installations are blocked by deny rules in `.claude/settings.json`\n")
	b.WriteString("- Use `qsdev enable` to add tools, never configure manually\n")
	b.WriteString("- Run `qsdev devenv doctor` after configuration changes\n")
	b.WriteString("- All security settings can only be strengthened, never weakened, via `.qsdev.local.yaml`\n\n")

	b.WriteString("## Troubleshooting\n\n")
	b.WriteString("### qsdev commands not found\nInstall qsdev: see project README for installation instructions.\n\n")
	b.WriteString("### devenv not activated\nRun `direnv allow` in the project root, then `devenv shell`.\n\n")
	b.WriteString("### Permission denied on tool operations\nCheck `.claude/settings.json` deny rules. Use `qsdev check --deny-rules` to validate.\n\n")

	return &types.GeneratedFile{
		Path:     ".claude/qsdev-reference.md",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.LibraryManaged,
	}, nil
}
