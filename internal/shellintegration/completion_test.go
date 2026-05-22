package shellintegration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// testRootCmd creates a minimal Cobra root command for testing completion
// generation. It has a subcommand so completions are non-trivial.
func testRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "qsdev",
		Short: "test root command",
	}
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "print version",
		Run:   func(_ *cobra.Command, _ []string) {},
	})
	return root
}

func TestCompletionInstaller_InstallBash(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-populate the RC file so we can verify existing content is preserved.
	if err := os.WriteFile(rcFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	if err := installer.Install(rootCmd, "bash", rcFile); err != nil {
		t.Fatalf("Install bash failed: %v", err)
	}

	// Verify completion file was written.
	completionFile := filepath.Join(tmpDir, ".qsdev", "completions", "qsdev.bash")
	data, err := os.ReadFile(completionFile)
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("completion file should not be empty")
	}

	// Verify RC file was updated.
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	rc := string(rcContent)

	if !strings.Contains(rc, completionMarkerStart()) {
		t.Error("RC file should contain completion start marker")
	}
	if !strings.Contains(rc, completionMarkerEnd()) {
		t.Error("RC file should contain completion end marker")
	}
	if !strings.Contains(rc, completionFile) {
		t.Errorf("RC file should reference completion file %s", completionFile)
	}
	if !strings.Contains(rc, "# existing") {
		t.Error("existing RC content should be preserved")
	}
}

func TestCompletionInstaller_InstallZsh(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".zshrc")

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	if err := installer.Install(rootCmd, "zsh", rcFile); err != nil {
		t.Fatalf("Install zsh failed: %v", err)
	}

	// Verify completion file.
	completionFile := filepath.Join(tmpDir, ".qsdev", "completions", "_qsdev")
	data, err := os.ReadFile(completionFile)
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("completion file should not be empty")
	}

	// Verify RC file has fpath and compinit.
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	rc := string(rcContent)

	completionDir := filepath.Join(tmpDir, ".qsdev", "completions")
	if !strings.Contains(rc, completionDir) {
		t.Error("RC file should reference completion directory in fpath")
	}
	if !strings.Contains(rc, "compinit") {
		t.Error("RC file should contain compinit line")
	}
}

func TestCompletionInstaller_InstallFish(t *testing.T) {
	tmpDir := t.TempDir()

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	// Fish does not need an RC file.
	if err := installer.Install(rootCmd, "fish", ""); err != nil {
		t.Fatalf("Install fish failed: %v", err)
	}

	// Verify completion file in fish's auto-load directory.
	completionFile := filepath.Join(tmpDir, ".config", "fish", "completions", "qsdev.fish")
	data, err := os.ReadFile(completionFile)
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("completion file should not be empty")
	}
}

func TestCompletionInstaller_InstallPowershell(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, "profile.ps1")

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	if err := installer.Install(rootCmd, "pwsh", rcFile); err != nil {
		t.Fatalf("Install powershell failed: %v", err)
	}

	// Verify completion file.
	completionFile := filepath.Join(tmpDir, ".qsdev", "completions", "qsdev.ps1")
	data, err := os.ReadFile(completionFile)
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("completion file should not be empty")
	}

	// Verify RC file sources the completion file.
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rcContent), completionFile) {
		t.Error("profile should reference completion file")
	}
}

func TestCompletionInstaller_UnsupportedShell(t *testing.T) {
	tmpDir := t.TempDir()

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	err := installer.Install(rootCmd, "nushell", "")
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompletionInstaller_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()

	// First install.
	if err := installer.Install(rootCmd, "bash", rcFile); err != nil {
		t.Fatalf("first Install failed: %v", err)
	}
	first, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	// Second install should produce the same result.
	rootCmd2 := testRootCmd()
	if err := installer.Install(rootCmd2, "bash", rcFile); err != nil {
		t.Fatalf("second Install failed: %v", err)
	}
	second, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Errorf("second install changed the RC file.\nFirst:\n%s\nSecond:\n%s", first, second)
	}
}

func TestCompletionInstaller_FullPathShell(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	installer := &CompletionInstaller{
		BinaryName: "qsdev",
		HomeDir:    tmpDir,
	}

	rootCmd := testRootCmd()
	// Use full path like /bin/bash.
	if err := installer.Install(rootCmd, "/bin/bash", rcFile); err != nil {
		t.Fatalf("Install with full path shell failed: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".qsdev", "completions", "qsdev.bash")
	if _, err := os.Stat(completionFile); err != nil {
		t.Errorf("completion file should exist: %v", err)
	}
}
