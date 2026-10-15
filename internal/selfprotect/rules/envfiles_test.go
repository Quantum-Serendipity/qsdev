package rules

import (
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestEnvFileRelocationVerdicts covers U18-WS1 round 2: a file that sets the
// environment the CLI runs in (devenv.local.nix, devenv.local.yaml, a shell
// startup file) can set or drop the org-config variable for every later
// regeneration, so a Write or Edit that changes such an assignment is denied,
// and a shell write of the file, whose content the hook cannot check, is
// denied outright. The Write and Bash paths agree. Round 3: only files the
// generated environment or the shell loads count (.env and .envrc.local are
// not loaded), and a command that only reads one (source, sed -n, the source
// of ln -s) is not a write; a write through a symlink still is.
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

	link := filepath.Join(dir, "notes.txt")
	if err := os.Symlink(plain, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
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
		{"Write devenv.local.yaml env import", write(filepath.Join(dir, "devenv.local.yaml"), "# "+orgEnv+"\nimports: [./x.nix]\n"), Deny},
		{"Write zshenv export", write(filepath.Join(home, ".zshenv"), "export "+orgEnv+"=/tmp/evil.yaml\n"), Deny},
		{"Write fish config", write(filepath.Join(home, ".config", "fish", "config.fish"), "set -gx "+orgEnv+" /tmp/evil.yaml\n"), Deny},
		{"Write devenv.nix env", write(filepath.Join(dir, "devenv.nix"), "{ env."+orgEnv+" = \"/tmp/e\"; }\n"), Deny},
		{"Write upper-case file name", write(filepath.Join(dir, "DEVENV.LOCAL.NIX"), "{ env."+orgEnv+" = \"/tmp/e\"; }\n"), Deny},
		{"Write split nix name", write(filepath.Join(dir, "devenv.local.nix"), "{ env.\"${\""+orgEnv[:6]+"\" + \""+orgEnv[6:]+"\"}\" = \"/tmp/e\"; }\n"), Deny},
		{"Write split shell name", write(filepath.Join(home, ".zshrc"), "export "+orgEnv[:6]+"\\\n"+orgEnv[6:]+"=/tmp/e\n"), Deny},
		{"Edit changes org config value", edit(existing, "/etc/org/defaults.yaml", "/tmp/evil.yaml"), Deny},
		{"Edit removes org config line", edit(existing, setLine, ""), Deny},
		{"Edit other value in file setting org config", edit(existing, "env.API = \"x\"", "env.API = \"y\""), Deny},
		{"Edit adds org config", edit(plain, "alias ll='ls -l'\n", "alias ll='ls -l'\nexport "+orgEnv+"=/tmp/e\n"), Deny},
		{"Edit without replacements", EvalContext{ToolName: "Edit", FilePath: plain, CanonicalPath: plain, CWD: dir}, Deny},
		{"append to bashrc", bash("echo 'export " + orgEnv + "=/tmp/e' >> ~/.bashrc"), Deny},
		{"append split name to bashrc", bash("printf 'export " + orgEnv[:6] + "%s=/tmp/e' " + orgEnv[6:] + " >> ~/.bashrc"), Deny},
		{"copy over devenv.local.nix", bash("cp /tmp/x devenv.local.nix"), Deny},
		{"tee devenv.local.nix", bash("echo x | tee devenv.local.nix"), Deny},
		{"sed in place on bashrc", bash("sed -i s/a/b/ ~/.bashrc"), Deny},
		{"sed clustered in place on bashrc", bash("sed -ni p ~/.bashrc"), Deny},
		{"sed long in place on bashrc", bash("sed --in-place s/a/b/ ~/.bashrc"), Deny},
		{"sed w command to bashrc", bash("sed -n 'w ~/.bashrc' /tmp/x"), Deny},
		{"ln over bashrc", bash("ln -sf /tmp/x ~/.bashrc"), Deny},
		{"ln into home directory", bash("ln -s /tmp/x/.bashrc ~/"), Deny},
		{"ln -t into home directory", bash("ln -st ~ /tmp/x/.bashrc"), Deny},
		{"append through symlink to rc file", bash("echo 'export X=1' >> notes.txt"), Deny},
		// A link into a directory an earlier command of the line creates
		// lands under the source's name, though the directory does not exist
		// when the hook runs.
		{"ln into directory mkdir creates", bash("mkdir -p fishcfg && ln -s /tmp/x/config.fish fishcfg"), Deny},
		{"ln into home directory mkdir creates", bash("mkdir -p ~/.config/qsdev-no-such-dir-4242/fish && ln -s /tmp/x/config.fish ~/.config/qsdev-no-such-dir-4242/fish"), Deny},
		{"ln into directory a sourced script may create", bash("source ./mk.sh; ln -s /tmp/x/.bashrc rcdir"), Deny},
		{"source process substitution naming bashrc", bash("source <(echo 'echo x >> ~/.bashrc')"), Deny},

		{"Write bashrc alias", write(filepath.Join(home, ".bashrc"), "alias ll='ls -l'\n"), Allow},
		{"Edit plain rc file", edit(plain, "ls -l", "ls -la"), Allow},
		{"Write other file mentioning org config", write(filepath.Join(dir, "README.md"), "Set "+orgEnv+" to use another overlay.\n"), Allow},
		{"Write unchanged file setting org config", write(existing, "{\n"+setLine+"  env.API = \"x\";\n}\n"), Allow},
		{"cat bashrc", bash("cat ~/.bashrc"), Allow},
		{"grep envrc.local", bash("grep -n PATH .envrc.local"), Allow},
		// Round 3: files the generated environment does not load.
		{"Write dotenv", write(filepath.Join(dir, ".env"), orgEnv+"=/tmp/evil.yaml\n"), Allow},
		{"Write envrc.local", write(filepath.Join(dir, ".envrc.local"), "export "+orgEnv+"=/tmp/evil.yaml\n"), Allow},
		{"copy dotenv example", bash("cp .env.example .env"), Allow},
		{"append to dotenv", bash("echo DATABASE_URL=postgres://x >> .env"), Allow},
		{"remove dotenv", bash("rm .env"), Allow},
		// Round 3: reads of a loaded file are not writes.
		{"source bashrc", bash("source ~/.bashrc"), Allow},
		{"dot bashrc", bash(". ~/.bashrc"), Allow},
		{"sed -n bashrc", bash("sed -n 1p ~/.bashrc"), Allow},
		{"sed -n expression bashrc", bash("sed -n -e 1,5p ~/.bashrc"), Allow},
		{"link to bashrc", bash("ln -s ~/.bashrc rc-link.txt"), Allow},
		{"link to bashrc after a read", bash("cat ~/.bashrc && ln -s ~/.bashrc rc-link.txt"), Allow},
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

// TestShellRewritesFile_GlobLocation pins the W1D-RESIDUAL precision of the
// SP-015 shell-write check for a glob: the shell expands it on the entries
// that exist in its directory, and one that matches nothing stays literal, so
// a glob counts only where its directory holds a guarded file. A find that
// only lists files writes none of its operands, while one with a mutating
// action keeps matching its name patterns anywhere below its start points.
// A glob whose directory is itself a glob or an expansion, or cannot be
// resolved, or follows a command that may create files, is judged by name
// alone (fail closed).
func TestShellRewritesFile_GlobLocation(t *testing.T) {
	t.Parallel()
	mkFiles := func(t *testing.T, root string, rels ...string) {
		t.Helper()
		for _, rel := range rels {
			p := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	// guarded holds devenv.yaml at its root; bare holds no guarded file.
	guarded, bare := t.TempDir(), t.TempDir()
	mkFiles(t, guarded, "devenv.yaml", "src/main.go", "dist/app.js")
	mkFiles(t, bare, "README.md", "src/main.go", "dist/app.js", "nested/src/devenv.yaml")
	if _, err := os.Stat(filepath.Join(homeDir(t), ".bashrc")); err != nil {
		t.Fatalf("TestMain's HOME holds no .bashrc: %v", err)
	}

	tests := []struct {
		name string
		cwd  string
		cmd  string
		want Verdict
	}{
		// The glob's directory holds no guarded file (s5 allowed these).
		{"negated class in src", guarded, "rm -rf src/[!x]*", Allow},
		{"negated class in dist", guarded, "rm -rf dist/[!k]*", Allow},
		{"range class in dist", guarded, "rm -rf dist/[a-z]*", Allow},
		{"negated class at a bare root", bare, "rm -rf ./[!.]*", Allow},
		{"posix class at a bare root", bare, "rm -rf [[:alpha:]]*", Allow},
		{"guarded-name glob in a directory without it", bare, "rm src/[d]evenv.yaml", Allow},
		{"startup-file glob in HOME without it", guarded, "rm ~/.[z]shenv", Allow},
		// A find that only lists files writes none of them.
		{"find name negated class in src", guarded, "find src -name '[!.]*'", Allow},
		{"find name negated class type f", guarded, "find . -name '[!.]*' -type f", Allow},
		{"find yaml names", guarded, "find . -name '*.yaml'", Allow},
		{"find name negated class at a bare root", bare, "find src -name '[!.]*'", Allow},
		{"find yaml names at a bare root", bare, "find . -name '*.yaml'", Allow},
		{"find yaml names counted", guarded, "find . -name '*.yaml' | wc -l", Allow},
		{"two listing finds", guarded, "find . -name '*.yaml'; find . -name '*.go'", Allow},
		// A cd joined by && in the line's own shell, to one existing directory
		// that CDPATH cannot redirect, moves where the glob expands: a failed
		// cd stops the line.
		{"negated class after cd ./src", guarded, "cd ./src && rm -rf [!x]*", Allow},
		{"bracket name after cd ./src", guarded, "cd ./src && rm -f [d]evenv.yaml", Allow},
		{"bracket name after an absolute cd", guarded, "cd " + filepath.Join(guarded, "src") + " && rm -f [d]evenv.yaml", Allow},
		{"bracket name after two cds", guarded, "cd ./src && cd .. && cd ./dist && rm -f [d]evenv.yaml", Allow},
		{"bracket name after cd into a directory holding it", bare, "cd ./nested/src && rm -f [d]evenv.yaml", Deny},
		// A `~` resolves with the hook's HOME, which TestMain's HOME does not
		// hold devenv.yaml in. A prefix HOME of the cd itself changes nothing:
		// the shell expands ~ before it applies it.
		{"bracket name after cd ~", guarded, "cd ./src && cd ~ && rm [d]evenv.yaml", Allow},
		{"bracket name after a prefix-HOME cd ~", guarded, "HOME=.. cd ~ && rm [d]evenv.yaml", Allow},

		// The glob's directory holds the guarded file.
		{"star suffix at the root", guarded, "rm -rf ./devenv.*", Deny},
		{"bracket name at the root", guarded, "rm -rf [d]evenv.yaml", Deny},
		{"negated class at the root", guarded, "rm -rf ./[!x]*", Deny},
		{"posix class at the root", guarded, "rm -rf [[:alpha:]]*", Deny},
		{"redirect to a bracket name at the root", guarded, "printf x > [d]evenv.yaml", Deny},
		{"bracket name in a nested directory holding it", bare, "rm nested/src/[d]evenv.yaml", Deny},
		{"startup-file glob in HOME holding it", guarded, "rm ~/.[b]ashrc", Deny},
		// A find with a mutating action matches its patterns anywhere below.
		{"find delete by name", guarded, "find . -name devenv.yaml -delete", Deny},
		{"find delete yaml names", guarded, "find . -name '*.yaml' -delete", Deny},
		{"find delete yaml names at a bare root", bare, "find . -name '*.yaml' -delete", Deny},
		// Fail closed: where the glob expands is not known.
		{"glob after an archive extraction", bare, "tar -xf a.tar; rm src/[d]evenv.yaml", Deny},
		{"glob directory segment", bare, "rm */[d]evenv.yaml", Deny},
		{"glob after cd to an expansion", bare, `cd "$X" && rm [d]evenv.yaml`, Deny},
		{"glob behind an expansion", bare, `rm "$X"/[d]evenv.yaml`, Deny},
		{"glob with an unknown working directory", "", "rm [d]evenv.yaml", Deny},
		{"git pathspec glob", bare, "git rm -- '*devenv.yaml'", Deny},
		// Fail closed, as on w1-s5: the line's directory model takes every cd
		// as succeeding and persisting, but the shell may expand the glob in
		// the root, where devenv.yaml lives.
		{"glob after a cd that may fail, then ;", guarded, "cd nosuch; rm -f [d]evenv.yaml", Deny},
		{"glob after a cd that may fail, then ||", guarded, "cd nosuch || rm -f [d]evenv.yaml", Deny},
		{"glob after a subshell cd, then &&", guarded, "(cd src) && rm -f [d]evenv.yaml", Deny},
		{"glob after a subshell cd, then ;", guarded, "(cd src); rm -f [d]evenv.yaml", Deny},
		{"glob after a background cd", guarded, "cd src & rm -f [d]evenv.yaml", Deny},
		{"glob after a pipeline cd", guarded, "cd src | rm -f [d]evenv.yaml", Deny},
		{"glob after cd -", guarded, "cd src; cd -; rm -f [d]evenv.yaml", Deny},
		{"glob after cd - joined by &&", guarded, "cd src && cd - && rm -f [d]evenv.yaml", Deny},
		{"glob after a physical cd", guarded, "cd -P src && rm -f [d]evenv.yaml", Deny},
		{"glob under ~- after cd", guarded, "cd src && rm -f ~-/[d]evenv.yaml", Deny},
		{"glob under ~+", guarded, "rm -f ~+/[d]evenv.yaml", Deny},
		// Fail closed, as on w1-s5: once the line may change HOME, a later `~`
		// is not the hook's HOME. Each of these reaches the root devenv.yaml in
		// a real shell.
		{"glob after cd ~ with HOME=..", guarded, "HOME=..; cd ./src && cd ~ && rm [d]evenv.yaml", Deny},
		{"glob after cd ~ with HOME=$PWD", guarded, "HOME=$PWD; cd ./src && cd ~ && rm [d]evenv.yaml", Deny},
		{"glob after cd ~ with HOME set mid-line", guarded, "cd ./src && HOME=.. && cd ~ && rm [d]evenv.yaml", Deny},
		{"edit after cd ~ with HOME=..", guarded, "HOME=..; cd ./src && cd ~ && sed -i s/a/b/ [d]evenv.yaml", Deny},
		{"copy after cd ~ with HOME=..", guarded, "HOME=..; cd ./src && cd ~ && cp /tmp/x [d]evenv.yaml", Deny},
		{"glob after cd ~ with an exported HOME", guarded, "export HOME=..; cd ./src && cd ~ && rm [d]evenv.yaml", Deny},
		{"glob after cd ~ with a read HOME", guarded, "cd ./src && read HOME && cd ~ && rm [d]evenv.yaml", Deny},
		{"glob under ~ with HOME=$PWD", guarded, "HOME=$PWD; rm ~/[d]evenv.yaml", Deny},
		{"negated class under ~ with HOME=.", guarded, "HOME=.; rm -rf ~/[!x]*", Deny},
		// Fail closed, as on w1-s5: the cd may land elsewhere than the model
		// says. A bare name may follow CDPATH or zsh's cdpath, zsh's
		// two-operand cd substitutes inside $PWD, and `cd +N` picks from the
		// directory stack; each lands in the root here.
		{"glob after a bare-name cd", guarded, "cd src && rm -f [d]evenv.yaml", Deny},
		{"negated class after a bare-name cd", guarded, "cd src && rm -rf [!x]*", Deny},
		{"glob after a two-operand cd", guarded, "cd ./src && cd src . && rm [d]evenv.yaml", Deny},
		{"glob after a two-operand cd with an empty operand", guarded, "cd ./src && cd /src '' && rm [d]evenv.yaml", Deny},
		{"edit after a two-operand cd", guarded, "cd ./src && cd src . && sed -i s/a/b/ [d]evenv.yaml", Deny},
		{"redirect after a two-operand cd", guarded, "cd ./src && cd src . && cat /tmp/x > [d]evenv.yaml", Deny},
		{"copy after a two-operand cd", guarded, "cd ./src && cd src . && cp /tmp/x [d]evenv.yaml", Deny},
		{"glob after a directory-stack cd", guarded, "pushd ./src && cd +1 && rm [d]evenv.yaml", Deny},
		{"glob after a directory-stack cd with -N", guarded, "pushd ./src && cd -1 && rm [d]evenv.yaml", Deny},
		{"glob after a cdpath cd", guarded, "cdpath=(..); cd proj && rm [d]evenv.yaml", Deny},
		{"glob after a cd to a missing directory", guarded, "cd ./nosuch && rm [d]evenv.yaml", Deny},
		{"glob after a bare cd", guarded, "cd ./src && cd && rm [d]evenv.yaml", Deny},
		{"redirect after a cd that may fail", guarded, "cd nosuch; printf x > [d]evenv.yaml", Deny},
		{"copy after a cd that may fail", guarded, "cd nosuch; cp /dev/null [d]evenv.yaml", Deny},
		{"move after a cd that may fail", guarded, "cd nosuch; mv src/main.go [d]evenv.yaml", Deny},
		// A listing find whose output a writing command consumes.
		{"find piped to xargs rm", guarded, "find . -name devenv.yaml | xargs rm", Deny},
		{"find print0 piped to xargs -0 rm", guarded, "find . -maxdepth 1 -name '[d]evenv.yaml' -print0 | xargs -0 rm -f", Deny},
		{"find list read by xargs rm", guarded, "find . -name devenv.yaml > list; xargs rm < list", Deny},
		{"find in a substitution", guarded, "rm $(find . -name '[d]evenv.yaml')", Deny},
		{"find piped to a while-read loop", guarded, `find . -name '[d]evenv.yaml' | while read f; do rm "$f"; done`, Deny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.cmd, CWD: tt.cwd}
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.want {
				t.Errorf("verdict = %v (first: %s), want %v", got, firstRuleID(matches), tt.want)
			}
			// A deleting find may also reach a protected file (SP-003).
			sp015 := slices.ContainsFunc(matches, func(m RuleMatch) bool { return m.Rule.ID == "SP-015" })
			if tt.want == Deny && !sp015 {
				t.Errorf("rules = %v, want SP-015 among them", matches)
			}
		})
	}
}

// TestEnvFileCommitGateVars covers U18-09's persistent half: a file that sets
// the environment later shells and git commits run in (devenv.nix,
// devenv.local.nix, a shell startup file) must not gain or lose an assignment
// of a commit-gate variable (canon.CommitGateEnvVars), which would skip or
// disable the pre-commit checks for every later commit. Only an assignment
// counts: the generated devenv.nix prints "SKIP: not a git repository", and a
// comment about skipping tests is not an assignment.
func TestEnvFileCommitGateVars(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	home := homeDir(t)

	// The generated enterTest text (addons/devenv security_defaults.go).
	const generated = "{\n  enterTest = ''\n    case \"$hooks_state\" in\n" +
		"      active) echo \"PASS: pre-commit hooks installed\" ;;\n" +
		"      *) echo \"SKIP: not a git repository; pre-commit hook check skipped\" ;;\n" +
		"    esac\n  '';\n  env.API = \"x\";\n}\n"
	devenvNix := filepath.Join(dir, "devenv.nix")
	if err := os.WriteFile(devenvNix, []byte(generated), 0o644); err != nil {
		t.Fatal(err)
	}
	withSkip := filepath.Join(dir, "with-skip", "devenv.local.nix")
	if err := os.MkdirAll(filepath.Dir(withSkip), 0o755); err != nil {
		t.Fatal(err)
	}
	const skipLine = "  env.SKIP = \"gitleaks\";\n"
	if err := os.WriteFile(withSkip, []byte("{\n"+skipLine+"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := filepath.Join(dir, "rc", ".bashrc")
	if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rc, []byte("alias ll='ls -l'\n"), 0o644); err != nil {
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
	tests := []struct {
		name string
		ctx  EvalContext
		want Verdict
	}{
		{"Write devenv.local.nix env.SKIP", write(filepath.Join(dir, "devenv.local.nix"), "{\n  env.SKIP = \"gitleaks\";\n}\n"), Deny},
		{"Write devenv.local.nix env.PREK_SKIP", write(filepath.Join(dir, "devenv.local.nix"), "{ env.PREK_SKIP = \"gitleaks\"; }\n"), Deny},
		{"Write devenv.nix env.PRE_COMMIT_ALLOW_NO_CONFIG", write(filepath.Join(dir, "devenv.nix"), "{ env.PRE_COMMIT_ALLOW_NO_CONFIG = \"1\"; }\n"), Deny},
		{"Write devenv.nix env.PREK_ALLOW_NO_CONFIG", write(filepath.Join(dir, "devenv.nix"), "{ env.PREK_ALLOW_NO_CONFIG = \"1\"; }\n"), Deny},
		{"Write devenv.nix env attrset SKIP", write(filepath.Join(dir, "devenv.nix"), "{ env = {\n    SKIP = \"gitleaks\";\n  };\n}\n"), Deny},
		{"Write devenv.nix quoted env.SKIP", write(filepath.Join(dir, "devenv.nix"), "{ env.\"SK\" + \"IP\" = \"gitleaks\"; }\n"), Deny},
		{"Edit adds env.SKIP to generated devenv.nix", edit(devenvNix, "  env.API = \"x\";\n", "  env.API = \"x\";\n  env.SKIP = \"gitleaks\";\n"), Deny},
		{"Edit removes env.SKIP", edit(withSkip, skipLine, ""), Deny},
		{"Edit adds export SKIP to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\nexport SKIP=gitleaks\n"), Deny},
		{"Write bashrc export SKIP", write(filepath.Join(home, ".bashrc"), "export SKIP=gitleaks\n"), Deny},
		{"Write fish config set SKIP", write(filepath.Join(home, ".config", "fish", "config.fish"), "set -gx SKIP gitleaks\n"), Deny},
		{"Write PowerShell profile env SKIP", write(filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), "$env:SKIP = 'gitleaks'\n"), Deny},

		{"Edit of generated devenv.nix enterTest", edit(devenvNix, "env.API = \"x\"", "env.API = \"y\""), Allow},
		{"Write generated devenv.nix", write(filepath.Join(dir, "devenv.nix"), generated), Allow},
		{"Edit adds a skip comment to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\n# skip slow tests\n"), Allow},
		{"Edit adds a SKIP test to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\n[ \"$SKIP\" == 1 ] && echo skipping\n"), Allow},
		// A POSIX test compares with a single `=`: removing the quotes and
		// the `$` must not turn the reference into an assignment.
		{"Edit adds a POSIX SKIP test to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\nif [ \"$SKIP\" = 1 ]; then :; fi\n"), Allow},
		{"Edit adds a braced PREK_SKIP test to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\ntest \"${PREK_SKIP}\" = x && echo prek\n"), Allow},
		{"Edit adds a SKIP default to bashrc", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\n: \"${SKIP:=gitleaks}\"\n"), Deny},
		{"Edit adds an unrelated SKIP-suffixed variable", edit(rc, "alias ll='ls -l'\n", "alias ll='ls -l'\nexport QSDEV_SKIP_SETUP=1\n"), Allow},
		{"Write other file setting SKIP", write(filepath.Join(dir, "Makefile"), "SKIP=gitleaks\n"), Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.ctx
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.want {
				t.Errorf("verdict = %v (first: %s), want %v", got, firstRuleID(matches), tt.want)
			}
			if tt.want == Deny && firstRuleID(matches) != "SP-015" {
				t.Errorf("first rule = %s, want SP-015", firstRuleID(matches))
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
		// Not .cache: the per-user cache directory (~/.cache/<app> without
		// XDG_CACHE_HOME) is protected, so ~/.cache is an ancestor too.
		{"unknown account other dir", bash("rm -rf " + unknown + "/.npm"), Allow},
		{"cd unknown account other dir", bash("cd " + unknown + "/.npm && rm -rf x"), Allow},
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
