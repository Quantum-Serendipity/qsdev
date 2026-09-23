package devinit

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// runJoin sets up a local development environment from an existing .qsdev.yaml.
// This is the "join" path for new team members cloning a project.
func runJoin(cmd *cobra.Command, opts InitOptions, projectRoot string) error {
	// 1. Build answers from config, detection, and optional overrides.
	answers, err := buildJoinAnswers(cmd, opts, projectRoot)
	if err != nil {
		return err
	}

	// 2. Auto-install missing prerequisites if --yes, otherwise warn.
	hasMissingPrereqs := joinPrerequisites(cmd, opts)

	// 3. Generate files via fragment accumulation.
	applyScopeFlags(opts, &answers)
	accResult, err := runAccumulator(answers, scopeFromAnswers(answers))
	if err != nil {
		return fmt.Errorf("generating files: %w", err)
	}
	allFiles := accResult.allFiles

	// 4. Generate local config template (only if it doesn't exist).
	localCfg := branding.Get().LocalConfig
	localConfigPath := filepath.Join(projectRoot, localCfg)
	if _, err := os.Stat(localConfigPath); os.IsNotExist(err) {
		localContent := GenerateLocalConfigTemplate(answers, answers.Detected)
		allFiles = append(allFiles, types.GeneratedFile{
			Path:    localCfg,
			Content: localContent,
			Mode:    fileutil.ModeReadWrite,
		})
	}

	// 5. Dry-run: preview and return before touching the working tree.
	if opts.DryRun {
		preview := generate.PreviewFiles(allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
	}

	// 6. Ensure local config is in .gitignore.
	if err := EnsureGitignoreEntry(projectRoot, localCfg); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	// 7. Write files and record results.
	if err := writeJoinResults(cmd, opts, projectRoot, answers, accResult, allFiles); err != nil {
		return err
	}

	// 8. Print join-specific summary.
	if !opts.Quiet {
		if hasMissingPrereqs {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNext: run '%s devenv setup --yes' to install missing prerequisites (nix, devenv, direnv).\n", branding.Get().AppName)
		}
	}

	return nil
}

// joinPrerequisites auto-installs missing prerequisites with --yes and warns
// otherwise, reporting whether any are still missing. Like the create path it
// does nothing for --dry-run (a preview must not install onto the host) or
// --claude-only (no devenv environment is generated).
func joinPrerequisites(cmd *cobra.Command, opts InitOptions) bool {
	if opts.DryRun || opts.ClaudeOnly {
		return false
	}
	prereqs := CheckPrerequisites(cmd.Context())
	if !prereqs.HasMissing() {
		return false
	}
	if !opts.Yes {
		fmt.Fprintln(cmd.ErrOrStderr(), "Note: some prerequisites are missing:")
		prereqs.PrintReport(cmd.ErrOrStderr())
		fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' after join to install them.\n", branding.Get().AppName)
		fmt.Fprintln(cmd.ErrOrStderr())
		return true
	}
	if err := devenv.AutoSetupPrerequisites(cmd.Context(), cmd.ErrOrStderr()); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: prerequisite installation failed: %v\n", err)
		fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' manually.\n\n", branding.Get().AppName)
		return true
	}
	return false
}

// buildJoinAnswers rebuilds the answers the project was created with from the
// committed .qsdev.yaml. Precedence, lowest first: the committed config, an
// --answers-file overlay (top-level keys it sets replace the config's), then
// explicitly-set flags. Settings the config has no key for get the same
// defaults and invariants the create path applies.
func buildJoinAnswers(cmd *cobra.Command, opts InitOptions, projectRoot string) (types.WizardAnswers, error) {
	// Parse project config.
	cfgFile := branding.Get().ConfigFile
	cfgPath := filepath.Join(projectRoot, cfgFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("parsing %s: %w", cfgFile, err)
	}

	detected := detect.Detect(cmdContext(cmd), projectRoot)
	answers := qsdevconfig.ConfigToAnswers(cfg, detected, projectRoot)

	if opts.AnswersFile != "" {
		answers, err = OverlayAnswersFile(answers, opts.AnswersFile)
		if err != nil {
			return types.WizardAnswers{}, err
		}
	}

	flagAnswers, err := AnswersFromFlags(opts, projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	answers = MergeFileWithFlags(answers, flagAnswers, flagSetToChangedMap(NewFlagSet(cmd), cmd))
	if opts.ProfileName != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Note: --profile applies only when creating a project; ignoring it in join mode (re-run with --mode create to apply it).")
	}

	if opts.AnswersFile != "" {
		if err := ValidateAnswersFileCompleteness(answers); err != nil {
			return types.WizardAnswers{}, err
		}
	}
	answers.ProjectRoot = projectRoot
	answers.ProjectName = filepath.Base(projectRoot)
	answers.Detected = detected
	answers.Confirmed = true

	cat, err := catalog.Default()
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("loading catalog for defaults: %w", err)
	}
	registry := toolreg.DefaultRegistry()
	applyJoinDefaults(&answers, cat, registry)

	// Validate answers.
	if err := ValidateAnswers(answers); err != nil {
		return types.WizardAnswers{}, err
	}

	// Augment EnabledTools with inferred tools (AlwaysOn, hooks-implied).
	toolreg.MergeInferredTools(&answers, registry)
	enforceAnswerInvariants(&answers)

	return answers, nil
}

// applyJoinDefaults gives a joiner the defaults the create path applies (via
// FillDefaults) for settings .qsdev.yaml has no key for: the self-protection
// and safety-block hook invariants, agent tools, and the tier-derived tool
// set. FillDefaults runs on a copy and only those fields are adopted, because
// its list defaulting must not apply here: an empty claude_code.mcp_servers or
// languages list in the committed config is the team's choice. The compliance
// level is not adopted either: security.level is persisted, so an empty one is
// what the project was created with.
//
// When the config records tool decisions (tools.enabled/disabled), they are
// authoritative for the hook and agent-tool toggles and are replayed instead
// of the agent-tool defaults.
func applyJoinDefaults(a *types.WizardAnswers, defaults types.DefaultsProvider, registry *toolreg.Registry) {
	toolsRecorded := len(a.EnabledTools) > 0

	filled := *a
	filled.Languages = slices.Clone(a.Languages)
	filled.MCPServers = slices.Clone(a.MCPServers)
	filled.EnvVars = maps.Clone(a.EnvVars)
	filled.EnabledTools = maps.Clone(a.EnabledTools)
	filled.FillDefaults(a.Detected, defaults)

	a.Hooks = filled.Hooks
	if toolsRecorded {
		restoreToolToggles(a, registry)
		return
	}
	a.AgentTools = filled.AgentTools
	a.EnabledTools = filled.EnabledTools
}

// restoreToolToggles replays each recorded tool decision's enable/disable
// function, keeping only its effect on the hook and agent-tool toggles. The
// MCP-server and skill lists are persisted verbatim under claude_code and stay
// authoritative, so a tool's edits to them are not replayed.
func restoreToolToggles(a *types.WizardAnswers, registry *toolreg.Registry) {
	for _, name := range slices.Sorted(maps.Keys(a.EnabledTools)) {
		tool, ok := registry.ByName(name)
		if !ok {
			continue
		}
		apply := (func(*types.WizardAnswers))(tool.DisableFunc)
		if a.EnabledTools[name] {
			apply = tool.EnableFunc
		}
		if apply == nil {
			continue
		}
		scratch := *a
		scratch.MCPServers = slices.Clone(a.MCPServers)
		scratch.Skills = slices.Clone(a.Skills)
		apply(&scratch)
		a.Hooks = scratch.Hooks
		a.AgentTools = scratch.AgentTools
	}
}

// runControlFlags are init flags that steer the run rather than describe the
// project, so they are never "ignored" configuration.
var runControlFlags = map[string]bool{
	"mode": true, "yes": true, "quiet": true, "theme": true, "dry-run": true, "force": true, "merge": true,
}

// warnIgnoredInitFlags reports configuration flags given to a run that
// generates nothing because the project is already set up, instead of
// dropping them silently.
func warnIgnoredInitFlags(cmd *cobra.Command) {
	var ignored []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if !runControlFlags[f.Name] {
			ignored = append(ignored, "--"+f.Name)
		}
	})
	if len(ignored) == 0 {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Note: ignoring %s because the project is already set up, so nothing was generated or previewed.\n"+
		"To regenerate the project from these flags, re-run with --yes --force (add --dry-run to preview first).\n",
		strings.Join(ignored, ", "))
}

// writeJoinResults writes generated files, records state, saves answers, and
// prints the join summary.
func writeJoinResults(
	cmd *cobra.Command,
	opts InitOptions,
	projectRoot string,
	answers types.WizardAnswers,
	accResult accumulatorResult,
	allFiles []types.GeneratedFile,
) error {
	// Write files. The merge funcs preserve user-owned keys (e.g. settings.json
	// "env", per-server .mcp.json fields, CLAUDE.md sections outside markers)
	// when join writes over an existing, unrecorded file.
	result, err := generate.WriteFiles(allFiles, generate.PipelineOptions{
		ProjectRoot:       projectRoot,
		SectionMergeFunc:  merge.SectionMarkersOrAppend,
		ThreeWayMergeFunc: merge.MergeOnCreate,
	})
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Record state with QsdevVersion (only for successfully written files).
	successfulFiles := result.SuccessfulFiles(allFiles)
	genState := state.RecordFiles(successfulFiles)
	genState.QsdevVersion = version.Info().Version
	genState.EnabledTools = answers.EnabledTools
	genState.Fragments = state.RecordFragments(accResult.fragments)
	stampTemplateVersions(&genState, accResult.claudeGenerated)
	stateFile := filepath.Join(projectRoot, stateFilePath())
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// Save answers, also after a partial write: they are what a re-run and
	// repair regenerate from.
	if err := saveAddonAnswers(cmd, projectRoot, answers, accResult); err != nil {
		return err
	}
	if result.HasFailures() {
		return partialWriteError(result, len(successfulFiles), "'"+branding.Get().AppName+" init --mode join'")
	}

	// Print join-specific summary.
	if !opts.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), joinOutcome(result))
		_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, accResult))
	}

	return nil
}

// joinOutcome is the join result line. It reports success only when every
// file matches what the committed config generates: a committed file with
// changes the config does not describe (e.g. a hand edit to devenv.nix) is
// kept and gets a sidecar, which the teammate must reconcile.
func joinOutcome(result generate.WriteResult) string {
	var sidecars int
	for _, fr := range result.Files {
		if fr.SidecarPath != "" {
			sidecars++
		}
	}
	cfgFile := branding.Get().ConfigFile
	if sidecars == 0 {
		return fmt.Sprintf("Joined project successfully from %s configuration.", cfgFile)
	}
	return fmt.Sprintf("Joined project from %s configuration, but %d committed file(s) differ from what it generates; "+
		"they were kept unchanged. Merge the %s file(s) listed above, or record the change with %s commands so %s describes it.",
		cfgFile, sidecars, generate.SidecarSuffix, branding.Get().AppName, cfgFile)
}
