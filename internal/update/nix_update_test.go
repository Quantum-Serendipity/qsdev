package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTestFile(%s): %v", name, err)
	}
	return path
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readTestFile(%s): %v", path, err)
	}
	return string(data)
}

func TestUpdateDevenvNix_Unmodified(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "old content\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("new content\n"),
		Status:      types.Unmodified,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixRegenerated {
		t.Errorf("expected NixRegenerated, got %d", result.Action)
	}

	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "new content\n" {
		t.Errorf("expected devenv.nix to have new content, got: %q", got)
	}

	// No sidecar should exist.
	sidecar := filepath.Join(dir, "devenv.nix.new")
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Error("expected no sidecar file for unmodified update")
	}
}

func TestUpdateDevenvNix_Modified_CreatesSidecar(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "original content\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("updated content\n"),
		Status:      types.Modified,
		Force:       false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixSidecarCreated {
		t.Errorf("expected NixSidecarCreated, got %d", result.Action)
	}

	// Original file should be unchanged.
	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "original content\n" {
		t.Errorf("expected devenv.nix unchanged, got: %q", got)
	}

	// Sidecar should contain new content.
	sidecar := filepath.Join(dir, "devenv.nix.new")
	sidecarContent := readTestFile(t, sidecar)
	if sidecarContent != "updated content\n" {
		t.Errorf("expected sidecar to have new content, got: %q", sidecarContent)
	}

	if result.DiffOutput == "" {
		t.Error("expected non-empty diff output")
	}
	if result.NewFilePath != sidecar {
		t.Errorf("expected NewFilePath=%q, got %q", sidecar, result.NewFilePath)
	}
}

func TestUpdateDevenvNix_Modified_Force(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "original content\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("forced content\n"),
		Status:      types.Modified,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixForceOverwritten {
		t.Errorf("expected NixForceOverwritten, got %d", result.Action)
	}

	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "forced content\n" {
		t.Errorf("expected devenv.nix overwritten, got: %q", got)
	}

	// No sidecar should exist.
	sidecar := filepath.Join(dir, "devenv.nix.new")
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Error("expected no sidecar file for force overwrite")
	}
}

func TestUpdateDevenvNix_Deleted_Skips(t *testing.T) {
	dir := t.TempDir()

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("new content\n"),
		Status:      types.Deleted,
		Force:       false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixSkipped {
		t.Errorf("expected NixSkipped, got %d", result.Action)
	}

	// File should not exist.
	if _, err := os.Stat(filepath.Join(dir, "devenv.nix")); !os.IsNotExist(err) {
		t.Error("expected devenv.nix to not be created when deleted and not forced")
	}
}

func TestUpdateDevenvNix_Deleted_Force(t *testing.T) {
	dir := t.TempDir()

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("recreated content\n"),
		Status:      types.Deleted,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixForceOverwritten {
		t.Errorf("expected NixForceOverwritten, got %d", result.Action)
	}

	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "recreated content\n" {
		t.Errorf("expected devenv.nix recreated, got: %q", got)
	}
}

func TestUpdateDevenvNix_New(t *testing.T) {
	dir := t.TempDir()

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("brand new content\n"),
		Status:      types.New,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixRegenerated {
		t.Errorf("expected NixRegenerated, got %d", result.Action)
	}

	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "brand new content\n" {
		t.Errorf("expected devenv.nix created, got: %q", got)
	}
}

func TestUpdateDevenvNix_StaleSidecarCleaned(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "original\n")

	// Create a stale sidecar with different content.
	staleSidecar := filepath.Join(dir, "devenv.nix.new")
	writeTestFile(t, dir, "devenv.nix.new", "stale sidecar\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("fresh content\n"),
		Status:      types.Modified,
		Force:       false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixSidecarCreated {
		t.Errorf("expected NixSidecarCreated, got %d", result.Action)
	}

	// The sidecar should now contain the fresh content, not the stale content.
	sidecarContent := readTestFile(t, staleSidecar)
	if sidecarContent != "fresh content\n" {
		t.Errorf("expected sidecar to have fresh content, got: %q", sidecarContent)
	}
}

func TestUpdateDevenvNix_DryRun_NoWrite(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "original content\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("new content\n"),
		Status:      types.Modified,
		Force:       false,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != NixSidecarCreated {
		t.Errorf("expected NixSidecarCreated, got %d", result.Action)
	}

	// Original file should be unchanged.
	got := readTestFile(t, filepath.Join(dir, "devenv.nix"))
	if got != "original content\n" {
		t.Errorf("expected devenv.nix unchanged, got: %q", got)
	}

	// Sidecar should NOT have been written.
	sidecar := filepath.Join(dir, "devenv.nix.new")
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Error("expected no sidecar file in dry-run mode")
	}

	// Diff should still be computed.
	if result.DiffOutput == "" {
		t.Error("expected non-empty diff output even in dry-run mode")
	}
}

func TestUpdateDevenvNix_DiffContent(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "devenv.nix", "line1\nline2\nline3\n")

	result, err := UpdateDevenvNix(NixUpdateOptions{
		ProjectRoot: dir,
		FilePath:    "devenv.nix",
		NewContent:  []byte("line1\nchanged\nline3\n"),
		Status:      types.Modified,
		Force:       false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	diff := result.DiffOutput
	if !strings.Contains(diff, "---") {
		t.Errorf("expected diff to contain '---', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++") {
		t.Errorf("expected diff to contain '+++', got:\n%s", diff)
	}
	if !strings.Contains(diff, "@@") {
		t.Errorf("expected diff to contain '@@', got:\n%s", diff)
	}
}
