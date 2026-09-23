package devinit

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/repair"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func stateFilePath() string {
	return state.InitStateFile()
}

func initCmd() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a security-hardened development environment",
		Long: `Initialize a complete development environment with security hardening.

Generates devenv.sh configuration (devenv.yaml, devenv.nix, .envrc) and
Claude Code configuration (.claude/settings.json, CLAUDE.md, hooks, skills)
for the current project. Detects existing languages and frameworks, applies
project-type profiles, and writes all files atomically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Update {
				return runUpdate(cmd, updateOptionsFromInit(opts))
			}
			return runInitWithModeDetection(cmd, opts)
		},
	}

	RegisterInitFlags(cmd, &opts)

	return cmd
}

// updateOptionsFromInit maps init flags onto the update flow. For init,
// --force means "overwrite", which covers both user-modified files and the
// version ratchet.
func updateOptionsFromInit(opts InitOptions) UpdateOptions {
	return UpdateOptions{
		Force:          opts.Force,
		AllowDowngrade: opts.Force,
		DryRun:         opts.DryRun,
	}
}

// runInitWithModeDetection auto-detects the onboarding mode and dispatches
// to the appropriate handler (create, join, update, repair).
func runInitWithModeDetection(cmd *cobra.Command, opts InitOptions) error {
	// a. Get project root. init creates (or re-initializes) the project in the
	// directory it is run from, so it does not walk up to an enclosing project.
	projectRoot, err := cmdutil.WorkingDir()
	if err != nil {
		return err
	}

	// b. Handle --list-profiles early return.
	if opts.ListProfiles {
		return listProfiles(cmd)
	}

	// c. Detect or override mode.
	var result *ModeDetectionResult
	if opts.Mode != "" {
		result, err = overrideMode(opts.Mode)
		if err != nil {
			return err
		}
	} else {
		result, err = DetectOnboardingMode(projectRoot)
		if err != nil {
			return fmt.Errorf("detecting onboarding mode: %w", err)
		}
	}

	// --force on a project that is already set up re-creates it; say so
	// instead of announcing "Nothing to do" and then regenerating everything.
	if result.Mode == ModeJoin && result.AlreadySetUp && opts.Force {
		result = &ModeDetectionResult{
			Mode:        ModeCreate,
			Explanation: "Project is already set up; --force regenerates its configuration from the given flags and detection.",
		}
	}

	slog.Info("onboarding mode detected", "mode", result.Mode)

	// d. Print explanation.
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", result.Mode, result.Explanation)

	// e. Dispatch to appropriate handler.
	switch result.Mode {
	case ModeCreate:
		return runCreate(cmd, opts, projectRoot)
	case ModeJoin:
		if result.AlreadySetUp {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Project is already set up.")
			warnIgnoredInitFlags(cmd)
			return nil
		}
		return runJoin(cmd, opts, projectRoot)
	case ModeUpdate:
		return runUpdate(cmd, updateOptionsFromInit(opts))
	case ModeRepair:
		return runRepair(cmd, opts)
	default:
		return fmt.Errorf("unexpected onboarding mode: %s", result.Mode)
	}
}

// runCreate is the original init flow for creating a project from scratch.
func runCreate(cmd *cobra.Command, opts InitOptions, projectRoot string) error {
	flagSet := NewFlagSet(cmd)
	detected := detect.Detect(cmd.Context(), projectRoot)
	slog.Debug("ecosystem detection complete",
		"ecosystems", len(detected.Ecosystems),
		"has_go", detected.HasGoMod,
		"has_node", detected.HasPackageJSON,
		"has_devenv_nix", detected.HasDevenvNix)

	if !detected.IsGitRepo {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: not a git repository. Some features (hooks, branch naming) require git.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Run 'git init' to initialize a repository.")
		fmt.Fprintln(cmd.ErrOrStderr())
	}

	warnOrInstallPrereqs(cmd, opts)

	answers, err := buildAnswersFromInputs(cmd, opts, projectRoot, detected, flagSet)
	if err != nil {
		return err
	}
	if !answers.Confirmed {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	if !opts.Force && !opts.Merge {
		existing := DetectExistingConfig(detected)
		if existing.NeedsMergeMode() {
			return fmt.Errorf("existing configuration found (%s); use --merge to merge %s configuration into it, or --force to overwrite",
				strings.Join(existing.Files, ", "), branding.Get().AppName)
		}
	}

	// Persist the generation scope so update and repair keep honouring it.
	applyScopeFlags(opts, &answers)
	accResult, err := runAccumulator(answers, scopeFromAnswers(answers))
	if err != nil {
		return fmt.Errorf("generating files: %w", err)
	}

	if opts.DryRun {
		preview := generate.PreviewFiles(accResult.allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
	}

	// Ignore the state directories before anything is written into them, so
	// a partial write never leaves them to be committed by `git add -A`.
	ensureProjectGitignore(projectRoot, answers)

	if err := writeAndRecordResults(cmd, opts, projectRoot, answers, accResult); err != nil {
		return err
	}

	return finalizeProject(cmd, opts, answers, projectRoot, accResult)
}

func warnOrInstallPrereqs(cmd *cobra.Command, opts InitOptions) {
	if opts.ClaudeOnly || opts.DryRun {
		return
	}
	prereqs := CheckPrerequisites(cmd.Context())
	if !prereqs.HasMissing() {
		return
	}
	if opts.Yes {
		if err := devenv.AutoSetupPrerequisites(cmd.Context(), cmd.ErrOrStderr()); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: prerequisite installation failed: %v\n", err)
			fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' manually after init.\n\n", branding.Get().AppName)
		}
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), "Note: some prerequisites are missing:")
		prereqs.PrintReport(cmd.ErrOrStderr())
		fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' after init to install them.\n", branding.Get().AppName)
		fmt.Fprintln(cmd.ErrOrStderr())
	}
}

func buildAnswersFromInputs(cmd *cobra.Command, opts InitOptions, projectRoot string, detected types.DetectedProject, flagSet *FlagSet) (types.WizardAnswers, error) {
	answers, err := AnswersFromFlags(opts, projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}

	if opts.AnswersFile != "" {
		fileAnswers, err := LoadAnswersFile(opts.AnswersFile)
		if err != nil {
			return types.WizardAnswers{}, err
		}
		fileAnswers.ProjectRoot = projectRoot
		fileAnswers.ProjectName = filepath.Base(projectRoot)

		changed := flagSetToChangedMap(flagSet, cmd)
		answers = MergeFileWithFlags(fileAnswers, answers, changed)

		if err := ValidateAnswersFileCompleteness(answers); err != nil {
			return types.WizardAnswers{}, err
		}

		answers.Confirmed = true
		answers.Detected = detected
	}

	if opts.ProfileName != "" {
		p, ok := ensureProfileRegistry().Get(opts.ProfileName)
		if !ok {
			return types.WizardAnswers{}, fmt.Errorf("unknown profile %q; use --list-profiles to see available profiles", opts.ProfileName)
		}
		profileAnswers, err := ProfileToAnswers(p, projectRoot, filepath.Base(projectRoot))
		if err != nil {
			return types.WizardAnswers{}, fmt.Errorf("profile %q: %w", opts.ProfileName, err)
		}
		// A profile does not configure agent tools, so start from the flag
		// answers (flag defaults included); otherwise --profile would silently
		// disable the postmortem and Version-Sentinel guardrails.
		profileAnswers.AgentTools = answers.AgentTools
		changed := flagSetToChangedMap(flagSet, cmd)
		answers = MergeProfileWithFlags(profileAnswers, answers, changed)
	}

	answers.Detected = detected

	if err := ValidateAnswers(answers); err != nil {
		return types.WizardAnswers{}, err
	}

	if opts.Yes {
		answers.Confirmed = true
	}
	// Answers that are already confirmed (--yes, --profile, --answers-file)
	// skip the wizard, so they all get the same catalog defaults.
	if answers.Confirmed {
		cat, err := catalog.Default()
		if err != nil {
			return types.WizardAnswers{}, fmt.Errorf("loading catalog for defaults: %w", err)
		}
		answers.FillDefaults(detected, cat)
	}

	if opts.Yes && !answers.IsComplete() {
		missing := incompleteAnswersMessage(answers)
		return types.WizardAnswers{}, fmt.Errorf("non-interactive mode (--yes) requires complete answers; missing:\n%s\nProvide --lang, --profile, or run in a project with detectable language files", missing)
	}

	if !answers.IsComplete() && !opts.Yes {
		answers, err = runInitWizard(opts, projectRoot, detected, answers, flagSet)
		if err != nil {
			return types.WizardAnswers{}, err
		}
	}

	// Infer tools only from the final answers: tools inferred from the
	// pre-wizard flags would outlive a wizard choice that turned them off.
	treg, err := toolreg.Default()
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("loading tool registry: %w", err)
	}
	toolreg.MergeInferredTools(&answers, treg)

	enforceAnswerInvariants(&answers)
	return answers, nil
}

// runInitWizard collects the remaining answers interactively and validates
// the result the same way as non-interactive answers.
func runInitWizard(opts InitOptions, projectRoot string, detected types.DetectedProject, partial types.WizardAnswers, flagSet *FlagSet) (types.WizardAnswers, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return types.WizardAnswers{}, fmt.Errorf("stdin is not a terminal; use --yes or --profile for non-interactive mode")
	}
	answers, err := RunWizard(projectRoot, detected, partial, flagSet, opts.Theme)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("running wizard: %w", err)
	}
	if !answers.Confirmed {
		return answers, nil
	}
	if err := ValidateAnswers(answers); err != nil {
		return types.WizardAnswers{}, err
	}
	return answers, nil
}

func writeAndRecordResults(cmd *cobra.Command, opts InitOptions, projectRoot string, answers types.WizardAnswers, accResult accumulatorResult) error {
	result, err := generate.WriteFiles(accResult.allFiles, generate.PipelineOptions{
		ProjectRoot:       projectRoot,
		Force:             opts.Force,
		SectionMergeFunc:  merge.SectionMarkersOrAppend,
		ThreeWayMergeFunc: merge.MergeOnCreate,
	})
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	if missing := generate.VerifyWritten(result, projectRoot); len(missing) > 0 {
		slog.Warn("post-generation verification: some files not found on disk", "missing", missing)
	}

	successfulFiles := result.SuccessfulFiles(accResult.allFiles)
	genState := state.RecordFiles(successfulFiles)
	genState.QsdevVersion = version.Info().Version
	genState.EnabledTools = answers.EnabledTools
	genState.Fragments = state.RecordFragments(accResult.fragments)
	stampTemplateVersions(&genState, accResult.claudeGenerated)
	stateFile := filepath.Join(projectRoot, stateFilePath())
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	slog.Info("files written",
		"created", result.Created,
		"updated", result.Updated,
		"skipped", result.Skipped,
		"failed", result.Failed)

	// The answers are saved even when some files failed: they are the input
	// a re-run and repair regenerate from.
	if err := saveAddonAnswers(cmd, projectRoot, answers, accResult); err != nil {
		return err
	}
	if result.HasFailures() {
		// .qsdev.yaml is only written once every file is, so the project
		// stays in create mode and the re-run finishes the setup.
		// The re-run builds its answers from its own flags again, so it must
		// repeat them; --merge gets past the files this run did write.
		rerun := "the same init command"
		if !opts.Force && !opts.Merge {
			rerun += " with --merge added"
		}
		return partialWriteError(result, len(successfulFiles), rerun)
	}

	if !opts.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
	}

	return nil
}

// partialWriteError reports the files a write failed on and the command that
// finishes the setup once their errors are fixed.
func partialWriteError(result generate.WriteResult, recorded int, rerun string) error {
	var details strings.Builder
	for _, ff := range result.FailedFiles() {
		fmt.Fprintf(&details, "\n  - %s: %v", ff.Path, ff.Error)
	}
	return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); fix the errors below and re-run %s to finish setup%s",
		result.Failed, recorded, rerun, details.String())
}

// saveAddonAnswers persists answers to the primary answers file and to each
// generated addon's copy.
func saveAddonAnswers(cmd *cobra.Command, projectRoot string, answers types.WizardAnswers, accResult accumulatorResult) error {
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	if accResult.devenvGenerated {
		if err := devenv.SaveAnswers(projectRoot, answers); err != nil {
			return fmt.Errorf("saving devenv answers: %w", err)
		}
	}
	if accResult.claudeGenerated {
		if err := claudecode.SaveAnswers(projectRoot, answers); err != nil {
			return fmt.Errorf("saving Claude Code answers: %w", err)
		}
		// Never report unqualified success when configured skills/MCP servers
		// were suppressed by the resolved tier.
		for _, w := range claudecode.SuppressedConfigWarnings(answers) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: "+w)
		}
	}
	return nil
}

// stampTemplateVersions records the Claude Code template and skill-library
// versions that produced the tracked files, so a later update reports a
// template change only when one actually happened.
func stampTemplateVersions(st *types.GeneratedState, claudeGenerated bool) {
	if !claudeGenerated {
		return
	}
	st.TemplateVersion = claudecode.ComputeTemplateVersion()
	st.SkillLibraryVersion = claudecode.ComputeSkillLibraryVersion()
}

// projectGitignoreEntries are the local-state paths every initialized project
// ignores: qsdev's state directories, the machine-specific local overrides file
// (join also ignores it, so init must too or every teammate's first join
// dirties .gitignore), the devenv/direnv caches, and the audit logs Claude Code
// hooks write under .claude/ (tool inputs and commands that can hold secrets,
// and must never be committed with the rest of .claude/).
func projectGitignoreEntries() []string {
	b := branding.Get()
	return []string{
		b.StateDir + "/", "." + b.AppName + "/", b.LocalConfig, ".direnv/", ".devenv/",
		claudecode.AddonDir + "/logs/", claudecode.AddonDir + "/hook-audit.log*",
	}
}

func finalizeProject(cmd *cobra.Command, opts InitOptions, answers types.WizardAnswers, projectRoot string, accResult accumulatorResult) error {
	qsdevCfg := qsdevconfig.AnswersToConfig(answers, version.Info().Version)
	qsdevCfgPath := filepath.Join(projectRoot, branding.Get().ConfigFile)
	if err := qsdevconfig.WriteProjectConfig(qsdevCfgPath, qsdevCfg); err != nil {
		return err
	}

	if !opts.Quiet {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, accResult))
	}

	return nil
}

// ensureProjectGitignore adds the qsdev state directories and the
// language-specific entries to .gitignore. Failures are logged, not fatal.
func ensureProjectGitignore(projectRoot string, answers types.WizardAnswers) {
	for _, entry := range projectGitignoreEntries() {
		if err := EnsureGitignoreEntry(projectRoot, entry); err != nil {
			slog.Warn("could not update .gitignore", "entry", entry, "error", err)
		}
	}

	var langNames []string
	for _, lc := range answers.Languages {
		langNames = append(langNames, lc.Name)
	}
	for _, entry := range gitignoreEntriesForLanguages(langNames) {
		if err := EnsureGitignoreEntry(projectRoot, entry); err != nil {
			slog.Warn("could not update .gitignore", "entry", entry, "error", err)
		}
	}
}

// runRepair delegates to the full repair command logic, which computes its
// own drift report.
func runRepair(cmd *cobra.Command, opts InitOptions) error {
	return runRepairCommand(cmd, repair.RepairOptions{
		Force:  opts.Force,
		DryRun: opts.DryRun,
	})
}

// listProfiles prints all available project-type profiles and returns.
func listProfiles(cmd *cobra.Command) error {
	profiles := ensureProfileRegistry().List()
	if len(profiles) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No profiles available.")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %s\n", "Profile", "Description")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 70))
	for _, p := range profiles {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %s\n", p.Name, p.Description)
	}
	return nil
}

// flagSetToChangedMap converts a FlagSet into the map[string]bool format
// expected by MergeProfileWithFlags. The keys use the WizardAnswers field
// names (not the CLI flag names).
func flagSetToChangedMap(fs *FlagSet, cmd *cobra.Command) map[string]bool {
	changed := make(map[string]bool)

	// Map CLI flag names to WizardAnswers field names used by MergeProfileWithFlags.
	flagToField := map[string]string{
		"lang":                    "languages",
		"service":                 "services",
		"direnv":                  "direnv",
		"claude-code":             "claude_code",
		"claude-permissions":      "permission_level",
		"claude-skills":           "skills",
		"claude-hooks":            "hooks",
		"git-hooks":               "git_hooks",
		"packages":                "extra_packages",
		"mcp":                     "mcp_servers",
		"tier":                    "tier",
		"infra-profile":           "profile_name",
		"yes":                     "confirmed",
		"env":                     "env_vars",
		"nix-hardening-guide":     "nix_hardening_guide",
		"profile":                 "project_type_profile",
		"agent-postmortem":        "agent_postmortem",
		"agent-version-sentinel":  "agent_version_sentinel",
		"agent-semble":            "agent_semble",
		"agent-semble-mode":       "agent_semble_mode",
		"agent-semble-text-files": "agent_semble_text_files",
	}

	for flagName, fieldName := range flagToField {
		if fs.IsSet(flagName) {
			changed[fieldName] = true
		}
	}

	// Language-specific flags implicitly change the languages field.
	langFlags := []string{
		"go-version", "node-version", "node-pkg-mgr",
		"python-version", "python-pkg-mgr", "rust-channel",
		"java-version", "java-build-tool",
	}
	for _, lf := range langFlags {
		if fs.IsSet(lf) {
			changed["languages"] = true
			break
		}
	}

	return changed
}
