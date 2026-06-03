package goldentest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil/goldentest"
)

// mockT implements just enough of testing.TB for error-case tests.
type mockT struct {
	testing.TB
	failed  bool
	fataled bool
}

func (m *mockT) Helper()                           {}
func (m *mockT) Fatalf(format string, args ...any) { m.fataled = true; m.failed = true }
func (m *mockT) Errorf(format string, args ...any) { m.failed = true }

func TestAssert_Match(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "expected.txt")
	content := []byte("hello world\n")
	if err := os.WriteFile(goldenPath, content, 0o644); err != nil {
		t.Fatalf("writing test fixture: %v", err)
	}

	goldentest.Assert(t, goldenPath, content)
}

func TestAssert_Mismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "expected.txt")
	if err := os.WriteFile(goldenPath, []byte("expected\n"), 0o644); err != nil {
		t.Fatalf("writing test fixture: %v", err)
	}

	mt := &mockT{}
	goldentest.Assert(mt, goldenPath, []byte("actual\n"))
	if !mt.failed {
		t.Error("Assert should have reported mismatch")
	}
}

func TestAssert_Update(t *testing.T) {
	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "new.txt")
	content := []byte("new content\n")

	t.Setenv("GOLDEN_UPDATE", "1")
	goldentest.Assert(t, goldenPath, content)

	got, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading updated golden file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("updated content = %q, want %q", got, content)
	}
}

func TestAssert_MissingFile(t *testing.T) {
	t.Parallel()
	mt := &mockT{}
	goldentest.Assert(mt, "/nonexistent/path/golden.txt", []byte("anything"))
	if !mt.fataled {
		t.Error("Assert should fatal on missing golden file")
	}
}

func TestAssert_NestedDir(t *testing.T) {
	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "sub", "dir", "file.txt")
	content := []byte("nested\n")

	t.Setenv("GOLDEN_UPDATE", "1")
	goldentest.Assert(t, goldenPath, content)

	got, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}
