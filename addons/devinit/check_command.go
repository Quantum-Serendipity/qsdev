package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func checkCmd() *cobra.Command {
	var (
		formatStr string
		auditStr  string
		autoFix   bool
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
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	// Build CheckContext.
	ctx := check.CheckContext{
		ProjectRoot:   projectRoot,
		BinaryVersion: version.Info().Version,
		StateFile:     filepath.Join(projectRoot, stateFilePath()),
	}

	// Parse config if present. The error travels in the context so the report
	// (including machine-readable formats) distinguishes "not found" from a
	// parse failure.
	cfgFile := branding.Get().ConfigFile
	ctx.QsdevConfig, ctx.ConfigErr = qsdevconfig.ParseQsdevConfig(filepath.Join(projectRoot, cfgFile))

	// Tool names from registry: all names for config validation, and the
	// always-on subset for the required-tools check.
	toolRegistry := toolreg.DefaultRegistry()
	ctx.ToolNames = toolRegistry.Names()
	for _, tool := range toolRegistry.All() {
		if tool.Default == toolreg.AlwaysOn {
			ctx.AlwaysOnToolNames = append(ctx.AlwaysOnToolNames, tool.Name)
		}
	}

	// Profile names from registry.
	ctx.ProfileNames = ensureProfileRegistry().Names()

	// Saved answers are the generator's input; they decide which deny rules
	// .claude/settings.json must contain.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not read saved answers: %v\n", err)
	}
	// The answers file is local (gitignored), so a CI checkout has none;
	// rebuild them from the committed config the way join does, so CI still
	// knows what the project must enforce.
	if answers.ProjectName == "" && ctx.QsdevConfig != nil {
		if answers, err = buildJoinAnswers(cmd, InitOptions{}, projectRoot); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not derive answers from %s: %v\n", cfgFile, err)
		}
	}

	// Required deny rules: every base rule the project's permission preset
	// generates, so deleting any of them from settings.json is caught.
	ctx.RequiredDenyRules, err = requiredDenyRules(answers, ctx.QsdevConfig)
	if err != nil {
		return err
	}

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

	// The generator's output for the saved answers is what the on-disk
	// settings.json must still enforce (hook registrations, bypass mode).
	var freshFiles map[string]types.GeneratedFile
	var genErr error
	if answers.ProjectName != "" {
		freshFiles, _, genErr = regenerateFreshFiles(answers)
		if genErr != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not regenerate expected files: %v\n", genErr)
		}
	}
	if settings, ok := freshFiles[check.ClaudeSettingsRelPath]; ok {
		ctx.ExpectedClaudeSettings = settings.Content
	}

	// Run all checks.
	report := check.RunAllChecks(ctx)

	// Auto-fix if requested.
	if autoFix {
		var regen check.RegenerateFunc
		if answers.ProjectName != "" {
			regen = func(_ string) (map[string]types.GeneratedFile, error) {
				return freshFiles, genErr
			}
		}
		report.Checks = check.ApplyAutoFixes(report.Checks, projectRoot, ctx.StateFile, regen)
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

	// Emit GitHub Actions annotations to stderr so they don't pollute JSON/SARIF output.
	if check.IsGitHubActions() {
		check.EmitGitHubAnnotations(report.Checks, cmd.ErrOrStderr())
	}

	// Check if we should fail.
	if check.ShouldFail(report.Checks, auditLevel) {
		if isMachineReadableFormat(format) {
			return &ExitError{Code: 1}
		}
		return &check.CheckFailedError{
			FailCount: check.FailCount(report.Checks, auditLevel),
			Level:     auditLevel,
		}
	}

	return nil
}

func isMachineReadableFormat(f check.OutputFormat) bool {
	switch f {
	case check.FormatJSON, check.FormatSARIF, check.FormatJUnit:
		return true
	default:
		return false
	}
}

// requiredDenyRules returns the base deny rules the generator emits for the
// project's permission preset; each must be present in .claude/settings.json.
// The rules come from the catalog's deny sets for that preset, so a preset
// that omits a set (e.g. supply-chain-only) is not required to carry it.
func requiredDenyRules(answers types.WizardAnswers, cfg *types.QsdevConfig) ([]string, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog for required deny rules: %w", err)
	}
	def, ok := cat.PermissionPreset(effectivePermissionPreset(answers, cfg))
	if !ok {
		// Mirror the generator, which falls back to the standard preset.
		if def, ok = cat.PermissionPreset(string(claudecode.PermissionPresetStandard)); !ok {
			return nil, fmt.Errorf("catalog has no %q permission preset", claudecode.PermissionPresetStandard)
		}
	}
	var rules []string
	for _, setName := range def.DenySets {
		rules = append(rules, cat.PermissionDenyRules(setName)...)
	}
	return rules, nil
}

// effectivePermissionPreset resolves the permission preset the same way
// settings generation does: an explicit permission level wins, then the
// tier's default preset, then standard. Saved answers are preferred; the
// project config is used when no answers were saved.
func effectivePermissionPreset(answers types.WizardAnswers, cfg *types.QsdevConfig) string {
	level, tierName, mcp := answers.PermissionLevel, answers.Tier, answers.MCPServers
	if level == "" && tierName == "" && cfg != nil {
		level, tierName = cfg.ClaudeCode.PermissionLevel, cfg.Tier
	}
	switch {
	case level != "":
		return level
	case tierName != "":
		return tier.Resolve(tierName, level, mcp).DefaultPermissionPreset()
	default:
		return string(claudecode.PermissionPresetStandard)
	}
}
