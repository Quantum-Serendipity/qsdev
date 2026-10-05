package cigeneration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixupCommitStep returns the run script of the fixup workflow's
// "Commit and push" step.
func fixupCommitStep(t *testing.T) string {
	t.Helper()

	var wf fixupWorkflow
	readRepoYAML(t, filepath.Join(".github", "workflows", "dependabot-fixup.yml"), &wf)
	for _, job := range wf.Jobs {
		for _, s := range job.Steps {
			if s.Name == "Commit and push" {
				return s.Run
			}
		}
	}
	t.Fatal("dependabot-fixup.yml has no \"Commit and push\" step")
	return ""
}

// fixupRepo is a scratch clone of a Dependabot branch with a bare origin.
type fixupRepo struct {
	dir, origin string
	env         []string
}

const fixupBranch = "dependabot/go_modules/example.com/mod-1.2.3"

func newFixupRepo(t *testing.T) *fixupRepo {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the fixup step runs on the Linux runner under bash")
	}
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed; the CI runner provides it", tool)
		}
	}
	root := t.TempDir()
	r := &fixupRepo{
		dir:    filepath.Join(root, "work"),
		origin: filepath.Join(root, "origin.git"),
		env: []string{
			"PATH=" + os.Getenv("PATH"), "HOME=" + root, "TMPDIR=" + root,
			"XDG_CONFIG_HOME=" + root, "GIT_CONFIG_NOSYSTEM=1",
			"REF_NAME=" + fixupBranch,
		},
	}
	r.git(t, root, "init", "-q", "--bare", r.origin)
	r.git(t, root, "init", "-q", r.dir)
	r.git(t, r.dir, "checkout", "-q", "-b", fixupBranch)
	// The repository's own normalisation rule: LF in the index.
	writeTestFile(t, filepath.Join(r.dir, ".gitattributes"), "* text=auto\n")
	writeTestFile(t, filepath.Join(r.dir, "vendor", "LICENSE"), "line one\nline two\n")
	r.git(t, r.dir, "add", "-A")
	r.git(t, r.dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "bump")
	r.git(t, r.dir, "remote", "add", "origin", r.origin)
	r.git(t, r.dir, "push", "-q", "origin", "HEAD:refs/heads/"+fixupBranch)
	return r
}

func (r *fixupRepo) git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// runStep runs script the way GitHub Actions runs a bash `run:` block.
func (r *fixupRepo) runStep(t *testing.T, script string) (string, error) {
	t.Helper()
	path := filepath.Join(filepath.Dir(r.dir), "step.sh")
	writeTestFile(t, path, script)
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", path)
	cmd.Dir = r.dir
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDependabotFixupCommitStep runs the fixup's commit step against a
// scratch repository. A vendored file that is CRLF upstream shows as
// modified in the working tree but stages to nothing, which is what most
// gomod bumps look like after `go mod vendor`: the step must treat that as
// nothing to do instead of failing `git commit`, and must still commit and
// push a real change.
func TestDependabotFixupCommitStep(t *testing.T) {
	t.Parallel()

	script := fixupCommitStep(t)

	tests := []struct {
		name       string
		change     func(t *testing.T, dir string)
		wantCommit bool
	}{
		{
			name:   "clean tree",
			change: func(*testing.T, string) {},
		},
		{
			name: "line-ending-only change",
			change: func(t *testing.T, dir string) {
				t.Helper()
				writeTestFile(t, filepath.Join(dir, "vendor", "LICENSE"), "line one\r\nline two\r\n")
			},
		},
		{
			name: "real change",
			change: func(t *testing.T, dir string) {
				t.Helper()
				writeTestFile(t, filepath.Join(dir, "vendor", "modules.txt"), "# example.com/mod v1.2.3\n")
			},
			wantCommit: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newFixupRepo(t)
			before := r.git(t, r.dir, "rev-parse", "HEAD")
			tc.change(t, r.dir)

			out, err := r.runStep(t, script)
			if err != nil {
				t.Fatalf("step failed: %v\n%s", err, out)
			}
			head := r.git(t, r.dir, "rev-parse", "HEAD")
			pushed := r.git(t, r.origin, "rev-parse", "refs/heads/"+fixupBranch)
			if committed := head != before; committed != tc.wantCommit {
				t.Errorf("committed = %v, want %v\n%s", committed, tc.wantCommit, out)
			}
			if pushed != head {
				t.Errorf("origin %s = %s, want local HEAD %s", fixupBranch, pushed, head)
			}

			// A re-run on the step's own result must be a no-op.
			out, err = r.runStep(t, script)
			if err != nil {
				t.Fatalf("re-run failed: %v\n%s", err, out)
			}
			if again := r.git(t, r.dir, "rev-parse", "HEAD"); again != head {
				t.Errorf("re-run committed again: HEAD %s -> %s\n%s", head, again, out)
			}
		})
	}
}
