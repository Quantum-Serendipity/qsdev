package answers_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestLoadPrimary_CorruptFileReturnsError pins F-CAP-2.5-1: a corrupt primary
// answers file must surface an error rather than being silently treated as
// empty state (which would let posture aggregation report against nothing).
func TestLoadPrimary_CorruptFileReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	b := branding.Get()
	dir := filepath.Join(tmpDir, b.StateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "."+b.AppName+"-init-answers.yaml")
	if err := os.WriteFile(path, []byte("{not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := answers.LoadPrimary(tmpDir)
	if err == nil {
		t.Fatal("expected error for corrupt primary answers file, got nil (silent data loss)")
	}
	if !strings.Contains(err.Error(), "parsing primary answers") {
		t.Errorf("error = %q, want 'parsing primary answers' context", err.Error())
	}
}

// TestLoadPrimary_MissingFileIsNotError confirms the absent-file path is still
// a benign zero-value return (the corruption fix must not change this).
func TestLoadPrimary_MissingFileIsNotError(t *testing.T) {
	tmpDir := t.TempDir()
	a, err := answers.LoadPrimary(tmpDir)
	if err != nil {
		t.Fatalf("missing primary file should not be an error, got: %v", err)
	}
	if a.ProjectName != "" || len(a.Languages) != 0 {
		t.Errorf("expected zero-value answers for missing file, got %+v", a)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	original := types.WizardAnswers{
		ProjectName: "round-trip-test",
		ProjectRoot: tmpDir,
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24", PackageManager: "gomod"},
			{Name: "javascript", Version: "22", PackageManager: "npm"},
		},
		Services: []types.ServiceChoice{
			{Name: "postgres", Version: "16"},
			{Name: "redis"},
		},
		Direnv:   true,
		GitHooks: []string{"ripsecrets"},
		EnvVars:  map[string]string{"FOO": "bar"},
	}

	if err := answers.SaveToDir(tmpDir, ".testdir", "answers.yaml", original); err != nil {
		t.Fatalf("SaveToDir failed: %v", err)
	}

	loaded, err := answers.LoadFromDir(tmpDir, ".testdir", "answers.yaml", "test")
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	if loaded.ProjectName != original.ProjectName {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, original.ProjectName)
	}
	if len(loaded.Languages) != len(original.Languages) {
		t.Fatalf("Languages count = %d, want %d", len(loaded.Languages), len(original.Languages))
	}
	for i, lang := range loaded.Languages {
		if lang.Name != original.Languages[i].Name {
			t.Errorf("Languages[%d].Name = %q, want %q", i, lang.Name, original.Languages[i].Name)
		}
		if lang.Version != original.Languages[i].Version {
			t.Errorf("Languages[%d].Version = %q, want %q", i, lang.Version, original.Languages[i].Version)
		}
	}
	if len(loaded.Services) != len(original.Services) {
		t.Fatalf("Services count = %d, want %d", len(loaded.Services), len(original.Services))
	}
	for i, svc := range loaded.Services {
		if svc.Name != original.Services[i].Name {
			t.Errorf("Services[%d].Name = %q, want %q", i, svc.Name, original.Services[i].Name)
		}
	}
	if loaded.Direnv != original.Direnv {
		t.Errorf("Direnv = %v, want %v", loaded.Direnv, original.Direnv)
	}
	if len(loaded.GitHooks) != len(original.GitHooks) {
		t.Errorf("GitHooks count = %d, want %d", len(loaded.GitHooks), len(original.GitHooks))
	}
	if loaded.EnvVars["FOO"] != "bar" {
		t.Errorf("EnvVars[FOO] = %q, want %q", loaded.EnvVars["FOO"], "bar")
	}
}

func TestLoadFromDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := answers.LoadFromDir(tmpDir, ".nonexistent", "answers.yaml", "test")
	if err == nil {
		t.Fatal("expected error when answers file does not exist, got nil")
	}

	// Verify the error message includes the command name hint.
	want := "run 'qsdev test init' first"
	if got := err.Error(); !containsStr(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

func TestSaveToDir_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	a := types.WizardAnswers{ProjectName: "dirtest"}

	if err := answers.SaveToDir(tmpDir, ".newdir", "answers.yaml", a); err != nil {
		t.Fatalf("SaveToDir failed: %v", err)
	}

	dir := filepath.Join(tmpDir, ".newdir")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".newdir should be a directory")
	}

	file := filepath.Join(dir, "answers.yaml")
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("answers file not created: %v", err)
	}
}

func TestSaveToDir_AtomicWrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file sharing semantics prevent atomic rename under continuous readers")
	}
	tmpDir := t.TempDir()
	dir := ".atomictest"
	filename := "answers.yaml"

	// Write initial content.
	initial := types.WizardAnswers{ProjectName: "initial"}
	if err := answers.SaveToDir(tmpDir, dir, filename, initial); err != nil {
		t.Fatalf("initial SaveToDir failed: %v", err)
	}

	replacement := types.WizardAnswers{ProjectName: "replacement"}

	// Start concurrent readers while writing.
	var wg sync.WaitGroup
	const readers = 10
	errCh := make(chan string, readers)
	stop := make(chan struct{})

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				path := answers.FilePath(tmpDir, dir, filename)
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				s := string(data)
				// Content should always be valid YAML with either
				// "initial" or "replacement", never partial/mixed.
				if !containsStr(s, "initial") && !containsStr(s, "replacement") {
					errCh <- s
					return
				}
			}
		}()
	}

	// Perform the atomic write while readers are running.
	if err := answers.SaveToDir(tmpDir, dir, filename, replacement); err != nil {
		t.Fatalf("SaveToDir failed: %v", err)
	}

	close(stop)
	wg.Wait()
	close(errCh)

	for partial := range errCh {
		t.Errorf("concurrent reader saw corrupt content: %q", partial)
	}

	// Final verification: file should contain replacement content.
	loaded, err := answers.LoadFromDir(tmpDir, dir, filename, "test")
	if err != nil {
		t.Fatalf("final LoadFromDir failed: %v", err)
	}
	if loaded.ProjectName != "replacement" {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, "replacement")
	}
}

func TestFilePath(t *testing.T) {
	got := answers.FilePath("/project", ".devenv", ".qsdev-answers.yaml")
	want := filepath.Join("/project", ".devenv", ".qsdev-answers.yaml")
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
