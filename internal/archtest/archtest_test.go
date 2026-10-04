package archtest

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	moduleRoot   = "../.."
	baselineFile = "baseline.txt"
	// baseEnv names a copy of the base branch's baseline.txt; CI sets it
	// from `git show origin/<base_ref>:internal/archtest/baseline.txt`.
	baseEnv = "ARCHTEST_BASE"
)

var update = flag.Bool("update", false, "regenerate baseline.txt from the current tree")

// TestArchitecture requires the computed violations to equal baseline.txt
// exactly: new sites and raised counts fail, and so do fixed sites that are
// still listed, so the baseline can only shrink.
func TestArchitecture(t *testing.T) {
	t.Parallel()
	repo, err := Load(moduleRoot)
	if err != nil {
		t.Fatalf("loading module: %v", err)
	}
	dropGitIgnored(t, repo)
	got := Collect(repo, Rules())

	if *update {
		if err := os.WriteFile(baselineFile, FormatBaseline(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", baselineFile, err)
		}
		t.Logf("wrote %s: %d entries", baselineFile, len(got))
		return
	}

	want := readBaseline(t, baselineFile)
	t.Logf("%s: %d entries", baselineFile, len(want))
	for _, p := range Compare(got, want) {
		t.Error(p)
	}
}

// TestBaselineMonotone rejects a baseline.txt that grew relative to the base
// branch, which closes the loophole of regenerating it with -update.
func TestBaselineMonotone(t *testing.T) {
	t.Parallel()
	basePath := os.Getenv(baseEnv)
	if basePath == "" {
		t.Skipf("%s not set: this diff check runs in CI against the base branch", baseEnv)
	}
	base := readBaseline(t, filepath.Clean(basePath))
	current := readBaseline(t, baselineFile)
	for _, p := range CompareMonotone(current, base) {
		t.Error(p)
	}
}

// dropGitIgnored removes the files git ignores from repo, so local-only
// trees (gitignored notes or scratch programs) neither fail the check nor
// leak into a regenerated baseline: the result matches a fresh checkout.
// Untracked files that are not ignored stay, so new code is still checked.
// Outside a git work tree every file is kept.
func dropGitIgnored(t *testing.T, repo *Repo) {
	t.Helper()
	out, err := exec.Command("git", "-C", moduleRoot, "ls-files", "-z",
		"--cached", "--others", "--exclude-standard", "--", "*.go").Output()
	if err != nil {
		t.Logf("listing files with git: %v; checking every file", err)
		return
	}
	visible := make(map[string]bool)
	for _, p := range bytes.Split(out, []byte{0}) {
		visible[string(p)] = true
	}
	kept := repo.Files[:0]
	for _, f := range repo.Files {
		if visible[f.Path] {
			kept = append(kept, f)
		}
	}
	repo.Files = kept
}

func readBaseline(t *testing.T, path string) Set {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening baseline: %v", err)
	}
	defer f.Close()
	s, err := ParseBaseline(f)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return s
}
