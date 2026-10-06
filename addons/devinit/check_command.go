package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	mcpadapters "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/conformance"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
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
		scan      bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run CI enforcement checks on the project configuration",
		Long: `Verify binary compatibility, config integrity, required tools,
generated file state, security hardening, and the credential isolation of
each configured cloud provider (checked statically; no cloud CLI runs).

Exit code is non-zero when checks fail at or above the audit level.
Use --format to select output format (human, json, sarif, junit).
Use --audit-level to control failure threshold (none, low, medium, high, critical).
Use --auto-fix to automatically fix issues where possible.

When the project has a .qsdev-policy.yaml, each of its custom conformance
requirements is also checked; a failing requirement is high severity. Use
--scan to run a fresh dependency vulnerability scan so requirements on
dependencies.totals can pass (without one they fail as inconclusive).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format := check.OutputFormat(formatStr)
			auditLevel := check.AuditLevel(auditStr)
			return runCheck(cmd, format, auditLevel, autoFix, scan)
		},
	}

	cmd.Flags().StringVar(&formatStr, "format", "human", "Output format: human, json, sarif, junit")
	cmd.Flags().StringVar(&auditStr, "audit-level", "medium", "Minimum severity to fail: none, low, medium, high, critical")
	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Automatically fix issues where possible")
	cmd.Flags().BoolVar(&scan, "scan", false,
		"Run a fresh dependency vulnerability scan for custom conformance requirements")

	// check reports a catalog that does not load as a config_catalog result
	// (so --format json still writes a report), alongside the checks that
	// need no catalog, instead of the root gate's bare error.
	return cmdutil.MarkCatalogOptional(cmdutil.MarkReadOnly(cmd, "", "auto-fix", "scan"))
}

func runCheck(cmd *cobra.Command, format check.OutputFormat, auditLevel check.AuditLevel, autoFix, scan bool) error {
	pc, err := cmdutil.Project(cmd)
	if err != nil {
		return err
	}
	projectRoot := pc.Root
	ctx := catalogFreeCheckContext(cmd, projectRoot)

	// A catalog that does not load (a project defaults file it rejects), or
	// a registry that cannot be built from it, is reported as a failing
	// check alongside the checks that need no catalog. No auto-fix runs: the
	// fixes regenerate from the catalog.
	toolRegistry, failure := loadCheckRegistry()
	if failure != nil {
		return emitCheckReport(cmd, check.RunCatalogUnavailable(ctx, *failure), format, auditLevel)
	}

	// Tool names from registry for config validation.
	ctx.ToolNames = toolRegistry.Names()

	// mcp.disabled_tools names MCP tools, a namespace separate from the
	// catalog: validate it against every tool the MCP server can mount.
	ctx.MCPToolNames = mcpserve.MountableToolNames(mcpadapters.All())

	// Profile names from registry.
	profiles, err := projectProfiles()
	if err != nil {
		return err
	}
	ctx.ProfileNames = profiles.Names()

	// Saved answers are the generator's input; they decide which deny rules
	// .claude/settings.json must contain.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not read saved answers: %v\n", err)
	}
	// The answers file is local (gitignored), so a CI checkout has none;
	// rebuild them from the committed config the way join does, so CI still
	// knows what the project must enforce.
	var answersErr error
	cfgFile := branding.Get().ConfigFile
	if answers.ProjectName == "" && ctx.QsdevConfig != nil {
		if answers, answersErr = buildJoinAnswers(cmd, InitOptions{}, projectRoot); answersErr != nil {
			answersErr = fmt.Errorf("deriving answers from %s: %w", cfgFile, answersErr)
		}
	}
	// The committed hooks block and claude_code.permissions are
	// authoritative for the hook policy and the extra permission rules (init,
	// join and update refresh them from .qsdev.yaml), so a policy committed
	// after the answers were saved is what settings.json must enforce, and
	// a checkout that has not run 'qsdev init --update' since fails.
	if ctx.QsdevConfig != nil {
		answers.HookPolicy = ctx.QsdevConfig.Hooks.Clone()
		answers.ClaudePermissions = ctx.QsdevConfig.ClaudeCode.Permissions.Clone()
	}

	// Settle the answers against the committed config, as every generation
	// path does, before deciding what is required: the always-on scope and
	// the opt-outs and the permission floor come from .qsdev.yaml, never from
	// the local answers file alone, so a local edit cannot narrow the
	// always-on tools or deny rules check enforces.
	settleCommittedScope(&answers, toolRegistry, ctx.QsdevConfig)

	// The required-tools check covers the always-on tools that apply to the
	// project: those that configure Claude Code only when it is enabled.
	for _, tool := range toolRegistry.All() {
		if tool.EnforcedFor(&answers) {
			ctx.AlwaysOnToolNames = append(ctx.AlwaysOnToolNames, tool.Name)
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
	ctx.AskRules = claudecode.AllBaseAskRules()
	builtinSkills := claudecode.BuiltinSkillDefinitions()
	ctx.SkillOps = make([]check.SkillOps, len(builtinSkills))
	for i, s := range builtinSkills {
		ctx.SkillOps[i] = check.SkillOps{
			Name:         s.Name,
			AllowedTools: s.AllowedTools,
			PreApproved:  s.PreApproved,
		}
	}
	ctx.ExpectedConflictKeys = claudecode.ExpectedConflicts()

	// The generator's output for the settled answers is what the on-disk
	// settings.json must still enforce (hook registrations, bypass mode), so
	// an opt-out the answers file records but .qsdev.yaml does not (e.g.
	// attach-guard: false) still expects the guard's hook registrations.
	var freshFiles map[string]types.GeneratedFile
	var genErr error
	if answers.ProjectName != "" {
		freshFiles, _, genErr = regenerateFreshFiles(answers)
	}
	if settings, ok := freshFiles[check.ClaudeSettingsRelPath]; ok {
		ctx.ExpectedClaudeSettings = settings.Content
	}
	// Every hook script is judged against the content qsdev writes for it,
	// seeded from the embedded templates so the judgement holds when the
	// generator cannot run or the config turns Claude Code off.
	ctx.GeneratedContent = claudecode.HookScriptContents()
	ctx.GuardSupportFiles = []string{claudecode.HookLibPath}
	for rel, f := range freshFiles {
		ctx.GeneratedContent[rel] = f.Content
	}
	ctx.ExpectedGenerationErr = expectedGenerationErr(answersErr, genErr)
	// The hand-edited devenv.nix (and devenv.local.nix) must still enable the
	// security hooks and strip the variables the generated one does.
	if ctx.ExpectedGenerationErr == nil {
		ctx.ExpectedGenerationErr = fillDevenvSecurity(&ctx, answers, freshFiles, projectRoot)
	}
	ctx.RequiredMCPServers = requiredMCPServers(answers, toolRegistry, freshFiles)
	if answers.ClaudeCode {
		for _, h := range claudecode.HooksWithoutPolicy(answers) {
			ctx.HooksWithoutPolicy = append(ctx.HooksWithoutPolicy, check.HookWithoutPolicy{Name: h.Name, PolicyKey: h.PolicyKey})
		}
	}

	ctx.CustomConformance = evaluateCustomConformance(projectRoot, scan)

	// Environment separation for cloud providers is judged from what the
	// devenv modules declare; no cloud CLI runs.
	ctx.DeclaredEnv, ctx.DeclaredEnvErr = devenv.ProjectDeclaredEnv(projectRoot)

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

	return emitCheckReport(cmd, report, format, auditLevel)
}

// loadCheckRegistry returns the tool registry check validates against, or the
// result that reports why it is unavailable. A catalog that does not load is
// reported with the catalog's own error, worded as the root catalog gate
// words it (not toolreg's re-wrap); a registry that cannot be built from a
// catalog that loaded is reported separately, so its cause is not blamed on
// the defaults file.
func loadCheckRegistry() (*toolreg.Registry, *check.CheckResult) {
	if _, err := catalog.Default(); err != nil {
		r := check.CatalogLoadFailure(err)
		return nil, &r
	}
	reg, err := toolreg.Default()
	if err != nil {
		r := check.ToolRegistryFailure(err)
		return nil, &r
	}
	return reg, nil
}

// catalogFreeCheckContext returns the check context for the project at
// projectRoot with every field that needs no catalog filled in: what
// check.RunCatalogUnavailable judges when the catalog does not load.
func catalogFreeCheckContext(cmd *cobra.Command, projectRoot string) check.CheckContext {
	ctx := check.CheckContext{
		ProjectRoot:   projectRoot,
		BinaryVersion: version.Info().Version,
		StateFile:     filepath.Join(projectRoot, stateFilePath()),
		ManifestFile:  filepath.Join(projectRoot, state.ManifestFile()),
		ProbeTool: func(binary, versionArg string) toolcheck.Info {
			return toolcheck.Detect(cmd.Context(), binary, versionArg)
		},
		ClaudeUserDir:   claudeUserDir(),
		OrgConfigDrift:  catalog.ProjectOrgConfigDrift(projectRoot),
		OrgConfigSource: catalog.ProjectOrgConfigSource(projectRoot),
	}
	// Parse config if present. The error travels in the context so the report
	// (including machine-readable formats) distinguishes "not found" from a
	// parse failure.
	ctx.QsdevConfig, ctx.ConfigErr = qsdevconfig.ParseQsdevConfig(filepath.Join(projectRoot, branding.Get().ConfigFile))
	return ctx
}

// emitCheckReport writes report to cmd's output in format, emits GitHub
// Actions annotations when running there, and returns the error that makes
// check exit 1 when a result fails at or above auditLevel.
func emitCheckReport(cmd *cobra.Command, report *check.CheckReport, format check.OutputFormat, auditLevel check.AuditLevel) error {
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

// evaluateCustomConformance evaluates the project's custom conformance policy
// (conformance.PolicyFileName) against a posture assessment, running a fresh
// dependency scan when scan is set. It returns nil when the project has no
// policy file or the file has no custom section, and only assesses the
// project when there are requirements to judge. A policy that cannot be
// loaded, or an assessment that fails, leaves every requirement unverifiable,
// which is reported as a single failing requirement.
func evaluateCustomConformance(projectRoot string, scan bool) *check.CustomConformance {
	policyFile := conformance.PolicyFileName()
	policy, err := conformance.LoadPolicy(projectRoot)
	var level *posture.ConformanceLevel
	switch {
	case err != nil:
		level = conformance.PolicyError(err)
	case policy == nil:
		return nil
	default:
		report, assessErr := posture.Assess(projectRoot, postureOptions(posture.AssessOptions{FreshScan: scan}))
		if assessErr != nil {
			level = conformance.PolicyError(fmt.Errorf(
				"cannot evaluate %s: assessing project posture: %w", policyFile, assessErr))
		} else {
			level = conformance.Evaluate(policy, report)
		}
	}
	custom := &check.CustomConformance{PolicyFile: policyFile}
	for _, c := range level.Checks {
		custom.Requirements = append(custom.Requirements, check.PolicyRequirement{
			Name:   string(c.Name),
			Pass:   c.Pass,
			Reason: c.Reason,
		})
	}
	return custom
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
// tier's default preset, then standard. The project config is used when no
// answers were saved; otherwise the saved answers, which are local, may only
// tighten the preset cfg commits (see qsdevconfig.TightenPermissionLevel), as
// a local layer may.
func effectivePermissionPreset(answers types.WizardAnswers, cfg *types.QsdevConfig) string {
	if cfg == nil {
		return qsdevconfig.EffectivePermissionLevel(answers.PermissionLevel, answers.Tier, answers.MCPServers)
	}
	floor := qsdevconfig.PermissionFloor(cfg)
	if answers.PermissionLevel == "" && answers.Tier == "" {
		return floor
	}
	level := qsdevconfig.EffectivePermissionLevel(answers.PermissionLevel, answers.Tier, answers.MCPServers)
	return qsdevconfig.TightenPermissionLevel(level, floor)
}

// settleCommittedScope settles answers against cfg, the committed
// .qsdev.yaml (nil when it did not load): Claude Code and the tier, which
// decide which always-on tools apply (see toolreg.Tool.EnforcedFor), are
// taken from cfg, as is the client MCP policy (see
// qsdevconfig.AdoptClientMCPPolicy), the permission level is raised to at
// least the one cfg commits (see effectivePermissionPreset), and the tools
// are reconciled against its tools block (see toolreg.Reconcile).
func settleCommittedScope(answers *types.WizardAnswers, reg *toolreg.Registry, cfg *types.QsdevConfig) {
	if answers.ProjectName == "" {
		return
	}
	var committed *types.ToolsConfig
	if cfg != nil {
		answers.ClaudeCode = qsdevconfig.ClaudeCodeEnabled(cfg)
		answers.Tier = qsdevconfig.ConfigTier(cfg)
		qsdevconfig.AdoptClientMCPPolicy(answers, cfg)
		answers.PermissionLevel = effectivePermissionPreset(*answers, cfg)
		committed = &cfg.Tools
	}
	toolreg.Reconcile(answers, reg, committed)
}

// requiredMCPServers maps each always-on tool that applies to answers and
// that they enable to its MCP server, for each server the expected
// .mcp.json in freshFiles configures (a server the client MCP policy blocks
// is never configured, so never required). The on-disk .mcp.json must
// configure every one of them.
func requiredMCPServers(answers types.WizardAnswers, reg *toolreg.Registry, freshFiles map[string]types.GeneratedFile) map[string]string {
	expected, ok := freshFiles[check.MCPConfigRelPath]
	if !ok {
		return nil
	}
	required := make(map[string]string)
	for _, tool := range reg.All() {
		if tool.MCPServer == "" || !tool.EnforcedFor(&answers) || !answers.EnabledTools[tool.Name] {
			continue
		}
		if surgery.JSONHasMCPServer(expected.Content, tool.MCPServer) {
			required[tool.Name] = tool.MCPServer
		}
	}
	return required
}

// fillDevenvSecurity records in ctx the security git hooks (see
// catalog.SecurityHookIDs for the answers' compliance level) the devenv.nix in
// freshFiles enables, the variables it strips and the git-hooks settings that
// shape those hooks, next to what the project's devenv modules declare. It
// records nothing when the generator writes no devenv.nix, and returns an
// error when the generated file cannot be read.
func fillDevenvSecurity(ctx *check.CheckContext, answers types.WizardAnswers, freshFiles map[string]types.GeneratedFile, projectRoot string) error {
	f, ok := freshFiles[toolreg.DevenvNixFile]
	if !ok {
		return nil
	}
	cat, err := catalog.Default()
	if err != nil {
		return fmt.Errorf("loading catalog for the devenv security floor: %w", err)
	}
	expected, err := devenv.DeclaredSecurity(string(f.Content))
	if err != nil {
		return fmt.Errorf("reading the generated %s: %w", toolreg.DevenvNixFile, err)
	}
	level := answers.ComplianceLevel
	if level == "" {
		level = cat.TierCompliance(answers.Tier)
	}
	security := cat.SecurityHookIDs(level)
	ctx.ExpectedDevenvHooks = slices.DeleteFunc(expected.Hooks, func(id string) bool { return !slices.Contains(security, id) })
	ctx.ExpectedUnsetVars = expected.UnsetVars
	ctx.ExpectedDevenvHookSettings = expected.SecuritySettings(ctx.ExpectedDevenvHooks)

	declared, err := devenv.ProjectDeclaredSecurity(projectRoot)
	ctx.DevenvHooks, ctx.DevenvUnsetVars, ctx.DevenvSecurityErr = declared.Hooks, declared.UnsetVars, err
	ctx.DevenvHookSettings = declared.SecuritySettings(ctx.ExpectedDevenvHooks)
	return nil
}

// expectedGenerationErr reports why the generator's output for the project
// is unknown, or nil when it is known or there was nothing to generate from.
// Without that output every hook-registration check has nothing to compare
// against, so check reports the failure (see check.CheckExpectedGeneration)
// rather than passing on what is left. An unreadable config is reported by
// its own check.
func expectedGenerationErr(answersErr, genErr error) error {
	if genErr != nil {
		return fmt.Errorf("regenerating expected files: %w", genErr)
	}
	return answersErr
}
