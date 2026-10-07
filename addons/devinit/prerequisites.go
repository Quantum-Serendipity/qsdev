package devinit

import (
	"context"
	"fmt"
	"io"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// PrerequisiteStatus describes whether a required tool is present on the
// system and meets its minimum version.
type PrerequisiteStatus struct {
	Name        string
	Found       bool
	Path        string
	Version     string
	MinVersion  string
	VersionOK   bool
	Required    bool
	InstallHint string
}

// needsSetup is doctor.ToolStatus.NeedsSetup for this tool: it is missing,
// or has a floor its version (if known) does not meet.
func (t PrerequisiteStatus) needsSetup() bool {
	return doctor.ToolStatus{Installed: t.Found, MinVersion: t.MinVersion, VersionOK: t.VersionOK}.NeedsSetup()
}

// PrerequisiteResult holds the results of checking all prerequisites.
type PrerequisiteResult struct {
	Tools []PrerequisiteStatus
}

// CheckPrerequisites checks for the tools the devenv environment requires
// (nix, devenv, direnv, git) with the doctor registry's checks, so init and
// "qsdev devenv doctor" always agree on what is required, how it is probed
// and the floor it must meet.
func CheckPrerequisites(ctx context.Context) PrerequisiteResult {
	checks := doctor.RequiredChecks()
	statuses := doctor.RunChecks(ctx, sysinfo.DetectOS(), checks)

	result := PrerequisiteResult{
		Tools: make([]PrerequisiteStatus, 0, len(checks)),
	}
	for i, st := range statuses {
		result.Tools = append(result.Tools, PrerequisiteStatus{
			Name:        st.Name,
			Found:       st.Installed,
			Path:        st.Path,
			Version:     st.Version,
			MinVersion:  st.MinVersion,
			VersionOK:   st.VersionOK,
			Required:    st.Required,
			InstallHint: checks[i].InstallHint,
		})
	}
	return result
}

// HasMissing returns true if any required prerequisite is missing or below
// its minimum version.
func (r PrerequisiteResult) HasMissing() bool {
	for _, t := range r.Tools {
		if t.Required && t.needsSetup() {
			return true
		}
	}
	return false
}

// PrintReport writes a human-readable prerequisite check report to w.
func (r PrerequisiteResult) PrintReport(w io.Writer) {
	for _, t := range r.Tools {
		status := "OK"
		switch {
		case !t.Found && t.Required:
			status = "MISSING"
		case !t.Found:
			status = "not found"
		case t.needsSetup():
			status = "below minimum " + t.MinVersion
		}

		if t.Found && t.Version != "" {
			fmt.Fprintf(w, "  %-10s %s (%s)\n", t.Name, status, t.Version)
		} else {
			fmt.Fprintf(w, "  %-10s %s\n", t.Name, status)
		}

		if !t.Found && t.InstallHint != "" {
			fmt.Fprintf(w, "             %s\n", t.InstallHint)
		}
	}
}
