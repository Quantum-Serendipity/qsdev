package devinit

import (
	"context"
	"fmt"
	"io"

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

// CheckPrerequisites checks for required development tools on the system.
// It checks for nix, devenv, direnv, and git.
func CheckPrerequisites(ctx context.Context) PrerequisiteResult {
	checks := []struct {
		name        string
		versionArg  string
		required    bool
		installHint string
	}{
		{
			name:        "nix",
			versionArg:  "--version",
			required:    true,
			installHint: "Install Nix: https://nixos.org/download.html",
		},
		{
			name:        "devenv",
			versionArg:  "version",
			required:    true,
			installHint: "Install devenv: https://devenv.sh/getting-started/",
		},
		{
			name:        "direnv",
			versionArg:  "version",
			required:    true,
			installHint: "Install direnv: https://direnv.net/docs/installation.html",
		},
		{
			name:        "git",
			versionArg:  "--version",
			required:    true,
			installHint: "Install git via your system package manager.",
		},
	}

	result := PrerequisiteResult{
		Tools: make([]PrerequisiteStatus, 0, len(checks)),
	}

	for _, c := range checks {
		info := toolcheck.Detect(ctx, c.name, c.versionArg)
		result.Tools = append(result.Tools, PrerequisiteStatus{
			Name:        c.name,
			Found:       info.Found,
			Path:        info.Path,
			Version:     info.Version,
			Required:    c.required,
			InstallHint: c.installHint,
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
