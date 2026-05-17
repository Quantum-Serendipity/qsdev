package drift

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectHookDrift_HooksInstalled(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectHookDrift(dir, enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when hooks are installed, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectHookDrift_MissingPreCommit(t *testing.T) {
	dir := t.TempDir()
	// Create .git/hooks directory but no pre-commit file.
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cat := detectHookDrift(dir, nil)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for missing pre-commit, got %d: %+v", len(cat.Findings), cat.Findings)
	}

	f := cat.Findings[0]
	if f.Severity != Warning {
		t.Errorf("expected severity %q, got %q", Warning, f.Severity)
	}
	if f.Subject != "pre-commit" {
		t.Errorf("expected subject %q, got %q", "pre-commit", f.Subject)
	}
}

func TestDetectHookDrift_NoGitDirectory(t *testing.T) {
	dir := t.TempDir()
	// No .git directory.

	cat := detectHookDrift(dir, nil)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for no git repo, got %d: %+v", len(cat.Findings), cat.Findings)
	}

	f := cat.Findings[0]
	if f.Severity != Info {
		t.Errorf("expected severity %q, got %q", Info, f.Severity)
	}
	if f.Subject != ".git" {
		t.Errorf("expected subject %q, got %q", ".git", f.Subject)
	}
}

func TestDetectHookDrift_CommitlintEnabled_MissingCommitMsg(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir) // Has pre-commit but no commit-msg.

	enabledTools := map[string]bool{
		"commitlint": true,
	}

	cat := detectHookDrift(dir, enabledTools)

	found := false
	for _, f := range cat.Findings {
		if f.Subject == "commit-msg" {
			found = true
			if f.Severity != Warning {
				t.Errorf("expected severity %q, got %q", Warning, f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected warning about missing commit-msg hook when commitlint is enabled")
	}
}

func TestDetectHookDrift_CommitlintEnabled_CommitMsgPresent(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)
	writeFileMode(t, filepath.Join(dir, ".git", "hooks", "commit-msg"), "#!/bin/sh\n", 0o755)

	enabledTools := map[string]bool{
		"commitlint": true,
	}

	cat := detectHookDrift(dir, enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when all hooks present, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectHookDrift_NotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit check is skipped on Windows")
	}

	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create pre-commit without executable bit.
	writeFileMode(t, filepath.Join(hooksDir, "pre-commit"), "#!/bin/sh\n", 0o644)

	cat := detectHookDrift(dir, nil)

	found := false
	for _, f := range cat.Findings {
		if f.Subject == "pre-commit" && f.Severity == Warning {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about non-executable pre-commit hook")
	}
}

func TestDetectHookDrift_CommitlintDisabled_NoCommitMsgCheck(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	// Commitlint is not in the enabled tools.
	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectHookDrift(dir, enabledTools)

	for _, f := range cat.Findings {
		if f.Subject == "commit-msg" {
			t.Error("should not check commit-msg when commitlint is not enabled")
		}
	}
}
