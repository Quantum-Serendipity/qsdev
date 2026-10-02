package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// orgConfigTarget is the org overlay named by <EnvPrefix>ORG_CONFIG for this
// test binary. canon reads the variable once, when it builds its protected
// table, so TestMain sets it before any test runs. TestMain also points the
// home directory at a fresh one, so verdicts that depend on what exists under
// it (a move onto ~/.config) do not depend on the machine running the tests.
var orgConfigTarget string

func TestMain(m *testing.M) {
	os.Exit(runWithOrgConfig(m))
}

func runWithOrgConfig(m *testing.M) int {
	dir, err := os.MkdirTemp("", "rules-org-config-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating org config dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(dir)
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	orgConfigTarget = filepath.Join(dir, "org-defaults.yaml")
	home := filepath.Join(dir, "home")
	if err := os.Mkdir(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "creating home dir: %v\n", err)
		return 1
	}
	for _, v := range []string{"HOME", "USERPROFILE"} {
		if err := os.Setenv(v, home); err != nil {
			fmt.Fprintf(os.Stderr, "setting %s: %v\n", v, err)
			return 1
		}
	}
	if err := os.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", orgConfigTarget); err != nil {
		fmt.Fprintf(os.Stderr, "setting org config: %v\n", err)
		return 1
	}
	return m.Run()
}

// TestGeneratorInputsVerdicts covers B3: the files the Claude settings
// generator reads (the answers, the devenv answers mirror, .envrc and the org
// overlay) cannot be rewritten by the agent through Write/Edit or Bash, while
// reading them stays allowed.
func TestGeneratorInputsVerdicts(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")
	b := branding.Get()
	answersRel := b.StateDir + "/." + b.AppName + "-init-answers.yaml"
	answersFile := filepath.Join(project, filepath.FromSlash(answersRel))
	devenvCopy := filepath.Join(project, ".devenv", "."+b.AppName+"-answers.yaml")
	envrc := filepath.Join(project, ".envrc")
	homeOverlay := filepath.Join(home, ".config", b.AppName, "defaults.yaml")
	overlayTilde := "~/.config/" + b.AppName + "/defaults.yaml"
	overlayHomeVar := "$HOME/.config/" + b.AppName + "/defaults.yaml"
	orgEnv := b.EnvPrefix + "ORG_CONFIG"
	sandboxPolicy := filepath.Join(home, "."+b.AppName, "cache", "sandbox-policy", "x.json")

	file := func(tool, p string) EvalContext {
		return EvalContext{ToolName: tool, FilePath: p, CanonicalPath: p, CWD: project, Content: "x"}
	}
	bash := func(cmd string) EvalContext {
		return EvalContext{ToolName: "Bash", Command: cmd, CWD: project}
	}

	tests := []struct {
		name string
		ctx  EvalContext
		want Verdict
	}{
		{"sed answers", bash("sed -i s/true/false/ " + answersRel), Deny},
		{"Write answers", file("Write", answersFile), Deny},
		{"Edit answers", file("Edit", answersFile), Deny},
		{"Write devenv answers copy", file("Write", devenvCopy), Deny},
		{"Write envrc", file("Write", envrc), Deny},
		{"append envrc", bash("echo x >> .envrc"), Deny},
		{"Write home org overlay", file("Write", homeOverlay), Deny},
		{"sed home org overlay", bash("sed -i s/a/b/ " + overlayTilde), Deny},
		{"Write org config env target", file("Write", orgConfigTarget), Deny},
		{"Write sandbox policy cache", file("Write", sandboxPolicy), Deny},
		{"remove state dir", bash("rm -rf " + b.StateDir), Deny},
		{"cd state dir then write answers", bash("cd " + b.StateDir + " && echo x > ." + b.AppName + "-init-answers.yaml"), Deny},
		{"move envrc", bash("mv .envrc x"), Deny},
		{"tee home org overlay", bash("echo x | tee " + overlayTilde), Deny},
		// A glob reaches a generator input as surely as its literal name.
		{"glob state dir", bash("sed -i s/true/false/ .devin?t/." + b.AppName + "-init-answers.yaml"), Deny},
		{"glob devenv answers copy", bash("sed -i s/a/b/ .devenv/." + b.AppName + "-answ*.yaml"), Deny},
		{"glob envrc", bash("echo x > .env?c"), Deny},
		{"cd glob state dir", bash("cd .devi* && sed -i s/a/b/ ." + b.AppName + "-init-answers.yaml"), Deny},
		{"glob home org overlay", bash("sed -i s/a/b/ ~/.config/" + b.AppName[:2] + "*/defaults.yaml"), Deny},
		// $HOME and the org-config variable name the overlay as surely as ~.
		{"sed $HOME org overlay", bash("sed -i s/a/b/ " + overlayHomeVar), Deny},
		{"sed ${HOME} org overlay", bash("sed -i s/a/b/ ${HOME}/.config/" + b.AppName + "/defaults.yaml"), Deny},
		{"printf to quoted $HOME org overlay", bash(`printf evil > "` + overlayHomeVar + `"`), Deny},
		{"tee $HOME org overlay", bash("echo evil | tee " + overlayHomeVar), Deny},
		{"cd $HOME overlay dir then write", bash("cd $HOME/.config/" + b.AppName + " && echo evil > defaults.yaml"), Deny},
		{"sed org config variable", bash(`sed -i s/a/b/ "$` + orgEnv + `"`), Deny},
		{"append org config variable", bash("echo 'mcp: evil' >> $" + orgEnv), Deny},
		{"append braced org config variable", bash("echo 'mcp: evil' >> ${" + orgEnv + "}"), Deny},
		// Spellings the scan cannot resolve statically still mention the
		// overlay, so a mutating command fails closed.
		{"sh -c tilde org overlay", bash(`sh -c "echo x > ~/.config/` + b.AppName + `/defaults.yaml"`), Deny},
		{"bash -c $HOME org overlay", bash(`bash -c "echo x > \$HOME/.config/` + b.AppName + `/defaults.yaml"`), Deny},
		{"dd of org overlay", bash("dd of=$HOME/.config/" + b.AppName + "/defaults.yaml if=/tmp/e"), Deny},
		{"tilde via variable org overlay", bash("U=~; echo x > $U/.config/" + b.AppName + "/defaults.yaml"), Deny},
		{"home via variable org overlay", bash("H=$HOME; echo x > $H/.config/" + b.AppName + "/defaults.yaml"), Deny},
		{"command substitution org overlay", bash("echo x > $(echo ~)/.config/" + b.AppName + "/defaults.yaml"), Deny},
		{"backticks org overlay", bash("echo x > `echo $HOME`/.config/" + b.AppName + "/defaults.yaml"), Deny},
		{"default expansion org overlay", bash(`echo x > "${HOME:-/x}/.config/` + b.AppName + `/defaults.yaml"`), Deny},
		{"sh -c org config target", bash(`sh -c "echo x > ` + filepath.ToSlash(orgConfigTarget) + `"`), Deny},
		// Relocating the overlay points a regeneration at an unprotected file.
		{"prefix-assign org config", bash(orgEnv + "=/tmp/evil.yaml " + b.AppName + " claude update --force"), Deny},
		{"export org config", bash("export " + orgEnv + "=/tmp/evil.yaml; " + b.AppName + " claude update --force"), Deny},
		{"env org config", bash("env " + orgEnv + "=/tmp/evil.yaml " + b.AppName + " claude update --force"), Deny},
		{"sh -c assign org config", bash(`sh -c "` + orgEnv + `=/tmp/evil.yaml ` + b.AppName + ` claude update"`), Deny},
		// The home overlay lives under the home directory, so a home variable
		// set for the CLI relocates it as surely as the org-config variable.
		{"prefix-assign home for cli", bash("HOME=/tmp/e " + b.AppName + " claude update --force"), Deny},
		{"prefix-assign home for init update", bash("HOME=/tmp/e " + b.AppName + " init --update"), Deny},
		{"export home then cli", bash("export HOME=/tmp/e; " + b.AppName + " init --update"), Deny},
		{"bare assign home then cli", bash("HOME=/tmp/e; " + b.AppName + " init --update"), Deny},
		{"env home for cli", bash("env HOME=/tmp/e " + b.AppName + " claude update"), Deny},
		{"env home for cli path", bash("env HOME=/tmp/e ./bin/" + b.AppName + " enable gitleaks"), Deny},
		{"userprofile for cli", bash("USERPROFILE=/tmp/e " + b.AppName + " claude update"), Deny},
		{"lowercase userprofile for cli exe", bash("userprofile=/tmp/e " + b.AppName + ".exe claude update"), Deny},
		{"sh -c home for cli", bash(`HOME=/tmp/e sh -c "` + b.AppName + ` init --update"`), Deny},
		{"sh -c assign home inside", bash(`sh -c "HOME=/tmp/e ` + b.AppName + ` init --update"`), Deny},
		{"unset home then cli", bash("unset HOME; " + b.AppName + " init --update"), Deny},
		{"env -u home for cli", bash("env -u HOME " + b.AppName + " init --update"), Deny},
		{"env -i for cli", bash("env -i " + b.AppName + " init --update"), Deny},
		{"home for expanded program", bash("HOME=/tmp/e $Q init --update"), Deny},
		// Every bash form that sets or clears HOME counts, and so does any
		// spelling of the program word. The CLI also ignores HOME for the
		// overlay (catalog.OrgConfigPath); these keep the fallback covered.
		{"for-loop home then cli", bash("for HOME in /tmp/e; do " + b.AppName + " init --update; done"), Deny},
		{"read home then cli", bash("read HOME <<< /tmp/e; " + b.AppName + " init --update"), Deny},
		{"printf -v home then cli", bash("printf -v HOME %s /tmp/e; " + b.AppName + " init --update"), Deny},
		{"mapfile home then cli", bash("mapfile -t HOME <<< /tmp/e; " + b.AppName + " init --update"), Deny},
		{"mapfile clears home then cli", bash("mapfile -t HOME < /dev/null; " + b.AppName + " claude update"), Deny},
		{"getopts home then cli", bash("getopts e HOME -e; " + b.AppName + " init --update"), Deny},
		{"arithmetic home then cli", bash("((HOME=5)); " + b.AppName + " init --update"), Deny},
		{"let home then cli", bash("let HOME=5; " + b.AppName + " init --update"), Deny},
		{"default-assign home then cli", bash(": ${HOME:=/tmp/e}; " + b.AppName + " init --update"), Deny},
		{"export computed name then cli", bash("X=HOME; export $X=/tmp/e; " + b.AppName + " init --update"), Deny},
		{"nameref home then cli", bash("declare -n R=HOME; R=/tmp/e; " + b.AppName + " init --update"), Deny},
		{"export -n home then cli", bash("export -n HOME; " + b.AppName + " init --update"), Deny},
		{"expanded read home then cli", bash("$R HOME <<< /tmp/e; " + b.AppName + " init --update"), Deny},
		{"home for ansi-c program", bash("HOME=/tmp/e $'" + b.AppName + "' init --update"), Deny},
		{"home for split ansi-c program", bash("HOME=/tmp/e " + b.AppName[:2] + "$'" + b.AppName[2:3] + "'" + b.AppName[3:] + " init --update"), Deny},
		{"home for brace program", bash("HOME=/tmp/e {" + b.AppName + ",} init --update"), Deny},
		{"home for glob program", bash("HOME=/tmp/e " + b.AppName[:len(b.AppName)-1] + "? init --update"), Deny},
		{"home for upper-case program", bash("HOME=/tmp/e " + strings.ToUpper(b.AppName) + " init --update"), Deny},
		{"env chdir then unset home", bash("env -C /tmp -u HOME " + b.AppName + " init --update"), Deny},
		{"env long chdir then unset home", bash("env --chdir /tmp -u HOME " + b.AppName + " init --update"), Deny},
		{"env abbreviated ignore-environment", bash("env --ignore-env " + b.AppName + " init --update"), Deny},
		{"env abbreviated unset", bash("env --un=HOME " + b.AppName + " init --update"), Deny},
		{"env cluster unset home", bash("env -iu HOME " + b.AppName + " init --update"), Deny},
		{"env.exe unset home", bash("env.exe -u HOME " + b.AppName + " init --update"), Deny},
		{"upper-case env -i", bash("ENV -i " + b.AppName + " init --update"), Deny},
		{"env split-string unset home", bash("env -S '-u HOME " + b.AppName + " init --update'"), Deny},
		{"sudo env unset home", bash("sudo env -u HOME " + b.AppName + " init --update"), Deny},
		// Deleting, moving or replacing a directory above the overlay removes
		// or plants it without naming it.
		{"remove overlay parent", bash("rm -rf ~/.config"), Deny},
		{"remove $HOME overlay parent", bash("rm -rf $HOME/.config"), Deny},
		{"remove home", bash("rm -rf ~"), Deny},
		{"move overlay parent away", bash("mv ~/.config ~/.config.bak"), Deny},
		{"move tree onto missing overlay parent", bash("mv /tmp/o ~/.config"), Deny},
		{"copy tree contents onto overlay parent", bash("cp -r /tmp/o/. ~/.config"), Deny},
		{"rsync tree contents onto overlay parent", bash("rsync -a /tmp/o/ ~/.config"), Deny},
		{"move app-named tree into home config", bash("mv -t ~/.config /tmp/" + b.AppName), Deny},
		{"glob overlay parent", bash("rm -rf ~/.conf*"), Deny},
		// An expansion the scan cannot render (a command substitution, a
		// variable set earlier on the line) can still be the home directory,
		// so an operand ending in the overlay's ancestor chain fails closed.
		{"command substitution overlay parent", bash(`rm -rf "$(echo ~)/.config"`), Deny},
		{"variable overlay parent", bash("H=~; rm -rf $H/.config"), Deny},
		{"variable overlay dir", bash(`rm -rf "$X/.config/` + b.AppName + `"`), Deny},
		{"move command substitution overlay parent", bash(`mv "$(echo ~)/.config" /tmp/x`), Deny},
		{"find starting at expanded overlay parent", bash(`find "$(echo ~)/.config" -delete`), Deny},
		// A cd to an expanded directory may land in the home directory, and
		// a plain echo substitution prints its words.
		{"cd $HOME config then remove overlay dir", bash("cd $HOME/.config && rm -rf " + b.AppName), Deny},
		{"cd expanded config then remove overlay dir", bash(`cd "$(cat /tmp/d)/.config" && rm -rf ` + b.AppName), Deny},
		{"cd expanded then config then remove overlay dir", bash(`cd "$X" && cd .config && rm -rf ` + b.AppName), Deny},
		{"cd glob config then remove overlay dir", bash("cd ~/.conf* && rm -rf " + b.AppName), Deny},
		{"echo substitution overlay parent", bash(`rm -rf "$(echo ~/.config)"`), Deny},
		{"split echo substitution overlay parent", bash(`rm -rf "$(echo ~/.co)nfig"`), Deny},
		{"echo substitution home then glob", bash(`rm -rf "$(echo ~)"/.conf*`), Deny},
		{"expanded prefix then glob", bash(`rm -rf "$(cat /tmp/h)"/.conf*`), Deny},
		{"cd home then relative glob", bash("cd ~ && rm -rf .conf*"), Deny},
		// Moving to the trash, gio and tar --remove-files remove the operand.
		{"trash-put overlay parent", bash("trash-put ~/.config"), Deny},
		{"gio trash overlay parent", bash("gio trash ~/.config"), Deny},
		{"gio move overlay parent", bash("gio move ~/.config /tmp/x"), Deny},
		{"tar remove-files overlay parent", bash("tar -cf /tmp/x.tar --remove-files ~/.config"), Deny},
		{"tar abbreviated remove-files overlay parent", bash("tar -cf /tmp/x.tar --remove-f ~/.config"), Deny},
		{"find delete overlay parent", bash("find ~/.config -delete"), Deny},
		{"find exec rm overlay parent", bash("find ~/.config -exec rm -rf {} +"), Deny},
		{"find exec mv overlay parent entries", bash(`find ~/.config -mindepth 1 -maxdepth 1 -exec mv {} /tmp/ \;`), Deny},
		// A find selects a generator input by name pattern without naming
		// its path.
		{"find exec sed answers", bash("find . -name '*init-answers.yaml' -exec sed -i s/true/false/ {} +"), Deny},
		{"find exec sed answers exact name", bash("find . -name '." + b.AppName + "-init-answers.yaml' -exec sed -i s/a/b/ {} +"), Deny},
		{"find delete answers", bash("find . -name '*init-answers.yaml' -delete"), Deny},
		{"find exec cp over answers", bash(`find . -name '*-answers.yaml' -exec cp /tmp/evil {} \;`), Deny},
		{"find path devenv answers copy", bash("find . -path '*.devenv/*answers.yaml' -delete"), Deny},
		{"find delete envrc", bash("find . -name .envrc -delete"), Deny},
		{"find exec sed home org overlay", bash("find ~ -name defaults.yaml -path '*" + b.AppName + "*' -exec sed -i s/a/b/ {} +"), Deny},
		{"find exec sed $HOME org overlay", bash("find $HOME/.config -name 'def*.yaml' -exec sed -i s/a/b/ {} +"), Deny},
		{"find delete org config target", bash("find " + filepath.ToSlash(filepath.Dir(orgConfigTarget)) + " -name '*.yaml' -delete"), Deny},

		{"cat answers", bash("cat " + answersRel), Allow},
		{"find answers without action", bash("find . -name '*init-answers.yaml'"), Allow},
		{"find exec on unrelated files", bash("find . -name '*.go' -exec gofmt -l {} +"), Allow},
		{"find exec other app overlay", bash("find ~ -name defaults.yaml -path '*zz-other*' -exec sed -i s/a/b/ {} +"), Allow},
		{"find delete outside project and home", bash("find /tmp/scratch -name '*init-answers.yaml' -delete"), Allow},
		{"Read answers", file("Read", answersFile), Allow},
		{"cat envrc", bash("cat .envrc"), Allow},
		{"direnv allow", bash("direnv allow"), Allow},
		{"direnv allow envrc", bash("direnv allow .envrc"), Allow},
		{"direnv allow dot", bash("direnv allow ."), Allow},
		{"direnv deny envrc", bash("direnv deny .envrc"), Allow},
		{"direnv status", bash("direnv status"), Allow},
		{"printenv org config variable", bash("printenv " + orgEnv), Allow},
		{"direnv edit envrc", bash("direnv edit .envrc"), Deny},
		{"direnv allow then append envrc", bash("direnv allow .envrc && echo x >> .envrc"), Deny},
		{"git status", bash("git status"), Allow},
		{"git diff envrc", bash("git diff .envrc"), Allow},
		{"cat devenv answers copy", bash("cat .devenv/." + b.AppName + "-answers.yaml"), Allow},
		{"grep state dir", bash("grep -r tier " + b.StateDir + "/"), Allow},
		{"ls state dir", bash("ls -la " + b.StateDir), Allow},
		{"glob unrelated dotfile", bash("echo x > .env?c.bak"), Allow},
		{"glob other config dir", bash("echo x > ~/.config/zz*/defaults.yaml"), Allow},
		{"cat org config variable", bash(`cat "$` + orgEnv + `"`), Allow},
		{"echo org config variable", bash("echo $" + orgEnv), Allow},
		{"ls org overlay dir", bash("ls ~/.config/" + b.AppName), Allow},
		{"cat org overlay", bash("cat ~/.config/" + b.AppName + "/defaults.yaml"), Allow},
		{"cat $HOME org overlay", bash("cat " + overlayHomeVar), Allow},
		{"cat org config target", bash("cat " + filepath.ToSlash(orgConfigTarget)), Allow},
		{"ls $HOME config", bash("ls $HOME/.config"), Allow},
		{"echo $HOME", bash("echo $HOME"), Allow},
		{"du overlay parent", bash("du -sh ~/.config"), Allow},
		{"copy file into home", bash("cp notes.txt ~"), Allow},
		{"remove sibling config dir", bash("rm -rf ~/.config/other"), Allow},
		{"remove lookalike of overlay parent", bash("rm -rf ~/.conf"), Allow},
		{"remove expanded build dir", bash(`rm -rf "$TMPDIR/build"`), Allow},
		{"remove expanded variable", bash(`rm -rf "$tmp"`), Allow},
		{"remove expanded sibling config dir", bash("rm -rf $X/.config/other"), Allow},
		{"remove literal config dir beside expansion", bash("rm -rf $TMPDIR/x .config"), Allow},
		{"cd expanded then remove unrelated dir", bash(`cd "$X" && rm -rf build`), Allow},
		{"cd $HOME config then remove sibling", bash("cd $HOME/.config && rm -rf other"), Allow},
		{"tar overlay parent without removal", bash("tar -cf /tmp/x.tar ~/.config"), Allow},
		{"gio info overlay parent", bash("gio info ~/.config"), Allow},
		{"trash-put unrelated file", bash("trash-put /tmp/scratch/x"), Allow},
		{"remove echo substitution elsewhere", bash(`rm -rf "$(echo /tmp/build)"`), Allow},
		// .devenv is devenv's runtime directory, routinely removed to reset
		// the environment; its answers mirror is read only when the protected
		// primary answers file is missing, so the directory stays removable.
		{"remove devenv runtime dir", bash("rm -rf .devenv"), Allow},
		// A home variable set for any other program does not move the
		// overlay the CLI reads.
		{"home for go test", bash("HOME=$(mktemp -d) go test ./..."), Allow},
		{"export home then make", bash("export HOME=/tmp/e; make"), Allow},
		{"cli without home change", bash(b.AppName + " status"), Allow},
		{"echo home with cli name", bash("echo HOME is $HOME; " + b.AppName + " status"), Allow},
		{"for-loop other variable with cli", bash("for f in a b; do " + b.AppName + " status; done"), Allow},
		{"read other variable with cli", bash("read -r line < notes.txt; " + b.AppName + " status"), Allow},
		{"env chdir for cli", bash("env -C /tmp " + b.AppName + " status"), Allow},
		{"read home for other program", bash("read HOME <<< /tmp/e; make"), Allow},
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
