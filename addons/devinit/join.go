package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
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
	prereqs := CheckPrerequisites(cmd.Context())
	hasMissingPrereqs := prereqs.HasMissing()
	if hasMissingPrereqs {
		if opts.Yes {
			if err := devenv.AutoSetupPrerequisites(cmd.Context(), cmd.ErrOrStderr()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: prerequisite installation failed: %v\n", err)
				fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' manually.\n\n", branding.Get().AppName)
			} else {
				hasMissingPrereqs = false
			}
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), "Note: some prerequisites are missing:")
			prereqs.PrintReport(cmd.ErrOrStderr())
			fmt.Fprintf(cmd.ErrOrStderr(), "Run '%s devenv setup' after join to install them.\n", branding.Get().AppName)
			fmt.Fprintln(cmd.ErrOrStderr())
		}
	}

	// 3. Generate files via fragment accumulation.
	accResult, err := runAccumulator(answers, struct {
		ClaudeOnly bool
		DevenvOnly bool
	}{})
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
			Mode:    0o644,
		})
	}

	// 5. Ensure local config is in .gitignore.
	if err := EnsureGitignoreEntry(projectRoot, localCfg); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	// 6. Dry-run: preview and return.
	if opts.DryRun {
		preview := generate.PreviewFiles(allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
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

// buildJoinAnswers parses the project config, runs detection, converts to
// wizard answers, and optionally merges answers-file overrides.
func buildJoinAnswers(cmd *cobra.Command, opts InitOptions, projectRoot string) (types.WizardAnswers, error) {
	// Parse project config.
	cfgFile := branding.Get().ConfigFile
	cfgPath := filepath.Join(projectRoot, cfgFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("parsing %s: %w", cfgFile, err)
	}

	// Run detection.
	detected := detect.Detect(projectRoot)

	// Convert config to answers for the generation pipeline.
	answers := configToAnswers(cfg, detected, projectRoot)

	// If --answers-file is set, merge file answers over config answers.
	if opts.AnswersFile != "" {
		fileAnswers, err := LoadAnswersFile(opts.AnswersFile)
		if err != nil {
			return types.WizardAnswers{}, err
		}
		fileAnswers.ProjectRoot = projectRoot
		fileAnswers.ProjectName = filepath.Base(projectRoot)

		flagSet := NewFlagSet(cmd)
		changed := flagSetToChangedMap(flagSet, cmd)
		flagAnswers, err := AnswersFromFlags(opts, projectRoot)
		if err != nil {
			return types.WizardAnswers{}, err
		}
		answers = MergeFileWithFlags(fileAnswers, flagAnswers, changed)

		if err := ValidateAnswersFileCompleteness(answers); err != nil {
			return types.WizardAnswers{}, err
		}

		answers.Confirmed = true
		answers.Detected = detected
	}

	// Validate answers.
	if err := ValidateAnswers(answers); err != nil {
		return types.WizardAnswers{}, err
	}

	// Augment EnabledTools with inferred tools (AlwaysOn, hooks-implied).
	toolreg.MergeInferredTools(&answers, toolreg.DefaultRegistry())

	return answers, nil
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
	devenvGenerated := accResult.devenvGenerated
	claudeGenerated := accResult.claudeGenerated

	// Write files.
	result, err := generate.WriteFiles(allFiles, generate.PipelineOptions{
		ProjectRoot: projectRoot,
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
	stateFile := filepath.Join(projectRoot, stateFilePath())
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	if result.HasFailures() {
		var details strings.Builder
		for _, ff := range result.FailedFiles() {
			fmt.Fprintf(&details, "\n  - %s: %v", ff.Path, ff.Error)
		}
		return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); run "+branding.Get().AppName+" repair to recover%s",
			result.Failed, len(successfulFiles), details.String())
	}

	// Save answers.
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}
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

	// Print join-specific summary.
	if !opts.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Joined project successfully from %s configuration.\n", branding.Get().ConfigFile)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, devenvGenerated, claudeGenerated))
	}

	return nil
}

// configToAnswers converts a parsed QsdevConfig into WizardAnswers for use
// by the generation pipeline during join mode.
func configToAnswers(cfg *types.QsdevConfig, detected types.DetectedProject, projectRoot string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectName: filepath.Base(projectRoot),
		ProjectRoot: projectRoot,
		Detected:    detected,
		Confirmed:   true,
		Direnv:      true,
	}

	// Map languages.
	for _, lang := range cfg.Languages {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:           lang.Name,
			Version:        lang.Version,
			PackageManager: lang.PackageManager,
		})
	}

	// Map services.
	for _, svc := range cfg.Services {
		answers.Services = append(answers.Services, types.ServiceChoice{
			Name:    svc.Name,
			Version: svc.Version,
		})
	}

	// Map Claude Code settings.
	if cfg.ClaudeCode.Enabled != nil {
		answers.ClaudeCode = *cfg.ClaudeCode.Enabled
	} else {
		// Default to enabled when not explicitly set.
		answers.ClaudeCode = true
	}
	if cfg.ClaudeCode.PermissionLevel != "" {
		answers.PermissionLevel = cfg.ClaudeCode.PermissionLevel
	} else if answers.ClaudeCode {
		answers.PermissionLevel = "standard"
	}
	answers.Skills = cfg.ClaudeCode.Skills
	answers.MCPServers = cfg.ClaudeCode.MCPServers

	// Map tools.
	if len(cfg.Tools.Enabled) > 0 {
		answers.EnabledTools = make(map[string]bool, len(cfg.Tools.Enabled))
		for _, t := range cfg.Tools.Enabled {
			answers.EnabledTools[t] = true
		}
	}

	// Map tier (infer from legacy fields if not explicit).
	if cfg.Tier != "" {
		answers.Tier = cfg.Tier
	} else {
		answers.Tier = inferTier(cfg)
	}

	// Map profile.
	if cfg.Profile != "" {
		answers.ProjectTypeProfile = cfg.Profile
	}

	// Map infrastructure config.
	answers.Infrastructure = cfg.Infrastructure
	if cfg.Infrastructure.RegistryProxy != "" || cfg.Infrastructure.NixCache != "" || cfg.Infrastructure.BuildCache != "" {
		answers.ProfileName = cfg.Profile
	}

	return answers
}

func inferTier(cfg *types.QsdevConfig) string {
	return tier.Infer(cfg.ClaudeCode.PermissionLevel, cfg.ClaudeCode.MCPServers).String()
}
