package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// AddonDir is the project-relative directory used by the claudecode addon for
// its configuration and state files.
const AddonDir = ".claude"

// statePath returns the path to the claude state file, using the branding app name.
func statePath() string {
	return ".claude/." + branding.Get().AppName + "-claude-state.yaml"
}

var validPermissionPresets = validation.PermissionPresets()
var validHookPresets = validation.HookPresets()

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
				return fmt.Errorf("unknown permission preset %q; valid presets: %v", preset, validPermissionPresets)
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
			detected := detect.Detect(projectRoot)

			// Build answers from flags.
			answers := buildClaudeAnswersFromFlags(projectRoot, preset, skills, mcpServers, yes, noSafetyBlock)
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

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot:      projectRoot,
				SectionMergeFunc: merge.SectionMarkers,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			successfulFiles := result.SuccessfulFiles(files)
			genState := state.RecordFiles(successfulFiles)
			stateFile := filepath.Join(projectRoot, statePath())
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
			answers.Detected = detect.Detect(projectRoot)

			// Load stored state.
			stateFile := filepath.Join(projectRoot, statePath())
			existingState, err := state.LoadStateFromFile(stateFile)
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			// Check modification status of all stored files.
			modStatus := state.CheckModified(existingState, projectRoot)

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

			// Write files respecting merge strategies.
			var writtenFiles []types.GeneratedFile
			mergedOriginals := make(map[string][]byte)
			created, updated, skipped := 0, 0, 0

			for _, f := range files {
				absPath := filepath.Join(projectRoot, f.Path)
				mode := f.Mode
				if mode == 0 {
					mode = fileutil.ModeReadWrite
				}

				fs, inState := modStatus[f.Path]
				if !inState {
					// New file — create it.
					if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
						return fmt.Errorf("writing %s: %w", f.Path, err)
					}
					writtenFiles = append(writtenFiles, f)
					created++
					continue
				}

				switch fs.Status {
				case types.Unmodified:
					switch f.Strategy {
					case types.SectionMarker, types.ThreeWayMerge:
						content, mergeErr := mergeFile(f, existingState, projectRoot)
						if mergeErr != nil {
							if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
								return fmt.Errorf("writing %s: %w", f.Path, err)
							}
							writtenFiles = append(writtenFiles, f)
						} else {
							if err := fileutil.WriteFileAtomic(absPath, content, mode); err != nil {
								return fmt.Errorf("writing merged %s: %w", f.Path, err)
							}
							writtenFiles = append(writtenFiles, types.GeneratedFile{
								Path: f.Path, Content: content, Mode: mode, Strategy: f.Strategy,
							})
							mergedOriginals[f.Path] = f.Content
						}
					default:
						if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
							return fmt.Errorf("writing %s: %w", f.Path, err)
						}
						writtenFiles = append(writtenFiles, f)
					}
					updated++

				case types.Modified:
					if force {
						if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
							return fmt.Errorf("writing %s: %w", f.Path, err)
						}
						writtenFiles = append(writtenFiles, f)
						updated++
					} else {
						content, err := mergeFile(f, existingState, projectRoot)
						if err != nil {
							_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: merge failed for %s: %v (skipping)\n", f.Path, err)
							skipped++
							continue
						}
						if err := fileutil.WriteFileAtomic(absPath, content, mode); err != nil {
							return fmt.Errorf("writing merged %s: %w", f.Path, err)
						}
						writtenFiles = append(writtenFiles, types.GeneratedFile{
							Path: f.Path, Content: content, Mode: mode, Strategy: f.Strategy,
						})
						mergedOriginals[f.Path] = f.Content
						updated++
					}

				case types.Deleted:
					if force {
						if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
							return fmt.Errorf("writing %s: %w", f.Path, err)
						}
						writtenFiles = append(writtenFiles, f)
						created++
					} else {
						skipped++
					}

				default:
					skipped++
				}
			}

			// Save updated state (merge new + old for skipped files).
			newState := state.RecordFiles(writtenFiles)
			// Correct BaseContent for merged ThreeWayMerge files: store the
			// original generated content (ours), not the merged result.
			for path, origContent := range mergedOriginals {
				if fs, ok := newState.Files[path]; ok && fs.Strategy == types.ThreeWayMerge {
					fs.BaseContent = origContent
					newState.Files[path] = fs
				}
			}
			for path, fs := range existingState.Files {
				if _, written := newState.Files[path]; !written {
					newState.Files[path] = fs
				}
			}
			newState.TemplateVersion = ComputeTemplateVersion()
			newState.SkillLibraryVersion = ComputeSkillLibraryVersion()
			if err := state.SaveStateToFile(stateFile, newState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			// Print version diff summary if applicable.
			vDiff := CompareVersions(existingState.TemplateVersion, existingState.SkillLibraryVersion)
			if vDiff.NeedsUpdate() {
				summary := BuildUpdateSummary(existingState, files, vDiff)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), summary.String())
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Update complete: %d created, %d updated, %d skipped.\n", created, updated, skipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite even if files have been modified")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

// mergeFile applies the appropriate merge strategy for a modified file.
func mergeFile(f types.GeneratedFile, storedState types.GeneratedState, projectRoot string) ([]byte, error) {
	absPath := filepath.Join(projectRoot, f.Path)
	theirs, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", f.Path, err)
	}

	switch f.Strategy {
	case types.ThreeWayMerge:
		var base []byte
		if fs, ok := storedState.Files[f.Path]; ok {
			base = fs.BaseContent
		}
		if f.Path == ".mcp.json" {
			return merge.MergeMcpJson(base, theirs, f.Content)
		}
		return merge.MergeSettings(base, theirs, f.Content)
	case types.SectionMarker:
		return merge.SectionMarkers(theirs, f.Content)
	case types.LibraryManaged:
		return f.Content, nil
	default:
		return nil, fmt.Errorf("no merge implementation for strategy %s on %s", f.Strategy, f.Path)
	}
}

// addItemSpec parameterizes the differences between add-skill and add-hook
// commands. The factory function makeAddItemCmd uses it to build a
// cobra.Command with identical control flow but type-specific behavior.
type addItemSpec struct {
	use       string
	short     string
	long      string
	validArgs []string

	validate   func(name string) error
	mutate     func(a *types.WizardAnswers, name string) error
	successMsg func(name string, filesWritten int) string
}

// makeAddItemCmd builds a cobra.Command that adds an item to the Claude Code
// configuration using the behavior described by spec.
func makeAddItemCmd(spec addItemSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:       spec.use,
		Short:     spec.short,
		Long:      spec.long,
		Args:      cobra.ExactArgs(1),
		ValidArgs: spec.validArgs,
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

			filesWritten, err := regenerateAndPersist(cmd, answers, projectRoot)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), spec.successMsg(name, filesWritten))
			return nil
		},
	}

	return cmd
}

// regenerateAndPersist generates files from answers, performs a merge-aware
// incremental write, and persists both state and answers. It returns the
// number of files written.
func regenerateAndPersist(cmd *cobra.Command, answers types.WizardAnswers, projectRoot string) (int, error) {
	registry := ecosystem.DefaultRegistry()
	gen := NewClaudeCodeGenerator(registry, addon.Config)
	files, err := gen.Generate(answers)
	if err != nil {
		return 0, fmt.Errorf("generating files: %w", err)
	}

	stFile := filepath.Join(projectRoot, statePath())
	existingState, err := state.LoadStateFromFile(stFile)
	if err != nil {
		return 0, fmt.Errorf("loading state: %w", err)
	}

	diskStatus := state.CheckModified(existingState, projectRoot)

	// Only write files that are new or whose content changed.
	var writtenFiles []types.GeneratedFile
	for _, f := range files {
		absPath := filepath.Join(projectRoot, f.Path)
		mode := f.Mode
		if mode == 0 {
			mode = fileutil.ModeReadWrite
		}

		if fs, ok := existingState.Files[f.Path]; ok {
			newHash := state.ComputeHash(f.Content)
			if newHash == fs.Hash {
				continue
			}

			if ds, found := diskStatus[f.Path]; found && ds.Status == types.Modified {
				merged, mergeErr := mergeFile(f, existingState, projectRoot)
				if mergeErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: merge failed for %s: %v (skipping)\n", f.Path, mergeErr)
					continue
				}
				f.Content = merged
			}
		}

		if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
			return 0, fmt.Errorf("writing %s: %w", f.Path, err)
		}
		writtenFiles = append(writtenFiles, f)
	}

	// Save state (merge new + existing for unchanged files).
	newState := state.RecordFiles(writtenFiles)
	for path, fs := range existingState.Files {
		if _, written := newState.Files[path]; !written {
			newState.Files[path] = fs
		}
	}
	if err := state.SaveStateToFile(stFile, newState); err != nil {
		return 0, fmt.Errorf("saving state: %w", err)
	}
	if err := saveAnswers(projectRoot, answers); err != nil {
		return 0, fmt.Errorf("saving answers: %w", err)
	}

	return len(writtenFiles), nil
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
		validArgs: validHookPresets,
		validate: func(name string) error {
			if !validation.IsValidHookPreset(name) {
				return fmt.Errorf("unknown hook preset %q; valid presets: %v", name, validHookPresets)
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
			hookPresetToChoices(name, &a.Hooks)
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
	case "credential-scan":
		hooks.CredentialScan = true
	case "destructive-prevention":
		hooks.DestructivePrevention = true
	case "soc2-audit":
		hooks.SOC2Audit = true
	case "file-boundary":
		hooks.FileBoundary = true
	case "tool-gates":
		hooks.ToolGates = true
	}
}
