package teardown

import (
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// stateFilesForTeardown returns the state file paths that are removed as part
// of the default teardown, built dynamically from branding.
func stateFilesForTeardown() []string {
	b := branding.Get()
	return []string{
		".devenv/." + b.AppName + "-state.yaml",
		".claude/." + b.AppName + "-claude-state.yaml",
		b.StateDir + "/." + b.AppName + "-init-state.yaml",
		b.StateDir + "/." + b.AppName + "-init-answers.yaml",
		b.ConfigFile,
	}
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
				Path:   cf.Path,
				Reason: "surgically remove " + branding.Get().AppName + " sections",
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
