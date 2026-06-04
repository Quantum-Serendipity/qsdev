package repair

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Repair classifies drift findings, creates backups, and writes fresh file
// content for auto-fixable issues. It returns the repair result and an
// updated copy of the generation state.
//
// freshFiles is a map from relative path to the freshly generated file content
// that should replace the drifted file on disk.
func Repair(
	projectRoot string,
	opts RepairOptions,
	genState types.GeneratedState,
	freshFiles map[string]types.GeneratedFile,
	driftReport *drift.Report,
) (*RepairResult, *types.GeneratedState, error) {
	if driftReport == nil {
		return &RepairResult{}, &genState, nil
	}

	actions := classifyFindings(driftReport, genState, opts)

	// Filter to a single file when --file is specified.
	if opts.TargetFile != "" {
		var filtered []RepairAction
		for _, a := range actions {
			if a.File == opts.TargetFile {
				filtered = append(filtered, a)
			}
		}
		actions = filtered
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

	return result, &updatedState, nil
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
			// Non-fatal: log but continue.
			action.Error = fmt.Errorf("pruning backups for %s: %w", action.File, err)
			return action
		}
	}

	// Determine file mode.
	mode := fresh.Mode
	if mode == 0 {
		mode = 0o644
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
	updatedState.Files[action.File] = types.FileState{
		Hash:     state.ComputeHash(fresh.Content),
		Strategy: fresh.Strategy,
		Mode:     mode,
		Owner:    fresh.Owner,
	}

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
