package devinit

import (
	"context"
	"fmt"
	"io"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// PrerequisiteStatus describes whether a required tool is present on the system.
type PrerequisiteStatus struct {
	Name        string
	Found       bool
	Path        string
	Version     string
	Required    bool
	InstallHint string
}

// PrerequisiteResult holds the results of checking all prerequisites.
type PrerequisiteResult struct {
	Tools []PrerequisiteStatus
}

// CheckPrerequisites checks for the tools the devenv environment requires
// (nix, devenv, direnv, git). The set, version flags and install hints come
// from the doctor registry so init and "qsdev devenv doctor" always agree on
// what is required.
func CheckPrerequisites(ctx context.Context) PrerequisiteResult {
	checks := doctor.RequiredChecks()

	result := PrerequisiteResult{
		Tools: make([]PrerequisiteStatus, 0, len(checks)),
	}

	for _, c := range checks {
		info := toolcheck.Detect(ctx, c.Binary, c.VersionFlag)
		version := info.Version
		if c.ParseVersion != nil && info.Output != "" {
			if parsed := c.ParseVersion(info.Output); parsed != "" {
				version = parsed
			}
		}
		result.Tools = append(result.Tools, PrerequisiteStatus{
			Name:        c.Name,
			Found:       info.Found,
			Path:        info.Path,
			Version:     version,
			Required:    c.Required,
			InstallHint: c.InstallHint,
		})
	}

	return result
}

// HasMissing returns true if any required prerequisite is not found.
func (r PrerequisiteResult) HasMissing() bool {
	for _, t := range r.Tools {
		if t.Required && !t.Found {
			return true
		}
	}
	return false
}

// PrintReport writes a human-readable prerequisite check report to w.
func (r PrerequisiteResult) PrintReport(w io.Writer) {
	for _, t := range r.Tools {
		status := "OK"
		if !t.Found {
			if t.Required {
				status = "MISSING"
			} else {
				status = "not found"
			}
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
