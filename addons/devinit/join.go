package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
	gdevconfig "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/config"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/detect"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/generate"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/profile"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// runJoin sets up a local development environment from an existing .gdev.yaml.
// This is the "join" path for new team members cloning a project.
func runJoin(cmd *cobra.Command, opts InitOptions, projectRoot string) error {
	// 1. Parse .gdev.yaml.
	gdevYaml := filepath.Join(projectRoot, ".gdev.yaml")
	cfg, err := gdevconfig.ParseGdevConfig(gdevYaml)
	if err != nil {
		return fmt.Errorf("parsing .gdev.yaml: %w", err)
	}

	// 2. Run detection.
	detected := detect.Detect(projectRoot)

	// 3. Convert config to answers using temporary bridge function.
	answers := configToAnswersTemp(cfg, detected, projectRoot)

	// 3b. If --answers-file is set, merge file answers over config answers.
	if opts.AnswersFile != "" {
		fileAnswers, err := LoadAnswersFile(opts.AnswersFile)
		if err != nil {
			return err
		}
		fileAnswers.ProjectRoot = projectRoot
		fileAnswers.ProjectName = filepath.Base(projectRoot)

		flagSet := NewFlagSet(cmd)
		changed := flagSetToChangedMap(flagSet, cmd)
		flagAnswers, err := AnswersFromFlags(opts, projectRoot)
		if err != nil {
			return err
		}
		answers = MergeFileWithFlags(fileAnswers, flagAnswers, changed)

		if err := ValidateAnswersFileCompleteness(answers); err != nil {
			return err
		}

		answers.Confirmed = true
		answers.Detected = detected
	}

	// 4. Check prerequisites.
	prereqs := CheckPrerequisites(cmd.Context())
	if prereqs.HasMissing() {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Prerequisite check:")
		prereqs.PrintReport(cmd.OutOrStdout())
		return fmt.Errorf("missing required prerequisites; install them and retry")
	}

	// 5. Validate answers.
	if err := ValidateAnswers(answers); err != nil {
		return err
	}

	// 6. Generate files.
	registry := ecosystem.DefaultRegistry()
	var allFiles []types.GeneratedFile
	devenvGenerated := false
	claudeGenerated := false

	gen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
	files, err := gen.Generate(answers)
	if err != nil {
		return fmt.Errorf("generating devenv files: %w", err)
	}
	allFiles = append(allFiles, files...)
	devenvGenerated = len(files) > 0

	if answers.ClaudeCode {
		cgen := claudecode.NewClaudeCodeGenerator(registry, claudecode.Config{})
		cfiles, err := cgen.Generate(answers)
		if err != nil {
			return fmt.Errorf("generating Claude Code files: %w", err)
		}
		allFiles = append(allFiles, cfiles...)
		claudeGenerated = len(cfiles) > 0
	}

	// 7. Generate .gdev.local.yaml template (only if it doesn't exist).
	localConfigPath := filepath.Join(projectRoot, ".gdev.local.yaml")
	if _, err := os.Stat(localConfigPath); os.IsNotExist(err) {
		localContent := GenerateLocalConfigTemplate(answers, detected)
		allFiles = append(allFiles, types.GeneratedFile{
			Path:           ".gdev.local.yaml",
			Content:        localContent,
			Mode:           0o644,
			SkipValidation: true,
		})
	}

	// 8. Ensure .gdev.local.yaml is in .gitignore.
	if err := EnsureGitignoreEntry(projectRoot, ".gdev.local.yaml"); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	// 9. Dry-run: preview and return.
	if opts.DryRun {
		preview := generate.PreviewFiles(allFiles, nil, projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil
	}

	// 10. Write files.
	result, err := generate.WriteFiles(allFiles, generate.PipelineOptions{
		ProjectRoot: projectRoot,
	})
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// 11. Record state with GdevVersion (only for successfully written files).
	successfulFiles := result.SuccessfulFiles(allFiles)
	genState := state.RecordFiles(successfulFiles)
	genState.GdevVersion = version.Info().Version
	stateFile := filepath.Join(projectRoot, statePath)
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	if result.HasFailures() {
		return fmt.Errorf("partial write: %d files failed (state saved for %d successful files); run gdev repair to recover",
			result.Failed, len(successfulFiles))
	}

	// 12. Save answers.
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

	// 13. Print join-specific summary.
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Joined project successfully from .gdev.yaml configuration.")
	_, _ = fmt.Fprint(cmd.OutOrStdout(), postGenerationMessage(answers, devenvGenerated, claudeGenerated))

	return nil
}

// configToAnswersTemp is a temporary bridge function that converts a parsed
// GdevConfig into WizardAnswers. This will be replaced by the full resolution
// engine (Unit 13.2) when it lands.
func configToAnswersTemp(cfg *types.GdevConfig, detected types.DetectedProject, projectRoot string) types.WizardAnswers {
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
