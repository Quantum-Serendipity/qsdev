package devenv

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// nixSecretsCheckDef returns the catalog definition of the always-on
// nix-secrets-check hook.
func nixSecretsCheckDef(t *testing.T) catalog.CustomHookDef {
	t.Helper()
	for _, def := range catalog.MustDefault().CustomHooks() {
		if def.ID == "nix-secrets-check" {
			return def
		}
	}
	t.Fatal("catalog has no nix-secrets-check hook")
	return catalog.CustomHookDef{}
}

func TestBuildNixSecretsCheckEntry_PatternIsNixEscaped(t *testing.T) {
	t.Parallel()
	def := nixSecretsCheckDef(t)
	if !strings.Contains(def.EnvPattern, `\`) {
		t.Skip("catalog env pattern has no backslashes; nothing to escape")
	}
	entry := buildNixSecretsCheckEntry(def)
	want := "envPattern = " + nixStr(def.EnvPattern) + ";"
	if !strings.Contains(entry, want) {
		t.Errorf("entry does not bind the Nix-escaped pattern %q:\n%s", want, entry)
	}
}

// TestBuildNixSecretsCheckEntry_PatternsEvaluateToCatalogRegexes evaluates the
// hook's let-bindings with Nix and checks the grep patterns the hook really
// runs are the catalog regexes, and that they catch a dotted env secret.
func TestBuildNixSecretsCheckEntry_PatternsEvaluateToCatalogRegexes(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	def := nixSecretsCheckDef(t)
	entry := buildNixSecretsCheckEntry(def)
	bindings, _, ok := strings.Cut(entry, "\n        in\n")
	if !ok {
		t.Fatalf("unexpected entry shape:\n%s", entry)
	}
	expr := bindings + "\nin { inherit envPattern credPattern; }"

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, nixInstantiate, "--eval", "--strict", "--json", "--expr", expr).Output()
	if err != nil {
		t.Fatalf("evaluating hook patterns: %v\n%s", err, expr)
	}
	var got struct {
		EnvPattern  string `json:"envPattern"`
		CredPattern string `json:"credPattern"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decoding %s: %v", out, err)
	}
	if got.EnvPattern != def.EnvPattern {
		t.Errorf("envPattern evaluates to %q, want catalog regex %q", got.EnvPattern, def.EnvPattern)
	}
	wantCred := "(" + strings.Join(def.CredentialPatterns, "|") + ")"
	if got.CredPattern != wantCred {
		t.Errorf("credPattern evaluates to %q, want %q", got.CredPattern, wantCred)
	}

	re := regexp.MustCompile(got.EnvPattern)
	for _, line := range []string{`env.AWS_SECRET_ACCESS_KEY = "x";`, `  env.STRIPE_API_KEY="x";`} {
		if !re.MatchString(line) {
			t.Errorf("evaluated env pattern misses %q", line)
		}
	}
	if re.MatchString(`env.GOFLAGS = "-mod=readonly";`) {
		t.Error("evaluated env pattern flags a non-secret env var")
	}
}

// runBash runs script with bash in dir with a minimal environment and returns
// its combined output.
func runBash(t *testing.T, dir, script string, env ...string) string {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	cmd := exec.Command(bash, "-c", script)
	cmd.Dir = dir
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir}, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash failed: %v\n%s", err, out)
	}
	return string(out)
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func TestMCPSecretsShell(t *testing.T) {
	t.Parallel()
	const report = `
printf 'GITHUB_TOKEN=[%s]\n' "${GITHUB_TOKEN:-}"
printf 'FOO_KEY=[%s]\n' "${FOO_KEY:-}"
printf 'BAR=[%s]\n' "${BAR:-}"
printf 'OTHER=[%s]\n' "${OTHER:-}"
printf 'LD_PRELOAD=[%s]\n' "${LD_PRELOAD:-}"
printf 'PATH_HIJACKED=[%s]\n' "$([ "$PATH" = /evil ] && echo yes)"
`
	secrets := strings.Join([]string{
		`FOO_KEY="hello world"`,
		`BAR=$(touch pwned)`,
		`PATH=/evil`,
		`LD_PRELOAD=/tmp/x.so`,
		`export OTHER=1`,
		`not a key value line`,
		`# comment`,
		``,
	}, "\n")

	tests := []struct {
		name    string
		mcpJSON string // "" means no .mcp.json
		want    []string
	}{
		{
			name:    "no mcp.json injects nothing",
			mcpJSON: "",
			want:    []string{"GITHUB_TOKEN=[]", "FOO_KEY=[]", "BAR=[]", "OTHER=[]"},
		},
		{
			name:    "only referenced names are injected, as data",
			mcpJSON: `{"mcpServers":{"x":{"env":{"A":"${FOO_KEY}","B":"${BAR:-d}","C":"${GITHUB_TOKEN}","D":"${PATH}","E":"${LD_PRELOAD}"}}}}`,
			want: []string{
				"GITHUB_TOKEN=[gh-test-token]",
				"FOO_KEY=[hello world]",
				"BAR=[$(touch pwned)]",
				"OTHER=[]",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, ".qsdev", "mcp-secrets.env"), secrets, 0o600)
			if tt.mcpJSON != "" {
				writeTestFile(t, filepath.Join(dir, ".mcp.json"), tt.mcpJSON, 0o644)
			}
			// Fake gh: authenticated, returns a fixed token.
			bin := filepath.Join(dir, "bin")
			writeTestFile(t, filepath.Join(bin, "gh"),
				"#!/bin/sh\n[ \"$1\" = auth ] && [ \"$2\" = token ] && echo gh-test-token\nexit 0\n", 0o755)

			out := runBash(t, dir, `PATH="`+bin+`:$PATH"`+"\n"+mcpSecretsShell+report)
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			// Values are never executed and loader/PATH variables never set,
			// even when .mcp.json references them.
			for _, bad := range []string{"PATH_HIJACKED=[yes]", "LD_PRELOAD=[/tmp/x.so]"} {
				if strings.Contains(out, bad) {
					t.Errorf("output contains %q:\n%s", bad, out)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
				t.Error("mcp-secrets.env value was executed as shell code")
			}
		})
	}
}

func TestHooksStateShell(t *testing.T) {
	t.Parallel()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	gitRun := func(t *testing.T, dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(git, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	state := func(t *testing.T, dir string) string {
		t.Helper()
		return strings.TrimSpace(runBash(t, dir, hooksStateShell+"\necho \"$hooks_state\"",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir)))
	}

	root := t.TempDir()
	plain := filepath.Join(root, "plain")
	repo := filepath.Join(root, "repo")
	worktree := filepath.Join(root, "wt")
	for _, d := range []string{plain, repo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, repo, "init", "-q")
	gitRun(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
	gitRun(t, repo, "worktree", "add", "-q", worktree)

	if got := state(t, plain); got != "unknown" {
		t.Errorf("outside a repository: hooks_state = %q, want unknown", got)
	}
	if got := state(t, repo); got != "missing" {
		t.Errorf("repo without hooks: hooks_state = %q, want missing", got)
	}
	// A linked worktree has a .git file, not a directory; it must not be
	// reported as active when no hook is installed.
	if got := state(t, worktree); got != "missing" {
		t.Errorf("worktree without hooks: hooks_state = %q, want missing", got)
	}

	writeTestFile(t, filepath.Join(repo, ".git", "hooks", "pre-commit"), "#!/bin/sh\n", 0o755)
	for _, dir := range []string{repo, worktree} {
		if got := state(t, dir); got != "active" {
			t.Errorf("%s with hook installed: hooks_state = %q, want active", filepath.Base(dir), got)
		}
	}
}
