package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devenv"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/detect"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules" // register all modules
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/generate"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/profile"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/state"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

const (
	statePath       = ".devinit/.gdev-init-state.yaml"
	answersDir      = ".devinit"
	answersFileName = ".gdev-init-answers.yaml"
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
			return runInit(cmd, opts)
		},
	}

	RegisterInitFlags(cmd, &opts)

	return cmd
}

func runInit(cmd *cobra.Command, opts InitOptions) error {
	// a. Get project root.
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	// b. Handle --list-profiles.
	if opts.ListProfiles {
		return listProfiles(cmd)
	}

	// c. Build FlagSet to track which flags were explicitly set.
	flagSet := NewFlagSet(cmd)

	// d. Run detection.
	detected := detect.Detect(projectRoot)

	// e. Build answers from flags.
	answers := AnswersFromFlags(opts, projectRoot)

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

	// p. Save state.
	genState := state.RecordFiles(allFiles)
	stateFile := filepath.Join(projectRoot, statePath)
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
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

	// r. Print summary + post-generation message.
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
	_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, devenvGenerated, claudeGenerated))

	return nil
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
