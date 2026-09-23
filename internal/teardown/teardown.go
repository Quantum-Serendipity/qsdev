package teardown

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Teardown orchestrates the full teardown operation: load state, classify
// files, build and display the plan, optionally archive and assess posture,
// execute the plan, and display the result.
func Teardown(
	opts TeardownOptions,
	registry *toolreg.Registry,
	confirm func(*TeardownPlan, io.Writer) bool,
	w io.Writer,
) (*TeardownResult, error) {
	// 1. Load and merge all states.
	loaded := loadAndMergeStates(opts.ProjectRoot)
	mergedState := loaded.merged
	for _, lerr := range loaded.errs {
		fmt.Fprintf(w, "Warning: %v\n", lerr)
	}
	// An unreadable state file hides which files qsdev generated; tearing
	// down without it would orphan them and destroy the only record.
	if len(loaded.unreadable) > 0 && !opts.Force {
		return nil, fmt.Errorf("cannot read state file(s) %s: fix or restore them, or re-run with --force to tear down without them",
			strings.Join(loaded.unreadable, ", "))
	}

	// 2. Classify files.
	classified := ClassifyFiles(mergedState, opts.ProjectRoot, registry)

	// 3. Build plan, keeping any unreadable state file for recovery.
	plan := BuildPlan(classified, opts)
	keepStateFiles(plan, loaded.unreadable)

	// 4. Display plan.
	DisplayPlan(plan, w)

	// 5. If DryRun, return without execution.
	if opts.DryRun {
		return &TeardownResult{
			Removed:     plan.Remove,
			Preserved:   plan.Preserve,
			Cleaned:     plan.Clean,
			DirsRemoved: plan.Dirs,
			Errors:      loaded.errs,
		}, nil
	}

	// 6. Confirm if needed.
	if confirm != nil && !opts.Force {
		if !confirm(plan, w) {
			return nil, fmt.Errorf("teardown aborted by user")
		}
	}

	var archivePath string
	var reportPath string

	// 7. If compliance or archive: create archive.
	if opts.Archive {
		files := collectAllFilePaths(mergedState, opts.ProjectRoot)
		var err error
		archivePath, err = CreateArchive(opts.ProjectRoot, files)
		if err != nil {
			return nil, fmt.Errorf("creating archive: %w", err)
		}
	}

	// 8. If compliance: generate final posture report.
	if opts.Profile == ProfileCompliance {
		report, err := posture.Assess(opts.ProjectRoot, posture.AssessOptions{})
		if err != nil {
			// Non-fatal: log warning but continue with teardown.
			fmt.Fprintf(w, "Warning: could not generate posture report: %v\n", err)
		} else {
			reportJSON, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				fmt.Fprintf(w, "Warning: could not marshal posture report: %v\n", err)
			} else {
				reportPath = filepath.Join(opts.ProjectRoot, "."+branding.Get().AppName+"-posture-final.json")
				if err := fileutil.WriteFileAtomic(reportPath, reportJSON, fileutil.ModeReadWrite); err != nil {
					fmt.Fprintf(w, "Warning: could not write posture report: %v\n", err)
					reportPath = ""
				}
			}
		}
	}

	// 9. Execute plan.
	result, err := Execute(plan, opts, registry)
	if err != nil {
		return nil, fmt.Errorf("executing teardown: %w", err)
	}

	result.ArchivePath = archivePath
	result.ReportPath = reportPath
	result.Errors = slices.Concat(loaded.errs, result.Errors)

	// 10. Display result.
	DisplayResult(result, w)

	return result, nil
}

// loadedStates is the merged view of all state files plus the problems found
// while loading them.
type loadedStates struct {
	merged     types.GeneratedState
	unreadable []string // state files (relative paths) that failed to load
	errs       []error  // load failures and rejected entries
}

// loadAndMergeStates loads all three state files and merges their file maps.
// State files may be committed to the repository, so every tracked path is
// validated; unsafe entries are dropped and reported.
func loadAndMergeStates(projectRoot string) loadedStates {
	loaded := loadedStates{
		merged: types.GeneratedState{Files: make(map[string]types.FileState)},
	}

	for _, relPath := range state.StateFilePaths() {
		absPath := filepath.Join(projectRoot, relPath)
		st, err := state.LoadStateFromFile(absPath)
		if err != nil {
			loaded.unreadable = append(loaded.unreadable, relPath)
			loaded.errs = append(loaded.errs, fmt.Errorf("loading state file %s: %w", relPath, err))
			continue
		}
		for k, v := range st.Files {
			if err := checkTrackedPath(k); err != nil {
				loaded.errs = append(loaded.errs, fmt.Errorf("state file %s: ignoring entry: %w", relPath, err))
				continue
			}
			loaded.merged.Files[k] = v
		}
		if st.QsdevVersion != "" {
			loaded.merged.QsdevVersion = st.QsdevVersion
		}
	}

	return loaded
}

// keepStateFiles makes sure the plan deletes none of the given state files:
// they move from the removal list to the preserve list, and any directory
// the plan would remove wholesale that contains one is dropped from it.
func keepStateFiles(plan *TeardownPlan, keep []string) {
	if len(keep) == 0 {
		return
	}
	const reason = "state file could not be read; kept for recovery"

	remove := plan.Remove[:0]
	for _, fa := range plan.Remove {
		if slices.Contains(keep, fa.Path) {
			plan.Preserve = append(plan.Preserve, FileAction{Path: fa.Path, Reason: reason})
			continue
		}
		remove = append(remove, fa)
	}
	plan.Remove = remove

	dirs := plan.Dirs[:0]
	for _, dir := range plan.Dirs {
		prefix := strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
		holdsKept := slices.ContainsFunc(keep, func(p string) bool {
			return strings.HasPrefix(filepath.ToSlash(p), prefix)
		})
		if holdsKept {
			plan.Preserve = append(plan.Preserve, FileAction{Path: dir, Reason: "contains an unreadable state file; kept for recovery"})
			continue
		}
		dirs = append(dirs, dir)
	}
	plan.Dirs = dirs
}

// collectAllFilePaths returns the tracked file paths from the merged state
// that are safe to read from inside projectRoot.
func collectAllFilePaths(genState types.GeneratedState, projectRoot string) []string {
	files := make([]string, 0, len(genState.Files))
	for path := range genState.Files {
		if _, err := resolveTrackedPath(projectRoot, path); err != nil {
			continue
		}
		files = append(files, path)
	}
	return files
}
