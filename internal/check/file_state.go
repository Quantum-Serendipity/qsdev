package check

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
				Message:  "No state file configured. Run qsdev init first.",
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
				Remediation: "Run 'qsdev init' to regenerate the state file",
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
				Message:  "No state file found. Run qsdev init first.",
			},
		}
	}

	statuses := state.CheckModified(genState, ctx.ProjectRoot)

	var results []CheckResult
	hasIssues := false
	var userEdited []string

	// Iterate in sorted order so report output is reproducible across runs.
	for _, relPath := range slices.Sorted(maps.Keys(statuses)) {
		fs := statuses[relPath]
		storedFile := genState.Files[relPath]

		switch fs.Status {
		case types.Modified:
			switch storedFile.Strategy {
			case types.ManualMerge, types.SectionMarker, types.ThreeWayMerge:
				// User-editable strategies: modification is expected, not a
				// failure (their security content is checked separately).
				userEdited = append(userEdited, relPath)
			default:
				hasIssues = true
				results = append(results, CheckResult{
					Category:    CategoryFileState,
					Name:        "file_unmodified_" + relPath,
					Status:      StatusFail,
					Severity:    SeverityMedium,
					Message:     fmt.Sprintf("Generated file %s has been modified", relPath),
					FilePath:    relPath,
					Remediation: "Run 'qsdev init --force' to regenerate, or commit intentional changes",
				})
			}
		case types.Deleted:
			hasIssues = true
			results = append(results, CheckResult{
				Category:    CategoryFileState,
				Name:        "file_exists_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     fmt.Sprintf("Generated file %s has been deleted", relPath),
				FilePath:    relPath,
				Remediation: "Run 'qsdev check --auto-fix' or 'qsdev repair' to restore",
				AutoFixable: true,
				Metadata:    map[string]string{"file": relPath},
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
		message := "All generated files are unmodified"
		if len(userEdited) > 0 {
			message = fmt.Sprintf("No machine-owned generated file is modified; %d user-editable file(s) carry local edits: %s",
				len(userEdited), strings.Join(userEdited, ", "))
		}
		results = append(results, CheckResult{
			Category: CategoryFileState,
			Name:     "generated_files",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  message,
		})
	}

	return append(results, checkGeneratedSyntax(ctx.ProjectRoot, statuses)...)
}

// checkGeneratedSyntax validates every tracked generated file still on disk
// with the syntax validator init applies before writing (nix-instantiate
// --parse for devenv.nix, JSON, YAML, shell). A generated file that no longer
// parses, e.g. a devenv.nix broken by a later edit or write, breaks the
// environment for everyone who pulls it.
func checkGeneratedSyntax(projectRoot string, statuses map[string]state.FileStatus) []CheckResult {
	var results []CheckResult
	for _, relPath := range slices.Sorted(maps.Keys(statuses)) {
		switch statuses[relPath].Status {
		case types.Unmodified, types.Modified:
		default:
			continue
		}
		content, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(relPath)))
		if err != nil {
			continue // reported by the modification check
		}
		if err := generate.ValidateContent(relPath, content); err != nil {
			results = append(results, CheckResult{
				Category:    CategoryFileState,
				Name:        "file_syntax_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     fmt.Sprintf("Generated file %s does not parse: %v", relPath, err),
				FilePath:    relPath,
				Remediation: "Fix the syntax error or restore the file from version control",
			})
		}
	}
	return results
}

// ClaudeSettingsRelPath is the project-relative, slash-separated path of the
// Claude Code settings file that carries the deny rules.
const ClaudeSettingsRelPath = ".claude/settings.json"

func checkDenyRules(ctx CheckContext) []CheckResult {
	if len(ctx.RequiredDenyRules) == 0 {
		return nil
	}

	settingsPath := filepath.Join(ctx.ProjectRoot, filepath.FromSlash(ClaudeSettingsRelPath))
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return []CheckResult{settingsUnavailableResult(ctx, err)}
	}

	settings, err := parseSettingsPosture(data)
	if err != nil {
		return []CheckResult{
			{
				Category:    CategoryFileState,
				Name:        "deny_rules_present",
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Could not parse .claude/settings.json: %v", err),
				FilePath:    ClaudeSettingsRelPath,
				Remediation: "Fix JSON syntax in .claude/settings.json",
			},
		}
	}

	existingDeny := make(map[string]bool, len(settings.Deny))
	for _, rule := range settings.Deny {
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
				FilePath: ClaudeSettingsRelPath,
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
			FilePath:    ClaudeSettingsRelPath,
			Remediation: "Run 'qsdev check --auto-fix' to add missing deny rules",
			AutoFixable: true,
			Metadata:    map[string]string{"rule": rule},
		})
	}

	return results
}

// settingsUnavailableResult reports a missing or unreadable settings.json. When
// the project is configured for Claude Code the file is the only carrier of the
// deny rules, so its absence is a high-severity failure (the agent would run
// with no deny rules at all); otherwise there is nothing to enforce and the
// result is only a warning.
func settingsUnavailableResult(ctx CheckContext, err error) CheckResult {
	message := fmt.Sprintf("Could not read %s: %v", ClaudeSettingsRelPath, err)
	if os.IsNotExist(err) {
		message = ClaudeSettingsRelPath + " not found; cannot verify deny rules"
	}

	if !claudeCodeConfigured(ctx) {
		return CheckResult{
			Category:    CategoryFileState,
			Name:        "deny_rules_present",
			Status:      StatusWarn,
			Severity:    SeverityMedium,
			Message:     message,
			FilePath:    ClaudeSettingsRelPath,
			Remediation: "Run 'qsdev init' with Claude Code enabled",
		}
	}

	return CheckResult{
		Category:    CategoryFileState,
		Name:        "deny_rules_present",
		Status:      StatusFail,
		Severity:    SeverityHigh,
		Message:     message + " (Claude Code is enabled, so no deny rules are enforced)",
		FilePath:    ClaudeSettingsRelPath,
		Remediation: "Run 'qsdev repair' or 'qsdev init --update' to restore " + ClaudeSettingsRelPath,
	}
}

// claudeCodeConfigured reports whether the project is expected to carry a
// qsdev-managed .claude/settings.json. An explicit claude_code.enabled value in
// the config is authoritative; when the config is silent, the state file
// recording settings.json as a generated file means Claude Code was set up.
func claudeCodeConfigured(ctx CheckContext) bool {
	if ctx.QsdevConfig != nil && ctx.QsdevConfig.ClaudeCode.Enabled != nil {
		return *ctx.QsdevConfig.ClaudeCode.Enabled
	}
	if ctx.StateFile == "" {
		return false
	}
	genState, err := state.LoadStateFromFile(ctx.StateFile)
	if err != nil {
		return false
	}
	_, tracked := genState.Files[ClaudeSettingsRelPath]
	return tracked
}
