package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestEnsureGitignoreEntry_CreatesMissing(t *testing.T) {
	dir := t.TempDir()

	err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml")
	if err != nil {
		t.Fatalf("EnsureGitignoreEntry: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	if !strings.Contains(string(content), ".qsdev.local.yaml") {
		t.Errorf(".gitignore does not contain entry, got:\n%s", content)
	}
	if !strings.Contains(string(content), fileutil.GitignoreSectionComment()) {
		t.Error(".gitignore should contain section comment")
	}
}

func TestEnsureGitignoreEntry_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()

	existing := "node_modules/\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml")
	if err != nil {
		t.Fatalf("EnsureGitignoreEntry: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "node_modules/") {
		t.Error("existing entries should be preserved")
	}
	if !strings.Contains(s, ".qsdev.local.yaml") {
		t.Error(".gitignore should contain the new entry")
	}
}

func TestEnsureGitignoreEntry_NoopWhenPresent(t *testing.T) {
	dir := t.TempDir()

	existing := "node_modules/\n.qsdev.local.yaml\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml")
	if err != nil {
		t.Fatalf("EnsureGitignoreEntry: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	// Count occurrences — should appear exactly once.
	count := strings.Count(string(content), ".qsdev.local.yaml")
	if count != 1 {
		t.Errorf("entry appears %d times, want 1; content:\n%s", count, content)
	}
}

func TestEnsureGitignoreEntry_HandlesNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	existing := "node_modules/"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml")
	if err != nil {
		t.Fatalf("EnsureGitignoreEntry: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, ".qsdev.local.yaml") {
		t.Errorf("entry not found in .gitignore:\n%s", s)
	}
	// Ensure no double entries from missing newline handling.
	if strings.Contains(s, "node_modules/.qsdev.local.yaml") {
		t.Errorf("entries should be on separate lines, got:\n%s", s)
	}
}

func TestEnsureGitignoreEntry_SectionCommentAddedOnce(t *testing.T) {
	dir := t.TempDir()

	// Add first entry.
	if err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml"); err != nil {
		t.Fatal(err)
	}

	// Add second entry.
	if err := EnsureGitignoreEntry(dir, ".devinit/"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	count := strings.Count(string(content), fileutil.GitignoreSectionComment())
	if count != 1 {
		t.Errorf("section comment appears %d times, want 1; content:\n%s", count, content)
	}
}

// TestFinalizeProject_IgnoresLocalConfig is the W166 regression test: join
// adds the machine-local overrides file to .gitignore, so init must as well,
// or every teammate's first join dirties the committed .gitignore.
func TestFinalizeProject_IgnoresLocalConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := &cobra.Command{}
	if err := finalizeProject(cmd, InitOptions{Quiet: true}, types.WizardAnswers{}, dir, false, false); err != nil {
		t.Fatalf("finalizeProject: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	local := branding.Get().LocalConfig
	if !slices.Contains(strings.Split(string(data), "\n"), local) {
		t.Errorf(".gitignore does not list %s:\n%s", local, data)
	}
	// Join's own EnsureGitignoreEntry call must then be a no-op.
	if err := EnsureGitignoreEntry(dir, local); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(data) {
		t.Errorf("join changed .gitignore after init:\n--- init\n%s\n--- join\n%s", data, after)
	}
}
