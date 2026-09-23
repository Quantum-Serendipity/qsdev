package claudecode_test

import (
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
for eco, mgr, seg, ps in dets:
    pkgs.extend(ps)
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
	start := strings.Index(src, "# 4. Check publication age")
	end := strings.Index(src, "if age_days is not None")
	if start < 0 || end < start {
		t.Fatal("age-check block not found in package-guard.py")
	}
	// OSV ecosystem name -> qsdev language name.
	languageOf := map[string]string{"npm": "javascript", "PyPI": "python", "crates.io": "rust"}
	var checked []string
	for _, m := range regexp.MustCompile(`ecosystem == "([^"]+)"`).FindAllStringSubmatch(src[start:end], -1) {
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
