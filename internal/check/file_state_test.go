package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckFileState_AllFilesMatch(t *testing.T) {
	dir := t.TempDir()

	// Create a file.
	content := []byte("hello world")
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create state tracking that file.
	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "test.txt",
			Content: content,
			Mode:    0o644,
		},
	})

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}

	// Should have a passing result.
	hasPass := false
	for _, r := range results {
		if r.Status == StatusPass {
			hasPass = true
			break
		}
	}
	if !hasPass {
		t.Error("expected at least one passing result")
	}
}

func TestCheckFileState_ModifiedFile(t *testing.T) {
	dir := t.TempDir()

	originalContent := []byte("original")
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, originalContent, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "test.txt",
			Content: originalContent,
			Mode:    0o644,
		},
	})

	// Now modify the file.
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityMedium {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a medium-severity failure for modified file")
	}
}

func TestCheckFileState_ModifiedUserEditableStrategy(t *testing.T) {
	strategies := []types.MergeStrategy{
		types.ManualMerge,
		types.SectionMarker,
		types.ThreeWayMerge,
	}

	for _, strat := range strategies {
		t.Run(strat.String(), func(t *testing.T) {
			dir := t.TempDir()

			originalContent := []byte("original")
			testFile := filepath.Join(dir, "test.txt")
			if err := os.WriteFile(testFile, originalContent, 0o644); err != nil {
				t.Fatal(err)
			}

			genState := state.RecordFiles([]types.GeneratedFile{
				{Path: "test.txt", Content: originalContent, Mode: 0o644, Strategy: strat},
			})

			if err := os.WriteFile(testFile, []byte("modified by user"), 0o644); err != nil {
				t.Fatal(err)
			}

			stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				t.Fatal(err)
			}

			results := checkGeneratedFiles(CheckContext{ProjectRoot: dir, StateFile: stateFile})

			for _, r := range results {
				if r.Status == StatusFail {
					t.Errorf("strategy %s: should not fail for user-editable file, got: %s", strat, r.Message)
				}
			}
		})
	}
}

func TestCheckFileState_DeletedFile(t *testing.T) {
	dir := t.TempDir()

	originalContent := []byte("original")

	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "deleted.txt",
			Content: originalContent,
			Mode:    0o644,
		},
	})

	// Don't create the file — simulate deletion.

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityHigh {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a high-severity failure for deleted file")
	}
}

func TestCheckFileState_NoStateFile(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   "",
	}

	results := checkGeneratedFiles(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckDenyRules_AllPresent(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/settings.json with all required deny rules.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				`Bash(rm -rf *)`,
				`Bash(git push --force *)`,
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		RequiredDenyRules: []string{
			`Bash(rm -rf *)`,
			`Bash(git push --force *)`,
		},
	}

	results := checkDenyRules(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}
}

func TestCheckDenyRules_MissingRules(t *testing.T) {
	dir := t.TempDir()

	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				`Bash(rm -rf *)`,
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		RequiredDenyRules: []string{
			`Bash(rm -rf *)`,
			`Bash(git push --force *)`,
		},
	}

	results := checkDenyRules(ctx)

	hasMissingFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Name == "deny_rule_missing" && r.AutoFixable {
			hasMissingFail = true
			break
		}
	}
	if !hasMissingFail {
		t.Error("expected an auto-fixable failure for missing deny rule")
	}
}

func TestCheckDenyRules_NoSettingsFile(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot:       dir,
		RequiredDenyRules: []string{`Bash(rm -rf *)`},
	}

	results := checkDenyRules(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusWarn {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusWarn)
	}
}

func TestCheckDenyRules_EmptyRequired(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot:       dir,
		RequiredDenyRules: nil,
	}

	results := checkDenyRules(ctx)

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty required rules, got %d", len(results))
	}
}

func TestCheckDenyRules_SettingsMissingFailsWhenClaudeCodeConfigured(t *testing.T) {
	t.Parallel()

	enabled, disabled := true, false

	// trackedState writes a state file recording settings.json as generated.
	trackedState := func(t *testing.T, dir string) string {
		t.Helper()
		genState := state.RecordFiles([]types.GeneratedFile{
			{Path: ClaudeSettingsRelPath, Content: []byte("{}"), Strategy: types.ThreeWayMerge},
		})
		stateFile := filepath.Join(dir, ".qsdev", "state.yaml")
		if err := state.SaveStateToFile(stateFile, genState); err != nil {
			t.Fatal(err)
		}
		return stateFile
	}

	tests := []struct {
		name         string
		cfg          *types.QsdevConfig
		trackInState bool
		wantStatus   CheckStatus
		wantSeverity CheckSeverity
	}{
		{
			name:         "config enables claude code",
			cfg:          &types.QsdevConfig{Version: 1, ClaudeCode: types.ClaudeCodeConfig{Enabled: &enabled}},
			wantStatus:   StatusFail,
			wantSeverity: SeverityHigh,
		},
		{
			name:         "config silent but state tracks settings.json",
			cfg:          &types.QsdevConfig{Version: 1},
			trackInState: true,
			wantStatus:   StatusFail,
			wantSeverity: SeverityHigh,
		},
		{
			name:         "config explicitly disables claude code",
			cfg:          &types.QsdevConfig{Version: 1, ClaudeCode: types.ClaudeCodeConfig{Enabled: &disabled}},
			trackInState: true,
			wantStatus:   StatusWarn,
			wantSeverity: SeverityMedium,
		},
		{
			name:         "claude code not configured",
			cfg:          &types.QsdevConfig{Version: 1},
			wantStatus:   StatusWarn,
			wantSeverity: SeverityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			ctx := CheckContext{
				ProjectRoot:       dir,
				QsdevConfig:       tt.cfg,
				RequiredDenyRules: []string{`Bash(rm -rf *)`},
			}
			if tt.trackInState {
				ctx.StateFile = trackedState(t, dir)
			}

			results := checkDenyRules(ctx)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Status != tt.wantStatus || results[0].Severity != tt.wantSeverity {
				t.Errorf("got %s/%s, want %s/%s", results[0].Status, results[0].Severity, tt.wantStatus, tt.wantSeverity)
			}
			if got := ShouldFail(results, AuditLevelMedium); got != (tt.wantStatus == StatusFail) {
				t.Errorf("ShouldFail(medium) = %v", got)
			}
		})
	}
}

func TestCheckGeneratedFiles_DeterministicOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var files []types.GeneratedFile
	for _, name := range []string{"e.txt", "a.txt", "d.txt", "c.txt", "b.txt"} {
		files = append(files, types.GeneratedFile{Path: name, Content: []byte(name)})
	}
	stateFile := filepath.Join(dir, ".qsdev", "state.yaml")
	if err := state.SaveStateToFile(stateFile, state.RecordFiles(files)); err != nil {
		t.Fatal(err)
	}
	// Every tracked file is absent on disk, so each yields one result.
	ctx := CheckContext{ProjectRoot: dir, StateFile: stateFile}

	want := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	for range 10 {
		results := checkGeneratedFiles(ctx)
		if len(results) != len(want) {
			t.Fatalf("got %d results, want %d", len(results), len(want))
		}
		for i, r := range results {
			if r.FilePath != want[i] {
				t.Fatalf("result %d = %s, want %s (order must be sorted)", i, r.FilePath, want[i])
			}
		}
	}
}

// TestCheckFileState_ReportsUnparseableGeneratedFile checks `qsdev check`
// fails on a tracked generated file that no longer parses (the syntax
// validation init applies), including a user-editable file whose edits are
// otherwise expected.
func TestCheckFileState_ReportsUnparseableGeneratedFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		strategy types.MergeStrategy
		content  string
		wantFail bool
	}{
		{"valid machine-owned json", types.Overwrite, `{"a": 1}`, false},
		{"broken machine-owned json", types.Overwrite, `{"a": `, true},
		{"broken user-editable json", types.ThreeWayMerge, `{"a": `, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			const rel = "cfg/x.json"
			if err := os.MkdirAll(filepath.Join(dir, "cfg"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, rel), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			st := state.RecordFiles([]types.GeneratedFile{{Path: rel, Content: []byte(tt.content), Strategy: tt.strategy}})
			stateFile := filepath.Join(dir, "state.yaml")
			if err := state.SaveStateToFile(stateFile, st); err != nil {
				t.Fatal(err)
			}

			results := CheckFileState(CheckContext{ProjectRoot: dir, StateFile: stateFile})
			var failed bool
			for _, r := range results {
				if r.Name == "file_syntax_"+rel && r.Status == StatusFail {
					failed = true
				}
			}
			if failed != tt.wantFail {
				t.Errorf("syntax failure reported = %v, want %v: %+v", failed, tt.wantFail, results)
			}
		})
	}
}

// TestCheckDenyRules_DecoyKeyCase checks a "Permissions" key in another case
// cannot stand in for the "permissions" key Claude Code reads.
func TestCheckDenyRules_DecoyKeyCase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, ClaudeSettingsRelPath,
		`{"permissions": {"deny": []}, "Permissions": {"deny": ["Bash(rm -rf *)"]}}`)
	results := checkDenyRules(CheckContext{ProjectRoot: dir, RequiredDenyRules: []string{`Bash(rm -rf *)`}})
	if !ShouldFail(results, AuditLevelLow) {
		t.Errorf("decoy-cased Permissions key satisfied the deny-rule check: %+v", results)
	}
}

// guardHooksSettings registers package-guard.py as a PreToolUse guard (in the
// fail-closed form the generator emits) and audit-log.sh as a PostToolUse
// hook.
const guardHooksSettings = `{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command",
    "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py || { echo blocked >&2; exit 2; }"}]}],
  "PostToolUse": [{"matcher": "*", "hooks": [{"type": "command",
    "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/audit-log.sh"}]}]
}}`

// TestFileSeverity_GuardScript checks that a machine-owned file a PreToolUse
// hook runs is a guard: modifying or deleting it is critical, so
// `check --audit-level critical` fails on a gutted guard, whether the hook
// comes from the generated settings or, without them, from the project's
// settings on disk. Other machine-owned files keep their severity.
func TestFileSeverity_GuardScript(t *testing.T) {
	t.Parallel()
	const (
		guard    = ".claude/hooks/package-guard.py"
		auditLog = ".claude/hooks/audit-log.sh"
		other    = ".gitleaks.toml"
	)
	tests := []struct {
		name         string
		expected     string            // ExpectedClaudeSettings
		onDisk       map[string]string // settings files written to the project
		gut          string            // tracked file emptied ("" for none)
		remove       string            // tracked file deleted ("" for none)
		wantName     string
		wantSeverity CheckSeverity
		wantCritical bool
	}{
		{name: "emptied guard", expected: guardHooksSettings, gut: guard,
			wantName: "file_unmodified_" + guard, wantSeverity: SeverityCritical, wantCritical: true},
		{name: "deleted guard", expected: guardHooksSettings, remove: guard,
			wantName: "file_exists_" + guard, wantSeverity: SeverityCritical, wantCritical: true},
		{name: "emptied guard registered in settings.json only",
			onDisk: map[string]string{claudesettings.ProjectRelPath: guardHooksSettings}, gut: guard,
			wantName: "file_unmodified_" + guard, wantSeverity: SeverityCritical, wantCritical: true},
		{name: "emptied guard registered in settings.local.json only",
			onDisk: map[string]string{claudesettings.LocalRelPath: guardHooksSettings}, gut: guard,
			wantName: "file_unmodified_" + guard, wantSeverity: SeverityCritical, wantCritical: true},
		{name: "modified PostToolUse script", expected: guardHooksSettings, gut: auditLog,
			wantName: "file_unmodified_" + auditLog, wantSeverity: SeverityMedium},
		{name: "modified non-hook machine-owned file", expected: guardHooksSettings, gut: other,
			wantName: "file_unmodified_" + other, wantSeverity: SeverityMedium},
		{name: "deleted non-hook machine-owned file", expected: guardHooksSettings, remove: other,
			wantName: "file_exists_" + other, wantSeverity: SeverityHigh},
		{name: "unregistered guard is not critical", gut: guard,
			wantName: "file_unmodified_" + guard, wantSeverity: SeverityMedium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var files []types.GeneratedFile
			for _, rel := range []string{guard, auditLog, other} {
				f := types.GeneratedFile{Path: rel, Content: []byte("content of " + rel + "\n"), Mode: 0o644}
				files = append(files, f)
				content := f.Content
				if rel == tt.gut {
					content = nil
				}
				if rel != tt.remove {
					writeProjectFile(t, dir, rel, string(content))
				}
			}
			for rel, content := range tt.onDisk {
				writeProjectFile(t, dir, rel, content)
			}
			stateFile := filepath.Join(dir, ".qsdev", "state.yaml")
			if err := state.SaveStateToFile(stateFile, state.RecordFiles(files)); err != nil {
				t.Fatal(err)
			}

			results := checkGeneratedFiles(CheckContext{
				ProjectRoot:            dir,
				StateFile:              stateFile,
				ExpectedClaudeSettings: []byte(tt.expected),
			})

			r := findResult(results, tt.wantName)
			if r == nil {
				t.Fatalf("no %s result in %+v", tt.wantName, results)
			}
			if r.Status != StatusFail || r.Severity != tt.wantSeverity {
				t.Errorf("%s = %s/%s, want fail/%s", tt.wantName, r.Status, r.Severity, tt.wantSeverity)
			}
			if tt.wantCritical {
				if want := "'qsdev update --configs-only --overwrite-modified' to restore the generated " + guard; !strings.Contains(r.Remediation, want) {
					t.Errorf("Remediation = %q, want it to name %q", r.Remediation, want)
				}
			}
			if got := ShouldFail(results, AuditLevelCritical); got != tt.wantCritical {
				t.Errorf("ShouldFail(critical) = %v, want %v", got, tt.wantCritical)
			}
		})
	}
}

func writeProjectFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestVerifyGeneratedFiles_UnjoinedCheckoutJoinsFirst pins the remediation in
// a checkout with a committed config but no local init state: repair, update
// and auto-fix all refuse or fail there, so the advice is to join first and
// the result is not offered for auto-fix.
func TestVerifyGeneratedFiles_UnjoinedCheckoutJoinsFirst(t *testing.T) {
	t.Parallel()
	for _, joined := range []bool{false, true} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte("version: 2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if joined {
			if err := state.SaveProjectState(dir, state.InitStateFile(), types.GeneratedState{}); err != nil {
				t.Fatal(err)
			}
		}
		expected := state.RecordFiles([]types.GeneratedFile{{Path: "deleted.txt", Content: []byte("x"), Mode: 0o644}})
		var r *CheckResult
		for _, res := range verifyGeneratedFiles(dir, expected, nil, nil, nil) {
			if res.Name == "file_exists_deleted.txt" {
				r = &res
			}
		}
		if r == nil {
			t.Fatalf("joined=%v: deleted file not reported", joined)
		}
		joinsFirst := strings.HasPrefix(r.Remediation, joinFirstRemediation)
		if joinsFirst == joined || r.AutoFixable == !joined {
			t.Errorf("joined=%v: remediation %q, auto-fixable %v", joined, r.Remediation, r.AutoFixable)
		}
	}
}

// Guard support file fixtures: a library every Python guard loads, judged like
// a guard whenever a registered guard is a .py script.
const (
	supportGuard   = ".claude/hooks/package-guard.py"
	supportLib     = ".claude/hooks/_qsdev_hooklib.py"
	supportGuardPy = "#!/usr/bin/env python3\nprint('guard')\n"
	supportLibPy   = "MIN_PYTHON = (3, 9)\n"
	// shellGuardSettings registers only a shell script as a PreToolUse guard.
	shellGuardSettings = `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command",
    "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/lsp-first-guard.sh"}]}]}}`
)

// supportCheckContext returns a context whose generator writes the Python
// guard and its support library with the fixture content, and with settings
// registering expectedSettings' guards.
func supportCheckContext(dir, stateFile, expectedSettings string) CheckContext {
	return CheckContext{
		ProjectRoot:            dir,
		StateFile:              stateFile,
		ExpectedClaudeSettings: []byte(expectedSettings),
		GeneratedContent: map[string][]byte{
			supportGuard: []byte(supportGuardPy),
			supportLib:   []byte(supportLibPy),
		},
		GuardSupportFiles: []string{supportLib},
	}
}

// TestCheck_GuardSupportFileModifiedFails checks that a tampered support
// library fails like a tampered guard: it runs inside every Python guard, so
// a recorded hash that was re-hashed along with it is no defence either.
func TestCheck_GuardSupportFileModifiedFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tampered := supportLibPy + "def audit_log(entry): pass\n"
	writeProjectFile(t, dir, supportGuard, supportGuardPy)
	writeProjectFile(t, dir, supportLib, tampered)
	stateFile := filepath.Join(dir, ".qsdev", "state.yaml")
	// The state records the tampered content, as a re-hash would.
	if err := state.SaveStateToFile(stateFile, state.RecordFiles([]types.GeneratedFile{
		{Path: supportGuard, Content: []byte(supportGuardPy), Mode: 0o644},
		{Path: supportLib, Content: []byte(tampered), Mode: 0o644},
	})); err != nil {
		t.Fatal(err)
	}

	results := checkGeneratedFiles(supportCheckContext(dir, stateFile, guardHooksSettings))

	r := findResult(results, "file_unmodified_"+supportLib)
	if r == nil {
		t.Fatalf("modified %s not reported: %+v", supportLib, results)
	}
	if r.Status != StatusFail || r.Severity != SeverityCritical {
		t.Errorf("%s = %s/%s, want fail/critical", r.Name, r.Status, r.Severity)
	}
	if !strings.Contains(r.Message, "support") || !strings.Contains(r.Remediation, supportLib) {
		t.Errorf("result does not name the support file: %+v", *r)
	}
}

// TestCheck_GuardSupportFileDeletedFails_NoState checks that a deleted support
// library fails on a checkout with no state file and no manifest, where only
// the generator's content says it should exist (a clean CI checkout).
func TestCheck_GuardSupportFileDeletedFails_NoState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, supportGuard, supportGuardPy)

	results := checkGeneratedFiles(supportCheckContext(dir, "", guardHooksSettings))

	r := findResult(results, "file_exists_"+supportLib)
	if r == nil {
		t.Fatalf("deleted %s not reported: %+v", supportLib, results)
	}
	if r.Status != StatusFail || r.Severity != SeverityCritical {
		t.Errorf("%s = %s/%s, want fail/critical", r.Name, r.Status, r.Severity)
	}
	if !ShouldFail(results, AuditLevelCritical) {
		t.Error("ShouldFail(critical) = false for a deleted guard support file")
	}
}

// TestCheck_NoSupportWhenNoPythonGuard checks that the support library is not
// judged as a guard when no registered guard is a Python script: nothing
// loads it then.
func TestCheck_NoSupportWhenNoPythonGuard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, supportLib, "tampered\n")

	ctx := supportCheckContext(dir, "", shellGuardSettings)
	if got := guardScripts(ctx); slices.Contains(got, supportLib) {
		t.Errorf("guardScripts = %v, want no support file without a Python guard", got)
	}
	for _, r := range checkGeneratedFiles(ctx) {
		if r.FilePath == supportLib && r.Status == StatusFail {
			t.Errorf("support file judged without a Python guard: %+v", r)
		}
	}
}
