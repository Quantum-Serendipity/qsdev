package claudecode_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

const (
	generatedBanner = "configuration generated"
	brokenSettings  = `{ "permissions": { broken`
)

// runClaudeInit runs `claude init` with args in dir and returns the command's
// combined output and error.
func runClaudeInit(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	t.Chdir(dir)
	return runClaudeCmd(t, append([]string{"init"}, args...)...)
}

// writeBrokenSettings creates .claude/settings.json holding unparseable JSON.
func writeBrokenSettings(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(settingsRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(brokenSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestClaudeInitFailsOnUnwritableSettings is the regression test for U14-01:
// a security file that failed to write was reported only as "failed 1" in
// the summary while init exited 0 and announced success. A directory where
// settings.json belongs cannot be fixed by --force, so init must fail, name
// the file, and still record the files it did write.
func TestClaudeInitFailsOnUnwritableSettings(t *testing.T) {
	dir := testutil.IsolatedDir(t)
	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(settingsRel)), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runClaudeInit(t, dir, "--yes", "--force")
	if err == nil {
		t.Fatalf("init succeeded although %s could not be written:\n%s", settingsRel, out)
	}
	if !strings.Contains(err.Error(), settingsRel) || !strings.Contains(err.Error(), "partial write") {
		t.Errorf("error should be a partial-write error naming %s, got: %v", settingsRel, err)
	}
	if strings.Contains(out, generatedBanner) {
		t.Errorf("output announces success after a failed write:\n%s", out)
	}
	if !strings.Contains(out, "FAILED "+settingsRel) {
		t.Errorf("summary should list the failed file:\n%s", out)
	}

	st, err := state.LoadStateFromFile(filepath.Join(dir, state.ClaudeStateFile()))
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	if _, ok := st.Files[settingsRel]; ok {
		t.Errorf("state records %s although it was not written", settingsRel)
	}
	if _, ok := st.Files["CLAUDE.md"]; !ok {
		t.Errorf("state should record the successfully written CLAUDE.md; got %d entries", len(st.Files))
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md should still be written: %v", err)
	}
}

// TestClaudeInitForceOverwritesBrokenSettings is the regression test for
// U14-V02: --force left an unparseable settings.json untouched (and reported
// it only as "failed 1"), unlike regen's --force, which overwrites it.
func TestClaudeInitForceOverwritesBrokenSettings(t *testing.T) {
	dir := testutil.IsolatedDir(t)
	path := writeBrokenSettings(t, dir)

	out, err := runClaudeInit(t, dir, "--yes", "--force")
	if err != nil {
		t.Fatalf("init --force over broken settings.json: %v\n%s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("settings.json is not valid JSON after --force: %v\n%s", err, data)
	}
	if len(settings.Permissions.Deny) == 0 {
		t.Errorf("settings.json has no permissions.deny after --force:\n%s", data)
	}
	if !strings.Contains(out, settingsRel+": ") || !strings.Contains(out, "(--force)") {
		t.Errorf("output should carry the --force overwrite note for %s:\n%s", settingsRel, out)
	}
	if !strings.Contains(out, generatedBanner) {
		t.Errorf("full success should announce the generated configuration:\n%s", out)
	}
}

// TestClaudeInitBrokenSettingsWithoutForceRefuses checks that without --force
// init refuses to touch an existing (even broken) settings.json.
func TestClaudeInitBrokenSettingsWithoutForceRefuses(t *testing.T) {
	dir := testutil.IsolatedDir(t)
	path := writeBrokenSettings(t, dir)

	out, err := runClaudeInit(t, dir, "--yes")
	if err == nil {
		t.Fatalf("init without --force should refuse an existing settings.json:\n%s", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != brokenSettings {
		t.Errorf("settings.json changed without --force:\n%s", data)
	}
}
