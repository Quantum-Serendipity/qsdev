package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cloneOf commits the initialised project at dir and returns a fresh clone of
// it: the committed tree only, with no local init state or answers.
func cloneOf(t *testing.T, env []string, dir string, cloneArgs ...string) string {
	t.Helper()
	git := func(wd string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = wd
		cmd.Env = append(env, "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git(dir, "add", "-A")
	git(dir, "-c", "commit.gpgsign=false", "commit", "-q", "--no-verify", "-m", "init")
	clone := filepath.Join(t.TempDir(), "clone")
	git(dir, append(append([]string{"clone", "-q"}, cloneArgs...), dir, clone)...)
	return clone
}

// gitStatus returns the clone's `git status --porcelain`: every committed
// file changed or removed and every non-ignored file created.
func gitStatus(t *testing.T, env []string, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	return string(out)
}

// TestFreshClone_ClaudeAndDevenvCommandsRequireJoin is the H0-3 follow-up: on
// a fresh clone of an initialised project, the claude and devenv commands
// that regenerate files refuse with the join hint and exit 3, leaving the
// team's committed .qsdev.yaml, CLAUDE.md and skills untouched, instead of
// pointing at `claude init` / `devenv init` and rewriting them.
func TestFreshClone_ClaudeAndDevenvCommandsRequireJoin(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"claude", "update"},
		{"claude", "init", "--yes"},
		{"claude", "init", "--yes", "--force"},
		{"claude", "add-hook", "audit-log"},
		{"claude", "add-skill", "review-pr"},
		{"devenv", "init", "--yes", "--force"},
		{"devenv", "update"},
		{"devenv", "add-package", "jq"},
		{"devenv", "add-language", "python"},
		{"devenv", "add-service", "redis"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			dir, _ := initialisedProject(t, env, false)
			clone := cloneOf(t, env, dir)
			out, code := runQsdev(t, env, clone, nil, args...)
			if code != 3 || !strings.Contains(out, "init --yes") {
				t.Fatalf("qsdev %s on a fresh clone: exit %d, want 3 with the join hint\n%s", strings.Join(args, " "), code, out)
			}
			if status := gitStatus(t, env, clone); status != "" {
				t.Errorf("qsdev %s changed the committed tree:\n%s", strings.Join(args, " "), status)
			}
		})
	}
}

// TestJoin_RenamedCloneRegeneratesCommittedFiles is the join regression for a
// clone checked out under another directory name: joining regenerates
// exactly the committed files (the project name comes from .qsdev.yaml, not
// the directory), reports no false local change, and keeps the gitignored
// .qsdev.local.yaml out of the committed manifest.
func TestJoin_RenamedCloneRegeneratesCommittedFiles(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir, _ := initialisedProject(t, env, false)
	clone := cloneOf(t, env, dir)
	out, code := runQsdev(t, env, clone, nil, "init", "--yes")
	if code != 0 {
		t.Fatalf("join: exit %d\n%s", code, out)
	}
	if strings.Contains(out, "local changes") {
		t.Errorf("join reported a local change in a pristine clone:\n%s", out)
	}
	if status := gitStatus(t, env, clone); status != "" {
		t.Errorf("join changed the committed tree:\n%s", status)
	}
}

// TestJoin_CRLFCheckoutRegeneratesNothing is the join regression for a clone
// whose text files Git checked out with CRLF line endings (core.autocrlf, the
// Git for Windows default): the committed files are the generated content in
// another line-ending spelling, so joining reports no local change and leaves
// the committed tree as Git sees it untouched.
func TestJoin_CRLFCheckoutRegeneratesNothing(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir, _ := initialisedProject(t, env, false)
	clone := cloneOf(t, env, dir, "-c", "core.autocrlf=true")
	data, err := os.ReadFile(filepath.Join(clone, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("\r\n")) {
		t.Fatal("core.autocrlf=true checked CLAUDE.md out without CRLF line endings")
	}
	// Scripts are pinned to LF by the generated .gitattributes: a CR on the
	// interpreter line would stop them starting.
	if data, err := os.ReadFile(filepath.Join(clone, ".envrc")); err != nil || bytes.Contains(data, []byte("\r")) {
		t.Fatalf(".envrc checked out with CR line endings (err %v)", err)
	}
	out, code := runQsdev(t, env, clone, nil, "init", "--yes")
	if code != 0 {
		t.Fatalf("join: exit %d\n%s", code, out)
	}
	if strings.Contains(out, "local changes") {
		t.Errorf("join reported a local change in a CRLF checkout:\n%s", out)
	}
	if status := gitStatus(t, env, clone); status != "" {
		t.Errorf("join changed the committed tree:\n%s", status)
	}
}
