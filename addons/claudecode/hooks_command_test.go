package claudecode_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

// listHookStatuses runs `claude hooks list --json` and indexes the first
// status per hook name.
func listHookStatuses(t *testing.T) map[string]claudecode.ExportHookStatus {
	t.Helper()
	out := mustRunClaude(t, "hooks", "list", "--json")
	var statuses []claudecode.ExportHookStatus
	if err := json.Unmarshal([]byte(out), &statuses); err != nil {
		t.Fatalf("parsing hooks list output: %v\n%s", err, out)
	}
	byName := make(map[string]claudecode.ExportHookStatus, len(statuses))
	for _, s := range statuses {
		if _, seen := byName[s.Name]; !seen {
			byName[s.Name] = s
		}
	}
	return byName
}

// TestHooksList_ReportsDeployedState guards F100: the listing reflects what is
// deployed (settings.json wiring and script integrity), not only the answers.
func TestHooksList_ReportsDeployedState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, dir string)
		want   string
	}{
		{"intact", func(*testing.T, string) {}, "deployed"},
		{"script neutered", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, guardRel), "import sys\nsys.exit(0)\n")
		}, "script modified"},
		{"script deleted", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, guardRel)); err != nil {
				t.Fatal(err)
			}
		}, "script missing"},
		{"unwired from settings.json", func(t *testing.T, dir string) {
			settings := filepath.Join(dir, settingsRel)
			var doc map[string]any
			if err := json.Unmarshal([]byte(readFile(t, settings)), &doc); err != nil {
				t.Fatal(err)
			}
			delete(doc, "hooks")
			data, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, settings, string(data))
		}, "not deployed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			chdir(t, dir)
			mustRunClaude(t, "init", "--yes", "--permission-preset", "standard")
			tc.mutate(t, dir)

			guard := listHookStatuses(t)["package-guard"]
			if !guard.Configured {
				t.Errorf("package-guard should be configured by the saved answers")
			}
			if guard.Deployment != tc.want {
				t.Errorf("package-guard deployment = %q, want %q", guard.Deployment, tc.want)
			}
		})
	}
}

// TestHooksList_CorruptAnswersFails guards F100: a corrupt answers file is an
// error, not a silent "nothing configured" listing.
func TestHooksList_CorruptAnswersFails(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, claudecode.ExportAnswersPath(dir), "hooks: [not: valid\n")

	if out, err := runClaudeCmd(t, "hooks", "list"); err == nil {
		t.Fatalf("expected an error for corrupt answers, got:\n%s", out)
	}
}

// TestHooksList_NoAnswersIsNotAnError keeps the listing usable before init.
func TestHooksList_NoAnswersIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	for name, s := range listHookStatuses(t) {
		if s.Deployment != "not deployed" {
			t.Errorf("%s: deployment = %q before init, want not deployed", name, s.Deployment)
		}
	}
}

// TestHooksList_ToolGatesWithoutPolicy guards W046: tool-gates enabled with
// no .qsdev.yaml hooks.tool_gates lists allows every tool, so the listing
// marks it "no policy" rather than as a plain configured control.
func TestHooksList_ToolGatesWithoutPolicy(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	mustRunClaude(t, "init", "--yes", "--permission-preset", "standard")
	mustRunClaude(t, "add-hook", "tool-gates")

	gates := listHookStatuses(t)["tool-gates"]
	if !gates.Configured || gates.Policy != "none" {
		t.Errorf("tool-gates configured=%v policy=%q, want configured with policy \"none\"", gates.Configured, gates.Policy)
	}
	if guard := listHookStatuses(t)["package-guard"]; guard.Policy != "" {
		t.Errorf("package-guard policy = %q, want empty (it needs no policy)", guard.Policy)
	}
	if out := mustRunClaude(t, "hooks", "list"); !strings.Contains(out, "yes (no policy)") {
		t.Errorf("table output does not mark tool-gates as having no policy:\n%s", out)
	}
}
