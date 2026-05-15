package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(string(content), gitignoreSectionComment) {
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

	count := strings.Count(string(content), gitignoreSectionComment)
	if count != 1 {
		t.Errorf("section comment appears %d times, want 1; content:\n%s", count, content)
	}
}
