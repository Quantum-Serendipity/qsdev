package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/check"
	gdevconfig "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/config"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
)

func checkCmd() *cobra.Command {
	var (
		formatStr   string
		auditStr    string
		autoFix     bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run CI enforcement checks on the project configuration",
		Long: `Verify binary compatibility, config integrity, required tools,
generated file state, and security hardening.

Exit code is non-zero when checks fail at or above the audit level.
Use --format to select output format (human, json, sarif, junit).
Use --audit-level to control failure threshold (none, low, medium, high, critical).
Use --auto-fix to automatically fix issues where possible.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format := check.OutputFormat(formatStr)
			auditLevel := check.AuditLevel(auditStr)
			return runCheck(cmd, format, auditLevel, autoFix)
		},
	}

	cmd.Flags().StringVar(&formatStr, "format", "human", "Output format: human, json, sarif, junit")
	cmd.Flags().StringVar(&auditStr, "audit-level", "medium", "Minimum severity to fail: none, low, medium, high, critical")
	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Automatically fix issues where possible")

	return cmd
}

func runCheck(cmd *cobra.Command, format check.OutputFormat, auditLevel check.AuditLevel, autoFix bool) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	// Build CheckContext.
	ctx := check.CheckContext{
		ProjectRoot:   projectRoot,
		BinaryVersion: version.Info().Version,
		StateFile:     filepath.Join(projectRoot, statePath),
	}

	// Parse .gdev.yaml if present.
	cfg, err := gdevconfig.ParseGdevConfig(projectRoot)
	if err != nil {
		// Log warning but continue — config_integrity checks will report the issue.
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not parse .gdev.yaml: %v\n", err)
	}
	ctx.GdevConfig = cfg

	// Tool names from registry.
	ctx.ToolNames = toolreg.DefaultRegistry().Names()

	// Profile names from registry.
	if profileRegistry != nil {
		ctx.ProfileNames = profileRegistry.Names()
	} else {
		reg := DefaultProjectProfileRegistry()
		ctx.ProfileNames = reg.Names()
	}

	// Required deny rules — the critical subset that should always be present.
	ctx.RequiredDenyRules = criticalDenyRules()

	// Deny rule conflict validation.
	ctx.DenyRules = claudecode.AllBaseDenyRules()
	builtinSkills := claudecode.BuiltinSkillDefinitions()
	ctx.SkillOps = make([]check.SkillOps, len(builtinSkills))
	for i, s := range builtinSkills {
		ctx.SkillOps[i] = check.SkillOps{
			Name:         s.Name,
			AllowedTools: s.AllowedTools,
		}
	}
	ctx.ExpectedConflictKeys = claudecode.ExpectedConflicts()

	// Run all checks.
	report := check.RunAllChecks(ctx)

	// Auto-fix if requested.
	if autoFix {
		report.Checks = check.ApplyAutoFixes(report.Checks, projectRoot)
		// Rebuild summary after fixes.
		report = check.BuildReport(report.Checks, report.Version, report.Project)
	}

	// Detect color support.
	useColor := false
	if f, ok := cmd.OutOrStdout().(*os.File); ok {
		useColor = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}

	// Format and write report.
	if err := check.FormatReport(report, format, cmd.OutOrStdout(), useColor); err != nil {
		return fmt.Errorf("formatting report: %w", err)
	}

	// Emit GitHub Actions annotations if running in CI.
	if check.IsGitHubActions() {
		check.EmitGitHubAnnotations(report.Checks, cmd.OutOrStdout())
	}

	// Check if we should fail.
	if check.ShouldFail(report.Checks, auditLevel) {
		return &check.CheckFailedError{
			FailCount: check.FailCount(report.Checks, auditLevel),
			Level:     auditLevel,
		}
	}

	return nil
}

// criticalDenyRules returns the deny rules that are considered critical and
// must always be present in .claude/settings.json. This is a focused subset
// of the full deny rule set — the most dangerous operations that agents must
// never perform without explicit human approval.
func criticalDenyRules() []string {
	return []string{
		`Bash(git push --force *)`,
		`Bash(git push * --force)`,
		`Bash(git reset --hard *)`,
		`Bash(rm -rf *)`,
		`Read(./.env)`,
		`Read(./.env.*)`,
		`Read(./secrets/**)`,
		`Bash(curl * | bash *)`,
		`Bash(curl * | bash)`,
		`Bash(curl * | sh *)`,
		`Bash(curl * | sh)`,
	}
}
