package archtest

import (
	"flag"
	"os"
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
