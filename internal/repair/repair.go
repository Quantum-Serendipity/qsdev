package repair

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Repair classifies drift findings, creates backups, and writes fresh file
// content for auto-fixable issues. It returns the repair result and an
// updated copy of the generation state.
//
// freshFiles is a map from relative path to the freshly generated file content
// that should replace the drifted file on disk. With opts.Reset every fresh
// file is regenerated, not only those with drift findings.
func Repair(
	projectRoot string,
	opts RepairOptions,
	genState types.GeneratedState,
	freshFiles map[string]types.GeneratedFile,
	driftReport *drift.Report,
) (*RepairResult, *types.GeneratedState, error) {
	if driftReport == nil && !opts.Reset {
		return &RepairResult{}, &genState, nil
	}

	actions := classifyFindings(driftReport, genState, opts)
	if opts.Reset {
		actions = append(actions, resetActions(actions, freshFiles)...)
	}

	// Filter to a single file when --file is specified.
	if opts.TargetFile != "" {
		target, err := normalizeTargetFile(projectRoot, opts.TargetFile)
		if err != nil {
			return nil, nil, err
		}
		actions = filterActions(actions, target)
		if len(actions) == 0 && !isManagedFile(target, genState, freshFiles) {
			return nil, nil, fmt.Errorf("%s is not a file managed by %s", opts.TargetFile, branding.Get().AppName)
		}
	}

	result := &RepairResult{}
	updatedState := copyState(genState)

	for _, action := range actions {
		if !action.AutoFixable {
			result.Skipped = append(result.Skipped, action)
			continue
		}

		if opts.DryRun {
			result.Fixed = append(result.Fixed, action)
			continue
		}

		repaired := executeRepair(projectRoot, action, freshFiles, &updatedState)
		switch {
		case repaired.Error != nil:
			result.Failed = append(result.Failed, repaired)
		default:
			result.Fixed = append(result.Fixed, repaired)
		}
	}

	result.Skipped = dropResolved(result.Skipped, result.Fixed)

	return result, &updatedState, nil
}

// resetActions synthesizes a regenerate action for every fresh file that no
// drift finding already covers, so --reset rewrites all generated files and
// not only drifted ones. Files on the never-auto-repair list are left alone.
func resetActions(existing []RepairAction, freshFiles map[string]types.GeneratedFile) []RepairAction {
	covered := make(map[string]bool, len(existing))
	for _, a := range existing {
		covered[a.File] = true
	}

	var actions []RepairAction
	for _, path := range slices.Sorted(maps.Keys(freshFiles)) {
		if covered[path] || neverAutoRepairFiles[path] {
			continue
		}
		actions = append(actions, RepairAction{
			File:        path,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("Regenerate %s (--reset)", path),
			ActionType:  ActionRegenerate,
			AutoFixable: true,
		})
	}
	return actions
}

// normalizeTargetFile converts a --file argument into the project-relative,
// slash-separated form used for drift subjects and generated-file keys. It
// accepts "./x", backslash separators and absolute paths inside projectRoot.
func normalizeTargetFile(projectRoot, target string) (string, error) {
	p := filepath.FromSlash(strings.ReplaceAll(target, "\\", "/"))
	if filepath.IsAbs(p) {
		rel, err := filepath.Rel(projectRoot, p)
		if err != nil {
			return "", fmt.Errorf("resolving %s against project root: %w", target, err)
		}
		p = rel
	}
	p = filepath.Clean(p)
	if !filepath.IsLocal(p) {
		return "", fmt.Errorf("%s is not inside the project", target)
	}
	return filepath.ToSlash(p), nil
}

// filterActions keeps the actions for target.
func filterActions(actions []RepairAction, target string) []RepairAction {
	var filtered []RepairAction
	for _, a := range actions {
		if a.File == target {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// isManagedFile reports whether path is a file qsdev generates or tracks.
func isManagedFile(path string, genState types.GeneratedState, freshFiles map[string]types.GeneratedFile) bool {
	if _, ok := genState.Files[path]; ok {
		return true
	}
	_, ok := freshFiles[path]
	return ok
}

// dropResolved removes skipped actions that a fixed action resolved in the
// same run (e.g. section-marker findings once CLAUDE.md is regenerated).
func dropResolved(skipped, fixed []RepairAction) []RepairAction {
	fixedFiles := make(map[string]bool, len(fixed))
	for _, a := range fixed {
		fixedFiles[a.File] = true
	}

	kept := skipped[:0:0]
	for _, a := range skipped {
		if a.ResolvedBy != "" && fixedFiles[a.ResolvedBy] {
			continue
		}
		kept = append(kept, a)
	}
	return kept
}

// executeRepair performs a single repair action: backup, write fresh content,
// and update the generation state.
func executeRepair(
	projectRoot string,
	action RepairAction,
	freshFiles map[string]types.GeneratedFile,
	updatedState *types.GeneratedState,
) RepairAction {
	fresh, ok := freshFiles[action.File]
	if !ok {
		action.Error = fmt.Errorf("no fresh content available for %s", action.File)
		return action
	}

	absPath := filepath.Join(projectRoot, action.File)

	// Create backup if the file exists on disk.
	if _, err := os.Stat(absPath); err == nil {
		backupPath, err := createBackup(projectRoot, action.File)
		if err != nil {
			action.Error = fmt.Errorf("creating backup for %s: %w", action.File, err)
			return action
		}
		action.BackupPath = backupPath

		// Prune old backups, keeping the 5 most recent.
		if err := pruneBackups(projectRoot, action.File, 5); err != nil {
			slog.Warn("pruning backups", "file", action.File, "error", err)
		}
	}

	// Determine file mode.
	mode := fresh.Mode
	if mode == 0 {
		mode = fileutil.ModeReadWrite
	}

	// Write the fresh content atomically.
	if err := fileutil.WriteFileAtomic(absPath, fresh.Content, mode); err != nil {
		action.Error = fmt.Errorf("writing %s: %w", action.File, err)
		return action
	}

	// Update state hash for the repaired file.
	if updatedState.Files == nil {
		updatedState.Files = make(map[string]types.FileState)
	}
	// Record the entry the same way generation does (including the
	// three-way-merge base), keyed by the repaired path.
	fresh.Path = action.File
	fresh.Mode = mode
	updatedState.Files[action.File] = state.RecordFiles([]types.GeneratedFile{fresh}).Files[action.File]

	return action
}

// copyState returns a deep copy of genState so callers can mutate the
// returned value without affecting the original.
func copyState(genState types.GeneratedState) types.GeneratedState {
	cp := genState

	// Deep copy Files, including the BaseContent byte slice in each entry.
	cp.Files = make(map[string]types.FileState, len(genState.Files))
	for k, v := range genState.Files {
		if v.BaseContent != nil {
			bc := make([]byte, len(v.BaseContent))
			copy(bc, v.BaseContent)
			v.BaseContent = bc
		}
		cp.Files[k] = v
	}

	// Deep copy EnabledTools.
	if genState.EnabledTools != nil {
		cp.EnabledTools = make(map[string]bool, len(genState.EnabledTools))
		maps.Copy(cp.EnabledTools, genState.EnabledTools)
	}

	// Deep copy Fragments (each value is a slice of ledger entries).
	if genState.Fragments != nil {
		cp.Fragments = make(map[string][]types.FragmentLedgerEntry, len(genState.Fragments))
		for k, v := range genState.Fragments {
			cp.Fragments[k] = slices.Clone(v)
		}
	}

	return cp
}
