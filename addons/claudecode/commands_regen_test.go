package claudecode_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const (
	claudeStateRel = ".claude/.qsdev-claude-state.yaml"
	settingsRel    = ".claude/settings.json"
	guardRel       = ".claude/hooks/package-guard.py"
)

// runClaudeCmd runs a `claude` subcommand in the current directory and returns
// its combined output and error.
func runClaudeCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// mustRunClaude runs a `claude` subcommand and fails the test on error.
func mustRunClaude(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runClaudeCmd(t, args...)
	if err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
	return out
}

// recordState adds a file record with content's hash to the state file at path.
func recordState(t *testing.T, path, rel string, content []byte) {
	t.Helper()
	st, err := state.LoadStateFromFile(path)
	if err != nil {
		t.Fatalf("loading state %s: %v", path, err)
	}
	st.Files[rel] = types.FileState{Hash: state.ComputeHash(content), Mode: 0o644}
	if err := state.SaveStateToFile(path, st); err != nil {
		t.Fatalf("saving state %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// initFullTier runs `claude init` and raises the saved tier to full so skills
// can be added.
func initFullTier(t *testing.T, dir string, extra ...string) {
	t.Helper()
	mustRunClaude(t, append([]string{"init", "--yes", "--permission-preset", "standard"}, extra...)...)
	a, err := claudecode.ExportLoadAnswers(dir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	a.Tier = "full"
	if err := claudecode.ExportSaveAnswers(dir, a); err != nil {
		t.Fatalf("saving answers: %v", err)
	}
}

// addUserSettings rewrites settings.json with a user env block and a custom
// allow rule merged into the generated content.
func addUserSettings(t *testing.T, path string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(readFile(t, path)), &doc); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	doc["env"] = map[string]any{"MY_TOKEN_VAR": "keep"}
	perms, _ := doc["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
		doc["permissions"] = perms
	}
	allow, _ := perms["allow"].([]any)
	perms["allow"] = append(allow, "Bash(make-user-custom *)")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

// TestUpdate_WithoutClaudeState_MergesUserSettings guards F091: after a
// top-level init the claude addon has no state record for settings.json, and
// `claude update` must still merge rather than overwrite user-owned keys.
func TestUpdate_WithoutClaudeState_MergesUserSettings(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	mustRunClaude(t, "init", "--yes", "--permission-preset", "standard")
	if err := os.Remove(filepath.Join(dir, claudeStateRel)); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(dir, settingsRel)
	addUserSettings(t, settings)

	for _, args := range [][]string{{"update"}, {"update", "--force"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			mustRunClaude(t, args...)
			got := readFile(t, settings)
			for _, want := range []string{"MY_TOKEN_VAR", "make-user-custom"} {
				if !strings.Contains(got, want) {
					t.Errorf("%s dropped from settings.json: %s", want, got)
				}
			}
		})
	}
}

// TestRegen_MalformedSettingsNotOverwritten guards F517: a merge failure on
// user content must fail the command and leave the file (and answers) alone.
func TestRegen_MalformedSettingsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	initFullTier(t, dir)
	settings := filepath.Join(dir, settingsRel)
	malformed := `{"env": {"MY_SECRET_ENDPOINT": "x"}, "permissions": {"allow": ["Bash(make:*)"],}}`

	tests := []struct {
		name string
		args []string
	}{
		{"add-skill", []string{"add-skill", "deploy"}},
		{"add-hook", []string{"add-hook", "audit-log"}},
		{"update", []string{"update"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writeFile(t, settings, malformed)
			out, err := runClaudeCmd(t, tc.args...)
			if err == nil {
				t.Fatalf("expected a merge error, got success:\n%s", out)
			}
			if !strings.Contains(err.Error(), "settings.json") {
				t.Errorf("error should name settings.json: %v", err)
			}
			if got := readFile(t, settings); got != malformed {
				t.Errorf("settings.json was modified despite the merge failure:\n%s", got)
			}
			a, loadErr := claudecode.ExportLoadAnswers(dir)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if a.Hooks.AuditLog || claudecode.ExportContains(a.Skills, "deploy") {
				t.Errorf("answers were persisted despite the failure: %+v", a)
			}
		})
	}

	t.Run("update --force overwrites", func(t *testing.T) {
		writeFile(t, settings, malformed)
		mustRunClaude(t, "update", "--force")
		var doc map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settings)), &doc); err != nil {
			t.Errorf("update --force should replace malformed settings.json with valid JSON: %v", err)
		}
	})
}

// TestRegen_RestoresTamperedGuardHook guards F094: a deleted or neutered guard
// hook must be restored by add-hook/add-skill/update, and only files actually
// written are counted.
func TestRegen_RestoresTamperedGuardHook(t *testing.T) {
	tamper := map[string]func(t *testing.T, guard string){
		"deleted": func(t *testing.T, guard string) {
			if err := os.Remove(guard); err != nil {
				t.Fatal(err)
			}
		},
		"neutered": func(t *testing.T, guard string) {
			writeFile(t, guard, "import sys\nsys.exit(0)\n")
		},
	}
	commands := [][]string{{"add-hook", "audit-log"}, {"add-skill", "deploy"}, {"update"}}
	for _, args := range commands {
		for _, how := range []string{"deleted", "neutered"} {
			t.Run(args[0]+"/"+how, func(t *testing.T) {
				dir := t.TempDir()
				chdir(t, dir)
				initFullTier(t, dir)
				guard := filepath.Join(dir, guardRel)
				original := readFile(t, guard)

				tamper[how](t, guard)
				out := mustRunClaude(t, args...)
				if got := readFile(t, guard); got != original {
					t.Fatalf("guard hook not restored after %v:\n%s", args, got)
				}
				if !strings.Contains(out, "security hook") {
					t.Errorf("expected a tamper warning, got:\n%s", out)
				}
			})
		}
	}

	t.Run("unchanged files are not counted", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		initFullTier(t, dir)
		mustRunClaude(t, "update") // realize the raised tier
		out := mustRunClaude(t, "update")
		if !strings.Contains(out, "0 created, 0 updated") {
			t.Errorf("a no-op update should report nothing written, got:\n%s", out)
		}
		out = mustRunClaude(t, "add-hook", "audit-log")
		if strings.Contains(out, " 0 file(s)") || strings.Contains(out, "unchanged") {
			t.Errorf("add-hook should report only the files it wrote, got:\n%s", out)
		}
	})
}

// TestRegen_PreservesMcpLifecycleStateAndVersions guards F093: regenerations
// must carry forward MCP lifecycle records and version stamps they do not own.
func TestRegen_PreservesMcpLifecycleStateAndVersions(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	initFullTier(t, dir)
	stPath := filepath.Join(dir, claudeStateRel)

	st, err := state.LoadStateFromFile(stPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.TemplateVersion == "" {
		t.Fatal("init should stamp the template version")
	}
	st.McpServers = map[string]types.McpServerState{"github": {InstalledVersion: "1.2.3"}}
	st.EnabledTools = map[string]bool{"semgrep": true}
	st.TemplateVersion = "tv1"
	if err := state.SaveStateToFile(stPath, st); err != nil {
		t.Fatal(err)
	}

	mustRunClaude(t, "add-skill", "deploy")
	after, err := state.LoadStateFromFile(stPath)
	if err != nil {
		t.Fatal(err)
	}
	if after.McpServers["github"].InstalledVersion != "1.2.3" {
		t.Errorf("mcp_servers lost after add-skill: %+v", after.McpServers)
	}
	if !after.EnabledTools["semgrep"] {
		t.Errorf("enabled_tools lost after add-skill: %+v", after.EnabledTools)
	}
	if after.TemplateVersion != "tv1" {
		t.Errorf("add-skill must not clear the template version, got %q", after.TemplateVersion)
	}

	mustRunClaude(t, "init", "--yes", "--force", "--permission-preset", "standard")
	reinit, err := state.LoadStateFromFile(stPath)
	if err != nil {
		t.Fatal(err)
	}
	if reinit.McpServers["github"].InstalledVersion != "1.2.3" {
		t.Errorf("mcp_servers lost after init --force: %+v", reinit.McpServers)
	}
}

// TestRegen_KeepsUnrecordedLegacyFlatSkill guards F102: a flat
// .claude/skills/<name>.md that qsdev did not generate (or that was edited) is
// never deleted by an unrelated regeneration.
func TestRegen_KeepsUnrecordedLegacyFlatSkill(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	initFullTier(t, dir, "--skills", "review-pr")

	legacy := filepath.Join(dir, ".claude", "skills", "review-pr.md")
	writeFile(t, legacy, "my own review notes")
	// A record whose hash does not match (the user edited the file) is
	// treated like no record at all.
	recordState(t, filepath.Join(dir, claudeStateRel), ".claude/skills/review-pr.md", []byte("generated long ago"))

	out := mustRunClaude(t, "add-hook", "audit-log")
	if got := readFile(t, legacy); got != "my own review notes" {
		t.Errorf("user-authored legacy skill file was changed: %q", got)
	}
	if !strings.Contains(out, "review-pr.md") {
		t.Errorf("expected a warning naming the kept legacy file, got:\n%s", out)
	}
}
