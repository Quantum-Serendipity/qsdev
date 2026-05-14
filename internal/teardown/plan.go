package teardown

import "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"

// State file paths that are removed as part of the default teardown.
var stateFiles = []string{
	".devenv/.gdev-state.yaml",
	".claude/.gdev-claude-state.yaml",
}

// BuildPlan creates a TeardownPlan from classified files and options.
func BuildPlan(classified []ClassifiedFile, opts TeardownOptions) *TeardownPlan {
	plan := &TeardownPlan{
		Profile: opts.Profile,
		Dirs:    []string{".devinit"},
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
					Reason: "exclusively owned by gdev",
				})
			}
		case toolreg.Shared:
			plan.Clean = append(plan.Clean, FileAction{
				Path:   cf.Path,
				Reason: "surgically remove gdev sections",
			})
		}
	}

	// Add state files to the remove list.
	for _, sf := range stateFiles {
		plan.Remove = append(plan.Remove, FileAction{
			Path:   sf,
			Reason: "gdev state file",
		})
	}

	return plan
}
