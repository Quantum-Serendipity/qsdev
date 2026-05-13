package devinit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Helpers ---

func readFileContent(t *testing.T, dir, relPath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, relPath))
	if err != nil {
		t.Fatalf("reading %s: %v", relPath, err)
	}
	return string(content)
}

func requireFileExists(t *testing.T, dir, relPath string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, relPath)); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", relPath)
	}
}

func requireFileNotExists(t *testing.T, dir, relPath string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, relPath)); err == nil {
		t.Errorf("expected file %s to not exist", relPath)
	}
}

func requireFileContains(t *testing.T, dir, relPath, substr string) {
	t.Helper()
	content := readFileContent(t, dir, relPath)
	if !strings.Contains(content, substr) {
		t.Errorf("file %s does not contain %q (len=%d)", relPath, substr, len(content))
	}
}

// --- Integration Tests ---

func TestIntegration_EmptyDir_GoWebProfile(t *testing.T) {
	dir := t.TempDir()
	output, err := executeInitCmd(t, dir, "--profile", "go-web", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Devenv files.
	requireFileExists(t, dir, "devenv.yaml")
	requireFileExists(t, dir, "devenv.nix")
	requireFileExists(t, dir, ".envrc")

	// Claude Code files.
	requireFileExists(t, dir, ".claude/settings.json")
	requireFileExists(t, dir, "CLAUDE.md")
	requireFileExists(t, dir, ".claude/hooks/package-guard.py")

	// Skills from go-web profile.
	requireFileExists(t, dir, ".claude/skills/deploy.md")
	requireFileExists(t, dir, ".claude/skills/security-review.md")

	// Rules for Go.
	requireFileExists(t, dir, ".claude/rules/go-conventions.md")
	requireFileExists(t, dir, ".claude/rules/security-rules.md")

	// State and answers saved.
	requireFileExists(t, dir, ".devinit/.gdev-init-state.yaml")
	requireFileExists(t, dir, ".devinit/.gdev-init-answers.yaml")
	requireFileExists(t, dir, ".devenv/.gdev-answers.yaml")
	requireFileExists(t, dir, ".claude/.gdev-claude-answers.yaml")

	// Content spot-checks.
	requireFileContains(t, dir, "devenv.nix", "go")
	requireFileContains(t, dir, ".envrc", "use devenv")
	requireFileContains(t, dir, "CLAUDE.md", "<!-- BEGIN GENERATED SECTION")
	requireFileContains(t, dir, "CLAUDE.md", "<!-- END GENERATED SECTION -->")
	requireFileContains(t, dir, "CLAUDE.md", "## Security")
}

func TestIntegration_GoProjectDetection(t *testing.T) {
	dir := t.TempDir()
	// Create a go.mod.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := executeInitCmd(t, dir, "--yes", "--force")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Go should be detected.
	requireFileContains(t, dir, "devenv.nix", "go")
	requireFileExists(t, dir, ".claude/rules/go-conventions.md")

	// Answers should include Go.
	answersContent := readFileContent(t, dir, ".devinit/.gdev-init-answers.yaml")
	if !strings.Contains(answersContent, "name: go") {
		t.Error("answers should contain Go language")
	}
}

func TestIntegration_UpdateCommand_UnmodifiedRegenerate(t *testing.T) {
	dir := t.TempDir()

	// Initial generation.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Record original devenv.yaml content.
	originalContent := readFileContent(t, dir, "devenv.yaml")

	// Run update.
	output, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\nOutput: %s", err, output)
	}

	// devenv.yaml should still be present with same content (regenerated but same template).
	newContent := readFileContent(t, dir, "devenv.yaml")
	if originalContent != newContent {
		t.Error("devenv.yaml content changed unexpectedly during update of unmodified file")
	}

	// State should be updated.
	requireFileExists(t, dir, ".devinit/.gdev-init-state.yaml")
}

func TestIntegration_UpdateCommand_ModifiedFile_MergeOrSkip(t *testing.T) {
	dir := t.TempDir()

	// Initial generation.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Modify devenv.yaml (which has Overwrite strategy).
	yamlPath := filepath.Join(dir, "devenv.yaml")
	yamlContent, _ := os.ReadFile(yamlPath)
	modifiedContent := append(yamlContent, []byte("\n# user customization\n")...)
	os.WriteFile(yamlPath, modifiedContent, 0o644)

	// Run update (no force).
	output, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\nOutput: %s", err, output)
	}

	// devenv.yaml was modified + strategy is Overwrite -> action is "skip" per buildUpdatePlan.
	// Verify the user modification is still present.
	content := readFileContent(t, dir, "devenv.yaml")
	if !strings.Contains(content, "# user customization") {
		t.Error("user modification in devenv.yaml should be preserved when not using --force")
	}
}

func TestIntegration_UpdateCommand_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()

	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Modify devenv.yaml.
	yamlPath := filepath.Join(dir, "devenv.yaml")
	yamlContent, _ := os.ReadFile(yamlPath)
	os.WriteFile(yamlPath, append(yamlContent, []byte("\n# user change\n")...), 0o644)

	// Update with --force.
	output, err := executeInitCmd(t, dir, "--update", "--force")
	if err != nil {
		t.Fatalf("update --force failed: %v\nOutput: %s", err, output)
	}

	// User modification should be gone.
	content := readFileContent(t, dir, "devenv.yaml")
	if strings.Contains(content, "# user change") {
		t.Error("user modification should be overwritten with --force")
	}
}

func TestIntegration_UpdateCommand_NoSavedAnswers(t *testing.T) {
	dir := t.TempDir()

	_, err := executeInitCmd(t, dir, "--update")
	if err == nil {
		t.Error("expected error when no saved answers exist")
	}
	if err != nil && !strings.Contains(err.Error(), "no saved answers") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIntegration_UpdateCommand_DeletedFile_NotRecreated(t *testing.T) {
	dir := t.TempDir()

	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Delete .envrc.
	os.Remove(filepath.Join(dir, ".envrc"))

	// Update without force.
	output, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\nOutput: %s", err, output)
	}

	// .envrc should not be recreated.
	requireFileNotExists(t, dir, ".envrc")
}

func TestIntegration_UpdateCommand_DryRun(t *testing.T) {
	dir := t.TempDir()

	// Initial generation.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Record state before update.
	stateBefore := readFileContent(t, dir, ".devinit/.gdev-init-state.yaml")

	// Dry-run update.
	output, err := executeInitCmd(t, dir, "--update", "--dry-run")
	if err != nil {
		t.Fatalf("update --dry-run failed: %v\nOutput: %s", err, output)
	}

	// Output should contain the preview table.
	if !strings.Contains(output, "File") || !strings.Contains(output, "Status") {
		t.Error("dry-run output missing preview table headers")
	}

	// State should NOT have changed.
	stateAfter := readFileContent(t, dir, ".devinit/.gdev-init-state.yaml")
	if stateBefore != stateAfter {
		t.Error("state file changed during dry-run")
	}
}

func TestIntegration_ProfilePlusFlagOverride(t *testing.T) {
	dir := t.TempDir()

	// Use go-web profile but override permissions to minimal.
	output, err := executeInitCmd(t, dir, "--profile", "go-web", "--claude-permissions", "minimal", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// devenv.nix should still have Go (from profile).
	requireFileContains(t, dir, "devenv.nix", "go")

	// settings.json should have minimal permissions (no defaultMode).
	settingsContent := readFileContent(t, dir, ".claude/settings.json")
	if strings.Contains(settingsContent, `"defaultMode"`) {
		t.Error("settings.json should not contain defaultMode with minimal preset")
	}
}

func TestIntegration_SecurityHardenedDefaults(t *testing.T) {
	dir := t.TempDir()

	_, err := executeInitCmd(t, dir, "--profile", "go-web", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// devenv.yaml: clean environment enabled.
	requireFileContains(t, dir, "devenv.yaml", "clean:")

	// devenv.nix: security hooks present.
	requireFileContains(t, dir, "devenv.nix", "ripsecrets")
	requireFileContains(t, dir, "devenv.nix", "DEVENV_SECURITY_HARDENED")

	// settings.json: comprehensive deny rules.
	settingsContent := readFileContent(t, dir, ".claude/settings.json")
	// Check representative deny rules from different categories.
	denyChecks := []string{
		"npm install",  // JS package managers
		"pip install",  // Python
		"curl",         // pipe-to-shell (partial match in deny rules)
		"rm -rf",       // destructive ops
	}
	for _, check := range denyChecks {
		if !strings.Contains(settingsContent, check) {
			t.Errorf("settings.json missing deny rule containing %q", check)
		}
	}

	// CLAUDE.md: security section.
	requireFileContains(t, dir, "CLAUDE.md", "## Security")
	requireFileContains(t, dir, "CLAUDE.md", "Lock files")

	// package-guard.py: exists and is executable.
	hookPath := filepath.Join(dir, ".claude/hooks/package-guard.py")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("package-guard.py not found: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Error("package-guard.py should be executable")
	}

	// security-rules.md exists.
	requireFileExists(t, dir, ".claude/rules/security-rules.md")
}

func TestIntegration_UpdateCommand_SectionMarkerMerge(t *testing.T) {
	dir := t.TempDir()

	// Initial generation.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Read CLAUDE.md and add user content after the end marker.
	claudeMd := readFileContent(t, dir, "CLAUDE.md")
	endMarker := "<!-- END GENERATED SECTION -->"
	endIdx := strings.Index(claudeMd, endMarker)
	if endIdx < 0 {
		t.Fatal("CLAUDE.md missing end marker")
	}
	afterMarker := endIdx + len(endMarker)
	// Find the next newline after marker.
	if afterMarker < len(claudeMd) && claudeMd[afterMarker] == '\n' {
		afterMarker++
	}

	// Replace everything after end marker with custom content.
	modified := claudeMd[:afterMarker] + "\n## My Custom Section\n\nDo not touch production.\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(modified), 0o644)

	// Run update.
	output, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\nOutput: %s", err, output)
	}

	// User content should be preserved.
	updatedContent := readFileContent(t, dir, "CLAUDE.md")
	if !strings.Contains(updatedContent, "My Custom Section") {
		t.Error("user content after end marker should be preserved")
	}
	if !strings.Contains(updatedContent, "Do not touch production") {
		t.Error("user content should survive section marker merge")
	}
	// Generated section should still be present.
	if !strings.Contains(updatedContent, "<!-- BEGIN GENERATED SECTION") {
		t.Error("begin marker should be present after merge")
	}
}

func TestIntegration_UpdateCommand_ThreeWayMerge_Settings(t *testing.T) {
	dir := t.TempDir()

	// Initial generation.
	_, err := executeInitCmd(t, dir, "--lang", "go", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add a custom allow rule to settings.json.
	settingsPath := filepath.Join(dir, ".claude/settings.json")
	content, _ := os.ReadFile(settingsPath)
	// Insert a custom rule into the allow array.
	modified := strings.Replace(string(content),
		`"Read(*)"`+",",
		`"Read(*)",`+"\n"+`    "Bash(my-custom-tool *)",`,
		1)
	// If no trailing comma after Read(*), try without comma.
	if modified == string(content) {
		modified = strings.Replace(string(content),
			`"Read(*)"`,
			`"Read(*)","Bash(my-custom-tool *)"`,
			1)
	}
	os.WriteFile(settingsPath, []byte(modified), 0o644)

	// Run update.
	output, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\nOutput: %s", err, output)
	}

	// User-added rule should be preserved.
	updatedSettings := readFileContent(t, dir, ".claude/settings.json")
	if !strings.Contains(updatedSettings, "my-custom-tool") {
		t.Error("user-added allow rule should survive three-way merge")
	}
	// Original generated rules should still be present.
	if !strings.Contains(updatedSettings, "Read(*)") {
		t.Error("generated allow rules should still be present")
	}
}
