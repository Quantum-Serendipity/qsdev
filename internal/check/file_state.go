package check

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
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

	expected := state.ExpectedState(genState, manifest)
	guards := guardScripts(ctx)
	// A guard the generator writes is verified even when nothing records it.
	generatesGuard := slices.ContainsFunc(guards, func(g string) bool { return ctx.GeneratedContent[g] != nil })
	if len(expected.Files) == 0 && !generatesGuard {
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

	return append(results, verifyGeneratedFiles(ctx.ProjectRoot, expected, guards, ctx.GuardSupportFiles, ctx.GeneratedContent)...)
}

// guardScripts returns the project hook scripts a PreToolUse hook runs: the
// guards that decide whether an agent action is allowed. The role comes from
// the hook registry's output, the generated settings, or, when those are not
// available, the project's effective settings on disk. An unreadable settings
// file yields no guards; its own check reports it. When any guard is a Python
// script, the guard support files (ctx.GuardSupportFiles) it loads are
// guards too.
func guardScripts(ctx CheckContext) []string {
	guards := registeredGuards(ctx)
	if slices.ContainsFunc(guards, func(g string) bool { return path.Ext(g) == ".py" }) {
		for _, f := range ctx.GuardSupportFiles {
			if !slices.Contains(guards, f) {
				guards = append(guards, f)
			}
		}
	}
	return guards
}

// registeredGuards returns the project hook scripts a PreToolUse hook runs
// (see guardScripts).
func registeredGuards(ctx CheckContext) []string {
	if len(ctx.ExpectedClaudeSettings) > 0 {
		if expected, err := claudesettings.Parse(ctx.ExpectedClaudeSettings); err == nil {
			return expected.Scripts(claudesettings.EventPreToolUse)
		}
	}
	effective, err := claudesettings.Read(ctx.ProjectRoot)
	if err != nil {
		return nil
	}
	return effective.Scripts(claudesettings.EventPreToolUse)
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

// verifyGeneratedFiles reports each expected file that was modified, deleted
// or no longer parses. A modified or deleted guard (one of guards, see
// guardScripts, which support lists the guard support files of) is
// critical: with it gone, the agent actions it vets run unchecked. A guard the
// generator writes (listed in generated) is judged against that content, line
// endings aside, whether or not expected lists it: the expected hashes come
// from the local state and the committed manifest, and a change to the guard
// can re-hash or drop its entry.
func verifyGeneratedFiles(projectRoot string, expected types.GeneratedState, guards, support []string, generated map[string][]byte) []CheckResult {
	statuses := state.CheckModified(expected, projectRoot)
	for _, guard := range guards {
		if content, ok := generated[guard]; ok {
			statuses[guard] = state.CheckContent(projectRoot, guard, content, expected.Files[guard].Mode)
		}
	}
	// In an un-joined checkout (a fresh clone or CI) repair and update
	// refuse to run until the checkout is joined, and auto-fix has no local
	// generation to restore from, so the remediation joins first.
	needsJoin, _ := state.NeedsJoin(projectRoot)
	remediate := func(r CheckResult) CheckResult {
		if needsJoin {
			r.Remediation = joinFirstRemediation + r.Remediation
			r.AutoFixable = false
		}
		return r
	}

	var results []CheckResult
	hasIssues := false
	var userEdited []string

	// Iterate in sorted order so report output is reproducible across runs.
	for _, relPath := range slices.Sorted(maps.Keys(statuses)) {
		status := statuses[relPath]
		storedFile := expected.Files[relPath]

		switch status.Status {
		case types.Modified:
			if storedFile.Strategy.IsHumanEdited() && !slices.Contains(guards, relPath) {
				// User-editable strategies: modification is expected, not a
				// failure (their security content is checked separately).
				userEdited = append(userEdited, relPath)
				continue
			}
			hasIssues = true
			r := CheckResult{
				Category:    CategoryFileState,
				Name:        "file_unmodified_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Generated file %s has been modified", relPath),
				FilePath:    relPath,
				Remediation: "Run 'qsdev repair' or 'qsdev init --force' to regenerate it; machine-owned generated files are not edited by hand",
			}
			if storedFile.Strategy == types.Skip {
				r.Remediation = skipModifiedRemediation(relPath)
			}
			if slices.Contains(guards, relPath) {
				r.Severity = SeverityCritical
				r.Message = fmt.Sprintf("%s has been modified", guardRole(relPath, support))
				if generated[relPath] != nil {
					r.Message = fmt.Sprintf("%s differs from the version this qsdev generates", guardRole(relPath, support))
				}
				r.Remediation = guardRemediation(relPath)
			}
			results = append(results, remediate(r))
		case types.Deleted:
			hasIssues = true
			r := CheckResult{
				Category:    CategoryFileState,
				Name:        "file_exists_" + relPath,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     fmt.Sprintf("Generated file %s has been deleted", relPath),
				FilePath:    relPath,
				Remediation: "Run 'qsdev check --auto-fix' or 'qsdev repair' to restore",
				AutoFixable: true,
				Metadata:    map[string]string{"file": relPath},
			}
			if slices.Contains(guards, relPath) {
				r.Severity = SeverityCritical
				r.Message = fmt.Sprintf("%s has been deleted", guardRole(relPath, support))
				r.Remediation = guardRemediation(relPath) + ", or 'qsdev check --auto-fix'"
			}
			results = append(results, remediate(r))
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

// guardRole names the guard at relPath for a result message: a guard script
// a PreToolUse hook runs, or a support file (one of support) the Python guards
// load.
func guardRole(relPath string, support []string) string {
	if slices.Contains(support, relPath) {
		return fmt.Sprintf("Guard support file %s, loaded by every Python guard,", relPath)
	}
	return fmt.Sprintf("Guard script %s, run by a PreToolUse hook,", relPath)
}

// joinFirstRemediation prefixes a restore remediation in an un-joined
// checkout, where the restoring commands refuse to run until it is joined.
const joinFirstRemediation = "Run 'qsdev init --yes' to join this checkout, then: "

// guardRemediation restores the generated version of the guard script at
// relPath. --configs-only keeps the update from replacing the binary.
func guardRemediation(relPath string) string {
	return fmt.Sprintf("Run 'qsdev update --configs-only --overwrite-modified' to restore the generated %s", relPath)
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
const ClaudeSettingsRelPath = claudesettings.ProjectRelPath

func checkDenyRules(ctx CheckContext) []CheckResult {
	if len(ctx.RequiredDenyRules) == 0 {
		return nil
	}

	settingsPath := filepath.Join(ctx.ProjectRoot, filepath.FromSlash(ClaudeSettingsRelPath))
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return []CheckResult{settingsUnavailableResult(ctx, err)}
	}

	settings, err := claudesettings.Parse(data)
	if err != nil {
		return []CheckResult{
			{
				Category:    CategoryFileState,
				Name:        "deny_rules_present",
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     fmt.Sprintf("Could not parse %s: %v", ClaudeSettingsRelPath, err),
				FilePath:    ClaudeSettingsRelPath,
				Remediation: "Fix JSON syntax in " + ClaudeSettingsRelPath,
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
// qsdev-managed .claude/settings.json (see ClaudeCodeConfigured).
func claudeCodeConfigured(ctx CheckContext) bool {
	return ClaudeCodeConfigured(ctx.QsdevConfig, ctx.StateFile)
}

// ClaudeCodeConfigured reports whether a project with config cfg and init
// state file stateFile is expected to carry a qsdev-managed
// .claude/settings.json. An explicit claude_code.enabled value in the config
// is authoritative; when the config is silent, the state file recording
// settings.json as a generated file means Claude Code was set up.
func ClaudeCodeConfigured(cfg *types.QsdevConfig, stateFile string) bool {
	if cfg != nil && cfg.ClaudeCode.Enabled != nil {
		return *cfg.ClaudeCode.Enabled
	}
	if stateFile == "" {
		return false
	}
	genState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return false
	}
	_, tracked := genState.Files[ClaudeSettingsRelPath]
	return tracked
}

// skipModifiedRemediation tells the user how to fix an edited
// skip-if-exists file: generation never writes over an existing one that
// differs from what qsdev generated, so 'qsdev repair' and
// 'qsdev init --force' keep the edit.
func skipModifiedRemediation(path string) string {
	return fmt.Sprintf("%s is created only when absent, so 'qsdev repair' and 'qsdev init --force' keep your edits; review the change and fix it by hand, or delete the file and run 'qsdev repair' to regenerate it", path)
}
