package posture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestDetectDrift_Aggregation(t *testing.T) {
	dir := t.TempDir()

	// Create a valid git repo with pre-commit hook.
	setupGitRepo(t, dir)

	// Create CLAUDE.md with markers for semgrep.
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"# CLAUDE\n<!-- gdev:semgrep -->\nSemgrep section\n<!-- /gdev:semgrep -->\n")

	// Create a generated file that matches its hash.
	content := []byte("hello world\n")
	writeFile(t, filepath.Join(dir, "test.txt"), string(content))

	genState := types.GeneratedState{
		GdevVersion: "dev", // Matches the default version.
		Files: map[string]types.FileState{
			"test.txt": {
				Hash:     state.ComputeHash(content),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	// Mock lookPath so semgrep is "found".
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	defer func() { lookPath = origLookPath }()

	report := DetectDrift(dir, genState, enabledTools)

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Should have 6 categories.
	if got := len(report.Categories); got != 6 {
		t.Errorf("expected 6 categories, got %d", got)
	}

	// Verify category names.
	expectedNames := []string{
		categoryFileModification,
		categoryVersionDrift,
		categoryToolAvailability,
		categoryMarkerIntegrity,
		categoryLockfileDrift,
		categoryHookDrift,
	}
	for i, name := range expectedNames {
		if report.Categories[i].Name != name {
			t.Errorf("category %d: expected %q, got %q", i, name, report.Categories[i].Name)
		}
	}

	// Verify BySeverity counts match TotalFindings.
	total := 0
	for _, count := range report.BySeverity {
		total += count
	}
	if total != report.TotalFindings {
		t.Errorf("BySeverity sum (%d) != TotalFindings (%d)", total, report.TotalFindings)
	}
}

func TestDetectDrift_EmptyState(t *testing.T) {
	dir := t.TempDir()

	// No git repo, no files.
	report := DetectDrift(dir, types.GeneratedState{}, nil)

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.BySeverity == nil {
		t.Error("expected non-nil BySeverity map")
	}
}

func BenchmarkDetectDrift(b *testing.B) {
	dir := b.TempDir()
	setupGitRepo(b, dir)
	writeFile(b, filepath.Join(dir, "CLAUDE.md"), "# CLAUDE\n")

	genState := types.GeneratedState{
		GdevVersion: "dev",
		Files:       map[string]types.FileState{},
	}
	enabledTools := map[string]bool{"semgrep": true}

	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	defer func() { lookPath = origLookPath }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectDrift(dir, genState, enabledTools)
	}
}

// setupGitRepo creates a minimal .git directory with a hooks subdirectory
// and executable pre-commit hook.
func setupGitRepo(tb testing.TB, dir string) {
	tb.Helper()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		tb.Fatal(err)
	}
	writeFileMode(tb, filepath.Join(hooksDir, "pre-commit"), "#!/bin/sh\n", 0o755)
}

// writeFile creates a file with the given content and default permissions.
func writeFile(tb testing.TB, path, content string) {
	tb.Helper()
	writeFileMode(tb, path, content, 0o644)
}

// writeFileMode creates a file with the given content and permissions.
func writeFileMode(tb testing.TB, path, content string, mode os.FileMode) {
	tb.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		tb.Fatal(err)
	}
}
