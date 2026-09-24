package canon

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
)

// TestClaudeConfigDirEntries pins the files protected in a Claude Code
// configuration directory relocated with CLAUDE_CONFIG_DIR: the same settings,
// hook, agent, command and skill entries as a .claude directory, plus
// .claude.json, and nothing when the variable is unset.
func TestClaudeConfigDirEntries(t *testing.T) {
	t.Parallel()
	dir := filepath.FromSlash("/home/alice/.config/claude")
	if runtime.GOOS == "windows" {
		// A rooted path without a drive letter is not absolute on Windows, so
		// the entries would gain the current drive; use a volume-qualified one.
		dir = `C:\Users\alice\.config\claude`
	}
	sep := string(filepath.Separator)
	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{"unset", "", nil},
		{"relocated", dir, []string{
			filepath.Join(dir, "settings.json"),
			filepath.Join(dir, "settings.local.json"),
			filepath.Join(dir, ".claude.json"),
			filepath.Join(dir, "hooks") + sep,
			filepath.Join(dir, "agents") + sep,
			filepath.Join(dir, "commands") + sep,
			filepath.Join(dir, "skills") + sep,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, e := range claudeConfigDirEntries(tt.dir) {
				if e.category != "claude-settings" {
					t.Errorf("entry %q category = %q, want claude-settings", e.path, e.category)
				}
				got = append(got, e.path)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("entries = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestManagedSettingsDirs pins where each platform's managed (policy) settings
// are protected.
func TestManagedSettingsDirs(t *testing.T) {
	t.Parallel()
	env := map[string]string{"ProgramFiles": `D:\Apps`}
	getenv := func(k string) string { return env[k] }
	tests := []struct {
		goos string
		want []string
	}{
		{"linux", []string{"/etc/claude-code"}},
		{"darwin", []string{"/etc/claude-code", "/Library/Application Support/ClaudeCode"}},
		{"windows", []string{
			"/etc/claude-code",
			filepath.Join(`D:\Apps`, "ClaudeCode"),
			filepath.Join(`C:\ProgramData`, "ClaudeCode"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			t.Parallel()
			if got := managedSettingsDirs(tt.goos, getenv); !slices.Equal(got, tt.want) {
				t.Errorf("managedSettingsDirs(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}

// TestIsProtected_RelocatedClaudeConfigDir checks that the settings of a
// configuration directory named by CLAUDE_CONFIG_DIR are protected, under the
// directory's written and symlink-resolved spellings, while its other content
// is not. It is sequential because it re-runs ensureInit with a different
// environment; the parallel tests resume only after its cleanup.
func TestIsProtected_RelocatedClaudeConfigDir(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "claude-config")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir, err := filepath.EvalSymlinks(cfgDir) // canonical, as the rules compare it
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "cfg")
	if err := os.Symlink(cfgDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv(ClaudeConfigDirEnv, link)
	resetInit := func() {
		initOnce = sync.Once{}
		initErr = nil
		protectedPrefixes = nil
		protectedSuffixes = nil
	}
	resetInit()
	t.Cleanup(resetInit)

	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(link, "settings.json"), true},
		{filepath.Join(cfgDir, "settings.json"), true},
		{filepath.Join(cfgDir, "settings.local.json"), true},
		{filepath.Join(cfgDir, "hooks", "guard.py"), true},
		{filepath.Join(cfgDir, ".claude.json"), true},
		{filepath.Join(cfgDir, "projects", "x.jsonl"), false},
		{filepath.Join(cfgDir, "notes.md"), false},
	}
	for _, tt := range tests {
		if got, cat := IsProtected(tt.path); got != tt.want || (got && cat != "claude-settings") {
			t.Errorf("IsProtected(%q) = (%v, %q), want (%v, claude-settings)", tt.path, got, cat, tt.want)
		}
	}
	dir, err := ClaudeConfigDir()
	if err != nil || dir != link {
		t.Errorf("ClaudeConfigDir() = (%q, %v), want %q", dir, err, link)
	}
}
