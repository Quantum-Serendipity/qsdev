package claudecode_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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

		// Fail-closed cases: exceeding the recursion cap, or an unparseable
		// segment, must be DETECTED (surfaced for validation) — never dropped.
		{"eight-deep nesting fails closed", deepNestFailClosed, true, nil},
		{"unbalanced quotes fail closed", `npm install "evil`, true, nil},

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
			cmd.Env = append(os.Environ(), "PG_PATH="+template, "PG_CMD="+tc.command)
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
