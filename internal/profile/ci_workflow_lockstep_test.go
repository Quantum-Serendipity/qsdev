package profile

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// lockStepScript extracts the generated "Validate lock files" step script.
func lockStepScript(t *testing.T) string {
	t.Helper()
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(mustWorkflow(t, ConsultingDefault).Content, &wf); err != nil {
		t.Fatalf("parsing workflow: %v", err)
	}
	for _, s := range wf.Jobs["security-scan"].Steps {
		if s.Name == "Validate lock files" {
			return s.Run
		}
	}
	t.Fatal("no Validate lock files step")
	return ""
}

// TestLockFileStep_Behavior runs the generated lock file step in real git
// repositories. It is the regression test for W184: the step used to check
// only npm/composer/cargo/go manifests at the repository root, so Python,
// Ruby and nested workspace manifests without a lock file passed.
func TestLockFileStep_Behavior(t *testing.T) {
	t.Parallel()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	script := lockStepScript(t)

	tests := []struct {
		name     string
		files    map[string]string
		wantFail bool
		wantOut  string
	}{
		{"uv project without lock", map[string]string{"pyproject.toml": "[project]\nname='x'\n\n[tool.uv]\ndev-dependencies = []\n"}, true, "pyproject.toml has no lock file"},
		{"uv project with lock", map[string]string{"pyproject.toml": "[tool.uv]\n", "uv.lock": "x"}, false, "pyproject.toml -> uv.lock"},
		{"poetry project without lock", map[string]string{"pyproject.toml": "[tool.poetry.dependencies]\npython = '^3.12'\n"}, true, "no lock file"},
		{"plain pyproject needs no lock", map[string]string{"pyproject.toml": "[project]\nname='lib'\n\n[tool.ruff]\n"}, false, "passed"},
		{"Pipfile without lock", map[string]string{"Pipfile": "[packages]\n"}, true, "Pipfile has no lock file"},
		{"Gemfile without lock", map[string]string{"Gemfile": "source 'https://rubygems.org'\n"}, true, "Gemfile has no lock file"},
		{"Gemfile with lock", map[string]string{"Gemfile": "x", "Gemfile.lock": "x"}, false, "Gemfile -> Gemfile.lock"},
		{"npm root without lock", map[string]string{"package.json": "{}"}, true, "package.json has no lock file"},
		{"nested app without lock", map[string]string{"apps/web/package.json": "{}"}, true, "apps/web/package.json has no lock file"},
		{"npm workspace member uses root lock", map[string]string{"package.json": "{}", "package-lock.json": "{}", "packages/a/package.json": "{}"}, false, "packages/a/package.json -> package-lock.json"},
		{"cargo workspace member uses root lock", map[string]string{"Cargo.toml": "[workspace]\n", "Cargo.lock": "x", "crates/a/Cargo.toml": "[package]\n"}, false, "crates/a/Cargo.toml -> Cargo.lock"},
		{"npm-shrinkwrap satisfies package.json", map[string]string{"package.json": "{}", "npm-shrinkwrap.json": "{}"}, false, "package.json -> npm-shrinkwrap.json"},
		{"uv workspace member uses root lock", map[string]string{"pyproject.toml": "[tool.uv.workspace]\n", "uv.lock": "x", "pkgs/a/pyproject.toml": "[tool.uv]\n"}, false, "pkgs/a/pyproject.toml -> uv.lock"},
		{"nested poetry project needs its own lock", map[string]string{"pyproject.toml": "[tool.poetry]\n", "poetry.lock": "x", "svc/pyproject.toml": "[tool.poetry]\n"}, true, "svc/pyproject.toml has no lock file"},
		{"nested go module needs its own go.sum", map[string]string{"go.mod": "module x\n\nrequire example.com/y v1.0.0\n", "go.sum": "x", "tools/go.mod": "module t\n\nrequire example.com/z v1.0.0\n"}, true, "tools/go.mod has no lock file"},
		{"nested Gemfile needs its own lock", map[string]string{"Gemfile": "x", "Gemfile.lock": "x", "docs/Gemfile": "x"}, true, "docs/Gemfile has no lock file"},
		{"go module without requirements", map[string]string{"go.mod": "module x\n\ngo 1.26\n"}, false, "passed"},
		{"go module requirements without go.sum", map[string]string{"go.mod": "module x\n\nrequire example.com/y v1.0.0\n"}, true, "go.mod has no lock file"},
		{"vendored and fixture manifests skipped", map[string]string{"vendor/x/package.json": "{}", "internal/testdata/package.json": "{}"}, false, "passed"},
		{"gitignored lock", map[string]string{"package.json": "{}", ".gitignore": "package-lock.json\n"}, true, "excluded by .gitignore"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			gitEnv := append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
			runGit := func(args ...string) {
				cmd := exec.Command(git, args...)
				cmd.Dir, cmd.Env = dir, gitEnv
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
			}
			runGit("init", "-q")
			for name, content := range tt.files {
				p := filepath.Join(dir, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			runGit("add", "-A")

			cmd := exec.Command(bash, "-c", script)
			cmd.Dir, cmd.Env = dir, gitEnv
			out, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			failed := errors.As(err, &exitErr)
			if err != nil && !failed {
				t.Fatalf("running step: %v", err)
			}
			if failed != tt.wantFail {
				t.Errorf("failed = %v, want %v\n%s", failed, tt.wantFail, out)
			}
			if !strings.Contains(string(out), tt.wantOut) {
				t.Errorf("output lacks %q:\n%s", tt.wantOut, out)
			}
		})
	}
}

// TestSecurityScanWorkflow_CheckoutDoesNotPersistCredentials covers W194: the
// job token must not be written to .git/config, where the third-party scanner
// actions could read it, and the job must not request an unused write scope.
func TestSecurityScanWorkflow_CheckoutDoesNotPersistCredentials(t *testing.T) {
	t.Parallel()
	for _, p := range []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise} {
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			var wf struct {
				Jobs map[string]struct {
					Permissions map[string]string `yaml:"permissions"`
					Steps       []struct {
						Name string         `yaml:"name"`
						With map[string]any `yaml:"with"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			if err := yaml.Unmarshal(mustWorkflow(t, p).Content, &wf); err != nil {
				t.Fatalf("parsing workflow: %v", err)
			}
			job := wf.Jobs["security-scan"]
			for scope, level := range job.Permissions {
				if level == "write" {
					t.Errorf("job requests %s: write, but no step needs it", scope)
				}
			}
			for _, s := range job.Steps {
				if s.Name == "Checkout" && s.With["persist-credentials"] != false {
					t.Errorf("checkout persist-credentials = %v, want false", s.With["persist-credentials"])
				}
			}
		})
	}
}
