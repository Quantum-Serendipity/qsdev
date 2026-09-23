package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// AddonDir is the project-relative directory used by the claudecode addon for
// its configuration and state files.
const AddonDir = ".claude"

// statePath returns the path to the claude state file, using the branding app name.
func statePath() string {
	return state.ClaudeStateFile()
}

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
		hooksCmd(),
	)

	return cmd
}

func initCmd() *cobra.Command {
	var (
		preset        string
		skills        []string
		mcpServers    []string
		yes           bool
		force         bool
		dryRun        bool
		noSafetyBlock bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Claude Code configuration for the project",
		Long:  "Generate .claude/settings.json, CLAUDE.md, hooks, skills, and rules for the current project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate permission preset before any work.
			if !validation.IsValidPermissionPreset(preset) {
				return fmt.Errorf("unknown permission preset %q; valid presets: %v", preset, validation.PermissionPresets())
			}

			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Check for existing settings.json unless --force is set.
			if !force {
				settingsPath := filepath.Join(projectRoot, ".claude", "settings.json")
				if _, err := os.Stat(settingsPath); err == nil {
					return fmt.Errorf(".claude/settings.json already exists; use --force to overwrite")
				}
			}

			// Detect project characteristics.
			detected := detect.Detect(cmd.Context(), projectRoot)

			// Build answers from flags, overlaid onto any saved answers so the
			// rest of the project's configuration survives a re-init.
			answers, err := overlayInitAnswers(projectRoot,
				buildClaudeAnswersFromFlags(projectRoot, preset, skills, mcpServers, yes, noSafetyBlock))
			if err != nil {
				return err
			}
			answers.Detected = detected

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, addon.Config)
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

			// Write files to disk. ThreeWayMergeFunc preserves user-owned
			// top-level keys (e.g. settings.json "env") when --force overwrites
			// an existing file.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot:       projectRoot,
				SectionMergeFunc:  merge.SectionMarkersOrAppend,
				ThreeWayMergeFunc: merge.MergeOnCreate,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers via the shared tail, version-stamped. A
			// re-init (--force) keeps state this command does not own, such as
			// MCP server lifecycle records from `mcp install`.
			stateFile := filepath.Join(projectRoot, statePath())
			existingState, err := state.LoadStateFromFile(stateFile)
			if err != nil {
				// init is the recovery path for a broken project, so an
				// unreadable state file is replaced rather than fatal.
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v; starting a fresh state file\n", err)
				existingState = types.GeneratedState{}
			}
			if err := persistRegenState(stateFile, projectRoot, answers, result.SuccessfulFiles(files), nil, existingState, true); err != nil {
				return err
			}

			// Warn when configured skills/MCP servers were suppressed by tier.
			for _, w := range SuppressedConfigWarnings(answers) {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: "+w)
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
	cmd.Flags().BoolVar(&noSafetyBlock, "no-safety-block", false, "Disable the safety block hook")

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
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Refresh detection.
			answers.Detected = detect.Detect(cmd.Context(), projectRoot)

			// Load stored state.
			stateFile := filepath.Join(projectRoot, statePath())
			existingState, err := state.LoadStateFromFile(stateFile)
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			// Generate new files.
			registry := ecosystem.DefaultRegistry()
			gen := NewClaudeCodeGenerator(registry, addon.Config)
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

			// Write files through the same merge-aware writer as add-skill and
			// add-hook, keeping user edits to non-mergeable files unless --force.
			opts := regenWriteOptions{force: force, keepUserChanges: true}
			plan, err := planRegenWrites(files, existingState, projectRoot, opts, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if err := applyRegenPlan(plan, projectRoot, cmd.ErrOrStderr()); err != nil {
				return err
			}

			// Save updated state and answers via the shared tail, stamping the
			// template/skill versions (the update path always version-stamps).
			if err := persistRegenState(stateFile, projectRoot, answers, plan.recorded, plan.mergedOriginals, existingState, true); err != nil {
				return err
			}

			// Print version diff summary if applicable.
			vDiff := CompareVersions(existingState.TemplateVersion, existingState.SkillLibraryVersion)
			if vDiff.NeedsUpdate() {
				summary := BuildUpdateSummary(existingState, files, vDiff)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), summary.String())
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Update complete: %d created, %d updated, %d unchanged, %d skipped.\n",
				plan.created, plan.updated, plan.unchanged, plan.skipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite even if files have been modified")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

// addItemSpec parameterizes the differences between add-skill and add-hook
// commands. The factory function makeAddItemCmd uses it to build a
// cobra.Command with identical control flow but type-specific behavior.
type addItemSpec struct {
	use       string
	short     string
	long      string
	validArgs func() []string

	validate   func(name string) error
	mutate     func(a *types.WizardAnswers, name string) error
	successMsg func(name string, filesWritten int) string
	// verify, when set, runs after planning with the set of paths that will
	// hold generated content. It lets a command fail loudly (instead of
	// reporting false success) when the item it was asked to add is not
	// generated — e.g. a skill suppressed by the resolved tier.
	verify func(name string, present map[string]bool) error
}

// makeAddItemCmd builds a cobra.Command that adds an item to the Claude Code
// configuration using the behavior described by spec.
func makeAddItemCmd(spec addItemSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:               spec.use,
		Short:             spec.short,
		Long:              spec.long,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cmdutil.CompleteFrom(spec.validArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate before any I/O.
			if spec.validate != nil {
				if err := spec.validate(name); err != nil {
					return err
				}
			}

			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			if err := spec.mutate(&answers, name); err != nil {
				return err
			}

			// verify runs inside regenerateAndPersist before anything is written
			// or persisted, so a failed check leaves the project untouched
			// instead of wedging a retry.
			verify := func(present map[string]bool) error {
				if spec.verify != nil {
					return spec.verify(name, present)
				}
				return nil
			}
			_, written, err := regenerateAndPersist(cmd, answers, projectRoot, verify)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), spec.successMsg(name, written))
			return nil
		},
	}

	return cmd
}

// regenerateAndPersist generates files from answers, performs a merge-aware
// incremental write (see planRegenWrites), and persists both state and
// answers. It returns the set of relative paths holding generated content and
// the number of files actually written.
//
// verify runs after planning but before anything is written or persisted, so a
// failed check (e.g. a skill suppressed by the current tier) leaves the project
// untouched and does not record the mutated answers, which would wedge a retry
// with "already configured".
func regenerateAndPersist(cmd *cobra.Command, answers types.WizardAnswers, projectRoot string, verify func(present map[string]bool) error) (map[string]bool, int, error) {
	registry := ecosystem.DefaultRegistry()
	gen := NewClaudeCodeGenerator(registry, addon.Config)
	files, err := gen.Generate(answers)
	if err != nil {
		return nil, 0, fmt.Errorf("generating files: %w", err)
	}

	stFile := filepath.Join(projectRoot, statePath())
	existingState, err := state.LoadStateFromFile(stFile)
	if err != nil {
		return nil, 0, fmt.Errorf("loading state: %w", err)
	}

	plan, err := planRegenWrites(files, existingState, projectRoot, regenWriteOptions{}, cmd.ErrOrStderr())
	if err != nil {
		return nil, 0, err
	}
	if verify != nil {
		if err := verify(plan.present); err != nil {
			return nil, 0, err
		}
	}
	if err := applyRegenPlan(plan, projectRoot, cmd.ErrOrStderr()); err != nil {
		return nil, 0, err
	}

	// Save state and answers via the shared tail (no version stamping here).
	if err := persistRegenState(stFile, projectRoot, answers, plan.recorded, plan.mergedOriginals, existingState, false); err != nil {
		return nil, 0, err
	}

	return plan.present, plan.written(), nil
}

// persistRegenState records the given files into the claude state, corrects the
// ThreeWayMerge base content (storing the un-merged generated "ours" content so
// future merges diff against it, not the merged result), carries forward state
// entries for untouched files, optionally stamps the template/skill versions,
// and saves the state file, the answers and the answer-derived keys of
// .qsdev.yaml. It is the shared tail of init,
// update, add-skill and add-hook.
//
// The new state starts from a copy of existingState, so everything this
// function does not own — MCP server lifecycle records written by
// `mcp install`, enabled tools, the fragment ledger, and the version stamps
// when stampVersions is false — is preserved.
func persistRegenState(
	stateFile, projectRoot string,
	answers types.WizardAnswers,
	recordedFiles []types.GeneratedFile,
	mergedOriginals map[string][]byte,
	existingState types.GeneratedState,
	stampVersions bool,
) error {
	fresh := state.RecordFiles(recordedFiles)
	for path, orig := range mergedOriginals {
		if fs, ok := fresh.Files[path]; ok && fs.Strategy == types.ThreeWayMerge {
			fs.BaseContent = orig
			fresh.Files[path] = fs
		}
	}
	for path, fs := range existingState.Files {
		if _, ok := fresh.Files[path]; !ok {
			fresh.Files[path] = fs
		}
	}

	newState := existingState
	newState.LastRun = fresh.LastRun
	newState.Files = fresh.Files
	if stampVersions {
		newState.TemplateVersion = ComputeTemplateVersion()
		newState.SkillLibraryVersion = ComputeSkillLibraryVersion()
	}
	if err := state.SaveStateToFile(stateFile, newState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}
	// Keep the committed .qsdev.yaml in step so a teammate's join rebuilds
	// the same Claude Code configuration.
	return qsdevconfig.SyncProjectConfig(projectRoot, answers)
}

func addSkillCmd() *cobra.Command {
	return makeAddItemCmd(addItemSpec{
		use:   "add-skill <name>",
		short: "Add a skill to the Claude Code configuration",
		long:  "Add a skill from the built-in library to the existing Claude Code configuration.",
		validate: func(name string) error {
			manifest, err := loadManifest()
			if err != nil {
				return fmt.Errorf("loading skill manifest: %w", err)
			}
			known := make(map[string]bool, len(manifest.Skills))
			for _, s := range manifest.Skills {
				known[s.Name] = true
			}
			if !known[name] {
				return fmt.Errorf("unknown skill %q; available skills are listed by 'qsdev claude list-skills'", name)
			}
			return nil
		},
		mutate: func(a *types.WizardAnswers, name string) error {
			if slices.Contains(a.Skills, name) {
				return fmt.Errorf("skill %q is already configured", name)
			}
			a.Skills = append(a.Skills, name)
			return nil
		},
		verify: func(name string, present map[string]bool) error {
			path := ".claude/skills/" + name + "/SKILL.md"
			if !present[path] {
				return fmt.Errorf("skill %q was not written (expected %s); it is suppressed by the current tier — raise the tier to standard or higher", name, path)
			}
			return nil
		},
		successMsg: func(name string, filesWritten int) string {
			return fmt.Sprintf("Added skill %q. %d file(s) updated.\n", name, filesWritten)
		},
	})
}

func addHookCmd() *cobra.Command {
	return makeAddItemCmd(addItemSpec{
		use:       "add-hook <name>",
		short:     "Enable a hook preset in the Claude Code configuration",
		long:      "Enable a hook preset (auto-format, safety-block, pre-commit, audit-log) in the existing configuration.",
		validArgs: validation.HookPresets,
		validate: func(name string) error {
			if !validation.IsValidHookPreset(name) {
				return fmt.Errorf("unknown hook preset %q; valid presets: %v", name, validation.HookPresets())
			}
			switch name {
			case "auto-format":
				return fmt.Errorf("hook preset %q is not yet implemented; it will be available in a future release", name)
			case "pre-commit":
				return fmt.Errorf("hook preset %q is managed by devenv, not Claude Code; use '%s devenv init' with git hooks enabled", name, branding.Get().AppName)
			}
			return nil
		},
		mutate: func(a *types.WizardAnswers, name string) error {
			if err := a.Hooks.EnableHook(name); err != nil {
				return fmt.Errorf("enabling hook preset: %w", err)
			}
			return nil
		},
		successMsg: func(name string, filesWritten int) string {
			return fmt.Sprintf("Enabled hook %q. %d file(s) updated.\n", name, filesWritten)
		},
	})
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

			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
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
func buildClaudeAnswersFromFlags(projectRoot, preset string, skills, mcpServers []string, yes, noSafetyBlock bool) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot:     projectRoot,
		ProjectName:     filepath.Base(projectRoot),
		ClaudeCode:      true,
		PermissionLevel: preset,
		Skills:          skills,
		MCPServers:      mcpServers,
		Confirmed:       yes,
		Hooks: types.HookChoices{
			SafetyBlock: !noSafetyBlock,
		},
	}

	return answers
}
