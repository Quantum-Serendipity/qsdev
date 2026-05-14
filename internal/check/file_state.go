package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// CheckFileState verifies that generated files have not been modified or
// deleted, and that settings.json contains required deny rules.
func CheckFileState(ctx CheckContext) []CheckResult {
	var results []CheckResult

	results = append(results, checkGeneratedFiles(ctx)...)
	results = append(results, checkDenyRules(ctx)...)

	return results
}

func checkGeneratedFiles(ctx CheckContext) []CheckResult {
	if ctx.StateFile == "" {
		return []CheckResult{
			{
				Category: CategoryFileState,
				Name:     "generated_files",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No state file configured. Run gdev init first.",
			},
		}
	}

	genState, err := state.LoadStateFromFile(ctx.StateFile)
	if err != nil {
		return []CheckResult{
			{
				Category:    CategoryFileState,
				Name:        "generated_files",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     fmt.Sprintf("Failed to load state file: %v", err),
				Remediation: "Run 'gdev init' to regenerate the state file",
			},
		}
	}

	if len(genState.Files) == 0 {
		return []CheckResult{
			{
				Category: CategoryFileState,
				Name:     "generated_files",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No state file found. Run gdev init first.",
			},
		}
	}

	statuses := state.CheckModified(genState, ctx.ProjectRoot)

	var results []CheckResult
	hasIssues := false

	for relPath, fs := range statuses {
		switch fs.Status {
		case types.Modified:
			hasIssues = true
			results = append(results, CheckResult{
				Category:    CategoryFileState,
				Name:        "file_unmodified_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Generated file %s has been modified", relPath),
				FilePath:    relPath,
				Remediation: "Run 'gdev init --force' to regenerate, or commit intentional changes",
			})
		case types.Deleted:
			hasIssues = true
			results = append(results, CheckResult{
				Category:    CategoryFileState,
				Name:        "file_exists_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     fmt.Sprintf("Generated file %s has been deleted", relPath),
				FilePath:    relPath,
				Remediation: "Run 'gdev init' to regenerate the file",
			})
		case types.Unknown:
			if fs.Error != nil {
				results = append(results, CheckResult{
					Category: CategoryFileState,
					Name:     "file_check_" + relPath,
					Status:   StatusWarn,
					Severity: SeverityLow,
					Message:  fmt.Sprintf("Could not check file %s: %v", relPath, fs.Error),
					FilePath: relPath,
				})
			}
		}
	}

	if !hasIssues && len(results) == 0 {
		results = append(results, CheckResult{
			Category: CategoryFileState,
			Name:     "generated_files",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "All generated files are unmodified",
		})
	}

	return results
}

func checkDenyRules(ctx CheckContext) []CheckResult {
	if len(ctx.RequiredDenyRules) == 0 {
		return nil
	}

	settingsPath := filepath.Join(ctx.ProjectRoot, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CheckResult{
				{
					Category:    CategoryFileState,
					Name:        "deny_rules_present",
					Status:      StatusWarn,
					Severity:    SeverityMedium,
					Message:     ".claude/settings.json not found; cannot verify deny rules",
					FilePath:    ".claude/settings.json",
					Remediation: "Run 'gdev init' with Claude Code enabled",
				},
			}
		}
		return []CheckResult{
			{
				Category: CategoryFileState,
				Name:     "deny_rules_present",
				Status:   StatusWarn,
				Severity: SeverityMedium,
				Message:  fmt.Sprintf("Could not read .claude/settings.json: %v", err),
				FilePath: ".claude/settings.json",
			},
		}
	}

	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return []CheckResult{
			{
				Category:    CategoryFileState,
				Name:        "deny_rules_present",
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Could not parse .claude/settings.json: %v", err),
				FilePath:    ".claude/settings.json",
				Remediation: "Fix JSON syntax in .claude/settings.json",
			},
		}
	}

	existingDeny := make(map[string]bool, len(settings.Permissions.Deny))
	for _, rule := range settings.Permissions.Deny {
		existingDeny[rule] = true
	}

	var missing []string
	for _, required := range ctx.RequiredDenyRules {
		if !existingDeny[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) == 0 {
		return []CheckResult{
			{
				Category: CategoryFileState,
				Name:     "deny_rules_present",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "All required deny rules are present in settings.json",
				FilePath: ".claude/settings.json",
			},
		}
	}

	var results []CheckResult
	for _, rule := range missing {
		results = append(results, CheckResult{
			Category:    CategoryFileState,
			Name:        "deny_rule_missing",
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("Required deny rule missing: %s", rule),
			FilePath:    ".claude/settings.json",
			Remediation: "Run 'gdev check --auto-fix' to add missing deny rules",
			AutoFixable: true,
		})
	}

	return results
}
