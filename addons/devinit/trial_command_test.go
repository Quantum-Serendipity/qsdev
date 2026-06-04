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

func TestValidateBranchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{name: "valid simple", branch: "feature-branch", wantErr: false},
		{name: "valid with slash", branch: "feature/branch", wantErr: false},
		{name: "valid with dots", branch: "v1.2.3", wantErr: false},
		{name: "valid with underscore", branch: "my_branch", wantErr: false},
		{name: "valid alphanumeric", branch: "abc123", wantErr: false},
		{name: "valid qsdev-trial", branch: "qsdev-trial", wantErr: false},
		{name: "empty", branch: "", wantErr: true},
		{name: "too long", branch: strings.Repeat("a", 251), wantErr: true},
		{name: "max length ok", branch: strings.Repeat("a", 250), wantErr: false},
		{name: "starts with dot", branch: ".hidden", wantErr: true},
		{name: "starts with dash", branch: "-flag", wantErr: true},
		{name: "starts with slash", branch: "/bad", wantErr: true},
		{name: "contains space", branch: "my branch", wantErr: true},
		{name: "contains double dot", branch: "a..b", wantErr: true},
		{name: "contains tilde", branch: "a~1", wantErr: true},
		{name: "contains caret", branch: "a^1", wantErr: true},
		{name: "contains colon", branch: "a:b", wantErr: true},
		{name: "ends with .lock", branch: "branch.lock", wantErr: true},
		{name: "ends with slash", branch: "branch/", wantErr: true},
		{name: "shell injection attempt", branch: "$(whoami)", wantErr: true},
		{name: "semicolon injection", branch: "branch;rm -rf /", wantErr: true},
		{name: "backtick injection", branch: "branch`id`", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBranchName(tt.branch)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBranchName(%q) error = %v, wantErr = %v", tt.branch, err, tt.wantErr)
			}
		})
	}
}

func TestTrialCmd_RejectsInvalidBranchName(t *testing.T) {
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
	cmd.SetArgs([]string{"--branch", "$(whoami)", "--path", "/tmp/nonexistent-trial"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid branch name, got nil")
	}
	if !strings.Contains(err.Error(), "validating branch name") {
		t.Errorf("error = %q, want it to contain 'validating branch name'", err.Error())
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
