package rules

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestEnvFileRelocationVerdicts covers U18-WS1 round 2: a file that sets the
// environment the CLI runs in (devenv.local.nix, .envrc.local, a shell
// startup file) can set or drop the org-config variable for every later
// regeneration, so a Write or Edit that changes such an assignment is denied,
// and a shell rewrite of the file, whose content the hook cannot check, is
// denied outright. The Write and Bash paths agree.
func TestEnvFileRelocationVerdicts(t *testing.T) {
	t.Parallel()
	b := branding.Get()
	orgEnv := b.EnvPrefix + "ORG_CONFIG"
	dir := t.TempDir()
	home := homeDir(t)

	existing := filepath.Join(dir, "devenv.local.nix")
	setLine := "  env." + orgEnv + " = \"/etc/org/defaults.yaml\";\n"
	if err := os.WriteFile(existing, []byte("{\n"+setLine+"  env.API = \"x\";\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, "plain", ".bashrc")
	if err := os.MkdirAll(filepath.Dir(plain), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte("alias ll='ls -l'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	write := func(p, content string) EvalContext {
		return EvalContext{ToolName: "Write", FilePath: p, CanonicalPath: p, CWD: dir, Content: content}
	}
	edit := func(p, old, repl string) EvalContext {
		return EvalContext{
			ToolName: "Edit", FilePath: p, CanonicalPath: p, CWD: dir,
			Edits: []TextEdit{{OldString: old, NewString: repl}},
		}
	}
	bash := func(cmd string) EvalContext {
		return EvalContext{ToolName: "Bash", Command: cmd, CWD: dir}
	}

	tests := []struct {
		name string
		ctx  EvalContext
		want Verdict
	}{
		{"Write bashrc export", write(filepath.Join(home, ".bashrc"), "export "+orgEnv+"=/tmp/evil.yaml\n"), Deny},
		{"Write envrc.local export", write(filepath.Join(dir, ".envrc.local"), "export "+orgEnv+"=/tmp/evil.yaml\n"), Deny},
		{"Write zshenv export", write(filepath.Join(home, ".zshenv"), "export "+orgEnv+"=/tmp/evil.yaml\n"), Deny},
		{"Write fish config", write(filepath.Join(home, ".config", "fish", "config.fish"), "set -gx "+orgEnv+" /tmp/evil.yaml\n"), Deny},
		{"Write dotenv", write(filepath.Join(dir, ".env"), orgEnv+"=/tmp/evil.yaml\n"), Deny},
		{"Write devenv.nix env", write(filepath.Join(dir, "devenv.nix"), "{ env."+orgEnv+" = \"/tmp/e\"; }\n"), Deny},
		{"Write upper-case file name", write(filepath.Join(dir, "DEVENV.LOCAL.NIX"), "{ env."+orgEnv+" = \"/tmp/e\"; }\n"), Deny},
		{"Write split nix name", write(filepath.Join(dir, "devenv.local.nix"), "{ env.\"${\""+orgEnv[:6]+"\" + \""+orgEnv[6:]+"\"}\" = \"/tmp/e\"; }\n"), Deny},
		{"Write split shell name", write(filepath.Join(dir, ".envrc.local"), "export "+orgEnv[:6]+"\\\n"+orgEnv[6:]+"=/tmp/e\n"), Deny},
		{"Edit changes org config value", edit(existing, "/etc/org/defaults.yaml", "/tmp/evil.yaml"), Deny},
		{"Edit removes org config line", edit(existing, setLine, ""), Deny},
		{"Edit other value in file setting org config", edit(existing, "env.API = \"x\"", "env.API = \"y\""), Deny},
		{"Edit adds org config", edit(plain, "alias ll='ls -l'\n", "alias ll='ls -l'\nexport "+orgEnv+"=/tmp/e\n"), Deny},
		{"Edit without replacements", EvalContext{ToolName: "Edit", FilePath: plain, CanonicalPath: plain, CWD: dir}, Deny},
		{"append to bashrc", bash("echo 'export " + orgEnv + "=/tmp/e' >> ~/.bashrc"), Deny},
		{"append split name to bashrc", bash("printf 'export " + orgEnv[:6] + "%s=/tmp/e' " + orgEnv[6:] + " >> ~/.bashrc"), Deny},
		{"copy over envrc.local", bash("cp /tmp/x .envrc.local"), Deny},
		{"tee devenv.local.nix", bash("echo x | tee devenv.local.nix"), Deny},

		{"Write bashrc alias", write(filepath.Join(home, ".bashrc"), "alias ll='ls -l'\n"), Allow},
		{"Edit plain rc file", edit(plain, "ls -l", "ls -la"), Allow},
		{"Write other file mentioning org config", write(filepath.Join(dir, "README.md"), "Set "+orgEnv+" to use another overlay.\n"), Allow},
		{"Write unchanged file setting org config", write(existing, "{\n"+setLine+"  env.API = \"x\";\n}\n"), Allow},
		{"cat bashrc", bash("cat ~/.bashrc"), Allow},
		{"grep envrc.local", bash("grep -n PATH .envrc.local"), Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.ctx
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.want {
				t.Errorf("verdict = %v (first: %s), want %v", got, firstRuleID(matches), tt.want)
			}
		})
	}
}

// TestTildeUserAncestorVerdicts covers the ~name tilde form: bash expands it
// to the home directory the user database records for name, so deleting or
// moving a directory above the overlay through it is denied like through ~.
// A ~name the database does not know, or a directory-stack form, may still be
// a home directory and fails closed on the overlay's ancestor chain.
func TestTildeUserAncestorVerdicts(t *testing.T) {
	t.Parallel()
	b := branding.Get()
	bash := func(cmd string) EvalContext {
		return EvalContext{ToolName: "Bash", Command: cmd, CWD: t.TempDir()}
	}
	const unknown = "~qsdev-no-such-account-4242"
	tests := []struct {
		name string
		ctx  EvalContext
		want Verdict
	}{
		{"unknown account overlay parent", bash("rm -rf " + unknown + "/.config"), Deny},
		{"unknown account overlay dir", bash("rm -rf " + unknown + "/.config/" + b.AppName), Deny},
		{"cd unknown account config then remove", bash("cd " + unknown + "/.config && rm -rf " + b.AppName), Deny},
		{"find delete unknown account config", bash("find " + unknown + "/.config -delete"), Deny},
		{"dirstack overlay parent", bash("rm -rf ~+/.config"), Deny},
		{"unknown account other dir", bash("rm -rf " + unknown + "/.cache"), Allow},
		{"cd unknown account other dir", bash("cd " + unknown + "/.cache && rm -rf x"), Allow},
	}
	// The account running the tests: its home directory is where the CLI
	// reads the overlay, and is protected even though TestMain points HOME
	// elsewhere. Checked only where ~name resolves it.
	if u, ok := resolvableAccount(); ok {
		acct := "~" + u.Username
		tests = append(tests, []struct {
			name string
			ctx  EvalContext
			want Verdict
		}{
			{"account overlay parent", bash("rm -rf " + acct + "/.config"), Deny},
			{"account home", bash("rm -rf " + acct), Deny},
			{"move account overlay parent", bash("mv " + acct + "/.config /tmp/x"), Deny},
			{"cd account config then remove", bash("cd " + acct + "/.config && rm -rf " + b.AppName), Deny},
			{"account other dir", bash("rm -rf " + acct + "/.cache/zz-unrelated"), Allow},
		}...)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.ctx
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.want {
				t.Errorf("verdict = %v (first: %s), want %v", got, firstRuleID(matches), tt.want)
			}
		})
	}
}

// resolvableAccount returns the account running the tests when its name can
// follow ~ as a plain shell word (a Windows DOMAIN\user name cannot) and
// ~name resolves to the account's home directory.
func resolvableAccount() (*user.User, bool) {
	u, err := user.Current()
	if err != nil || u.Username == "" ||
		strings.Trim(strings.ToLower(u.Username), "abcdefghijklmnopqrstuvwxyz0123456789._-") != "" {
		return nil, false
	}
	account, err := userhome.Account()
	if err != nil {
		return nil, false
	}
	named, err := userhome.Named(u.Username)
	return u, err == nil && named == account
}
