package teardown

import (
	"path"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// stateFilesForTeardown returns the state file paths that are removed as part
// of the default teardown: every addon state file (from state.StateFilePaths,
// the single source of truth) plus the saved answers and project config.
func stateFilesForTeardown() []string {
	b := branding.Get()
	statePaths := state.StateFilePaths()
	files := make([]string, 0, len(statePaths)+2)
	files = append(files, statePaths[:]...)
	return append(files,
		path.Join(answers.PrimaryDir(), answers.PrimaryFilename()),
		b.ConfigFile,
	)
}

// BuildPlan creates a TeardownPlan from classified files and options.
func BuildPlan(classified []ClassifiedFile, opts TeardownOptions) *TeardownPlan {
	plan := &TeardownPlan{
		Profile: opts.Profile,
		Dirs:    []string{branding.Get().StateDir},
	}

	if opts.Profile == ProfileQuick {
		// Quick profile: only dirs, no file operations.
		return plan
	}

	// Default and Compliance profiles process all classified files.
	for _, cf := range classified {
		if cf.Deleted {
			continue
		}

		switch cf.Ownership {
		case toolreg.Exclusive:
			if cf.Modified {
				plan.Preserve = append(plan.Preserve, FileAction{
					Path:     cf.Path,
					Reason:   "file has been modified by user",
					Modified: true,
				})
			} else {
				plan.Remove = append(plan.Remove, FileAction{
					Path:   cf.Path,
					Reason: "exclusively owned by " + branding.Get().AppName,
				})
			}
		case toolreg.Shared:
			plan.Clean = append(plan.Clean, FileAction{
				Path:        cf.Path,
				Reason:      "surgically remove " + branding.Get().AppName + " sections",
				BaseContent: cf.BaseContent,
			})
		}
	}

	// Add state files to the remove list.
	for _, sf := range stateFilesForTeardown() {
		plan.Remove = append(plan.Remove, FileAction{
			Path:   sf,
			Reason: branding.Get().AppName + " state file",
		})
	}

	return plan
}
