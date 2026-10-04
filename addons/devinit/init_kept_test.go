package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitMergeReportsKeptEnvrc is the regression test for U14-04: a user's
// .envrc kept by the Skip strategy during `init --yes --merge` was logged at
// info level only, so the summary showed it as a bare "skipped 1". The
// summary now counts it as kept and names the file.
func TestInitMergeReportsKeptEnvrc(t *testing.T) {
	dir := createGoFixture(t)
	const envrc = "export FOO=mine\n"
	if err := os.WriteFile(filepath.Join(dir, ".envrc"), []byte(envrc), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--yes", "--merge", "--lang", "go")
	if err != nil {
		t.Fatalf("init --merge: %v\n%s", err, out)
	}
	for _, want := range []string{"kept 1", "kept existing .envrc"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if got := readProjectFile(t, dir, ".envrc"); got != envrc {
		t.Errorf(".envrc changed:\n%s", got)
	}
}

// TestInitPartialWriteStillPrintsSummary verifies a partial write prints the
// summary before failing, so the kept notice for .envrc is not lost when
// another file fails in the same run.
func TestInitPartialWriteStillPrintsSummary(t *testing.T) {
	dir := createGoFixture(t)
	const envrc = "export FOO=mine\n"
	if err := os.WriteFile(filepath.Join(dir, ".envrc"), []byte(envrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "security-overview.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--yes", "--merge", "--lang", "go")
	if err == nil || !strings.Contains(err.Error(), "partial write") {
		t.Fatalf("want a partial-write error, got %v\n%s", err, out)
	}
	for _, want := range []string{"kept 1", "kept existing .envrc", "FAILED docs/security-overview.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
