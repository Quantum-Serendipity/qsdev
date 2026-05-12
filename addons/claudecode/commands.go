package claudecode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/detect"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules" // register all modules
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/generate"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/state"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

const (
	statePath  = ".claude/.gdev-claude-state.yaml"
	answersDir = ".claude"
)

var validPermissionPresets = []string{"minimal", "standard", "permissive", "custom"}
var validHookPresets = []string{"auto-format", "safety-block", "pre-commit", "audit-log"}

func claudeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Manage Claude Code project configuration",
		Long:  "Create, update, and extend Claude Code settings, skills, hooks, and MCP servers.",
	}

	cmd.AddCommand(
		initCmd(),
		updateCmd(),
		addSkillCmd(),
		addHookCmd(),
		listSkillsCmd(),
	)

	return cmd
}

func initCmd() *cobra.Command {
	var (
		preset     string
		skills     []string
		mcpServers []string
		yes        bool
		force      bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Claude Code configuration for the project",
		Long:  "Generate .claude/settings.json, CLAUDE.md, hooks, skills, and rules for the current project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determining project root: %w", err)
			}

			// Check for existing settings.json unless --force is set.
			if !force {
				settingsPath := filepath.Join(projectRoot, ".claude", "settings.json")
				if _, err := os.Stat(settingsPath); err == nil {
					return fmt.Errorf(".claude/settings.json already exists; use --force to overwrite")
				}
			}

			// Detect project characteristics.
			detected := detect.Detect(projectRoot)

			// Build answers from flags.
			answers := buildClaudeAnswersFromFlags(projectRoot, preset, skills, mcpServers, yes)
			answers.Detected = detected

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, Config{})
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Dry-run: show preview and exit.
			if dryRun {
				preview := generate.PreviewFiles(files, nil, projectRoot)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
				return nil
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			genState := state.RecordFiles(files)
			stateFile := filepath.Join(projectRoot, statePath)
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			// Print summary.
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Claude Code configuration generated. Review .claude/settings.json and CLAUDE.md.")

			return nil
		},
	}

	cmd.Flags().StringVar(&preset, "permission-preset", "standard", "Permission preset (minimal, standard, permissive, custom)")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skills to install (e.g. deploy,review-pr)")
	cmd.Flags().StringSliceVar(&mcpServers, "mcp", nil, "MCP servers to configure (e.g. github,filesystem)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

func updateCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Regenerate Claude Code files from saved answers",
		Long:  "Re-run generation using previously saved wizard answers, incorporating any detection changes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determining project root: %w", err)
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Refresh detection.
			answers.Detected = detect.Detect(projectRoot)

			// Check for modified files unless --force is set.
			if !force {
				stateFile := filepath.Join(projectRoot, statePath)
				existingState, err := state.LoadStateFromFile(stateFile)
				if err != nil {
					return fmt.Errorf("loading state: %w", err)
				}
				modified := state.CheckModified(existingState, projectRoot)
				for path, status := range modified {
					if status.Status == types.Modified {
						return fmt.Errorf("file %s has been modified; use --force to overwrite", path)
					}
				}
			}

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, Config{})
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Dry-run: show preview and exit.
			if dryRun {
				preview := generate.PreviewFiles(files, nil, projectRoot)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
				return nil
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			genState := state.RecordFiles(files)
			stateFile := filepath.Join(projectRoot, statePath)
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite even if files have been modified")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

func addSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-skill <name>",
		Short: "Add a skill to the Claude Code configuration",
		Long:  "Add a skill from the built-in library to the existing Claude Code configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := args[0]

			// Validate against manifest.
			manifest, err := loadManifest()
			if err != nil {
				return fmt.Errorf("loading skill manifest: %w", err)
			}
			known := make(map[string]bool, len(manifest.Skills))
			for _, s := range manifest.Skills {
				known[s.Name] = true
			}
			if !known[skillName] {
				return fmt.Errorf("unknown skill %q; available skills are listed by 'gdev claude list-skills'", skillName)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determining project root: %w", err)
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Check for duplicate.
			if contains(answers.Skills, skillName) {
				return fmt.Errorf("skill %q is already configured", skillName)
			}

			// Add skill.
			answers.Skills = append(answers.Skills, skillName)

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, Config{})
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			genState := state.RecordFiles(files)
			stateFile := filepath.Join(projectRoot, statePath)
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added skill %q.\n%s\n", skillName, result.Summary())
			return nil
		},
	}

	return cmd
}

func addHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "add-hook <name>",
		Short:     "Enable a hook preset in the Claude Code configuration",
		Long:      "Enable a hook preset (auto-format, safety-block, pre-commit, audit-log) in the existing configuration.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validHookPresets,
		RunE: func(cmd *cobra.Command, args []string) error {
			hookName := args[0]

			// Validate hook name.
			if !contains(validHookPresets, hookName) {
				return fmt.Errorf("unknown hook preset %q; valid presets: %v", hookName, validHookPresets)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determining project root: %w", err)
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Apply hook preset to answers.
			hookPresetToChoices(hookName, &answers.Hooks)

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, Config{})
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			genState := state.RecordFiles(files)
			stateFile := filepath.Join(projectRoot, statePath)
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enabled hook %q.\n%s\n", hookName, result.Summary())
			return nil
		},
	}

	return cmd
}

func listSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-skills",
		Short: "List available skills from the built-in library",
		Long:  "Show all available skills and mark those that are currently installed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := loadManifest()
			if err != nil {
				return fmt.Errorf("loading skill manifest: %w", err)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determining project root: %w", err)
			}

			// Load answers, tolerating missing file.
			answers, loadErr := loadAnswers(projectRoot)
			installed := make(map[string]bool)
			if loadErr == nil {
				for _, s := range answers.Skills {
					installed[s] = true
				}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-50s  %s\n", "Name", "Description", "Status")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "-------------------------------------------------------------------------------------")

			for _, skill := range manifest.Skills {
				status := ""
				if installed[skill.Name] {
					status = "(installed)"
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-50s  %s\n", skill.Name, skill.Description, status)
			}

			return nil
		},
	}

	return cmd
}

// buildClaudeAnswersFromFlags constructs a WizardAnswers from CLI flag values.
func buildClaudeAnswersFromFlags(projectRoot, preset string, skills, mcpServers []string, yes bool) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot:     projectRoot,
		ProjectName:     filepath.Base(projectRoot),
		ClaudeCode:      true,
		PermissionLevel: preset,
		Skills:          skills,
		MCPServers:      mcpServers,
		Confirmed:       yes,
		Hooks: types.HookChoices{
			SafetyBlock: true,
		},
	}

	return answers
}

// hookPresetToChoices maps a hook preset name to the corresponding field in
// HookChoices, setting it to true.
func hookPresetToChoices(name string, hooks *types.HookChoices) {
	switch name {
	case "auto-format":
		hooks.AutoFormat = true
	case "safety-block":
		hooks.SafetyBlock = true
	case "pre-commit":
		hooks.PreCommit = true
	case "audit-log":
		hooks.AuditLog = true
	}
}
