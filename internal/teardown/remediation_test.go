package teardown

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func settingsRegistry() *toolreg.Registry {
	r := toolreg.NewRegistry()
	_ = r.Register(toolreg.Tool{
		Name:     "attach-guard",
		Category: toolreg.CategorySecurity,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: ".claude/settings.json", Ownership: toolreg.Shared, SectionID: "attach-guard"},
		},
	})
	return r
}

const generatedSettings = `{
  "permissions": {
    "defaultMode": "default",
    "allow": [],
    "deny": ["Read(./.env)", "Bash(curl:*)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/scan-secrets.py", "timeout": 10}]},
      {"matcher": "Write", "hooks": [{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/guard.sh"}]}
    ]
  }
}`

func writeFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatal(err)
	}
}

func decodeSettings(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v\n%s", err, data)
	}
	return m
}

func TestExecute_CleansSettingsJSONAgainstGeneratedBase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// On disk: the generated settings plus user additions (an env block, a
	// deny rule, and a hook added to qsdev's Bash matcher group).
	onDisk := `{
  "env": {"FOO": "bar"},
  "permissions": {
    "defaultMode": "default",
    "allow": [],
    "deny": ["Read(./.env)", "Bash(curl:*)", "Read(./secret-user)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [
        {"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/scan-secrets.py", "timeout": 10},
        {"type": "command", "command": "my-own-hook"}
      ]},
      {"matcher": "Write", "hooks": [{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/guard.sh"}]}
    ]
  }
}`
	path := filepath.Join(dir, ".claude", "settings.json")
	writeFile(t, path, []byte(onDisk), 0o644)

	plan := &TeardownPlan{Clean: []FileAction{{
		Path:        ".claude/settings.json",
		BaseContent: []byte(generatedSettings),
	}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: dir}, settingsRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Cleaned) != 1 {
		t.Fatalf("Cleaned = %d, want 1", len(result.Cleaned))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	for _, gone := range []string{"scan-secrets.py", "guard.sh", "Bash(curl:*)", "Read(./.env)", "defaultMode"} {
		if strings.Contains(s, gone) {
			t.Errorf("generated entry %q survived teardown:\n%s", gone, s)
		}
	}
	for _, kept := range []string{"FOO", "Read(./secret-user)", "my-own-hook", `"matcher": "Bash"`} {
		if !strings.Contains(s, kept) {
			t.Errorf("user entry %q was removed:\n%s", kept, s)
		}
	}
	decodeSettings(t, got)
}

func TestExecute_SettingsJSONWithoutBaseIsReportedNotCleaned(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "settings.json")
	writeFile(t, path, []byte(generatedSettings), 0o644)

	plan := &TeardownPlan{Clean: []FileAction{{Path: ".claude/settings.json"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: dir}, settingsRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cleaned) != 0 {
		t.Errorf("Cleaned = %v, want none (qsdev entries could not be identified)", result.Cleaned)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors = %v, want 1 explaining the file was not cleaned", result.Errors)
	}
}

func TestExecute_UnchangedSharedFileNotRewrittenOrReportedCleaned(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.md")
	writeFile(t, path, []byte("# Only user content\n"), 0o644)
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{Clean: []FileAction{{Path: "shared.md"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: dir}, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cleaned) != 0 {
		t.Errorf("Cleaned = %v, want none", result.Cleaned)
	}
	if len(result.Preserved) != 1 {
		t.Errorf("Preserved = %v, want the unchanged shared file", result.Preserved)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(past) {
		t.Error("unchanged shared file was rewritten")
	}
}

func TestExecute_CleanPreservesModeAndSymlink(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks and Unix permission bits are not reliable on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "AGENTS.md")
	writeFile(t, target, []byte("user\n<!-- qsdev:test-tool -->\ngen\n<!-- /qsdev:test-tool -->\n"), 0o600)
	if err := os.Symlink("AGENTS.md", filepath.Join(dir, "shared.md")); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{Clean: []FileAction{{Path: "shared.md"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: dir}, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 || len(result.Cleaned) != 1 {
		t.Fatalf("Cleaned = %v, Errors = %v", result.Cleaned, result.Errors)
	}
	if fi, err := os.Lstat(filepath.Join(dir, "shared.md")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("shared.md symlink was replaced (err=%v)", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("target mode = %o, want 600", perm)
	}
	got, _ := os.ReadFile(target)
	if strings.Contains(string(got), "gen") || !strings.Contains(string(got), "user") {
		t.Errorf("target content = %q", got)
	}
}

func TestTeardown_RejectsStatePathsOutsideProject(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	victim := filepath.Join(parent, "victim.txt")
	victimContent := []byte("do not delete\n")
	writeFile(t, victim, victimContent, 0o644)

	statePath := filepath.Join(dir, state.StateFilePaths()[2])
	writeFile(t, statePath, nil, 0o644)
	if err := state.SaveStateToFile(statePath, types.GeneratedState{Files: map[string]types.FileState{
		"../victim.txt": {Hash: state.ComputeHash(victimContent)},
	}}); err != nil {
		t.Fatal(err)
	}

	result, err := Teardown(TeardownOptions{Profile: ProfileDefault, Force: true, ProjectRoot: dir},
		toolreg.NewRegistry(), nil, io.Discard)
	if err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("file outside the project was removed: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected the unsafe state entry to be reported")
	}
}

func TestExecute_RefusesRemovalThroughSymlinkedDir(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks are not reliable on Windows")
	}
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	outside := filepath.Join(parent, "outside")
	writeFile(t, filepath.Join(outside, "x.txt"), []byte("x"), 0o644)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{Remove: []FileAction{{Path: "link/x.txt"}}}
	result, err := Execute(plan, TeardownOptions{ProjectRoot: dir}, toolreg.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "x.txt")); err != nil {
		t.Fatalf("file behind symlinked directory was removed: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors = %v, want 1", result.Errors)
	}
}

func TestTeardown_UnreadableStateFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		force bool
	}{
		{"aborts_without_force", false},
		{"keeps_state_file_with_force", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			rel := state.StateFilePaths()[2] // .claude state
			statePath := filepath.Join(dir, rel)
			writeFile(t, statePath, []byte("files: [unterminated\n"), 0o644)
			hook := filepath.Join(dir, ".claude", "hooks", "guard.sh")
			writeFile(t, hook, []byte("#!/bin/sh\n"), 0o755)

			result, err := Teardown(TeardownOptions{Profile: ProfileDefault, Force: tt.force, ProjectRoot: dir},
				toolreg.NewRegistry(), func(*TeardownPlan, io.Writer) bool { return true }, io.Discard)
			if !tt.force {
				if err == nil {
					t.Fatal("expected teardown to abort on an unreadable state file")
				}
			} else {
				if err != nil {
					t.Fatalf("Teardown: %v", err)
				}
				if len(result.Errors) == 0 {
					t.Error("unreadable state file not reported in result errors")
				}
			}
			if _, err := os.Stat(statePath); err != nil {
				t.Errorf("unreadable state file was deleted: %v", err)
			}
		})
	}
}

func TestClassifyFiles_StatErrorPreserves(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs Unix permissions enforced for a non-root user")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	writeFile(t, filepath.Join(locked, "exclusive.txt"), []byte("x"), 0o644)
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	classified := ClassifyFiles(types.GeneratedState{Files: map[string]types.FileState{
		"locked/exclusive.txt": {Hash: state.ComputeHash([]byte("x"))},
	}}, dir, testRegistry())
	if len(classified) != 1 {
		t.Fatalf("classified = %d, want 1", len(classified))
	}
	if classified[0].Deleted || !classified[0].Modified {
		t.Errorf("stat error classified as Deleted=%v Modified=%v; want preserved (Modified)",
			classified[0].Deleted, classified[0].Modified)
	}
}
