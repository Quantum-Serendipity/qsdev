package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateQsdevReference produces a .claude/qsdev-reference.md file — a comprehensive
// reference document loaded via @-import from CLAUDE.md. This keeps the main
// CLAUDE.md lean while providing full CLI and workflow documentation.
func GenerateQsdevReference(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	var b strings.Builder
	app := branding.Get().AppName

	fmt.Fprintf(&b, "# %s Reference\n\n", app)
	b.WriteString("## CLI Commands\n\n")

	commands := []struct{ Cmd, Desc string }{
		{app + " init", "Initialize project with detection, wizard, and generation"},
		{app + " init --update", "Update generated files to latest templates"},
		{app + " init --mode join", "Join an existing " + app + "-managed project"},
		{app + " devenv doctor", "Check system prerequisites and project health"},
		{app + " devenv setup", "Install missing prerequisites"},
		{app + " enable <tool>", "Enable a tool (updates configs, adds to shared files)"},
		{app + " disable <tool>", "Disable a tool (cleans up all owned files)"},
		{app + " status", "Show enabled/disabled tools and configuration state"},
		{app + " list", "Show all available tools with categories"},
		{app + " check", "Validate configuration for CI enforcement"},
		{app + " check --format json", "Machine-readable check output"},
		{app + " check --audit-level medium", "Set minimum severity that fails CI"},
		{app + " config migrate", "Migrate " + branding.Get().ConfigFile + " to latest schema version"},
	}
	for _, c := range commands {
		fmt.Fprintf(&b, "### `%s`\n%s\n\n", c.Cmd, c.Desc)
	}

	b.WriteString("## Security Policy\n\n")
	b.WriteString("- The package guard hook validates all package installs for vulnerabilities and age\n")
	fmt.Fprintf(&b, "- Use `%s enable <tool>` to add ecosystem tools (run `%s list` to see all)\n", app, app)
	fmt.Fprintf(&b, "- Use `%s devenv add-package <name>` to add system packages\n", app)
	fmt.Fprintf(&b, "- Use `%s devenv add-language <name>` to add language ecosystems\n", app)
	fmt.Fprintf(&b, "- Use `%s devenv add-service <name>` to add services (databases, caches)\n", app)
	fmt.Fprintf(&b, "- Never edit devenv.nix or .nix files directly — use %s commands\n", app)
	fmt.Fprintf(&b, "- Run `%s devenv doctor` after configuration changes\n\n", app)

	b.WriteString("## Common Workflows\n\n")
	b.WriteString("### Add a project dependency\n")
	b.WriteString("Use the project's package manager within the devenv shell:\n")
	b.WriteString("- `pnpm add <package>` (npm/pnpm projects)\n")
	b.WriteString("- `cargo add <crate>` (Rust projects)\n")
	b.WriteString("- Add to pyproject.toml/requirements.txt (Python projects)\n")
	b.WriteString("- `go get <module>` (Go projects)\n")
	b.WriteString("The package guard hook validates safety automatically. Commit the lockfile after.\n\n")
	b.WriteString("### Add a system tool\n")
	fmt.Fprintf(&b, "`%s devenv add-package <name>` — then run `direnv allow` to activate.\n\n", app)
	b.WriteString("### Enable a security/AI tool\n")
	fmt.Fprintf(&b, "`%s enable <tool>` — run `%s list` to see available tools.\n\n", app, app)

	b.WriteString("## Troubleshooting\n\n")
	fmt.Fprintf(&b, "### %s commands not found\nInstall %s: see project README for installation instructions.\n\n", app, app)
	b.WriteString("### devenv not activated\nRun `direnv allow` in the project root, then `devenv shell`.\n\n")
	fmt.Fprintf(&b, "### Permission denied on tool operations\nCheck `.claude/settings.json` deny rules. Use `%s check --deny-rules` to validate.\n\n", app)

	return &types.GeneratedFile{
		Path:     ".claude/qsdev-reference.md",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.LibraryManaged,
	}, nil
}
