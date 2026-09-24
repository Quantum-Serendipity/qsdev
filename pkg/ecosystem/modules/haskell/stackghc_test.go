package haskell

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeStackYAML writes content as stack.yaml in a new temp dir and returns
// the dir.
func writeStackYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stack.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSnapshotGHC(t *testing.T) {
	t.Parallel()
	tests := []struct {
		snapshot string
		want     string
	}{
		{"lts-22.0", "9.6.3"},
		{"lts-22.6", "9.6.3"},
		{"lts-22.7", "9.6.4"},
		{"lts-22.43", "9.6.6"},
		{"lts-22.44", "9.6.7"},
		{"lts-22", "9.6.7"}, // closed series: its last release
		{"lts-23.28", "9.8.4"},
		{"lts-24.11", "9.10.2"},
		{"lts-24.12", "9.10.3"},
		{"lts-24.60", "9.10.3"},
		{"lts-24.61", ""}, // newer than the verified data
		{"lts-24", ""},    // open series: its latest release is unknown
		{"lts-25.0", ""},
		{"lts-11.22", ""}, // older than the table
		{"lts-12.0", "8.4.3"},
		{" lts-21.25 ", "9.4.8"},
		{"ghc-9.6.7", "9.6.7"},
		{"ghc-9.6", ""},
		{"nightly-2025-01-01", ""},
		{"https://raw.githubusercontent.com/commercialhaskell/stackage-snapshots/master/lts/22/44.yaml", "9.6.7"},
		{"https://example.test/custom.yaml", ""},
		{"./snapshot.yaml", ""},
		{"lts-22.x", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.snapshot, func(t *testing.T) {
			t.Parallel()
			if got := snapshotGHC(tt.snapshot); got != tt.want {
				t.Errorf("snapshotGHC(%q) = %q, want %q", tt.snapshot, got, tt.want)
			}
		})
	}
}

func TestLTSCompilersOrdered(t *testing.T) {
	t.Parallel()
	for i := 1; i < len(ltsCompilers); i++ {
		prev, cur := ltsCompilers[i-1], ltsCompilers[i]
		if cur.major < prev.major || cur.major == prev.major && cur.minor <= prev.minor {
			t.Errorf("ltsCompilers[%d] (lts-%d.%d) is not after lts-%d.%d", i, cur.major, cur.minor, prev.major, prev.minor)
		}
		if !ghcVersionRe.MatchString(cur.ghc) {
			t.Errorf("ltsCompilers[%d] GHC %q is not a full version", i, cur.ghc)
		}
		if cur.major != prev.major && cur.minor != 0 {
			t.Errorf("lts-%d starts at minor %d, want 0", cur.major, cur.minor)
		}
	}
	if last := ltsCompilers[len(ltsCompilers)-1]; last.major != ltsVerifiedMajor {
		t.Errorf("last entry is lts-%d, want the verified series lts-%d", last.major, ltsVerifiedMajor)
	}
}

func TestReadStackCompiler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		content    string
		wantGHC    string
		wantSource string
		wantErr    error
	}{
		{name: "snapshot key", content: "snapshot: lts-22.44\n", wantGHC: "9.6.7", wantSource: `snapshot "lts-22.44"`},
		{name: "resolver key", content: "resolver: lts-21.25\npackages: [.]\n", wantGHC: "9.4.8", wantSource: `snapshot "lts-21.25"`},
		{name: "snapshot wins over resolver", content: "snapshot: lts-23.1\nresolver: lts-21.25\n", wantGHC: "9.8.4"},
		{name: "compiler override", content: "resolver: lts-22.44\ncompiler: ghc-9.6.6\n", wantGHC: "9.6.6", wantSource: `compiler "ghc-9.6.6"`},
		{name: "ghcjs compiler", content: "resolver: lts-22.44\ncompiler: ghcjs-0.2.1\n", wantGHC: "", wantSource: `compiler "ghcjs-0.2.1"`},
		{name: "nightly", content: "resolver: nightly-2025-01-01\n", wantGHC: "", wantSource: `snapshot "nightly-2025-01-01"`},
		{name: "mapping snapshot", content: "snapshot:\n  url: https://example.test/s.yaml\n", wantGHC: "", wantSource: "a custom snapshot"},
		{name: "no snapshot", content: "packages: [.]\n", wantErr: errNoStackSnapshot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sc, err := readStackCompiler(writeStackYAML(t, tt.content))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("readStackCompiler() error = %v, want %v", err, tt.wantErr)
			}
			if sc.ghc != tt.wantGHC {
				t.Errorf("ghc = %q, want %q", sc.ghc, tt.wantGHC)
			}
			if tt.wantSource != "" && sc.source != tt.wantSource {
				t.Errorf("source = %q, want %q", sc.source, tt.wantSource)
			}
		})
	}
}

func TestReadStackCompiler_Errors(t *testing.T) {
	t.Parallel()
	if _, err := readStackCompiler(t.TempDir()); err == nil {
		t.Error("missing stack.yaml: want an error")
	}
	if _, err := readStackCompiler(writeStackYAML(t, "resolver: [unclosed\n")); err == nil {
		t.Error("invalid YAML: want an error")
	}
}

func TestCompilerAttr(t *testing.T) {
	t.Parallel()
	for v, want := range map[string]string{"9.6.7": "ghc967", "9.10.3": "ghc9103", "9.6": "ghc96"} {
		if got := compilerAttr(v); got != want {
			t.Errorf("compilerAttr(%q) = %q, want %q", v, got, want)
		}
	}
}
