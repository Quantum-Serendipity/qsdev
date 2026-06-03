package goldentest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

// Assert compares got against the golden file at goldenPath.
// If GOLDEN_UPDATE=1 is set, writes got to goldenPath instead.
func Assert(t testing.TB, goldenPath string, got []byte) {
	t.Helper()

	if os.Getenv("GOLDEN_UPDATE") == "1" {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("updating golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s: %v\n(run with GOLDEN_UPDATE=1 to create)", goldenPath, err)
	}

	if string(got) != string(want) {
		diff := unifiedDiff(string(want), string(got), goldenPath)
		t.Errorf("output differs from golden file %s:\n%s", goldenPath, diff)
	}
}

func unifiedDiff(want, got, path string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(want),
		B:        difflib.SplitLines(got),
		FromFile: path + " (golden)",
		ToFile:   path + " (actual)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}
