package devenv_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
)

func TestBuildAnswersFromFlags(t *testing.T) {
	projectRoot := "/tmp/test-project"
	langs := []string{"go", "javascript"}
	services := []string{"postgres", "redis"}
	direnvEnabled := true

	answers := devenv.ExportBuildAnswersFromFlags(projectRoot, langs, services, direnvEnabled)

	if answers.ProjectRoot != projectRoot {
		t.Errorf("ProjectRoot = %q, want %q", answers.ProjectRoot, projectRoot)
	}
	if answers.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q, want %q", answers.ProjectName, "test-project")
	}
	if answers.Direnv != direnvEnabled {
		t.Errorf("Direnv = %v, want %v", answers.Direnv, direnvEnabled)
	}
	if len(answers.Languages) != 2 {
		t.Fatalf("Languages count = %d, want 2", len(answers.Languages))
	}
	if answers.Languages[0].Name != "go" {
		t.Errorf("Languages[0].Name = %q, want %q", answers.Languages[0].Name, "go")
	}
	if answers.Languages[1].Name != "javascript" {
		t.Errorf("Languages[1].Name = %q, want %q", answers.Languages[1].Name, "javascript")
	}
	if len(answers.Services) != 2 {
		t.Fatalf("Services count = %d, want 2", len(answers.Services))
	}
	if answers.Services[0].Name != "postgres" {
		t.Errorf("Services[0].Name = %q, want %q", answers.Services[0].Name, "postgres")
	}
	if answers.Services[1].Name != "redis" {
		t.Errorf("Services[1].Name = %q, want %q", answers.Services[1].Name, "redis")
	}
}

func TestBuildAnswersFromFlags_Empty(t *testing.T) {
	answers := devenv.ExportBuildAnswersFromFlags("/tmp/empty", nil, nil, false)

	if answers.Direnv != false {
		t.Errorf("Direnv = %v, want false", answers.Direnv)
	}
	if len(answers.Languages) != 0 {
		t.Errorf("Languages count = %d, want 0", len(answers.Languages))
	}
	if len(answers.Services) != 0 {
		t.Errorf("Services count = %d, want 0", len(answers.Services))
	}
}

func TestValidServices(t *testing.T) {
	expected := map[string]bool{
		"postgres":      true,
		"redis":         true,
		"mysql":         true,
		"mongodb":       true,
		"elasticsearch": true,
		"rabbitmq":      true,
	}

	services := devenv.ExportValidServices
	if len(services) != len(expected) {
		t.Fatalf("validServices has %d entries, want %d", len(services), len(expected))
	}

	for _, svc := range services {
		if !expected[svc] {
			t.Errorf("unexpected service in validServices: %q", svc)
		}
	}
}

func TestValidLanguages(t *testing.T) {
	expected := map[string]bool{
		"go":         true,
		"javascript": true,
		"python":     true,
		"rust":       true,
		"java":       true,
		"dotnet":     true,
		"docker":     true,
		"terraform":  true,
	}

	languages := devenv.ExportValidLanguages
	if len(languages) != len(expected) {
		t.Fatalf("validLanguages has %d entries, want %d", len(languages), len(expected))
	}

	for _, lang := range languages {
		if !expected[lang] {
			t.Errorf("unexpected language in validLanguages: %q", lang)
		}
	}
}

// chdir changes the working directory to dir and registers a cleanup to restore
// the original directory when the test finishes.
func chdir(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

func TestInitCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--dry-run", "--yes", "--lang", "go"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --dry-run failed: %v", err)
	}

	output := buf.String()

	// Dry-run should show a preview containing file paths.
	if !strings.Contains(output, "devenv.nix") {
		t.Error("dry-run output should mention devenv.nix")
	}

	// Verify no files were actually written.
	for _, name := range []string{"devenv.nix", "devenv.yaml"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("dry-run should not write %s", name)
		}
	}
}

func TestInitCmd_WritesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify key generated files exist.
	expectedFiles := []string{
		filepath.Join(tmpDir, "devenv.yaml"),
		filepath.Join(tmpDir, "devenv.nix"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %q was not created: %v", f, err)
		}
	}

	// Verify devenv.nix has content.
	data, err := os.ReadFile(filepath.Join(tmpDir, "devenv.nix"))
	if err != nil {
		t.Fatalf("reading devenv.nix: %v", err)
	}
	if len(data) == 0 {
		t.Error("devenv.nix should not be empty")
	}
}

func TestInitCmd_ExistingDevenvNix_NoForce(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Create existing devenv.nix.
	nixPath := filepath.Join(tmpDir, "devenv.nix")
	if err := os.WriteFile(nixPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when devenv.nix exists without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestInitCmd_ExistingDevenvNix_Force(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Create existing devenv.nix.
	nixPath := filepath.Join(tmpDir, "devenv.nix")
	if err := os.WriteFile(nixPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --force should succeed: %v", err)
	}

	// Verify devenv.nix was overwritten (no longer "existing").
	data, err := os.ReadFile(nixPath)
	if err != nil {
		t.Fatalf("reading devenv.nix: %v", err)
	}
	if string(data) == "existing" {
		t.Error("devenv.nix should have been overwritten by --force")
	}
}

func TestInitCmd_SavesState(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	stateFile := filepath.Join(tmpDir, ".devenv", ".gdev-state.yaml")
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("state file not saved: %v", err)
	}

	// Verify state file has content.
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("reading state file: %v", err)
	}
	if len(data) == 0 {
		t.Error("state file should not be empty")
	}
}

func TestInitCmd_SavesAnswers(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	answersFile := filepath.Join(tmpDir, ".devenv", ".gdev-answers.yaml")
	if _, err := os.Stat(answersFile); err != nil {
		t.Errorf("answers file not saved: %v", err)
	}

	// Verify answers can be loaded back.
	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if len(answers.Languages) != 1 || answers.Languages[0].Name != "go" {
		t.Errorf("loaded answers should have go language, got: %v", answers.Languages)
	}
}

func TestUpdateCmd_NoSavedAnswers(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"update"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no saved answers exist")
	}
	if !strings.Contains(err.Error(), "no saved answers") {
		t.Errorf("error should mention 'no saved answers', got: %v", err)
	}
}

func TestUpdateCmd_AfterInit(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Now, update.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"update", "--force"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Verify files still exist after update.
	for _, name := range []string{"devenv.yaml", "devenv.nix"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q should still exist after update: %v", name, err)
		}
	}
}

func TestAddServiceCmd_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add service.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-service", "postgres"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-service postgres failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "postgres") {
		t.Errorf("output should mention added service, got: %s", output)
	}

	// Verify answers now include the service.
	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}

	found := false
	for _, svc := range answers.Services {
		if svc.Name == "postgres" {
			found = true
			break
		}
	}
	if !found {
		t.Error("answers should include postgres service after add-service")
	}
}

func TestAddServiceCmd_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try invalid service.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-service", "invalid-service"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for invalid service")
	}
	if !strings.Contains(err.Error(), "unknown service") {
		t.Errorf("error should mention 'unknown service', got: %v", err)
	}
	if !strings.Contains(err.Error(), "valid services") {
		t.Errorf("error should list valid services, got: %v", err)
	}
}

func TestAddServiceCmd_Duplicate_NoForce(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init with postgres already included.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--services", "postgres", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try adding postgres again without --force.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-service", "postgres"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate service without --force")
	}
	if !strings.Contains(err.Error(), "already configured") {
		t.Errorf("error should mention 'already configured', got: %v", err)
	}
}

func TestAddLanguageCmd_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init with go.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add python.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-language", "python"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-language python failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "python") {
		t.Errorf("output should mention added language, got: %s", output)
	}

	// Verify answers now include python.
	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}

	found := false
	for _, lang := range answers.Languages {
		if lang.Name == "python" {
			found = true
			break
		}
	}
	if !found {
		t.Error("answers should include python language after add-language")
	}
}

func TestAddLanguageCmd_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// First, init.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try invalid language.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-language", "invalid-lang"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for invalid language")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Errorf("error should mention 'unknown language', got: %v", err)
	}
	if !strings.Contains(err.Error(), "valid languages") {
		t.Errorf("error should list valid languages, got: %v", err)
	}
}

func TestAddLanguageCmd_Duplicate_NoForce(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init with go.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try adding go again without --force.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-language", "go"})

	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate language without --force")
	}
	if !strings.Contains(err.Error(), "already configured") {
		t.Errorf("error should mention 'already configured', got: %v", err)
	}
}

func TestInitCmd_MultipleLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go,python", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with multiple languages failed: %v", err)
	}

	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if len(answers.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(answers.Languages))
	}

	names := make(map[string]bool)
	for _, lang := range answers.Languages {
		names[lang.Name] = true
	}
	if !names["go"] || !names["python"] {
		t.Errorf("expected go and python, got: %v", names)
	}
}

func TestInitCmd_WithServices(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--services", "postgres,redis", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with services failed: %v", err)
	}

	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if len(answers.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(answers.Services))
	}

	names := make(map[string]bool)
	for _, svc := range answers.Services {
		names[svc.Name] = true
	}
	if !names["postgres"] || !names["redis"] {
		t.Errorf("expected postgres and redis, got: %v", names)
	}
}

func TestInitCmd_DirenvEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--direnv", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with direnv failed: %v", err)
	}

	// Verify .envrc was created when direnv is enabled.
	envrcPath := filepath.Join(tmpDir, ".envrc")
	if _, err := os.Stat(envrcPath); err != nil {
		t.Errorf(".envrc should be created when direnv is enabled: %v", err)
	}
}

func TestInitCmd_NoDirenv(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--direnv=false", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init without direnv failed: %v", err)
	}

	// Verify .envrc was NOT created when direnv is disabled.
	envrcPath := filepath.Join(tmpDir, ".envrc")
	if _, err := os.Stat(envrcPath); err == nil {
		t.Error(".envrc should NOT be created when direnv is disabled")
	}
}

func TestUpdateCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init first.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Get modification time of devenv.nix before update.
	nixPath := filepath.Join(tmpDir, "devenv.nix")
	infoBefore, err := os.Stat(nixPath)
	if err != nil {
		t.Fatalf("stat devenv.nix before update: %v", err)
	}

	// Run update with --dry-run.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"update", "--dry-run"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("update --dry-run failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "devenv.nix") {
		t.Error("dry-run output should mention devenv.nix")
	}

	// Verify file was NOT re-written (same mod time).
	infoAfter, err := os.Stat(nixPath)
	if err != nil {
		t.Fatalf("stat devenv.nix after dry-run: %v", err)
	}
	if infoAfter.ModTime() != infoBefore.ModTime() {
		t.Error("dry-run should not modify files on disk")
	}
}

func TestAddServiceCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init first.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add service with --dry-run.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-service", "postgres", "--dry-run"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-service --dry-run failed: %v", err)
	}

	// Verify the service was NOT actually persisted.
	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	for _, svc := range answers.Services {
		if svc.Name == "postgres" {
			t.Error("dry-run should not persist the new service to answers")
		}
	}
}

func TestAddLanguageCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)

	// Init first.
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init", "--lang", "go", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add language with --dry-run.
	cmd2 := devenv.ExportDevenvCmd()
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"add-language", "python", "--dry-run"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add-language --dry-run failed: %v", err)
	}

	// Verify the language was NOT actually persisted.
	answers, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	for _, lang := range answers.Languages {
		if lang.Name == "python" {
			t.Error("dry-run should not persist the new language to answers")
		}
	}
}
