package devinit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/update"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// UpdateOptions holds configuration for the update command.
type UpdateOptions struct {
	Force  bool
	DryRun bool
}

// UpdateAction describes what the update will do to a file.
type UpdateAction int

const (
	UpdateActionRegenerate UpdateAction = iota
	UpdateActionMerge
	UpdateActionSkip
	UpdateActionCreate
	UpdateActionSidecar
)

// FileUpdatePlan describes the planned action for a single file during update.
type FileUpdatePlan struct {
	Path       string
	Status     types.ModificationStatus
	Strategy   types.MergeStrategy
	Action     UpdateAction
	NewContent []byte
	OldContent []byte
	NewMode    os.FileMode
	Reason     string
}

// UpdatePlan holds the complete update plan for all files.
type UpdatePlan struct {
	Files   []FileUpdatePlan
	NixPlan *FileUpdatePlan // separate tracking for devenv.nix
}

func runUpdate(cmd *cobra.Command, opts UpdateOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	// 1. Load saved answers — fail if none exist.
	answers, err := loadAnswers(projectRoot)
	if err != nil {
		return err
	}

	// 2. Refresh detection.
	answers.Detected = detect.Detect(projectRoot)
	answers.ProjectRoot = projectRoot

	// 3. Load stored state.
	stateFile := filepath.Join(projectRoot, stateFilePath())
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// 4. Check modification status of all stored files.
	modStatus := state.CheckModified(existingState, projectRoot)

	// 4b. Version ratchet check — warn if current binary is older than last run.
	if !opts.Force {
		if ratchet := qsdevconfig.CheckVersionRatchet(version.Info().Version, existingState.QsdevVersion); ratchet != nil {
			return ratchet
		}
	}

	// 5. Generate new files from both generators using saved answers.
	registry := ecosystem.DefaultRegistry()
	var allFiles []types.GeneratedFile

	if !answers.ClaudeCode || answers.MergeMode != "claude-only" {
		gen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		files, err := gen.Generate(answers)
		if err != nil {
			return fmt.Errorf("generating devenv files: %w", err)
		}
		allFiles = append(allFiles, files...)
	}

	if answers.ClaudeCode {
		gen := claudecode.NewClaudeCodeGenerator(registry, claudecode.CurrentConfig())
		files, err := gen.Generate(answers)
		if err != nil {
			return fmt.Errorf("generating Claude Code files: %w", err)
		}
		allFiles = append(allFiles, files...)
	}

	// 6. Build update plan.
	plan := buildUpdatePlan(allFiles, modStatus, existingState, opts)

	// 7. Preview.
	if opts.DryRun {
		previewUpdatePlan(plan, cmd.OutOrStdout())
		return nil
	}

	// 8. Show plan summary.
	previewUpdatePlan(plan, cmd.OutOrStdout())

	// 9. Execute plan.
	writtenFiles, nixResult, err := executeUpdatePlan(plan, projectRoot, opts)
	if err != nil {
		return fmt.Errorf("executing update: %w", err)
	}

	// 10. If nix sidecar was created, show instructions.
	if nixResult != nil && nixResult.Action == update.NixSidecarCreated {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), nixResult.DiffOutput)
		fmt.Fprintln(cmd.OutOrStdout(), nixResult.Message)
	}

	// 11. Save updated state.
	// Merge: new state for written files + old state for skipped files.
	newState := state.RecordFiles(writtenFiles)
	newState.QsdevVersion = version.Info().Version
	// Preserve state entries for files we didn't touch.
	for path, fs := range existingState.Files {
		if _, written := newState.Files[path]; !written {
			newState.Files[path] = fs
		}
	}
	// Set version metadata.
	if answers.ClaudeCode {
		newState.TemplateVersion = claudecode.ComputeTemplateVersion()
		newState.SkillLibraryVersion = claudecode.ComputeSkillLibraryVersion()
	}
	if err := state.SaveStateToFile(stateFile, newState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// 12. Re-save answers (detection may have changed).
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	// 13. Print version diff summary if applicable.
	vDiff := claudecode.CompareVersions(existingState.TemplateVersion, existingState.SkillLibraryVersion)
	if vDiff.NeedsUpdate() {
		summary := claudecode.BuildUpdateSummary(existingState, allFiles, vDiff)
		fmt.Fprintln(cmd.OutOrStdout(), summary.String())
	}

	// 14. Print result summary.
	created, updated, skipped := 0, 0, 0
	for _, fp := range plan.Files {
		switch fp.Action {
		case UpdateActionCreate:
			created++
		case UpdateActionRegenerate, UpdateActionMerge:
			updated++
		case UpdateActionSkip, UpdateActionSidecar:
			skipped++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nUpdate complete: %d created, %d updated, %d skipped.\n", created, updated, skipped)

	return nil
}

func buildUpdatePlan(
	newFiles []types.GeneratedFile,
	modStatus map[string]state.FileStatus,
	storedState types.GeneratedState,
	opts UpdateOptions,
) UpdatePlan {
	var plan UpdatePlan

	for _, f := range newFiles {
		fp := FileUpdatePlan{
			Path:       f.Path,
			Strategy:   f.Strategy,
			NewContent: f.Content,
			NewMode:    f.Mode,
		}

		fs, inState := modStatus[f.Path]
		if !inState {
			// File not in stored state — it's new.
			fp.Status = types.New
			fp.Action = UpdateActionCreate
			fp.Reason = "new file"
			plan.Files = append(plan.Files, fp)
			continue
		}

		fp.Status = fs.Status

		switch fs.Status {
		case types.Unmodified:
			fp.Action = UpdateActionRegenerate
			fp.Reason = "unmodified, safe to update"

		case types.Modified:
			if opts.Force {
				fp.Action = UpdateActionRegenerate
				fp.Reason = "modified, force overwrite"
			} else {
				// Route based on merge strategy.
				switch f.Strategy {
				case types.ThreeWayMerge:
					fp.Action = UpdateActionMerge
					fp.Reason = "modified, three-way merge"
					// Load current disk content for merge.
					fp.OldContent = readFileForMerge(storedState, f.Path)
				case types.SectionMarker:
					fp.Action = UpdateActionMerge
					fp.Reason = "modified, section marker merge"
				case types.ManualMerge:
					fp.Action = UpdateActionSidecar
					fp.Reason = "modified, manual merge required"
				case types.LibraryManaged:
					fp.Action = UpdateActionRegenerate
					fp.Reason = "library-managed, updating to latest"
				default:
					fp.Action = UpdateActionSkip
					fp.Reason = "modified, use --force to overwrite"
				}
			}

		case types.Deleted:
			if opts.Force {
				fp.Action = UpdateActionCreate
				fp.Reason = "deleted by user, force recreate"
			} else {
				fp.Action = UpdateActionSkip
				fp.Reason = "deleted by user, not recreating"
			}

		case types.Unknown:
			fp.Action = UpdateActionSkip
			fp.Reason = fmt.Sprintf("unknown status: %v", fs.Error)

		default:
			fp.Action = UpdateActionSkip
			fp.Reason = "unexpected status"
		}

		plan.Files = append(plan.Files, fp)
	}

	return plan
}

// readFileForMerge returns the base content for three-way merge from stored state.
func readFileForMerge(storedState types.GeneratedState, path string) []byte {
	if fs, ok := storedState.Files[path]; ok {
		return fs.BaseContent
	}
	return nil
}

func previewUpdatePlan(plan UpdatePlan, w io.Writer) {
	fmt.Fprintf(w, "\n%-50s  %-12s  %-12s  %s\n", "File", "Status", "Action", "Reason")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 110))

	for _, fp := range plan.Files {
		statusStr := fp.Status.String()
		actionStr := updateActionString(fp.Action)
		fmt.Fprintf(w, "%-50s  %-12s  %-12s  %s\n", fp.Path, statusStr, actionStr, fp.Reason)
	}
	fmt.Fprintln(w)
}

func updateActionString(a UpdateAction) string {
	switch a {
	case UpdateActionRegenerate:
		return "regenerate"
	case UpdateActionMerge:
		return "merge"
	case UpdateActionSkip:
		return "skip"
	case UpdateActionCreate:
		return "create"
	case UpdateActionSidecar:
		return "sidecar"
	default:
		return "unknown"
	}
}

func executeUpdatePlan(
	plan UpdatePlan,
	projectRoot string,
	opts UpdateOptions,
) ([]types.GeneratedFile, *update.NixUpdateResult, error) {
	var writtenFiles []types.GeneratedFile
	var nixResult *update.NixUpdateResult

	for _, fp := range plan.Files {
		absPath := filepath.Join(projectRoot, fp.Path)
		mode := fp.NewMode
		if mode == 0 {
			mode = 0o644
		}

		switch fp.Action {
		case UpdateActionCreate, UpdateActionRegenerate:
			if err := fileutil.WriteFileAtomic(absPath, fp.NewContent, mode); err != nil {
				return writtenFiles, nixResult, fmt.Errorf("writing %s: %w", fp.Path, err)
			}
			writtenFiles = append(writtenFiles, types.GeneratedFile{
				Path: fp.Path, Content: fp.NewContent, Mode: mode, Strategy: fp.Strategy,
			})

		case UpdateActionMerge:
			merged, err := dispatchMerge(fp, projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: merge failed for %s: %v (will retry on next update)\n", fp.Path, err)
				continue
			}
			if err := fileutil.WriteFileAtomic(absPath, merged, mode); err != nil {
				return writtenFiles, nixResult, fmt.Errorf("writing merged %s: %w", fp.Path, err)
			}
			writtenFiles = append(writtenFiles, types.GeneratedFile{
				Path: fp.Path, Content: merged, Mode: mode, Strategy: fp.Strategy,
			})

		case UpdateActionSidecar:
			result, err := update.UpdateDevenvNix(update.NixUpdateOptions{
				ProjectRoot: projectRoot,
				FilePath:    fp.Path,
				NewContent:  fp.NewContent,
				NewMode:     mode,
				Status:      fp.Status,
				Force:       opts.Force,
				DryRun:      opts.DryRun,
			})
			if err != nil {
				return writtenFiles, nixResult, fmt.Errorf("updating %s: %w", fp.Path, err)
			}
			nixResult = result
			// Only record in written files if actually written.
			if result.Action == update.NixRegenerated || result.Action == update.NixForceOverwritten {
				writtenFiles = append(writtenFiles, types.GeneratedFile{
					Path: fp.Path, Content: fp.NewContent, Mode: mode, Strategy: fp.Strategy,
				})
			}

		case UpdateActionSkip:
			// Do nothing.
		}
	}

	return writtenFiles, nixResult, nil
}

func dispatchMerge(fp FileUpdatePlan, projectRoot string) ([]byte, error) {
	absPath := filepath.Join(projectRoot, fp.Path)

	// Read current on-disk content ("theirs").
	theirs, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fp.Path, err)
	}

	switch fp.Strategy {
	case types.ThreeWayMerge:
		base := fp.OldContent
		switch {
		case fp.Path == ".mcp.json" || strings.HasSuffix(fp.Path, ".mcp.json"):
			return merge.MergeMcpJson(base, theirs, fp.NewContent)
		case strings.HasSuffix(fp.Path, "settings.json"):
			return merge.MergeSettings(base, theirs, fp.NewContent)
		default:
			return nil, fmt.Errorf("no three-way merge handler for %s; add one to dispatchMerge()", fp.Path)
		}
	case types.SectionMarker:
		return merge.SectionMarkers(theirs, fp.NewContent)
	default:
		return nil, fmt.Errorf("merge strategy %s not implemented for %s", fp.Strategy, fp.Path)
	}
}
