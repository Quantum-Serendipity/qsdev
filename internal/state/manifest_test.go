package state

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestBuildManifest_MachineOwnedOnly(t *testing.T) {
	t.Parallel()

	st := RecordFiles([]types.GeneratedFile{
		{Path: ".claude/hooks/package-guard.py", Content: []byte("hook"), Strategy: types.Overwrite},
		{Path: ".claude/skills/x/SKILL.md", Content: []byte("skill"), Strategy: types.LibraryManaged},
		{Path: ".envrc", Content: []byte("use devenv"), Strategy: types.Skip},
		{Path: "CLAUDE.md", Content: []byte("md"), Strategy: types.SectionMarker},
		{Path: "devenv.nix", Content: []byte("{}"), Strategy: types.ManualMerge},
		{Path: ".claude/settings.json", Content: []byte("{}"), Strategy: types.ThreeWayMerge},
		{Path: "merged.yaml", Content: []byte("a: 1"), Strategy: types.Merge},
	})

	got := BuildManifest(st)
	want := Manifest{
		".claude/hooks/package-guard.py": ComputeHash([]byte("hook")),
		".claude/skills/x/SKILL.md":      ComputeHash([]byte("skill")),
		".envrc":                         ComputeHash([]byte("use devenv")),
	}
	if !maps.Equal(got, want) {
		t.Errorf("BuildManifest() = %v, want %v", got, want)
	}
}

func TestManifest_MarshalRoundTrip(t *testing.T) {
	t.Parallel()

	m := Manifest{
		"b/two.sh":       ComputeHash([]byte("two")),
		"a one.txt":      ComputeHash([]byte("one")),
		".gitleaks.toml": ComputeHash([]byte("three")),
	}
	data, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// sha256sum text format, sorted by path, bare hex digests.
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	wantPaths := []string{".gitleaks.toml", "a one.txt", "b/two.sh"}
	if len(lines) != len(wantPaths) {
		t.Fatalf("Marshal() wrote %d lines, want %d:\n%s", len(lines), len(wantPaths), data)
	}
	for i, line := range lines {
		want := strings.TrimPrefix(m[wantPaths[i]], HashPrefix) + "  " + wantPaths[i]
		if line != want {
			t.Errorf("line %d = %q, want %q", i+1, line, want)
		}
	}

	parsed, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest() error: %v", err)
	}
	if !maps.Equal(parsed, m) {
		t.Errorf("round trip = %v, want %v", parsed, m)
	}
}

func TestManifest_MarshalRejectsBadEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Manifest
	}{
		{"hash without prefix", Manifest{"a.txt": strings.Repeat("a", 64)}},
		{"short hash", Manifest{"a.txt": HashPrefix + "abc"}},
		{"path escapes project", Manifest{"../a.txt": ComputeHash(nil)}},
		{"path with newline", Manifest{"a\nb.txt": ComputeHash(nil)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.m.Marshal(); err == nil {
				t.Error("Marshal() succeeded, want an error")
			}
		})
	}
}

func TestParseManifest(t *testing.T) {
	t.Parallel()

	sum := strings.Repeat("0123456789abcdef", 4)
	tests := []struct {
		name    string
		data    string
		want    Manifest
		wantErr string
	}{
		{name: "empty", data: "", want: Manifest{}},
		{name: "text mode", data: sum + "  a.txt\n", want: Manifest{"a.txt": HashPrefix + sum}},
		{name: "binary mode marker", data: sum + " *a.txt\n", want: Manifest{"a.txt": HashPrefix + sum}},
		{name: "CRLF, comments and blank lines", data: "# header\r\n\r\n" + sum + "  a.txt\r\n", want: Manifest{"a.txt": HashPrefix + sum}},
		{name: "missing separator", data: sum + "a.txt\n", wantErr: "line 1"},
		{name: "single space", data: sum + " a.txt\n", wantErr: "line 1"},
		{name: "uppercase hex", data: strings.ToUpper(sum) + "  a.txt\n", wantErr: "not a lowercase SHA-256"},
		{name: "short hex", data: "abc  a.txt\n", wantErr: "not a lowercase SHA-256"},
		{name: "parent traversal", data: sum + "  ../etc/passwd\n", wantErr: "unsafe path"},
		{name: "absolute path", data: sum + "  /etc/passwd\n", wantErr: "unsafe path"},
		{name: "unclean path", data: sum + "  a/./b.txt\n", wantErr: "unsafe path"},
		{name: "backslash", data: sum + "  a\\b.txt\n", wantErr: "backslash"},
		{name: "empty path", data: sum + "  \n", wantErr: "empty path"},
		{name: "duplicate path", data: sum + "  a.txt\n" + sum + "  a.txt\n", wantErr: "more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseManifest([]byte(tt.data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseManifest() error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseManifest() error: %v", err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("ParseManifest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadManifest_MissingWrapsNotExist(t *testing.T) {
	t.Parallel()

	_, err := LoadManifest(filepath.Join(t.TempDir(), ManifestFile()))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("LoadManifest() error = %v, want one wrapping fs.ErrNotExist", err)
	}
}

// TestSaveInitState_WritesStateAndManifest checks that the init state and the
// committed manifest are written together and agree.
func TestSaveInitState_WritesStateAndManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	st := RecordFiles([]types.GeneratedFile{
		{Path: ".claude/hooks/guard.py", Content: []byte("hook"), Mode: 0o755, Strategy: types.Overwrite},
		{Path: "CLAUDE.md", Content: []byte("md"), Strategy: types.SectionMarker},
	})
	if err := SaveInitState(dir, st); err != nil {
		t.Fatalf("SaveInitState() error: %v", err)
	}

	saved, err := LoadStateFromFile(filepath.Join(dir, InitStateFile()))
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	if len(saved.Files) != 2 {
		t.Errorf("state tracks %d files, want 2", len(saved.Files))
	}

	manifestPath := filepath.Join(dir, ManifestFile())
	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}
	if !maps.Equal(m, BuildManifest(st)) {
		t.Errorf("manifest = %v, want %v", m, BuildManifest(st))
	}
	// The state directory is gitignored; the manifest must be committable.
	if strings.Contains(ManifestFile(), "/") {
		t.Errorf("manifest %s is not at the project root", ManifestFile())
	}
	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o022 != 0 {
		t.Errorf("manifest mode = %v, want no group/other write", info.Mode().Perm())
	}
}
