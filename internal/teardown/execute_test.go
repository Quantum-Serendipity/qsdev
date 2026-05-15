package teardown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

func TestExecute_RemoveExclusiveFiles(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "remove-me.txt")
	if err := os.WriteFile(filePath, []byte("delete this"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "remove-me.txt", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// File should be removed.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("file should have been removed, but still exists")
	}

	if len(result.Removed) != 1 {
		t.Errorf("Removed = %d, want 1", len(result.Removed))
	}
}

func TestExecute_CleanSharedFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# Project\n<!-- qsdev:test-tool -->\ngenerated stuff\n<!-- /qsdev:test-tool -->\n\nUser content\n")
	mdPath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(mdPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	registry := toolreg.NewRegistry()
	_ = registry.Register(toolreg.Tool{
		Name:     "test-tool",
		Category: toolreg.CategorySecurity,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: "README.md", Ownership: toolreg.Shared, SectionID: "test-tool"},
		},
	})

	plan := &TeardownPlan{
		Clean: []FileAction{
			{Path: "README.md", Reason: "remove qsdev sections"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Cleaned) != 1 {
		t.Errorf("Cleaned = %d, want 1", len(result.Cleaned))
	}

	// Read the file back and verify the section was removed.
	updated, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}

	if contains(updated, "<!-- qsdev:test-tool -->") {
		t.Errorf("expected qsdev section markers to be removed from README.md")
	}
	if !contains(updated, "User content") {
		t.Errorf("expected user content to be preserved in README.md")
	}
}

func TestExecute_RemoveDirs(t *testing.T) {
	dir := t.TempDir()
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Put a file in the directory.
	if err := os.WriteFile(filepath.Join(devinitDir, "state.yaml"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Dirs: []string{".devinit"},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(devinitDir); !os.IsNotExist(err) {
		t.Errorf("directory .devinit should have been removed")
	}

	if len(result.DirsRemoved) != 1 {
		t.Errorf("DirsRemoved = %d, want 1", len(result.DirsRemoved))
	}
}

func TestExecute_DryRun(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "keep-me.txt")
	if err := os.WriteFile(filePath, []byte("keep this"), 0o644); err != nil {
		t.Fatal(err)
	}
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "keep-me.txt", Reason: "exclusive"},
		},
		Dirs: []string{".devinit"},
	}

	opts := TeardownOptions{ProjectRoot: dir, DryRun: true}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// DryRun: nothing should be changed.
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file should still exist in dry-run mode")
	}
	if _, err := os.Stat(devinitDir); err != nil {
		t.Errorf("directory should still exist in dry-run mode")
	}

	// But result should list what would be done.
	if len(result.Removed) != 1 {
		t.Errorf("Removed = %d, want 1 (dry-run still populates result)", len(result.Removed))
	}
	if len(result.DirsRemoved) != 1 {
		t.Errorf("DirsRemoved = %d, want 1 (dry-run still populates result)", len(result.DirsRemoved))
	}
}

func TestExecute_RemoveEmptyDirs(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/skills/ with a single file that we will remove.
	skillsDir := filepath.Join(dir, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillsDir, "test.md")
	if err := os.WriteFile(skillFile, []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: ".claude/skills/test.md", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	_, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// .claude/skills/ should be removed (now empty).
	if _, err := os.Stat(skillsDir); !os.IsNotExist(err) {
		t.Errorf(".claude/skills/ should be removed after becoming empty")
	}

	// .claude/ should also be removed (now empty).
	claudeDir := filepath.Join(dir, ".claude")
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Errorf(".claude/ should be removed after becoming empty")
	}
}

func TestExecute_NonexistentFileRemoval(t *testing.T) {
	dir := t.TempDir()

	plan := &TeardownPlan{
		Remove: []FileAction{
			{Path: "doesnt-exist.txt", Reason: "exclusive"},
		},
	}

	opts := TeardownOptions{ProjectRoot: dir}
	registry := toolreg.NewRegistry()

	result, err := Execute(plan, opts, registry)
	if err != nil {
		t.Fatal(err)
	}

	// Should succeed (os.Remove on nonexistent file is treated as success).
	if len(result.Removed) != 1 {
		t.Errorf("Removed = %d, want 1 (nonexistent file removal is a no-op success)", len(result.Removed))
	}
}

func contains(data []byte, needle string) bool {
	for i := 0; i <= len(data)-len(needle); i++ {
		if string(data[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}
