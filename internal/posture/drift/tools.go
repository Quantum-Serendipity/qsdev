package drift

import (
	"fmt"
	"os/exec"
)

const categoryToolAvailability = "Tool Availability"

// lookPath is a package-level function variable wrapping exec.LookPath
// to allow test injection.
var lookPath = exec.LookPath

// toolBinaries maps tool names to the binary that must be on PATH for
// the tool to function.
var toolBinaries = map[string]string{
	"semgrep":            "semgrep",
	"gitleaks":           "gitleaks",
	"attach-guard":       "python3",
	"container-security": "grype",
	"commitlint":         "commitlint",
	"license-compliance": "scancode",
}

// detectToolAvailability checks whether the binaries for each enabled tool
// are available on the system PATH.
func detectToolAvailability(enabledTools map[string]bool) Category {
	cat := Category{Name: categoryToolAvailability}

	for tool, enabled := range enabledTools {
		if !enabled {
			continue
		}

		binary, hasBinary := toolBinaries[tool]
		if !hasBinary {
			// Tools without binary requirements (MCP servers, skills) are skipped.
			continue
		}

		if _, err := lookPath(binary); err != nil {
			cat.Findings = append(cat.Findings, Finding{
				Category:    categoryToolAvailability,
				Severity:    Warning,
				Subject:     tool,
				Description: fmt.Sprintf("Required binary %q for tool %q is not available on PATH", binary, tool),
				Expected:    binary,
				Remediation: fmt.Sprintf("Install %s or add it to your PATH", binary),
			})
		}
	}

	return cat
}
