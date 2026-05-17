package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrialCmd_HasCorrectUseAndFlags(t *testing.T) {
	cmd := trialCmd()

	if cmd.Use != "trial" {
		t.Errorf("Use = %q, want %q", cmd.Use, "trial")
	}

	expectedFlags := []string{"branch", "path", "profile", "dry-run"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q not found", name)
		}
	}

	if cmd.Flags().ShorthandLookup("b") == nil {
		t.Error("expected shorthand -b for --branch")
	}
	if cmd.Flags().ShorthandLookup("p") == nil {
		t.Error("expected shorthand -p for --path")
	}
}

func TestTrialCmd_DryRun(t *testing.T) {
	// Create a temp dir that looks like a git repo.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("setup .git: %v", err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := trialCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("dry-run failed: %v\nOutput: %s", err, buf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Would create:") {
		t.Errorf("output should contain 'Would create:', got:\n%s", output)
	}
	if !strings.Contains(output, "Worktree:") {
		t.Errorf("output should contain 'Worktree:', got:\n%s", output)
	}
	if !strings.Contains(output, "Branch:") {
		t.Errorf("output should contain 'Branch:', got:\n%s", output)
	}
}

func TestTrialCmd_FailsOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := trialCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error = %q, want it to contain 'not a git repository'", err.Error())
	}
}

func TestTrialCmd_FailsWhenPathExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("setup .git: %v", err)
	}

	// Create the target path so it already exists.
	targetPath := filepath.Join(dir, "existing-target")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("setup target: %v", err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := trialCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", targetPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when path exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want it to contain 'already exists'", err.Error())
	}
}

func TestIsGitRepo(t *testing.T) {
	t.Run("directory with .git dir", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if !isGitRepo(dir) {
			t.Error("expected true for directory with .git")
		}
	})

	t.Run("directory with .git file (worktree)", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../main/.git/worktrees/trial"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if !isGitRepo(dir) {
			t.Error("expected true for directory with .git file")
		}
	})

	t.Run("directory without .git", func(t *testing.T) {
		dir := t.TempDir()
		if isGitRepo(dir) {
			t.Error("expected false for directory without .git")
		}
	})
}
