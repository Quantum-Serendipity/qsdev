package check

import (
	"errors"
	"fmt"
	"io/fs"
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

// checkGeneratedFiles verifies the generated files on disk against what qsdev
// recorded writing. Two records exist: the local generation state (gitignored,
// so absent on a CI checkout) and the committed manifest of machine-owned
// files. The manifest is the authority for the files it lists, which is what
// lets a clean CI checkout detect a deleted or edited hook; the local state
// adds file modes, the human-edited files and anything the manifest does not
// cover.
func checkGeneratedFiles(ctx CheckContext) []CheckResult {
	var results []CheckResult

	genState, loadFailure := loadGenerationState(ctx.StateFile)
	if loadFailure != nil {
		results = append(results, *loadFailure)
	}
	manifest, manifestResults := loadCommittedManifest(ctx, genState)
	results = append(results, manifestResults...)

	expected := expectedGeneratedState(genState, manifest)
	if len(expected.Files) == 0 {
		if len(results) > 0 {
			return results
		}
		message := "No state file found. Run qsdev init first."
		if ctx.StateFile == "" {
			message = "No state file configured. Run qsdev init first."
		}
		return []CheckResult{{
			Category: CategoryFileState,
			Name:     "generated_files",
			Status:   StatusSkip,
			Severity: SeverityInfo,
			Message:  message,
		}}
	}

	return append(results, verifyGeneratedFiles(ctx.ProjectRoot, expected)...)
}

// loadGenerationState loads the local generation state. A missing file yields
// an empty state; an unreadable one also yields a failure result.
func loadGenerationState(stateFile string) (types.GeneratedState, *CheckResult) {
	empty := types.GeneratedState{Files: map[string]types.FileState{}}
	if stateFile == "" {
		return empty, nil
	}
	genState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return empty, &CheckResult{
			Category:    CategoryFileState,
			Name:        "generated_files",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     fmt.Sprintf("Failed to load state file: %v", err),
			Remediation: "Run 'qsdev init' to regenerate the state file",
		}
	}
	return genState, nil
}

// loadCommittedManifest loads the committed generated-file manifest. When the
// project has a config but no usable manifest, CI cannot tell whether a
// generated file was edited or deleted, so that is a high-severity failure. It
// is auto-fixable when a local generation state exists to rebuild it from.
func loadCommittedManifest(ctx CheckContext, genState types.GeneratedState) (state.Manifest, []CheckResult) {
	if ctx.ManifestFile == "" {
		return nil, nil
	}
	name := state.ManifestFile()
	manifest, err := state.LoadManifest(ctx.ManifestFile)
	if err == nil && len(manifest) == 0 && projectConfigured(ctx) {
		// Every qsdev setup generates machine-owned files, so an empty
		// manifest verifies nothing: treat it like a missing one rather than
		// letting a truncated file turn the check into a skip.
		err = errEmptyManifest
	}
	if err == nil {
		return manifest, manifestCoverage(manifest, genState)
	}

	missing := errors.Is(err, fs.ErrNotExist)
	if missing && !projectConfigured(ctx) {
		return nil, nil
	}
	message := fmt.Sprintf("%s is missing, so generated files cannot be verified on a clean checkout (CI)", name)
	if !missing {
		message = fmt.Sprintf("%s cannot be used: %v", name, err)
	}
	result := CheckResult{
		Category:    CategoryFileState,
		Name:        manifestCheckName,
		Status:      StatusFail,
		Severity:    SeverityHigh,
		Message:     message,
		FilePath:    name,
		Remediation: fmt.Sprintf("Run 'qsdev init --update' locally to regenerate it, then commit %s", name),
	}
	if len(genState.Files) > 0 {
		result.AutoFixable = true
		result.Remediation = fmt.Sprintf("Run 'qsdev check --auto-fix' or 'qsdev init --update' to write it from the local generation state, then commit %s", name)
	}
	return nil, []CheckResult{result}
}

// errEmptyManifest reports a manifest that lists no files for a configured
// project.
var errEmptyManifest = errors.New("it lists no generated files")

// manifestCheckName names the result reporting a missing or unusable manifest;
// ApplyAutoFixes rewrites the manifest for it.
const manifestCheckName = "generated_manifest"

// manifestCoverage warns about machine-owned files the local state tracks but
// the committed manifest does not list: CI does not verify those.
func manifestCoverage(manifest state.Manifest, genState types.GeneratedState) []CheckResult {
	var uncovered []string
	for relPath := range state.BuildManifest(genState) {
		if _, ok := manifest[relPath]; !ok {
			uncovered = append(uncovered, relPath)
		}
	}
	if len(uncovered) == 0 {
		return nil
	}
	slices.Sort(uncovered)
	name := state.ManifestFile()
	return []CheckResult{{
		Category:    CategoryFileState,
		Name:        "generated_manifest_coverage",
		Status:      StatusWarn,
		Severity:    SeverityLow,
		Message:     fmt.Sprintf("%d generated file(s) are not listed in %s, so CI does not verify them: %s", len(uncovered), name, strings.Join(uncovered, ", ")),
		FilePath:    name,
		Remediation: fmt.Sprintf("Run 'qsdev init --update' and commit %s", name),
	}}
}

// projectConfigured reports whether the project has a config file, parsed or
// not: such a project was set up by qsdev and must carry a manifest.
func projectConfigured(ctx CheckContext) bool {
	return ctx.QsdevConfig != nil || configParseFailed(ctx)
}

// expectedGeneratedState overlays the committed manifest on the local state:
// a manifest entry sets the expected hash of its file (keeping the strategy
// and mode the local state records, if any), so a file matching the committed
// manifest is not reported as modified just because the local state is older.
func expectedGeneratedState(genState types.GeneratedState, manifest state.Manifest) types.GeneratedState {
	if len(manifest) == 0 {
		return genState
	}
	expected := types.GeneratedState{Files: make(map[string]types.FileState, len(genState.Files)+len(manifest))}
	maps.Copy(expected.Files, genState.Files)
	for relPath, hash := range manifest {
		entry := expected.Files[relPath]
		entry.Hash = hash
		expected.Files[relPath] = entry
	}
	return expected
}

// verifyGeneratedFiles reports each expected file that was modified, deleted
// or no longer parses.
func verifyGeneratedFiles(projectRoot string, expected types.GeneratedState) []CheckResult {
	statuses := state.CheckModified(expected, projectRoot)

	var results []CheckResult
	hasIssues := false
	var userEdited []string

	// Iterate in sorted order so report output is reproducible across runs.
	for _, relPath := range slices.Sorted(maps.Keys(statuses)) {
		status := statuses[relPath]
		storedFile := expected.Files[relPath]

		switch status.Status {
		case types.Modified:
			if storedFile.Strategy.IsHumanEdited() {
				// User-editable strategies: modification is expected, not a
				// failure (their security content is checked separately).
				userEdited = append(userEdited, relPath)
				continue
			}
			hasIssues = true
			results = append(results, CheckResult{
				Category:    CategoryFileState,
				Name:        "file_unmodified_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Generated file %s has been modified", relPath),
				FilePath:    relPath,
				Remediation: "Run 'qsdev repair' or 'qsdev init --force' to regenerate it; machine-owned generated files are not edited by hand",
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
				Remediation: "Run 'qsdev check --auto-fix' or 'qsdev repair' to restore",
				AutoFixable: true,
				Metadata:    map[string]string{"file": relPath},
			})
		case types.Unknown:
			if status.Error != nil {
				results = append(results, CheckResult{
					Category: CategoryFileState,
					Name:     "file_check_" + relPath,
					Status:   StatusWarn,
					Severity: SeverityLow,
					Message:  fmt.Sprintf("Could not check file %s: %v", relPath, status.Error),
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

	return append(results, checkGeneratedSyntax(projectRoot, statuses)...)
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
