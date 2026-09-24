package devinit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/update"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// UpdateOptions holds configuration for the update command.
type UpdateOptions struct {
	// Force overwrites managed files the user has modified or deleted, and
	// untracked files that already exist at a generated path.
	Force bool
	// AllowDowngrade lets a binary older than the one that last generated the
	// project regenerate its files (bypasses the version ratchet).
	AllowDowngrade bool
	// OverwriteFlag is the CLI flag that sets Force, named in skip reasons.
	// Defaults to "--force".
	OverwriteFlag string
	DryRun        bool
	// SkipContainer opts out of generating Gateway container configuration for
	// detected frameworks that lack native hook enforcement (Unit 32.10).
	SkipContainer bool
}

// overwriteFlag returns the flag that enables Force, for user-facing reasons.
func (o UpdateOptions) overwriteFlag() string {
	if o.OverwriteFlag != "" {
		return o.OverwriteFlag
	}
	return "--force"
}

// UpdateAction describes what the update will do to a file.
type UpdateAction int

const (
	UpdateActionRegenerate UpdateAction = iota
	UpdateActionMerge
	UpdateActionSkip
	UpdateActionCreate
	UpdateActionSidecar
	// UpdateActionRemove deletes an unmodified file the generators no longer
	// produce and stops tracking it.
	UpdateActionRemove
	// UpdateActionUntrack stops tracking a file the generators no longer
	// produce but leaves it on disk (the user modified or deleted it).
	UpdateActionUntrack
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
	// MCPPolicy is the client MCP policy merged files must honour.
	MCPPolicy types.MCPPolicy
}

// fileFailure records a file that could not be updated. Execution continues
// past these; the file keeps its previous state entry so it is retried.
type fileFailure struct {
	Path string
	Err  error
}

// updateOutcome records what executeUpdatePlan actually did, as opposed to
// what the plan intended.
type updateOutcome struct {
	written   []types.GeneratedFile
	dropped   map[string]bool // orphaned paths removed from disk or untracked
	failures  []fileFailure
	nixResult *update.NixUpdateResult
}

func runUpdate(cmd *cobra.Command, opts UpdateOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	// 1. Load answers, refresh detection, infer tools.
	answers, err := loadAndRefreshForUpdate(cmd.Context(), cmd.ErrOrStderr(), projectRoot)
	if err != nil {
		return err
	}

	// 2. Load stored state.
	stateFile := filepath.Join(projectRoot, stateFilePath())
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// 3. Check modification status of all stored files.
	modStatus := state.CheckModified(existingState, projectRoot)

	// 3b. Version ratchet check — refuse if current binary is older than last run.
	if !opts.AllowDowngrade {
		if ratchet := qsdevconfig.CheckVersionRatchet(version.Info().Version, existingState.QsdevVersion); ratchet != nil {
			return ratchet
		}
	}

	// 4. Generate new files via fragment accumulation, honouring the
	// generation scope the project was initialized with. Generation also
	// sees .qsdev.local.yaml; the answers and .qsdev.yaml saved below do not.
	genAnswers, err := localGenerationAnswers(projectRoot, answers)
	if err != nil {
		return err
	}
	accResult, err := runAccumulator(genAnswers, scopeFromAnswers(genAnswers))
	if err != nil {
		return fmt.Errorf("generating files: %w", err)
	}
	allFiles := accResult.allFiles

	// 5. Build update plan, including cleanup of files no longer generated.
	plan := buildUpdatePlan(allFiles, modStatus, existingState, projectRoot, opts)
	plan.MCPPolicy = genAnswers.MCPPolicy
	plan.Files = append(plan.Files, planOrphans(existingState, allFiles, modStatus, genAnswers)...)

	// 6. Preview.
	previewUpdatePlan(plan, cmd.OutOrStdout())
	if opts.DryRun {
		return nil
	}

	// 7. Execute plan.
	outcome, execErr := executeUpdatePlan(plan, projectRoot, opts)
	reportUpdateFailures(cmd.ErrOrStderr(), outcome.failures)

	// 8. If nix sidecar was created, show instructions.
	if nixResult := outcome.nixResult; nixResult != nil && nixResult.Action == update.NixSidecarCreated {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), nixResult.DiffOutput)
		fmt.Fprintln(cmd.OutOrStdout(), nixResult.Message)
	}

	// 9. Save state and answers. This runs even when execution stopped
	// partway: files already rewritten must be recorded, or the next update
	// would mistake qsdev's own output for user modifications.
	if err := saveUpdateResults(plan, outcome, existingState, answers, accResult, projectRoot); err != nil {
		return errors.Join(execErr, err)
	}
	if execErr != nil {
		return fmt.Errorf("executing update: %w", execErr)
	}

	// 10. Print version diff summary if Claude Code files were generated.
	if accResult.claudeGenerated {
		vDiff := claudecode.CompareVersions(existingState.TemplateVersion, existingState.SkillLibraryVersion)
		if vDiff.NeedsUpdate() {
			summary := claudecode.BuildUpdateSummary(existingState, allFiles, vDiff)
			fmt.Fprintln(cmd.OutOrStdout(), summary.String())
		}
	}

	// 11. Print result summary.
	printUpdateSummary(cmd.OutOrStdout(), plan, outcome)

	// 12. Best-effort Gateway container config for hookless frameworks (Unit
	// 32.10). This is intentionally additive and non-fatal: a failure or a
	// project with no gateway-needing framework leaves the rest of the update
	// untouched and produces no output.
	maybeGenerateContainerConfig(cmd, projectRoot, answers, opts)

	if n := len(outcome.failures); n > 0 {
		return fmt.Errorf("%d file(s) could not be updated; they keep their previous state and will be retried on the next update", n)
	}
	return nil
}

// reportUpdateFailures writes one warning per file that could not be updated.
func reportUpdateFailures(w io.Writer, failures []fileFailure) {
	for _, f := range failures {
		fmt.Fprintf(w, "Warning: %s: %v\n", f.Path, f.Err)
	}
}

// printUpdateSummary prints per-action counts derived from what was actually
// written, removed or failed, rather than from the plan alone.
func printUpdateSummary(w io.Writer, plan UpdatePlan, out updateOutcome) {
	written := make(map[string]bool, len(out.written))
	for _, f := range out.written {
		written[f.Path] = true
	}
	failed := make(map[string]bool, len(out.failures))
	for _, f := range out.failures {
		failed[f.Path] = true
	}

	var created, updated, skipped, removed int
	for _, fp := range plan.Files {
		switch {
		case failed[fp.Path]:
			// Reported separately below.
		case written[fp.Path] && fp.Action == UpdateActionCreate:
			created++
		case written[fp.Path]:
			updated++
		case fp.Action == UpdateActionRemove && out.dropped[fp.Path]:
			removed++
		default:
			skipped++
		}
	}

	heading := "Update complete"
	if len(failed) > 0 {
		heading = "Update finished with errors"
	}
	fmt.Fprintf(w, "\n%s: %d created, %d updated, %d skipped, %d removed", heading, created, updated, skipped, removed)
	if len(failed) > 0 {
		fmt.Fprintf(w, ", %d failed", len(failed))
	}
	fmt.Fprintln(w, ".")
}

// loadAndRefreshForUpdate loads saved answers, refreshes ecosystem detection,
// applies the committed security floor and client policy (warning on w about
// any setting the floor raised), and augments enabled tools with inferred
// entries.
func loadAndRefreshForUpdate(ctx context.Context, w io.Writer, projectRoot string) (types.WizardAnswers, error) {
	answers, err := loadAnswers(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}

	// Refresh detection.
	answers.Detected = detect.Detect(ctx, projectRoot)
	answers.ProjectRoot = projectRoot

	if err := applyCommittedPolicy(w, projectRoot, &answers); err != nil {
		return types.WizardAnswers{}, err
	}

	// Augment EnabledTools with inferred tools (AlwaysOn, hooks-implied).
	toolreg.MergeInferredTools(&answers, toolreg.DefaultRegistry())
	adoptCommittedTier(projectRoot, &answers)
	enforceAnswerInvariants(&answers)

	return answers, nil
}

// adoptCommittedTier gives answers saved before the tier was always recorded
// the tier committed in .qsdev.yaml, so update never replaces the team's
// recorded tier with an inferred one. Only when neither file records a valid
// tier does enforceAnswerInvariants infer it. An unreadable config is left to
// SyncProjectConfig, which reports it.
func adoptCommittedTier(projectRoot string, a *types.WizardAnswers) {
	if a.Tier != "" {
		return
	}
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(projectRoot, branding.Get().ConfigFile))
	if err != nil || cfg.Tier == "" {
		return
	}
	if _, err := tier.ParseTier(cfg.Tier); err != nil {
		return
	}
	a.Tier = cfg.Tier
}

// saveUpdateResults persists the new state (merging written and skipped files)
// and re-saves answers after an update execution.
func saveUpdateResults(
	plan UpdatePlan,
	outcome updateOutcome,
	existingState types.GeneratedState,
	answers types.WizardAnswers,
	accResult accumulatorResult,
	projectRoot string,
) error {
	// Merge: new state for written files + old state for skipped files.
	newState := state.RecordFiles(outcome.written)
	// Correct BaseContent for merged ThreeWayMerge files: store the
	// original generated content (ours), not the merged result.
	for _, fp := range plan.Files {
		if fp.Action == UpdateActionMerge && fp.Strategy == types.ThreeWayMerge {
			if fs, ok := newState.Files[fp.Path]; ok {
				fs.BaseContent = fp.NewContent
				newState.Files[fp.Path] = fs
			}
		}
	}
	newState.QsdevVersion = version.Info().Version
	newState.EnabledTools = answers.EnabledTools
	newState.Fragments = state.RecordFragments(accResult.fragments)
	// Preserve state entries for files we didn't touch (skipped, failed, or
	// not reached), dropping orphans that were removed or untracked.
	for path, fs := range existingState.Files {
		if _, written := newState.Files[path]; written || outcome.dropped[path] {
			continue
		}
		newState.Files[path] = fs
	}
	stampTemplateVersions(&newState, accResult.claudeGenerated)
	if err := state.SaveInitState(projectRoot, newState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// Re-save answers (detection may have changed).
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}
	if err := qsdevconfig.SyncProjectConfig(projectRoot, answers); err != nil {
		return err
	}

	return nil
}

func buildUpdatePlan(
	newFiles []types.GeneratedFile,
	modStatus map[string]state.FileStatus,
	storedState types.GeneratedState,
	projectRoot string,
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
			plan.Files = append(plan.Files, planUntrackedFile(fp, projectRoot, opts))
			continue
		}

		fp.Status = fs.Status

		switch fs.Status {
		case types.Unmodified:
			switch f.Strategy {
			case types.SectionMarker:
				fp.Action = UpdateActionMerge
				fp.Reason = "unmodified, section marker merge"
			case types.ThreeWayMerge:
				fp.Action = UpdateActionMerge
				fp.Reason = "unmodified, three-way merge"
				fp.OldContent = readFileForMerge(storedState, f.Path)
			default:
				fp.Action = UpdateActionRegenerate
				fp.Reason = "unmodified, safe to update"
			}

		case types.Modified:
			switch {
			case f.Strategy == types.Skip:
				// Skip-if-exists: once the user edits the file it is theirs;
				// --force does not override the strategy.
				fp.Action = UpdateActionSkip
				fp.Reason = "modified, kept (skip-if-exists file)"
			case opts.Force:
				fp.Action = UpdateActionRegenerate
				fp.Reason = "modified, force overwrite"
			default:
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
					fp.Reason = fmt.Sprintf("modified, use %s to overwrite", opts.overwriteFlag())
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

// planUntrackedFile plans a generated file that has no state entry. A missing
// file is simply created. A file that already exists was not written by
// qsdev (or its state was lost), so it is never blindly overwritten: mergeable
// strategies merge with it the way init does, a Skip (skip-if-exists) file is
// always kept, and anything else is skipped unless overwriting was explicitly
// requested.
func planUntrackedFile(fp FileUpdatePlan, projectRoot string, opts UpdateOptions) FileUpdatePlan {
	fp.Status = types.New

	absPath := filepath.Join(projectRoot, fp.Path)
	info, err := os.Lstat(absPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fp.Action = UpdateActionCreate
		fp.Reason = "new file"
	case err != nil:
		fp.Action = UpdateActionSkip
		fp.Reason = fmt.Sprintf("cannot inspect existing file: %v", err)
	case info.Mode().IsRegular() && fileHasContent(absPath, fp.NewContent):
		// Already exactly what would be generated: nothing to lose, so start
		// tracking it instead of reporting it as untracked on every update.
		fp.Action = UpdateActionRegenerate
		fp.Reason = "exists untracked with generated content, tracking"
	case fp.Strategy == types.ThreeWayMerge || fp.Strategy == types.SectionMarker:
		// OldContent stays nil: there is no recorded base.
		fp.Action = UpdateActionMerge
		fp.Reason = "exists but untracked, merging"
	case fp.Strategy == types.Skip:
		// Skip-if-exists: the user's own file is never replaced, not even
		// with --force.
		fp.Action = UpdateActionSkip
		fp.Reason = fmt.Sprintf("exists but not generated by %s, kept (skip-if-exists file)", branding.Get().AppName)
	case opts.Force:
		fp.Action = UpdateActionRegenerate
		fp.Reason = "exists but untracked, force overwrite"
	default:
		fp.Action = UpdateActionSkip
		fp.Reason = fmt.Sprintf("exists but not tracked by %s, use %s to overwrite",
			branding.Get().AppName, opts.overwriteFlag())
	}
	return fp
}

// fileHasContent reports whether the file at path holds exactly content.
func fileHasContent(path string, content []byte) bool {
	existing, err := os.ReadFile(path)
	return err == nil && bytes.Equal(existing, content)
}

// planOrphans plans cleanup for tracked files the generators no longer
// produce (language removed, ecosystem no longer detected, tier lowered,
// template retired). Unmodified orphans are removed; ones the user modified
// or already deleted are left alone and simply no longer tracked.
func planOrphans(
	storedState types.GeneratedState,
	newFiles []types.GeneratedFile,
	modStatus map[string]state.FileStatus,
	answers types.WizardAnswers,
) []FileUpdatePlan {
	generatedOwners := make(map[string]bool)
	for _, f := range newFiles {
		if f.Owner != "" {
			generatedOwners[f.Owner] = true
		}
	}

	var plans []FileUpdatePlan
	for _, path := range state.OrphanedFiles(storedState, newFiles) {
		stored := storedState.Files[path]
		if !updateOwnsOrphan(path, stored, answers, generatedOwners) {
			continue
		}

		fp := FileUpdatePlan{Path: path, Strategy: stored.Strategy}
		st, ok := modStatus[path]
		switch {
		case !ok:
			fp.Status = types.Unknown
			fp.Action = UpdateActionSkip
			fp.Reason = "no longer generated, status unknown"
		case !filepath.IsLocal(filepath.FromSlash(path)):
			// Never delete outside the project, whatever the state file says.
			fp.Status = st.Status
			fp.Action = UpdateActionUntrack
			fp.Reason = "no longer generated, path outside project; untracking"
		case st.Status == types.Unmodified:
			fp.Status = st.Status
			fp.Action = UpdateActionRemove
			fp.Reason = "no longer generated, removing"
		case st.Status == types.Modified:
			fp.Status = st.Status
			fp.Action = UpdateActionUntrack
			fp.Reason = "no longer generated, modified; left in place and untracked"
		case st.Status == types.Deleted:
			fp.Status = st.Status
			fp.Action = UpdateActionUntrack
			fp.Reason = "no longer generated, already deleted"
		default:
			fp.Status = st.Status
			fp.Action = UpdateActionSkip
			fp.Reason = fmt.Sprintf("no longer generated, unknown status: %v", st.Error)
		}
		plans = append(plans, fp)
	}
	return plans
}

// updateOwnsOrphan reports whether update is responsible for cleaning up a
// tracked file it no longer generates. Files owned by a still-enabled tool
// belong to the enable/disable lifecycle, unless that tool's generator ran in
// this update (generatedOwners): its output is then authoritative, so a file it
// used to produce and no longer does was retired and is cleaned up here. When
// the tool generated nothing (its addon is out of the generation scope, or it
// yields no files at this tier) its tracked files are left alone. The
// per-developer local config is created once by join and never regenerated.
func updateOwnsOrphan(path string, stored types.FileState, answers types.WizardAnswers, generatedOwners map[string]bool) bool {
	if stored.Owner != "" && answers.EnabledTools[stored.Owner] && !generatedOwners[stored.Owner] {
		return false
	}
	return path != branding.Get().LocalConfig
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
	case UpdateActionRemove:
		return "remove"
	case UpdateActionUntrack:
		return "untrack"
	default:
		return "unknown"
	}
}

// executeUpdatePlan applies the plan. Per-file merge and removal failures are
// recorded in the outcome and execution continues; a write failure stops
// execution and is returned together with everything done up to that point,
// so the caller can still record it.
func executeUpdatePlan(
	plan UpdatePlan,
	projectRoot string,
	opts UpdateOptions,
) (updateOutcome, error) {
	out := updateOutcome{dropped: make(map[string]bool)}

	for _, fp := range plan.Files {
		absPath := filepath.Join(projectRoot, fp.Path)
		mode := fp.NewMode
		if mode == 0 {
			mode = fileutil.ModeReadWrite
		}
		// Same containment guarantee as generate.WriteFiles: never write
		// through a symlink (or a crafted path) to outside the project.
		if fp.Action != UpdateActionSkip {
			if err := generate.ValidateDestination(projectRoot, fp.Path); err != nil {
				return out, fmt.Errorf("refusing to write %s: %w", fp.Path, err)
			}
		}

		switch fp.Action {
		case UpdateActionCreate, UpdateActionRegenerate:
			if skipExistingUserFile(fp, absPath) {
				continue
			}
			if stop, err := out.writeValidated(projectRoot, fp, fp.NewContent, mode); stop {
				return out, err
			}

		case UpdateActionMerge:
			merged, err := dispatchMerge(fp, projectRoot, plan.MCPPolicy)
			if err != nil {
				out.failures = append(out.failures, fileFailure{Path: fp.Path, Err: fmt.Errorf("merge failed: %w", err)})
				continue
			}
			if stop, err := out.writeValidated(projectRoot, fp, merged, mode); stop {
				return out, err
			}

		case UpdateActionSidecar:
			// Validate before the regenerate-or-sidecar decision, so neither
			// the file nor its sidecar receives content init would reject.
			if err := generate.ValidateContent(fp.Path, fp.NewContent); err != nil {
				out.failures = append(out.failures, fileFailure{Path: fp.Path, Err: err})
				continue
			}
			if err := executeSidecar(fp, projectRoot, mode, opts, &out); err != nil {
				return out, err
			}

		case UpdateActionRemove:
			if err := os.Remove(absPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				out.failures = append(out.failures, fileFailure{Path: fp.Path, Err: fmt.Errorf("removing orphaned file: %w", err)})
				continue
			}
			out.dropped[fp.Path] = true

		case UpdateActionUntrack:
			out.dropped[fp.Path] = true

		case UpdateActionSkip:
			// Do nothing.
		}
	}

	return out, nil
}

// writeValidated writes content for fp through the validated generated-file
// writer and records it. Content that fails validation is recorded as a
// per-file failure (the file keeps its previous content and state); stop is
// true only for a write error, which ends execution.
func (o *updateOutcome) writeValidated(projectRoot string, fp FileUpdatePlan, content []byte, mode os.FileMode) (stop bool, err error) {
	err = generate.WriteGeneratedFile(projectRoot, types.GeneratedFile{Path: fp.Path, Content: content, Mode: mode})
	switch {
	case errors.Is(err, generate.ErrInvalidContent):
		o.failures = append(o.failures, fileFailure{Path: fp.Path, Err: err})
		return false, nil
	case err != nil:
		return true, err
	}
	o.recordWrite(fp, content, mode)
	return false, nil
}

// recordWrite notes a file written with content and mode.
func (o *updateOutcome) recordWrite(fp FileUpdatePlan, content []byte, mode os.FileMode) {
	o.written = append(o.written, types.GeneratedFile{
		Path: fp.Path, Content: content, Mode: mode, Strategy: fp.Strategy,
	})
}

// executeSidecar applies a manual-merge file update (devenv.nix), which
// either regenerates the file or writes a .new sidecar for the user to merge.
func executeSidecar(fp FileUpdatePlan, projectRoot string, mode os.FileMode, opts UpdateOptions, out *updateOutcome) error {
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
		return fmt.Errorf("updating %s: %w", fp.Path, err)
	}
	out.nixResult = result
	// Only record in written files if actually written.
	if result.Action == update.NixRegenerated || result.Action == update.NixForceOverwritten {
		out.recordWrite(fp, fp.NewContent, mode)
	}
	return nil
}

// skipExistingUserFile reports whether a planned create must leave an
// existing file alone: a Skip-strategy file that qsdev never recorded (so it
// is "new" to the plan) but that already exists on disk belongs to the user,
// e.g. a project's own .npmrc or .bazelrc that init also declined to replace.
func skipExistingUserFile(fp FileUpdatePlan, absPath string) bool {
	if fp.Action != UpdateActionCreate || fp.Strategy != types.Skip {
		return false
	}
	_, err := os.Lstat(absPath)
	return err == nil
}

// dispatchMerge merges generated content into a user-modified file and then
// drops the MCP servers policy forbids, which the merge would otherwise keep
// as user-added.
func dispatchMerge(fp FileUpdatePlan, projectRoot string, policy types.MCPPolicy) ([]byte, error) {
	absPath := filepath.Join(projectRoot, fp.Path)

	// Read current on-disk content ("theirs").
	theirs, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fp.Path, err)
	}

	// An empty on-disk file has nothing to preserve (matches the init
	// pipeline); write the generated content.
	if len(bytes.TrimSpace(theirs)) == 0 {
		return fp.NewContent, nil
	}

	// fp.OldContent is the recorded base for this file. Delegate to the shared
	// merge.Dispatch table so all write paths route identically.
	merged, err := merge.Dispatch(fp.Path, fp.Strategy, fp.OldContent, theirs, fp.NewContent)
	if err != nil {
		return nil, err
	}
	return merge.EnforceMCPPolicy(fp.Path, merged, policy)
}
