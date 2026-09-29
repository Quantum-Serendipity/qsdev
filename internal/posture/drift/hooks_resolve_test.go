package drift

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// findingSubjects returns the subjects of a category's findings.
func findingSubjects(cat Category) map[string]bool {
	subjects := make(map[string]bool, len(cat.Findings))
	for _, f := range cat.Findings {
		subjects[f.Subject] = true
	}
	return subjects
}

// writeGitdirWorktree lays out a main repository git dir and a linked
// worktree whose .git is a "gitdir:" file with a commondir back-link, the
// shape `git worktree add` creates. It returns the worktree directory and the
// main repository's hooks directory.
func writeGitdirWorktree(t *testing.T) (worktree, commonHooks string) {
	t.Helper()
	root := t.TempDir()
	mainGit := filepath.Join(root, "main", ".git")
	wtGitDir := filepath.Join(mainGit, "worktrees", "feature")
	if err := os.MkdirAll(wtGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	commonHooks = filepath.Join(mainGit, "hooks")
	if err := os.MkdirAll(commonHooks, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileMode(t, filepath.Join(wtGitDir, "commondir"), "../..\n", 0o644)

	worktree = filepath.Join(root, "feature")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileMode(t, filepath.Join(worktree, ".git"), "gitdir: "+wtGitDir+"\n", 0o644)
	return worktree, commonHooks
}

// TestDetectHookDrift_GitdirFileWorktree is the F343/F546 regression test: in a
// worktree or submodule .git is a file, and the hooks live in the git dir it
// points at. Missing hooks there must be reported, not silently dropped.
func TestDetectHookDrift_GitdirFileWorktree(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		install      []string
		wantSubjects []string
	}{
		{name: "no hooks installed", wantSubjects: []string{"pre-commit", "commit-msg"}},
		{name: "hooks installed in common dir", install: []string{"pre-commit", "commit-msg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			worktree, commonHooks := writeGitdirWorktree(t)
			for _, h := range tt.install {
				writeFileMode(t, filepath.Join(commonHooks, h), "#!/bin/sh\n", 0o755)
			}

			cat := detectHookDrift(worktree, map[string]bool{"commitlint": true})

			got := findingSubjects(cat)
			if len(cat.Findings) != len(tt.wantSubjects) {
				t.Fatalf("got %d findings %+v, want subjects %v", len(cat.Findings), cat.Findings, tt.wantSubjects)
			}
			for _, s := range tt.wantSubjects {
				if !got[s] {
					t.Errorf("missing finding for %q: %+v", s, cat.Findings)
				}
			}
		})
	}
}

// TestDetectHookDrift_MalformedGitFile pins that an unusable .git file is
// reported instead of reading as a clean result.
func TestDetectHookDrift_MalformedGitFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFileMode(t, filepath.Join(dir, ".git"), "not a gitdir link\n", 0o644)

	cat := detectHookDrift(dir, nil)
	if len(cat.Findings) == 0 {
		t.Fatal("expected a finding for an unusable .git file, got none")
	}
}

// runGit runs git in dir with a fixed identity, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com", "-C", dir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = repoScopedEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestDetectHookDrift_RealGit exercises the git-resolved hooks path against
// real repositories: a linked worktree without hooks, and core.hooksPath.
func TestDetectHookDrift_RealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Isolate from the developer's global and system git config (a global
	// core.hooksPath would otherwise decide where hooks are looked up).
	// t.Setenv rules out t.Parallel for this test; its subtests still run
	// in parallel with each other.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	t.Run("linked worktree without hooks", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		repo := filepath.Join(root, "repo")
		if err := os.MkdirAll(repo, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, repo, "init", "-q")
		runGit(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
		wt := filepath.Join(root, "feature")
		runGit(t, repo, "worktree", "add", "-q", wt)

		cat := detectHookDrift(wt, nil)
		if !findingSubjects(cat)["pre-commit"] {
			t.Errorf("expected missing pre-commit finding in worktree, got %+v", cat.Findings)
		}
	})

	t.Run("core.hooksPath honoured", func(t *testing.T) {
		t.Parallel()
		repo := t.TempDir()
		runGit(t, repo, "init", "-q")
		runGit(t, repo, "config", "core.hooksPath", ".githooks")
		hooks := filepath.Join(repo, ".githooks")
		if err := os.MkdirAll(hooks, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFileMode(t, filepath.Join(hooks, "pre-commit"), "#!/bin/sh\n", 0o755)

		cat := detectHookDrift(repo, nil)
		if len(cat.Findings) != 0 {
			t.Errorf("expected no findings with hooks under core.hooksPath, got %+v", cat.Findings)
		}
	})
}
