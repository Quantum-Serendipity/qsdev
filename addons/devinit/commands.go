package devinit

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/internal/generate"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/repair"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const (
	statePath       = ".devinit/.qsdev-init-state.yaml"
	answersDir      = ".devinit"
	answersFileName = ".qsdev-init-answers.yaml"
)

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
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
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

	// d2. Warn about missing critical dependencies (non-blocking).
	if !opts.ClaudeOnly {
		prereqs := CheckPrerequisites(cmd.Context())
		if prereqs.HasMissing() {
			fmt.Fprintln(cmd.ErrOrStderr(), "Note: some prerequisites are missing:")
			prereqs.PrintReport(cmd.ErrOrStderr())
			fmt.Fprintln(cmd.ErrOrStderr(), "Run 'qsdev devenv setup' after init to install them.")
			fmt.Fprintln(cmd.ErrOrStderr())
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

	// i. If not complete and --yes, fill defaults from detection.
	if !answers.IsComplete() && opts.Yes {
		answers.Confirmed = true
		answers.FillDefaults(detected)
	}

	// j. Run wizard for missing answers.
	if !answers.IsComplete() && !opts.Yes {
		wizardAnswers, err := RunWizard(projectRoot, detected, answers, flagSet)
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

	// Create a shared ecosystem registry for both generators.
	registry := ecosystem.DefaultRegistry()

	var allFiles []types.GeneratedFile
	devenvGenerated := false
	claudeGenerated := false

	// l. Generate devenv files (if not --claude-only).
	if !opts.ClaudeOnly {
		gen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		files, err := gen.Generate(answers)
		if err != nil {
			return fmt.Errorf("generating devenv files: %w", err)
		}
		allFiles = append(allFiles, files...)
		devenvGenerated = len(files) > 0
		slog.Info("devenv files generated", "count", len(files))
	}

	// m. Generate Claude Code files (if not --devenv-only and Claude Code enabled).
	if !opts.DevenvOnly && answers.ClaudeCode {
		gen := claudecode.NewClaudeCodeGenerator(registry, claudecode.Config{})
		files, err := gen.Generate(answers)
		if err != nil {
			return fmt.Errorf("generating Claude Code files: %w", err)
		}
		allFiles = append(allFiles, files...)
		claudeGenerated = len(files) > 0
		slog.Info("claude code files generated", "count", len(files))
	}

	// n. Dry-run: preview and return.
	if opts.DryRun {
		preview := generate.PreviewFiles(allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
	}

	// o. Write files to disk.
	result, err := generate.WriteFiles(allFiles, generate.PipelineOptions{
		ProjectRoot: projectRoot,
	})
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// p. Save state immediately after write, recording only successfully
	// written files so partial writes leave a recoverable state.
	successfulFiles := result.SuccessfulFiles(allFiles)
	genState := state.RecordFiles(successfulFiles)
	genState.QsdevVersion = version.Info().Version
	stateFile := filepath.Join(projectRoot, statePath)
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
		return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); run qsdev repair to recover%s",
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

	// r. Generate .qsdev.yaml project config.
	qsdevCfg := buildQsdevConfig(answers, version.Info().Version)
	qsdevCfgPath := filepath.Join(projectRoot, ".qsdev.yaml")
	if err := writeQsdevConfig(qsdevCfgPath, qsdevCfg); err != nil {
		return fmt.Errorf("writing .qsdev.yaml: %w", err)
	}

	// s. Add managed directories to .gitignore.
	for _, entry := range []string{".devinit/", ".qsdev/", ".direnv/", ".devenv/"} {
		if err := EnsureGitignoreEntry(projectRoot, entry); err != nil {
			slog.Warn("could not update .gitignore", "entry", entry, "error", err)
		}
	}

	// t. Print summary + post-generation message.
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
	_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, devenvGenerated, claudeGenerated))

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
