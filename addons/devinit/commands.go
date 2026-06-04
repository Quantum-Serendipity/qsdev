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
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/repair"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
)

func stateFilePath() string {
	b := branding.Get()
	return b.StateDir + "/." + b.AppName + "-init-state.yaml"
}

func answersDirectory() string {
	return branding.Get().StateDir
}

func answersFile() string {
	return "." + branding.Get().AppName + "-init-answers.yaml"
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
				return runUpdate(cmd, UpdateOptions{
					Force:  opts.Force,
					DryRun: opts.DryRun,
				})
			}
			return runInitWithModeDetection(cmd, opts)
		},
	}

	RegisterInitFlags(cmd, &opts)

	return cmd
}

// runInitWithModeDetection auto-detects the onboarding mode and dispatches
// to the appropriate handler (create, join, update, repair).
func runInitWithModeDetection(cmd *cobra.Command, opts InitOptions) error {
	// a. Get project root.
	projectRoot, err := cmdutil.ProjectRoot()
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
		result, err = overrideMode(opts.Mode, projectRoot)
		if err != nil {
			return err
		}
	} else {
		result, err = DetectOnboardingMode(projectRoot)
		if err != nil {
			return fmt.Errorf("detecting onboarding mode: %w", err)
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
		if result.AlreadySetUp && opts.Force {
			return runCreate(cmd, opts, projectRoot)
		}
		if result.AlreadySetUp {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Project is already set up.")
			return nil
		}
		return runJoin(cmd, opts, projectRoot)
	case ModeUpdate:
		return runUpdate(cmd, UpdateOptions{
			Force:  opts.Force,
			DryRun: opts.DryRun,
		})
	case ModeRepair:
		return runRepair(cmd, opts, projectRoot, result)
	default:
		return fmt.Errorf("unexpected onboarding mode: %s", result.Mode)
	}
}

// runCreate is the original init flow for creating a project from scratch.
func runCreate(cmd *cobra.Command, opts InitOptions, projectRoot string) error {
	// c. Build FlagSet to track which flags were explicitly set.
	flagSet := NewFlagSet(cmd)

	// d. Run detection.
	detected := detect.Detect(projectRoot)
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

	// d2. Auto-install missing prerequisites if --yes, otherwise warn.
	// Skip for --dry-run: preview should not have side effects.
	if !opts.ClaudeOnly && !opts.DryRun {
		prereqs := CheckPrerequisites(cmd.Context())
		if prereqs.HasMissing() {
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
	}

	// e. Build answers from flags.
	answers, err := AnswersFromFlags(opts, projectRoot)
	if err != nil {
		return err
	}

	// e2. If --answers-file is set, load from file and merge with CLI flag overrides.
	if opts.AnswersFile != "" {
		fileAnswers, err := LoadAnswersFile(opts.AnswersFile)
		if err != nil {
			return err
		}
		fileAnswers.ProjectRoot = projectRoot
		fileAnswers.ProjectName = filepath.Base(projectRoot)

		changed := flagSetToChangedMap(flagSet, cmd)
		answers = MergeFileWithFlags(fileAnswers, answers, changed)

		if err := ValidateAnswersFileCompleteness(answers); err != nil {
			return err
		}

		answers.Confirmed = true
		answers.Detected = detected
	}

	// f. If --profile set, load profile and merge.
	if opts.ProfileName != "" {
		if profileRegistry == nil {
			profileRegistry = DefaultProjectProfileRegistry()
		}
		p, ok := profileRegistry.Get(opts.ProfileName)
		if !ok {
			return fmt.Errorf("unknown profile %q; use --list-profiles to see available profiles", opts.ProfileName)
		}
		profileAnswers := ProfileToAnswers(p, projectRoot, filepath.Base(projectRoot))
		changed := flagSetToChangedMap(flagSet, cmd)
		answers = MergeProfileWithFlags(profileAnswers, answers, changed)
	}

	// g. Set detected results on answers.
	answers.Detected = detected

	// h. Validate answers.
	if err := ValidateAnswers(answers); err != nil {
		return err
	}

	// i. Fill defaults (agent tools, MCP servers, etc.) for any unset fields.
	if opts.Yes {
		answers.Confirmed = true
		answers.FillDefaults(detected)
	}

	// i2. Augment EnabledTools with inferred tools (AlwaysOn, hooks-implied).
	toolreg.MergeInferredTools(&answers, toolreg.DefaultRegistry())

	// i3. In non-interactive mode, bail if answers are incomplete.
	if opts.Yes && !answers.IsComplete() {
		missing := incompleteAnswersMessage(answers)
		return fmt.Errorf("non-interactive mode (--yes) requires complete answers; missing:\n%s\nProvide --lang, --profile, or run in a project with detectable language files", missing)
	}

	// j. Run wizard for missing answers.
	if !answers.IsComplete() && !opts.Yes {
		if !term.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("stdin is not a terminal; use --yes or --profile for non-interactive mode")
		}
		wizardAnswers, err := RunWizard(projectRoot, detected, answers, flagSet, opts.Theme)
		if err != nil {
			return fmt.Errorf("running wizard: %w", err)
		}
		if !wizardAnswers.Confirmed {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
		answers = wizardAnswers
	}

	// k. Check for existing configs unless --force.
	if !opts.Force {
		existing := DetectExistingConfig(detected)
		if existing.NeedsMergeMode() {
			return fmt.Errorf("existing configuration found (%s); use --force to overwrite",
				strings.Join(existing.Files, ", "))
		}
	}

	// l-m. Generate files via fragment accumulation.
	accResult, err := runAccumulator(answers, struct {
		ClaudeOnly bool
		DevenvOnly bool
	}{ClaudeOnly: opts.ClaudeOnly, DevenvOnly: opts.DevenvOnly})
	if err != nil {
		return fmt.Errorf("generating files: %w", err)
	}
	allFiles := accResult.allFiles
	devenvGenerated := accResult.devenvGenerated
	claudeGenerated := accResult.claudeGenerated

	// n. Dry-run: preview and return.
	if opts.DryRun {
		preview := generate.PreviewFiles(allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
	}

	// o. Write files to disk.
	result, err := generate.WriteFiles(allFiles, generate.PipelineOptions{
		ProjectRoot:      projectRoot,
		SectionMergeFunc: merge.SectionMarkers,
	})
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	if missing := generate.VerifyWritten(result, projectRoot); len(missing) > 0 {
		slog.Warn("post-generation verification: some files not found on disk", "missing", missing)
	}

	// p. Save state immediately after write, recording only successfully
	// written files so partial writes leave a recoverable state.
	successfulFiles := result.SuccessfulFiles(allFiles)
	genState := state.RecordFiles(successfulFiles)
	genState.QsdevVersion = version.Info().Version
	genState.EnabledTools = answers.EnabledTools
	genState.Fragments = state.RecordFragments(accResult.fragments)
	stateFile := filepath.Join(projectRoot, stateFilePath())
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	slog.Info("files written",
		"created", result.Created,
		"updated", result.Updated,
		"skipped", result.Skipped,
		"failed", result.Failed)

	if result.HasFailures() {
		var details strings.Builder
		for _, ff := range result.FailedFiles() {
			fmt.Fprintf(&details, "\n  - %s: %v", ff.Path, ff.Error)
		}
		return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); run "+branding.Get().AppName+" repair to recover%s",
			result.Failed, len(successfulFiles), details.String())
	}

	// q. Save unified answers.
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	// Also save per-addon answers so each addon's update command works.
	if devenvGenerated {
		if err := devenv.SaveAnswers(projectRoot, answers); err != nil {
			return fmt.Errorf("saving devenv answers: %w", err)
		}
	}
	if claudeGenerated {
		if err := claudecode.SaveAnswers(projectRoot, answers); err != nil {
			return fmt.Errorf("saving Claude Code answers: %w", err)
		}
	}

	// r. Generate project config.
	qsdevCfg := buildQsdevConfig(answers, version.Info().Version)
	qsdevCfgPath := filepath.Join(projectRoot, branding.Get().ConfigFile)
	if err := writeQsdevConfig(qsdevCfgPath, qsdevCfg); err != nil {
		return fmt.Errorf("writing %s: %w", branding.Get().ConfigFile, err)
	}

	// s. Add managed directories to .gitignore.
	for _, entry := range []string{".devinit/", "." + branding.Get().AppName + "/", ".direnv/", ".devenv/"} {
		if err := EnsureGitignoreEntry(projectRoot, entry); err != nil {
			slog.Warn("could not update .gitignore", "entry", entry, "error", err)
		}
	}

	// s2. Add ecosystem-specific .gitignore entries (build artifacts, secrets).
	var langNames []string
	for _, lc := range answers.Languages {
		langNames = append(langNames, lc.Name)
	}
	for _, entry := range gitignoreEntriesForLanguages(langNames) {
		if err := EnsureGitignoreEntry(projectRoot, entry); err != nil {
			slog.Warn("could not update .gitignore", "entry", entry, "error", err)
		}
	}

	// t. Print summary + post-generation message.
	if !opts.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
		_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, devenvGenerated, claudeGenerated))
	}

	return nil
}

// runRepair delegates to the full repair command logic.
func runRepair(cmd *cobra.Command, opts InitOptions, _ string, _ *ModeDetectionResult) error {
	return runRepairCommand(cmd, repair.RepairOptions{
		Force:  opts.Force,
		DryRun: opts.DryRun,
	})
}

// listProfiles prints all available project-type profiles and returns.
func listProfiles(cmd *cobra.Command) error {
	if profileRegistry == nil {
		profileRegistry = DefaultProjectProfileRegistry()
	}
	profiles := profileRegistry.List()
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
		"lang":               "languages",
		"service":            "services",
		"direnv":             "direnv",
		"claude-code":        "claude_code",
		"claude-permissions": "permission_level",
		"claude-skills":      "skills",
		"claude-hooks":       "hooks",
		"git-hooks":          "git_hooks",
		"packages":           "extra_packages",
		"mcp":                "mcp_servers",
		"tier":               "tier",
		"infra-profile":      "profile_name",
		"yes":                "confirmed",
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
