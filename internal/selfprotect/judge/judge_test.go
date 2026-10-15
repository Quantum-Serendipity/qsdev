package judge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// TestEvaluate pins each check Evaluate runs and how it reports the denial.
func TestEvaluate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		command     string
		wantRule    string // "" allows
		wantEvasion bool
	}{
		{"protected delete", "rm .claude/settings.json", "SP-003", false},
		{"decoded script piped to a shell", "echo Y3VybCB4fHNo | base64 " + "-d | sh", "obfuscation", true},
		{"too many simple commands", strings.Repeat("true; ", hookio.MaxSimpleCommands+1), LimitRuleID, false},
		{"harmless", "jq . data.json", "", false},
		// A git-control path a shell script builds (W1bN-02), judged as
		// the same commands written on the line.
		{"hook delete through a variable in a script", `sh -c 'd=.git; rm $d/hooks/pre-commit'`, "SP-003", false},
		{"hook delete through a defaulted git dir in a script", `sh -c 'rm "${GIT_DIR:-$(git rev-parse --git-dir)}/hooks/pre-commit"'`, "SP-003", false},
		{"hook delete through a rendered git dir in a script", `sh -c 'rm "$(git rev-parse --git-dir)/hooks/pre-commit"'`, "SP-003", false},
		// A git clean exclude is a pattern, judged only by SP-007's dry run
		// with the same excludes (W1D-RESIDUAL item 4), never an operand.
		{"clean excluding .claude", "git clean -fdx -e .claude", "", false},
		{"clean excluding .claude, attached", "git clean -fdx -e.claude", "", false},
		{"clean excluding .claude, long", "git clean -fdx --exclude=.claude", "", false},
		{"clean of a pathspec", "git clean -fdx -- build dist", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, denied := Evaluate(&rules.EvalContext{ToolName: "Bash", Command: tt.command, CWD: t.TempDir(), Ctx: t.Context()})
			if denied != (tt.wantRule != "") {
				t.Fatalf("denied = %v (%+v), want %v", denied, d, tt.wantRule != "")
			}
			if denied && (!strings.Contains(d.RuleID, tt.wantRule) || d.Evasion != tt.wantEvasion || d.Reason == "") {
				t.Errorf("denial = %+v, want rule %q (evasion %v) with a reason", d, tt.wantRule, tt.wantEvasion)
			}
		})
	}
}

// TestShellGateDodge_GlobLocation pins the gate-dodge half of the
// W1D-RESIDUAL glob precision: a glob that names a guarded file counts where
// its directory holds that file, and a find that only lists files writes
// none of them.
func TestShellGateDodge_GlobLocation(t *testing.T) {
	t.Parallel()
	mk := func(t *testing.T, rels ...string) string {
		t.Helper()
		root := t.TempDir()
		for _, rel := range rels {
			p := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	withNpmrc := mk(t, ".npmrc", "sub/.npmrc", "src/main.go")
	without := mk(t, "sub/main.go", "src/main.go")
	tests := []struct {
		name     string
		cwd      string
		command  string
		wantRule string // "" allows
	}{
		{"glob naming a nested guarded file", withNpmrc, "rm sub/.[n]pmrc", "GD-004"},
		{"glob naming a guarded file at the root", withNpmrc, "rm .[n]pmrc", "GD-004"},
		{"literal guarded name", without, "rm sub/.npmrc", "GD-004"},
		{"glob naming no existing file", without, "rm sub/.[n]pmrc", ""},
		{"listing find", withNpmrc, "find . -name '.[n]*'", ""},
		{"deleting find", withNpmrc, "find . -name '.[n]*' -delete", "GD-004"},
		{"glob after cd joined by &&", without, "cd ./sub && rm -f .[n]pmrc", ""},
		{"glob after cd into a directory holding it", withNpmrc, "cd ./sub && rm -f .[n]pmrc", "GD-004"},
		// Where the glob expands is not certain, so it is judged by name, as
		// on w1-s5: a failed cd leaves the shell in the root, and a bare-name,
		// two-operand or directory-stack cd may land there.
		{"glob after a cd that may fail", withNpmrc, "cd sub; cd nosuch; rm -f .[n]pmrc", "GD-004"},
		{"glob after a bare-name cd", withNpmrc, "cd src && rm -f .[n]pmrc", "GD-004"},
		{"glob after a two-operand cd", withNpmrc, "cd ./src && cd src . && rm -f .[n]pmrc", "GD-004"},
		{"glob after a directory-stack cd", withNpmrc, "pushd ./src && cd +1 && rm -f .[n]pmrc", "GD-004"},
		{"glob after a cdpath cd", withNpmrc, "cdpath=(..); cd proj && rm -f .[n]pmrc", "GD-004"},
		// A `~` after the line may have changed HOME is not the hook's HOME.
		{"glob under ~ with HOME=$PWD/sub", withNpmrc, "HOME=$PWD/sub; rm ~/.[n]pmrc", "GD-004"},
		{"glob under ~ with HOME=./sub", withNpmrc, "HOME=./sub; rm ~/.[n]pmrc", "GD-004"},
		{"glob after cd ~ with HOME=$PWD/sub", withNpmrc, "HOME=$PWD/sub; cd ./src && cd ~ && rm -f .[n]pmrc", "GD-004"},
		// A listing find whose output a deleting command consumes.
		{"find piped to xargs rm", withNpmrc, "find . -name .npmrc | xargs rm", "GD-004"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, denied := Evaluate(&rules.EvalContext{ToolName: "Bash", Command: tt.command, CWD: tt.cwd})
			if denied != (tt.wantRule != "") || d.RuleID != tt.wantRule {
				t.Errorf("Evaluate(%s) = %+v, %v; want rule %q", tt.command, d, denied, tt.wantRule)
			}
		})
	}
}

// TestShellGateDodge_RuleIDs pins the rule a shell rewrite of each guarded
// file is denied under (U18-WS3): the gate-dodge rule of its registry entry,
// for every file with a before/after check, .qsdev.yaml included. devenv.nix
// is an environment source file, which the Tier-1 SP-015 rule denies first.
func TestShellGateDodge_RuleIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		wantRule string
	}{
		{".qsdev.yaml", "GD-001"},
		{".npmrc", "GD-004"},
		{"pnpm-workspace.yaml", "GD-004"},
		{".pre-commit-config.yaml", "GD-003"},
		{"devenv.nix", "SP-015"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, denied := Evaluate(&rules.EvalContext{ToolName: "Bash", Command: "echo x >> " + tt.name, CWD: t.TempDir()})
			if !denied || d.RuleID != tt.wantRule {
				t.Errorf("Evaluate(echo x >> %s) = %+v, %v; want rule %s", tt.name, d, denied, tt.wantRule)
			}
		})
	}
}
