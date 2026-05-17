package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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
	b.WriteString("- The package guard hook validates all package installs for vulnerabilities and age\n")
	b.WriteString("- Use `qsdev enable <tool>` to add ecosystem tools (run `qsdev list` to see all)\n")
	b.WriteString("- Use `qsdev devenv add-package <name>` to add system packages\n")
	b.WriteString("- Use `qsdev devenv add-language <name>` to add language ecosystems\n")
	b.WriteString("- Use `qsdev devenv add-service <name>` to add services (databases, caches)\n")
	b.WriteString("- Never edit devenv.nix or .nix files directly — use qsdev commands\n")
	b.WriteString("- Run `qsdev devenv doctor` after configuration changes\n\n")

	b.WriteString("## Common Workflows\n\n")
	b.WriteString("### Add a project dependency\n")
	b.WriteString("Use the project's package manager within the devenv shell:\n")
	b.WriteString("- `pnpm add <package>` (npm/pnpm projects)\n")
	b.WriteString("- `cargo add <crate>` (Rust projects)\n")
	b.WriteString("- Add to pyproject.toml/requirements.txt (Python projects)\n")
	b.WriteString("- `go get <module>` (Go projects)\n")
	b.WriteString("The package guard hook validates safety automatically. Commit the lockfile after.\n\n")
	b.WriteString("### Add a system tool\n")
	b.WriteString("`qsdev devenv add-package <name>` — then run `direnv allow` to activate.\n\n")
	b.WriteString("### Enable a security/AI tool\n")
	b.WriteString("`qsdev enable <tool>` — run `qsdev list` to see available tools.\n\n")

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
