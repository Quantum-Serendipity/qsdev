package canon

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestResolver_MatchesUncached checks that one Resolver shared by many paths
// answers each exactly as the uncached functions do, on first lookup and when
// remembered, for existing, missing, symlinked and failing paths alike.
func TestResolver_MatchesUncached(t *testing.T) {
	t.Parallel()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "real", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	rels := []string{
		".", "real", "real/sub", "real/sub/../x", "file", "file/x",
		"missing", "missing/a/b", "missing/../real/sub", "missing/a/../../file",
		"real/missing/../../real", strings.Repeat("a/", 40) + "x",
	}
	if runtime.GOOS != "windows" {
		for name, target := range map[string]string{
			"lnk":      filepath.Join(dir, "real"),
			"rel":      "real/sub",
			"dangling": filepath.Join(dir, "gone", "target"),
			"loop1":    "loop2",
			"loop2":    "loop1",
		} {
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
		rels = append(rels, "lnk", "lnk/sub", "lnk/x/y", "rel", "rel/../x", "dangling", "dangling/x",
			"missing/../lnk/z", "missing/lnk/x", "loop1", "loop1/x", "real/../lnk/../dangling")
	}
	paths := []string{"~", "~/" + filepath.Base(dir), "relative/path"}
	for _, rel := range rels {
		paths = append(paths, dir+string(filepath.Separator)+filepath.FromSlash(rel))
	}

	var r Resolver
	for _, pass := range []string{"first", "remembered"} {
		for _, p := range paths {
			got, gotErr := r.Canonicalize(p)
			want, wantErr := Canonicalize(p)
			if got != want || (gotErr != nil) != (wantErr != nil) {
				t.Errorf("%s Canonicalize(%q) = (%q, %v), want (%q, %v)", pass, p, got, gotErr, want, wantErr)
			}
			gotEval, gotErr := r.EvalSymlinks(p)
			wantEval, wantErr := filepath.EvalSymlinks(p)
			if gotEval != wantEval || (gotErr != nil) != (wantErr != nil) {
				t.Errorf("%s EvalSymlinks(%q) = (%q, %v), want (%q, %v)", pass, p, gotEval, gotErr, wantEval, wantErr)
			}
			gotInfo, gotErr := r.Stat(p)
			wantInfo, wantErr := os.Stat(p)
			if (gotErr != nil) != (wantErr != nil) || gotErr == nil && gotInfo.IsDir() != wantInfo.IsDir() {
				t.Errorf("%s Stat(%q) = (%v, %v), want (%v, %v)", pass, p, gotInfo, gotErr, wantInfo, wantErr)
			}
		}
	}
}

// TestResolver_NilAsksAfresh checks that a nil *Resolver is usable and
// answers as the uncached functions do.
func TestResolver_NilAsksAfresh(t *testing.T) {
	t.Parallel()
	var r *Resolver
	p := filepath.Join(t.TempDir(), "missing", "x")
	got, err := r.Canonicalize(p)
	want, wantErr := Canonicalize(p)
	if got != want || (err != nil) != (wantErr != nil) {
		t.Errorf("nil Canonicalize(%q) = (%q, %v), want (%q, %v)", p, got, err, want, wantErr)
	}
	if _, err := r.Stat(p); err == nil {
		t.Errorf("nil Stat(%q) succeeded for a missing path", p)
	}
	if _, err := r.EvalSymlinks(p); err == nil {
		t.Errorf("nil EvalSymlinks(%q) succeeded for a missing path", p)
	}
}
