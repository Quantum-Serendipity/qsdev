package teardown

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// State file paths to load (same as posture.stateFilePaths).
var allStatePaths = [3]string{
	".devinit/.gdev-init-state.yaml",
	".devenv/.gdev-state.yaml",
	".claude/.gdev-claude-state.yaml",
}

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
	mergedState := loadAndMergeStates(opts.ProjectRoot)

	// 2. Classify files.
	classified := ClassifyFiles(mergedState, opts.ProjectRoot, registry)

	// 3. Build plan.
	plan := BuildPlan(classified, opts)

	// 4. Display plan.
	DisplayPlan(plan, w)

	// 5. If DryRun, return without execution.
	if opts.DryRun {
		return &TeardownResult{
			Removed:     plan.Remove,
			Preserved:   plan.Preserve,
			Cleaned:     plan.Clean,
			DirsRemoved: plan.Dirs,
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
		files := collectAllFilePaths(mergedState)
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
				reportPath = filepath.Join(opts.ProjectRoot, ".gdev-posture-final.json")
				if err := os.WriteFile(reportPath, reportJSON, 0o644); err != nil {
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

	// 10. Display result.
	DisplayResult(result, w)

	return result, nil
}

// loadAndMergeStates loads all three state files and merges their file maps.
func loadAndMergeStates(projectRoot string) types.GeneratedState {
	merged := types.GeneratedState{
		Files: make(map[string]types.FileState),
	}

	for _, relPath := range allStatePaths {
		absPath := filepath.Join(projectRoot, relPath)
		st, err := state.LoadStateFromFile(absPath)
		if err != nil {
			continue
		}
		for k, v := range st.Files {
			merged.Files[k] = v
		}
		if st.GdevVersion != "" {
			merged.GdevVersion = st.GdevVersion
		}
	}

	return merged
}

// collectAllFilePaths returns all tracked file paths from the merged state.
func collectAllFilePaths(genState types.GeneratedState) []string {
	files := make([]string, 0, len(genState.Files))
	for path := range genState.Files {
		files = append(files, path)
	}
	return files
}
