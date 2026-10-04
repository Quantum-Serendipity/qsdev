package state

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// legacyState mirrors GeneratedState as v0.7.10 and earlier wrote it, with
// BaseContent a plain []byte, which yaml.v3 encodes as one integer per line.
type legacyState struct {
	LastRun         time.Time                  `yaml:"last_run"`
	Files           map[string]legacyFileState `yaml:"files"`
	TemplateVersion string                     `yaml:"template_version"`
}

type legacyFileState struct {
	Hash        string              `yaml:"hash"`
	Strategy    types.MergeStrategy `yaml:"strategy"`
	Mode        os.FileMode         `yaml:"mode"`
	BaseContent []byte              `yaml:"base_content,omitempty"`
}

const (
	mergeBase   = `{"permissions":{"allow":["Read"]},"env":{"A":"1"}}`
	mergeTheirs = `{"permissions":{"allow":["Read","Bash(ls:*)"]},"env":{"A":"1"}}`
	mergeOurs   = `{"permissions":{"allow":["Read","Grep"]},"env":{"A":"2"}}`
)

// legacyByteList matches a YAML sequence item holding a byte value, the shape
// a []byte base took in the old format.
var legacyByteList = regexp.MustCompile(`(?m)^\s*- \d{1,3}$`)

// TestLoadLegacyByteListBase loads a state file in the old integer-list form
// and checks the three-way merge from its base gives the same result as from
// the original bytes.
func TestLoadLegacyByteListBase(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.yaml")
	legacy := legacyState{
		LastRun:         time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		TemplateVersion: "v1",
		Files: map[string]legacyFileState{
			".claude/settings.json": {Hash: "sha256:x", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: []byte(mergeBase)},
		},
	}
	data, err := yaml.Marshal(legacy)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if !legacyByteList.Match(data) {
		t.Fatalf("fixture is not in the legacy integer-list form:\n%s", data)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadStateFromFile(path)
	if err != nil {
		t.Fatalf("LoadStateFromFile: %v", err)
	}
	base := loaded.Files[".claude/settings.json"].BaseContent
	if string(base) != mergeBase {
		t.Fatalf("BaseContent = %q, want %q", base, mergeBase)
	}

	want, err := merge.Dispatch(".claude/settings.json", types.ThreeWayMerge, []byte(mergeBase), []byte(mergeTheirs), []byte(mergeOurs))
	if err != nil {
		t.Fatalf("merge from original base: %v", err)
	}
	got, err := merge.Dispatch(".claude/settings.json", types.ThreeWayMerge, base, []byte(mergeTheirs), []byte(mergeOurs))
	if err != nil {
		t.Fatalf("merge from loaded base: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("merge from legacy base = %s, want %s", got, want)
	}
}

// TestSaveStateWritesCompactBaseContent pins the new format: a base is one
// base64 scalar, not a per-byte integer list, and it loads back unchanged.
func TestSaveStateWritesCompactBaseContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte(strings.Repeat(mergeBase+"\n", 200)) // ~10 KB
	st := types.GeneratedState{
		TemplateVersion: "v1",
		Files: map[string]types.FileState{
			".claude/settings.json": {Hash: "sha256:x", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: content},
		},
	}
	if err := SaveProjectState(dir, InitStateFile(), st); err != nil {
		t.Fatalf("SaveProjectState: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, InitStateFile()))
	if err != nil {
		t.Fatal(err)
	}
	if legacyByteList.Match(data) {
		t.Errorf("state file still holds an integer byte list:\n%.400s", data)
	}
	if !bytes.Contains(data, []byte("base_content: !!binary")) {
		t.Errorf("state file has no !!binary base_content:\n%.400s", data)
	}
	// Base64 costs 4/3 of the content plus line breaks and indentation; the
	// integer list cost about seven times the content.
	if limit := len(content)*3/2 + 1024; len(data) > limit {
		t.Errorf("state file is %d bytes for %d bytes of base content, want at most %d", len(data), len(content), limit)
	}

	loaded, err := LoadStateFromFile(filepath.Join(dir, InitStateFile()))
	if err != nil {
		t.Fatalf("LoadStateFromFile: %v", err)
	}
	if got := loaded.Files[".claude/settings.json"].BaseContent; !bytes.Equal(got, content) {
		t.Errorf("BaseContent did not round-trip: got %d bytes, want %d", len(got), len(content))
	}
}

// BenchmarkLoadState loads a state file holding about 17 KB of three-way
// merge bases, the size a single-ecosystem init records, in the current and
// the legacy encoding.
func BenchmarkLoadState(b *testing.B) {
	base := []byte(strings.Repeat(mergeBase+"\n", 170))
	current := types.GeneratedState{TemplateVersion: "v1", Files: map[string]types.FileState{
		".claude/settings.json": {Hash: "sha256:x", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: base},
		".mcp.json":             {Hash: "sha256:y", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: base},
	}}
	legacy := legacyState{TemplateVersion: "v1", Files: map[string]legacyFileState{
		".claude/settings.json": {Hash: "sha256:x", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: base},
		".mcp.json":             {Hash: "sha256:y", Strategy: types.ThreeWayMerge, Mode: 0o644, BaseContent: base},
	}}
	for _, bc := range []struct {
		name  string
		state any
	}{{"current", current}, {"legacy", legacy}} {
		b.Run(bc.name, func(b *testing.B) {
			data, err := yaml.Marshal(bc.state)
			if err != nil {
				b.Fatal(err)
			}
			path := filepath.Join(b.TempDir(), "state.yaml")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, err := LoadStateFromFile(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
