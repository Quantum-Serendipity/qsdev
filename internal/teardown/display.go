package teardown

import (
	"fmt"
	"io"
)

// DisplayPlan writes a human-readable summary of the teardown plan to w.
func DisplayPlan(plan *TeardownPlan, w io.Writer) {
	fmt.Fprintf(w, "Teardown plan (profile: %s)\n", plan.Profile)
	fmt.Fprintln(w, "")

	if len(plan.Remove) > 0 {
		fmt.Fprintf(w, "Files to remove (%d):\n", len(plan.Remove))
		for _, fa := range plan.Remove {
			fmt.Fprintf(w, "  - %s  (%s)\n", fa.Path, fa.Reason)
		}
		fmt.Fprintln(w, "")
	}

	if len(plan.Clean) > 0 {
		fmt.Fprintf(w, "Shared files to clean (%d):\n", len(plan.Clean))
		for _, fa := range plan.Clean {
			fmt.Fprintf(w, "  ~ %s  (%s)\n", fa.Path, fa.Reason)
		}
		fmt.Fprintln(w, "")
	}

	if len(plan.Preserve) > 0 {
		fmt.Fprintf(w, "Modified files preserved (%d):\n", len(plan.Preserve))
		for _, fa := range plan.Preserve {
			fmt.Fprintf(w, "  ! %s  (%s)\n", fa.Path, fa.Reason)
		}
		fmt.Fprintln(w, "")
	}

	if len(plan.Dirs) > 0 {
		fmt.Fprintf(w, "Directories to remove (%d):\n", len(plan.Dirs))
		for _, d := range plan.Dirs {
			fmt.Fprintf(w, "  - %s/\n", d)
		}
		fmt.Fprintln(w, "")
	}
}

// DisplayResult writes a human-readable summary of the teardown result to w.
func DisplayResult(result *TeardownResult, w io.Writer) {
	fmt.Fprintln(w, "Teardown complete.")

	if len(result.Removed) > 0 {
		fmt.Fprintf(w, "  Removed %d file(s).\n", len(result.Removed))
	}
	if len(result.Cleaned) > 0 {
		fmt.Fprintf(w, "  Cleaned %d shared file(s).\n", len(result.Cleaned))
	}
	if len(result.Preserved) > 0 {
		fmt.Fprintf(w, "  Preserved %d modified file(s).\n", len(result.Preserved))
	}
	if len(result.DirsRemoved) > 0 {
		fmt.Fprintf(w, "  Removed %d directory(ies).\n", len(result.DirsRemoved))
	}
	if result.ArchivePath != "" {
		fmt.Fprintf(w, "  Archive: %s\n", result.ArchivePath)
	}
	if result.ReportPath != "" {
		fmt.Fprintf(w, "  Posture report: %s\n", result.ReportPath)
	}
	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "  Errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Fprintf(w, "    - %s\n", err)
		}
	}
}
