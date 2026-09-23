package devinit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/update"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Enable and disable never edit shared files (CLAUDE.md, devenv.nix,
// .mcp.json, settings.json) by hand-written surgery. They regenerate them
// through the same generators and update planner that init/update use, so
// every command renders a tool's contribution from one source, in each file's
// own format, with the file's merge strategy and recorded state preserved.

// toolChange is the file work one enable or disable performs, computed in full
// before anything is written so a refusal leaves the project untouched.
type toolChange struct {
	// exclusive holds the tool's own files to write (enable only).
	exclusive []types.GeneratedFile
	// shared is the update plan for the shared files the tool contributes to.
	shared UpdatePlan
	// notices explains shared files left untouched.
	notices []string
}

// toolChangeResult reports what applying a toolChange did.
type toolChangeResult struct {
	written   []types.GeneratedFile
	notices   []string
	nixResult *update.NixUpdateResult
}

// runEnable enables a tool: validates prerequisites, generates files, and
// updates persisted answers and state.
func runEnable(cmd *cobra.Command, toolName string, opts enableOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	answers, tool, err := loadToolForEnable(cmdContext(cmd), projectRoot, toolName)
	if err != nil {
		return err
	}

	// Already enabled — no-op.
	if answers.EnabledTools[toolName] {
		fmt.Fprintf(cmd.OutOrStdout(), "Tool %q is already enabled.\n", toolName)
		return nil
	}

	// Validate prerequisites and conflicts.
	if err := toolreg.ValidateEnable(toolreg.DefaultRegistry(), toolName, answers.EnabledTools); err != nil {
		return err
	}

	// Call the tool's enable function to update answers.
	if tool.EnableFunc != nil {
		tool.EnableFunc(&answers)
	}
	answers.EnabledTools[toolName] = true

	if opts.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would enable %q.\n", tool.DisplayName)
		printOwnedFiles(cmd, tool, "would write")
		return nil
	}

	stateFile := filepath.Join(projectRoot, stateFilePath())
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	change, err := planToolEnable(tool, toolName, projectRoot, answers, existingState, opts.Force)
	if err != nil {
		return err
	}
	result, err := applyToolChange(projectRoot, change, existingState)
	if err != nil {
		return err
	}

	if err := saveToolState(stateFile, existingState, toolName, true); err != nil {
		return err
	}
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Enabled %q.\n", tool.DisplayName)
	if len(result.written) > 0 {
		printWrittenFiles(cmd, result.written, "wrote")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  No files generated (tool enabled but no output needed for current project configuration).\n")
	}
	printChangeNotices(cmd, result)
	return nil
}

// loadToolForEnable loads saved answers, infers enabled tools, and looks up
// the named tool in the registry.
func loadToolForEnable(ctx context.Context, projectRoot, toolName string) (types.WizardAnswers, *toolreg.Tool, error) {
	answers, err := loadLifecycleAnswers(ctx, projectRoot)
	if err != nil {
		return types.WizardAnswers{}, nil, err
	}

	tool, ok := toolreg.DefaultRegistry().ByName(toolName)
	if !ok {
		return types.WizardAnswers{}, nil, fmt.Errorf("unknown tool %q; use '%s list' to see available tools", toolName, branding.Get().AppName)
	}

	return answers, tool, nil
}

// loadLifecycleAnswers loads saved answers (empty if no prior init) and
// refreshes them exactly as update does — current detection plus inferred
// tools — so the shared files enable/disable regenerate match what the next
// update would produce.
func loadLifecycleAnswers(ctx context.Context, projectRoot string) (types.WizardAnswers, error) {
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, fmt.Errorf("loading answers: %w", err)
	}
	answers.ProjectRoot = projectRoot
	answers.Detected = detect.Detect(ctx, projectRoot)
	toolreg.MergeInferredTools(&answers, toolreg.DefaultRegistry())
	return answers, nil
}

// lifecycleAccumulatorMode mirrors update's generator selection: a
// claude-only project never has its devenv files regenerated.
func lifecycleAccumulatorMode(answers types.WizardAnswers) generationScope {
	return generationScope{ClaudeOnly: answers.ClaudeCode && answers.MergeMode == mergeModeClaudeOnly}
}

// generateToolFiles renders the tool's exclusive files (its GenerateFunc plus
// any file the generators emit under its exclusive paths) and the current
// content of every shared file it contributes to, from the given answers.
func generateToolFiles(tool *toolreg.Tool, toolName string, answers types.WizardAnswers) (exclusive, shared []types.GeneratedFile, err error) {
	have := make(map[string]bool)
	if tool.GenerateFunc != nil {
		generated, err := tool.GenerateFunc(answers)
		if err != nil {
			return nil, nil, fmt.Errorf("generating files for %q: %w", toolName, err)
		}
		for _, f := range generated {
			f.Owner = toolName
			exclusive = append(exclusive, f)
			have[f.Path] = true
		}
	}

	acc, err := runAccumulator(answers, lifecycleAccumulatorMode(answers))
	if err != nil {
		return nil, nil, fmt.Errorf("generating files: %w", err)
	}
	sharedPaths := make(map[string]bool)
	for _, sf := range tool.SharedFiles() {
		sharedPaths[sf.Path] = true
	}
	for _, f := range acc.allFiles {
		switch {
		case sharedPaths[f.Path]:
			shared = append(shared, f)
		case !have[f.Path] && tool.OwnsExclusively(f.Path):
			f.Owner = toolName
			exclusive = append(exclusive, f)
			have[f.Path] = true
		}
	}
	return exclusive, shared, nil
}

// planToolEnable computes every file an enable writes and refuses — before
// anything is written — when the tool would produce nothing, when a write
// would leave the project, or when it would overwrite a file qsdev did not
// generate (unless force).
func planToolEnable(
	tool *toolreg.Tool, toolName, projectRoot string,
	answers types.WizardAnswers, existingState types.GeneratedState, force bool,
) (toolChange, error) {
	exclusive, shared, err := generateToolFiles(tool, toolName, answers)
	if err != nil {
		return toolChange{}, err
	}

	// Honesty guard: a tool whose own generator produced none of its files,
	// or that produced nothing at all, is being suppressed (by a tier gate, or
	// because the project gives it nothing to do). Refuse loudly instead of
	// reporting a false success that advertises the tool in CLAUDE.md while
	// its files are missing (BL-P1-9). Declared exclusive files may be
	// conditional (e.g. semble's sub-agent only exists in sub-agent mode), so
	// tools without a generator of their own are judged on all their output.
	generatorProducedNothing := tool.GenerateFunc != nil && len(tool.ExclusiveFiles()) > 0 && len(exclusive) == 0
	producedNothing := len(tool.OwnedFiles) > 0 && len(exclusive) == 0 && len(shared) == 0
	switch {
	case generatorProducedNothing:
		return toolChange{}, noToolOutputError(toolName, tool.ExclusiveFiles(), answers)
	case producedNothing:
		return toolChange{}, noToolOutputError(toolName, tool.OwnedFiles, answers)
	}

	modStatus := state.CheckModified(existingState, projectRoot)
	if err := checkExclusiveWrites(projectRoot, exclusive, modStatus, force); err != nil {
		return toolChange{}, err
	}

	change := toolChange{exclusive: exclusive}
	change.shared, change.notices, err = planSharedFiles(tool, projectRoot, shared, modStatus, existingState, false)
	if err != nil {
		return toolChange{}, err
	}
	return change, nil
}

// noToolOutputError builds an actionable error for a tool that produced none
// of its files. It only blames the tier when a higher tier exists: at full
// tier the cause is the project configuration, and "raise the tier" advice
// would send the user in a loop.
func noToolOutputError(toolName string, missing []toolreg.FileOwnership, answers types.WizardAnswers) error {
	app := branding.Get().AppName
	t := tier.Resolve(answers.Tier, answers.PermissionLevel, answers.MCPServers)
	declared := make([]string, 0, len(missing))
	for _, f := range missing {
		declared = append(declared, f.Path)
	}
	files := strings.Join(declared, ", ")
	if t < tier.Full {
		return fmt.Errorf(
			"tool %q generated none of its files (%s) at the %q tier: it may require a higher tier to produce them. "+
				"Raise the tier (e.g. run '%s init --tier full') and re-run '%s enable %s'",
			toolName, files, t.String(), app, app, toolName)
	}
	return fmt.Errorf(
		"tool %q generated none of its files (%s) for the current project configuration (tier %q), so it was not enabled; "+
			"it has nothing to configure in this project yet (for example, no declared services or inputs it depends on)",
		toolName, files, t.String())
}

// checkExclusiveWrites validates every exclusive destination before any
// write. Paths must stay inside the project (never overridable), and an
// existing file is only replaced when qsdev generated it and it is unmodified,
// when it already has the new content, or when force is set.
func checkExclusiveWrites(projectRoot string, files []types.GeneratedFile, modStatus map[string]state.FileStatus, force bool) error {
	var unsafe, foreign []string
	for _, f := range files {
		if err := generate.ValidateDestination(projectRoot, f.Path); err != nil {
			unsafe = append(unsafe, err.Error())
			continue
		}
		existing, err := os.ReadFile(filepath.Join(projectRoot, f.Path))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return fmt.Errorf("checking %s: %w", f.Path, err)
		case bytes.Equal(existing, f.Content):
			continue
		}
		if st, ok := modStatus[f.Path]; ok && st.Status == types.Unmodified {
			continue
		}
		if !force {
			foreign = append(foreign, f.Path)
		}
	}
	if len(unsafe) > 0 {
		return fmt.Errorf("refusing to write outside the project:\n  %s", strings.Join(unsafe, "\n  "))
	}
	if len(foreign) > 0 {
		return fmt.Errorf(
			"refusing to overwrite existing files that %s did not generate (or that were modified since):\n  %s\n"+
				"Move them aside, or re-run with --force to overwrite them",
			branding.Get().AppName, strings.Join(foreign, "\n  "))
	}
	return nil
}

// planSharedFiles plans the regenerated shared files with update's planner,
// so each file keeps its merge strategy (section markers for CLAUDE.md,
// three-way merge for .mcp.json/settings.json, sidecar for a modified
// devenv.nix). Files the user owns are never clobbered: an existing file qsdev
// does not track is merged or skipped, never created over. When removing
// (disable), files are never newly created.
func planSharedFiles(
	tool *toolreg.Tool, projectRoot string, shared []types.GeneratedFile,
	modStatus map[string]state.FileStatus, existingState types.GeneratedState, removing bool,
) (UpdatePlan, []string, error) {
	var notices []string
	produced := make(map[string]bool, len(shared))
	for _, f := range shared {
		produced[f.Path] = true
		if err := generate.ValidateDestination(projectRoot, f.Path); err != nil {
			return UpdatePlan{}, nil, fmt.Errorf("refusing to write outside the project: %w", err)
		}
	}

	plan := buildUpdatePlan(shared, modStatus, existingState, projectRoot, UpdateOptions{})
	kept := plan.Files[:0]
	for _, fp := range plan.Files {
		// buildUpdatePlan already merges an existing untracked file with a
		// mergeable strategy and skips anything else; a manual-merge file
		// (devenv.nix) gets a sidecar instead so the tool's contribution is
		// still offered to the user.
		if fp.Status == types.New && fp.Action == UpdateActionSkip && fp.Strategy == types.ManualMerge &&
			fileutil.FileExists(projectRoot, fp.Path) {
			untrackedExisting(&fp)
		}
		if removing && fp.Action == UpdateActionCreate {
			continue
		}
		if fp.Action == UpdateActionSkip {
			notices = append(notices, fmt.Sprintf("%s: not updated (%s)", fp.Path, fp.Reason))
		}
		kept = append(kept, fp)
	}
	plan.Files = kept

	if !removing {
		for _, sf := range tool.SharedFiles() {
			if !produced[sf.Path] {
				notices = append(notices, fmt.Sprintf("%s: not generated for this project configuration; its %q section was not written", sf.Path, sf.SectionID))
			}
		}
	}
	return plan, dedupeStrings(notices), nil
}

// untrackedExisting re-plans a "create" for a file that already exists but
// is not in qsdev's state: merge it by strategy, or leave it alone.
func untrackedExisting(fp *FileUpdatePlan) {
	fp.Status = types.Modified
	switch fp.Strategy {
	case types.SectionMarker, types.ThreeWayMerge:
		fp.Action = UpdateActionMerge
		fp.Reason = "exists but untracked, merge"
	case types.ManualMerge:
		fp.Action = UpdateActionSidecar
		fp.Reason = "exists but untracked, manual merge required"
	default:
		fp.Action = UpdateActionSkip
		fp.Reason = "exists but was not generated by " + branding.Get().AppName
	}
}

// applyToolChange writes the planned files and records them in state,
// preserving each shared file's merge strategy and recording the generated
// base (not the merged result) for three-way-merged files.
func applyToolChange(projectRoot string, change toolChange, st types.GeneratedState) (toolChangeResult, error) {
	var result toolChangeResult
	for _, f := range change.exclusive {
		if f.Mode == 0 {
			f.Mode = fileutil.ModeReadWrite
		}
		if err := fileutil.WriteFileAtomic(filepath.Join(projectRoot, f.Path), f.Content, f.Mode); err != nil {
			return result, fmt.Errorf("writing %s: %w", f.Path, err)
		}
		result.written = append(result.written, f)
	}

	outcome, err := executeUpdatePlan(change.shared, projectRoot, UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("updating shared files: %w", err)
	}
	sharedWritten := outcome.written
	result.nixResult = outcome.nixResult
	result.notices = append(result.notices, change.notices...)

	writtenPaths := make(map[string]bool, len(sharedWritten))
	for _, f := range sharedWritten {
		writtenPaths[f.Path] = true
	}
	for _, fp := range change.shared.Files {
		if fp.Action == UpdateActionMerge && !writtenPaths[fp.Path] {
			result.notices = append(result.notices, fmt.Sprintf("%s: merge failed; not updated (re-run after resolving, or run '%s update')", fp.Path, branding.Get().AppName))
		}
	}
	result.written = append(result.written, sharedWritten...)

	recorded := state.RecordFiles(result.written)
	for _, fp := range change.shared.Files {
		if fp.Action == UpdateActionMerge && fp.Strategy == types.ThreeWayMerge {
			if entry, ok := recorded.Files[fp.Path]; ok {
				entry.BaseContent = fp.NewContent
				recorded.Files[fp.Path] = entry
			}
		}
	}
	for path, entry := range recorded.Files {
		st.Files[path] = entry
	}
	return result, nil
}

// saveToolState records the tool's enabled flag and persists the state.
func saveToolState(stateFile string, st types.GeneratedState, toolName string, enabled bool) error {
	if st.EnabledTools == nil {
		st.EnabledTools = make(map[string]bool)
	}
	st.EnabledTools[toolName] = enabled
	st.LastRun = time.Now().UTC()
	if err := state.SaveStateToFile(stateFile, st); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

// runDisable disables a tool: validates dependents, removes files, and
// updates persisted answers and state.
func runDisable(cmd *cobra.Command, toolName string, opts disableOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	registry := toolreg.DefaultRegistry()

	answers, err := loadLifecycleAnswers(cmdContext(cmd), projectRoot)
	if err != nil {
		return err
	}

	tool, ok := registry.ByName(toolName)
	if !ok {
		return fmt.Errorf("unknown tool %q; use '%s list' to see available tools", toolName, branding.Get().AppName)
	}

	// Already disabled — no-op.
	if !answers.EnabledTools[toolName] {
		fmt.Fprintf(cmd.OutOrStdout(), "Tool %q is already disabled.\n", toolName)
		return nil
	}

	// Validate that the tool can be disabled.
	if err := toolreg.ValidateDisable(registry, toolName, answers.EnabledTools); err != nil {
		var alwaysOnErr *toolreg.AlwaysOnError
		if errors.As(err, &alwaysOnErr) && opts.Force {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: disabling always-on tool %q.\n", toolName)
		} else {
			return err
		}
	}

	stateFile := filepath.Join(projectRoot, stateFilePath())
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Plan everything (and run the modification check) before deleting.
	removal, err := planExclusiveRemoval(registry, tool, toolName, projectRoot, existingState, opts.Force)
	if err != nil {
		return err
	}

	// Call the tool's disable function to update answers.
	if tool.DisableFunc != nil {
		tool.DisableFunc(&answers)
	}
	answers.EnabledTools[toolName] = false

	change, err := planToolDisable(tool, toolName, projectRoot, answers, existingState)
	if err != nil {
		return err
	}

	removed, err := removal.apply(projectRoot, existingState)
	if err != nil {
		return err
	}
	result, err := applyToolChange(projectRoot, change, existingState)
	if err != nil {
		return err
	}
	result.notices = append(removal.notices, result.notices...)
	result.notices = append(result.notices, removeStaleSections(tool, projectRoot, change, existingState)...)

	if err := saveToolState(stateFile, existingState, toolName, false); err != nil {
		return err
	}
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Disabled %q.\n", tool.DisplayName)
	if len(removed) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Files removed:")
		for _, p := range removed {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p)
		}
	}
	printWrittenFiles(cmd, result.written, "updated")
	printChangeNotices(cmd, result)
	return nil
}

// planToolDisable regenerates the tool's shared files with the tool disabled.
func planToolDisable(
	tool *toolreg.Tool, toolName, projectRoot string,
	answers types.WizardAnswers, existingState types.GeneratedState,
) (toolChange, error) {
	_, shared, err := generateToolFiles(tool, toolName, answers)
	if err != nil {
		return toolChange{}, err
	}
	modStatus := state.CheckModified(existingState, projectRoot)
	plan, notices, err := planSharedFiles(tool, projectRoot, shared, modStatus, existingState, true)
	if err != nil {
		return toolChange{}, err
	}
	return toolChange{shared: plan, notices: notices}, nil
}

// exclusiveRemoval is the validated set of tool files a disable deletes.
type exclusiveRemoval struct {
	files   []string // files to delete (project-relative)
	dirs    []string // declared exclusive directories to prune when empty
	notices []string
}

// planExclusiveRemoval collects the tool's exclusive files — declared paths,
// every tracked file beneath a declared directory (e.g. the opengrep rule
// library under .opengrep/rules/core), and tracked files recorded as owned by
// the tool — and refuses, before anything is deleted, when any was modified by
// the user (unless force). Files that are shared with other tools are never
// candidates, whatever owner an older state recorded for them.
func planExclusiveRemoval(
	registry *toolreg.Registry, tool *toolreg.Tool, toolName, projectRoot string,
	st types.GeneratedState, force bool,
) (exclusiveRemoval, error) {
	sharedAnywhere := make(map[string]bool)
	for _, t := range registry.All() {
		for _, sf := range t.SharedFiles() {
			sharedAnywhere[sf.Path] = true
		}
	}

	candidates := make(map[string]bool)
	for _, ef := range tool.ExclusiveFiles() {
		candidates[ef.Path] = true
	}
	for p, entry := range st.Files {
		if !sharedAnywhere[p] && (tool.OwnsExclusively(p) || entry.Owner == toolName) {
			candidates[p] = true
		}
	}
	paths := make([]string, 0, len(candidates))
	for p := range candidates {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	modStatus := state.CheckModified(st, projectRoot)
	var r exclusiveRemoval
	var modified []string
	for _, p := range paths {
		if err := generate.ValidateDestination(projectRoot, p); err != nil {
			return exclusiveRemoval{}, fmt.Errorf("refusing to remove a path outside the project: %w", err)
		}
		info, err := os.Lstat(filepath.Join(projectRoot, p))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			delete(st.Files, p) // already gone; forget it
			continue
		case err != nil:
			return exclusiveRemoval{}, fmt.Errorf("checking %s: %w", p, err)
		case info.IsDir():
			r.dirs = append(r.dirs, p)
			continue
		}
		status, tracked := modStatus[p]
		switch {
		case !tracked && !force:
			r.notices = append(r.notices, fmt.Sprintf("%s: left in place (not generated by %s; use --force to remove)", p, branding.Get().AppName))
			continue
		case tracked && status.Status != types.Unmodified && !force:
			modified = append(modified, p)
		}
		r.files = append(r.files, p)
	}
	if len(modified) > 0 {
		return exclusiveRemoval{}, fmt.Errorf(
			"the following files have been modified by the user:\n  %s\nUse --force to remove them anyway",
			strings.Join(modified, "\n  "))
	}
	return r, nil
}

// apply deletes the planned files, forgets them in state, and prunes the
// directories they leave empty (never the project root itself).
func (r exclusiveRemoval) apply(projectRoot string, st types.GeneratedState) ([]string, error) {
	var removed []string
	pruneFrom := append([]string(nil), r.dirs...)
	for _, p := range r.files {
		if err := os.Remove(filepath.Join(projectRoot, p)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return removed, fmt.Errorf("removing %s: %w", p, err)
		}
		delete(st.Files, p)
		removed = append(removed, p)
		pruneFrom = append(pruneFrom, filepath.Dir(p))
	}
	// Deepest first, so a parent is only tried after its children.
	sort.Slice(pruneFrom, func(i, j int) bool { return len(pruneFrom[i]) > len(pruneFrom[j]) })
	for _, dir := range pruneFrom {
		pruneEmptyDirs(projectRoot, dir)
	}
	return removed, nil
}

// pruneEmptyDirs removes dir and then each parent while they are empty,
// stopping at the project root.
func pruneEmptyDirs(projectRoot, dir string) {
	for dir != "." && dir != "" && dir != string(filepath.Separator) {
		if err := os.Remove(filepath.Join(projectRoot, dir)); err != nil {
			return // not empty (or already gone): stop climbing
		}
		dir = filepath.Dir(dir)
	}
}

// removeStaleSections handles a disable whose shared file the generators no
// longer produce at all (e.g. .mcp.json once no MCP server remains at the
// standard tier): the tool's entry is removed in place from the qsdev-tracked
// file so it does not linger. Untracked files are never edited.
func removeStaleSections(tool *toolreg.Tool, projectRoot string, change toolChange, st types.GeneratedState) []string {
	planned := make(map[string]bool, len(change.shared.Files))
	for _, fp := range change.shared.Files {
		planned[fp.Path] = true
	}
	var notices []string
	for _, sf := range tool.SharedFiles() {
		entry, tracked := st.Files[sf.Path]
		if planned[sf.Path] || !tracked {
			continue
		}
		updated, err := removeSectionInPlace(projectRoot, sf.Path, sf.SectionID)
		switch {
		case err != nil:
			notices = append(notices, fmt.Sprintf("%s: could not remove the %q section: %v", sf.Path, sf.SectionID, err))
			continue
		case updated == nil:
			continue
		}
		if err := generate.ValidateDestination(projectRoot, sf.Path); err != nil {
			notices = append(notices, fmt.Sprintf("%s: not updated: %v", sf.Path, err))
			continue
		}
		if err := fileutil.WriteFileAtomic(filepath.Join(projectRoot, sf.Path), updated, fileutil.ModeReadWrite); err != nil {
			notices = append(notices, fmt.Sprintf("%s: could not write: %v", sf.Path, err))
			continue
		}
		// Keep the recorded strategy, base and owner; only the content changed.
		entry.Hash = state.ComputeHash(updated)
		st.Files[sf.Path] = entry
	}
	return notices
}

// removeSectionInPlace removes a tool's entry from a shared file on disk. It
// returns nil when the file is absent, unchanged, or of a format that has no
// addressable per-tool section.
func removeSectionInPlace(projectRoot, relPath, sectionID string) ([]byte, error) {
	existing, err := os.ReadFile(filepath.Join(projectRoot, relPath))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relPath, err)
	}

	var updated []byte
	switch base, ext := filepath.Base(relPath), filepath.Ext(relPath); {
	case base == ".mcp.json":
		// MCP servers are keyed by name; the section ID is the server name.
		updated, err = surgery.JSONRemoveMCPServer(existing, sectionID)
	case ext == ".md":
		updated, err = surgery.MarkdownRemoveSection(existing, sectionID)
	case ext == ".nix":
		updated, err = surgery.NixRemoveSection(existing, sectionID)
	default:
		return nil, nil
	}
	if err != nil || bytes.Equal(updated, existing) {
		return nil, err
	}
	return updated, nil
}

// printChangeNotices reports shared files left untouched and any devenv.nix
// sidecar, so a partial result is never presented as complete.
func printChangeNotices(cmd *cobra.Command, result toolChangeResult) {
	if result.nixResult != nil && result.nixResult.Action == update.NixSidecarCreated {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), result.nixResult.DiffOutput)
		fmt.Fprintln(cmd.OutOrStdout(), result.nixResult.Message)
	}
	if len(result.notices) == 0 {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Notes:")
	for _, n := range dedupeStrings(result.notices) {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", n)
	}
}

// dedupeStrings returns items without duplicates, preserving order.
func dedupeStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := items[:0:0]
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// runList prints all registered tools grouped by category.
func runList(cmd *cobra.Command, opts listOptions) error {
	registry := toolreg.DefaultRegistry()

	// Load project state for enabled/disabled display.
	projectRoot, _ := cmdutil.ProjectRoot()
	var enabledTools map[string]bool
	if projectRoot != "" {
		if ans, err := loadAnswersOrEmpty(projectRoot); err == nil {
			ans.ProjectRoot = projectRoot
			toolreg.InferEnabledTools(&ans, registry)
			enabledTools = ans.EnabledTools
		}
	}

	var tools []*toolreg.Tool
	if opts.Category != "" {
		cat := toolreg.ToolCategory(opts.Category)
		tools = registry.ByCategory(cat)
		if len(tools) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No tools found in category %q.\n", opts.Category)
			return nil
		}
		printToolGroup(cmd, cat, tools, enabledTools)
		return nil
	}

	tools = registry.All()
	if len(tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools registered.")
		return nil
	}

	// Group by category, preserving sort order from registry.All().
	groups := groupByCategory(tools)
	for _, g := range groups {
		printToolGroup(cmd, g.category, g.tools, enabledTools)
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

type categoryGroup struct {
	category toolreg.ToolCategory
	tools    []*toolreg.Tool
}

// groupByCategory groups a pre-sorted tool slice into category groups,
// preserving the input order.
func groupByCategory(tools []*toolreg.Tool) []categoryGroup {
	var groups []categoryGroup
	seen := make(map[toolreg.ToolCategory]int) // category -> index in groups

	for _, t := range tools {
		idx, ok := seen[t.Category]
		if !ok {
			idx = len(groups)
			seen[t.Category] = idx
			groups = append(groups, categoryGroup{category: t.Category})
		}
		groups[idx].tools = append(groups[idx].tools, t)
	}
	return groups
}

func printToolGroup(cmd *cobra.Command, cat toolreg.ToolCategory, tools []*toolreg.Tool, enabledTools map[string]bool) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", cat.DisplayName())

	// Sort by name within category for consistent output.
	sorted := make([]*toolreg.Tool, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	for _, t := range sorted {
		defaultStr := t.Default.String()
		stateStr := ""
		if enabledTools != nil {
			if enabledTools[t.Name] {
				stateStr = "[enabled]  "
			} else {
				stateStr = "[disabled] "
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-25s  %-15s  %s%s\n", t.Name, "("+defaultStr+")", stateStr, t.Description)
	}
}

// printOwnedFiles prints the list of files a tool owns.
func printOwnedFiles(cmd *cobra.Command, tool *toolreg.Tool, verb string) {
	if len(tool.OwnedFiles) == 0 {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Files %s:\n", verb)
	for _, f := range tool.OwnedFiles {
		ownerType := f.Ownership.String()
		fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s)\n", f.Path, ownerType)
	}
}

// printWrittenFiles prints the list of files that were actually written during
// an enable operation.
func printWrittenFiles(cmd *cobra.Command, files []types.GeneratedFile, verb string) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Files %s:\n", verb)
	for _, f := range files {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Path)
	}
}
