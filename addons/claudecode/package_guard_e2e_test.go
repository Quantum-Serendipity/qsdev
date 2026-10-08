package claudecode_test

import (
	"crypto/sha1" // git object ids, not a security hash
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// pgResult is what testdata/hooks/package_guard_driver.py reports for one run
// of the package-guard hook.
type pgResult struct {
	Exit   int `json:"exit"`
	Output *struct {
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
		UpdatedInput             *struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		} `json:"updatedInput"`
	} `json:"output"`
	Stderr  string `json:"stderr"`
	Lookups []struct {
		Source  string  `json:"source"`
		Name    string  `json:"name"`
		Version *string `json:"version"`
	} `json:"lookups"`
	// Subprocesses is every argv the hook passed to subprocess.run.
	Subprocesses [][]string `json:"subprocesses"`
}

// reason is everything the hook told the user: its decision reason and stderr.
func (r pgResult) reason() string {
	if r.Output == nil {
		return r.Stderr
	}
	return r.Stderr + r.Output.PermissionDecisionReason
}

func (r pgResult) decision() string {
	switch {
	case r.Exit == 2:
		return "deny"
	case r.Output == nil:
		return "allow"
	case r.Output.PermissionDecision == "":
		return "allow"
	default:
		return r.Output.PermissionDecision
	}
}

func (r pgResult) rewritten() string {
	if r.Output == nil || r.Output.UpdatedInput == nil {
		return ""
	}
	return r.Output.UpdatedInput.Command
}

// runPackageGuard feeds one hook envelope through the whole package-guard
// hook (envelope to decision, including main() and its error handling) with
// the registries stubbed out; env configures the stub (PG_VULN, PG_FRESH,
// PG_URL_ERROR, PG_INTERNAL_ERROR, PG_GIT_LSTREE).
func runPackageGuard(t *testing.T, tool string, input map[string]any, env ...string) pgResult {
	t.Helper()
	return runPackageGuardIn(t, "", tool, input, env...)
}

// runPackageGuardIn is runPackageGuard with the envelope's cwd set to cwd
// (omitted when empty). input is any JSON value, so a malformed tool_input
// can be sent.
func runPackageGuardIn(t *testing.T, cwd, tool string, input any, env ...string) pgResult {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping package-guard end-to-end test")
	}
	hook, err := filepath.Abs(filepath.Join("templates", "hooks", "package-guard.py"))
	if err != nil {
		t.Fatal(err)
	}
	driver, err := filepath.Abs(filepath.Join("testdata", "hooks", "package_guard_driver.py"))
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]any{"tool_name": tool, "tool_input": input}
	if cwd != "" {
		fields["cwd"] = cwd
	}
	envelope, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, driver)
	cmd.Env = hookEnv(t, append([]string{"PG_PATH=" + hook, "PG_ENVELOPE=" + string(envelope)}, env...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("driver failed: %v (stdout %q)", err, out)
	}
	var res pgResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("bad driver output %q: %v", out, err)
	}
	return res
}

// TestPackageGuard_EndToEnd runs whole hook invocations (W016): the decision,
// the reason, the rewritten command and the registry lookups, for every tool
// that runs shell commands (W033) and for the fail-closed paths.
func TestPackageGuard_EndToEnd(t *testing.T) {
	t.Parallel()
	bash := func(c string) map[string]any { return map[string]any{"command": c} }

	cases := []struct {
		name          string
		tool          string
		input         map[string]any
		env           []string
		want          string
		reasonHas     string
		rewrittenHas  string
		wantLookupFor string
		// keepsDescription: updatedInput replaces the whole tool input, so the
		// rewrite must carry the other fields (Monitor requires description).
		keepsDescription string
	}{
		{name: "clean npm install is rewritten with safety flags", tool: "Bash", input: bash("npm install left-pad"),
			want: "ask", rewrittenHas: "npm install --ignore-scripts left-pad", wantLookupFor: "left-pad"},
		{name: "vulnerable package", tool: "Bash", input: bash("npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "freshly published package", tool: "Bash", input: bash("pip install fresh"), env: []string{"PG_FRESH=fresh"},
			want: "deny", reasonHas: "days ago"},
		{name: "pinned version is checked", tool: "Bash", input: bash("pip install requests==2.31.0"),
			want: "ask", rewrittenHas: "pip install --only-binary :all: requests==2.31.0", wantLookupFor: "requests"},
		{name: "network failure fails closed", tool: "Bash", input: bash("npm install left-pad"), env: []string{"PG_URL_ERROR=1"},
			want: "deny", reasonHas: "Failing closed"},
		{name: "internal error fails closed", tool: "Bash", input: bash("npm install left-pad"), env: []string{"PG_INTERNAL_ERROR=1"},
			want: "deny"},
		{name: "imperative nix install", tool: "Bash", input: bash("nix-env -iA nixpkgs.hello"), want: "deny", reasonHas: "nix-env"},
		{name: "not an install", tool: "Bash", input: bash("go test ./..."), want: "allow"},
		{name: "compound command rewrites the install segment", tool: "Bash", input: bash("cd web && npm install left-pad && npm test"),
			want: "ask", rewrittenHas: "npm install --ignore-scripts left-pad &&"},
		// W033: the same commands through PowerShell and Monitor.
		{name: "powershell vulnerable package", tool: "PowerShell", input: bash("npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor vulnerable package", tool: "Monitor", input: map[string]any{"command": "npm install evil", "description": "d"},
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor rewrite keeps the other input fields", tool: "Monitor",
			input: map[string]any{"command": "npm install left-pad", "description": "watch install"},
			want:  "ask", rewrittenHas: "--ignore-scripts", keepsDescription: "watch install"},
		{name: "powershell npm.cmd launcher", tool: "PowerShell", input: bash("npm.cmd install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "powershell windows path to pip.exe", tool: "PowerShell", input: bash(`C:\Python312\Scripts\pip.exe install evil`),
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor websocket source", tool: "Monitor", input: map[string]any{"ws": map[string]any{"url": "wss://x.example"}, "description": "d"},
			want: "allow"},
		// U17-07: terminal multiplexers run their command detached.
		{name: "tmux new runs a script", tool: "Bash", input: bash("tmux new -d 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "tmux new-session runs a script", tool: "Bash", input: bash("tmux new-session -d 'npm install evil'"),
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen runs its command", tool: "Bash", input: bash("screen -dmS s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		// U17-07 review: screen -X/-Q, tmux prefixes/chains/shell commands, script FILE -c.
		{name: "screen -X exec npm install evil", tool: "Bash", input: bash("screen -X exec npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -X screen npm install evil", tool: "Bash", input: bash("screen -X screen npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -S work -X screen -t t npm install evil", tool: "Bash", input: bash("screen -S work -X screen -t t npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -x s -X exec npm install evil", tool: "Bash", input: bash("screen -x s -X exec npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -r s -X exec npm install evil", tool: "Bash", input: bash("screen -r s -X exec npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -Q exec npm install evil", tool: "Bash", input: bash("screen -Q exec npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "tmux new-s -d npm install evil", tool: "Bash", input: bash("tmux new-s -d 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "tmux new-w npm install evil", tool: "Bash", input: bash("tmux new-w 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "tmux run-shell npm install evil", tool: "Bash", input: bash("tmux run-shell 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "tmux ls \\; new -d npm install evil", tool: "Bash", input: bash("tmux ls \\; new -d 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "script /dev/null -c npm install evil", tool: "Bash", input: bash("script /dev/null -c 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "script -q /dev/null -c npm install evil", tool: "Bash", input: bash("script -q /dev/null -c 'npm install evil'"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "sudo -h h npm install evil", tool: "Bash", input: bash("sudo -h h npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		// screen -R creates a session running the command when none matches.
		{name: "screen -R s npm install evil", tool: "Bash", input: bash("screen -R s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -RR npm install evil", tool: "Bash", input: bash("screen -RR npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -RR npm npm install evil", tool: "Bash", input: bash("screen -RR npm npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -dR s npm install evil", tool: "Bash", input: bash("screen -dR s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -dRR s npm install evil", tool: "Bash", input: bash("screen -dRR s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -xRR s npm install evil", tool: "Bash", input: bash("screen -xRR s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -d -RR s npm install evil", tool: "Bash", input: bash("screen -d -RR s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -D -RR s npm install evil", tool: "Bash", input: bash("screen -D -RR s npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "script -qc 'screen -R s npm install evil' /dev/null", tool: "Bash", input: bash("script -qc 'screen -R s npm install evil' /dev/null"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "screen -R", tool: "Bash", input: bash("screen -R"), env: []string{"PG_VULN=evil"}, want: "allow"},
		{name: "screen -dRR s", tool: "Bash", input: bash("screen -dRR s"), env: []string{"PG_VULN=evil"}, want: "allow"},
		{name: "screen -r s", tool: "Bash", input: bash("screen -r s"), env: []string{"PG_VULN=evil"}, want: "allow"},
		{name: "non-shell tool", tool: "Write", input: map[string]any{"file_path": "x", "content": "npm install evil"},
			env: []string{"PG_VULN=evil"}, want: "allow"},
		// U17-03: `--` before the verb ends the manager's global options.
		{name: "npm -- install evil", tool: "Bash", input: bash("npm -- install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "npm -- i evil", tool: "Bash", input: bash("npm -- i evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "pnpm -- add evil", tool: "Bash", input: bash("pnpm -- add evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "yarn -- add evil", tool: "Bash", input: bash("yarn -- add evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "bun -- add evil", tool: "Bash", input: bash("bun -- add evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "npm --global -- install evil", tool: "Bash", input: bash("npm --global -- install evil"),
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "uv -- pip install evil", tool: "Bash", input: bash("uv -- pip install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "npm install -- evil", tool: "Bash", input: bash("npm install -- evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "npm -- install is rewritten after the verb", tool: "Bash", input: bash("npm -- install left-pad"),
			want: "ask", rewrittenHas: "npm -- install --ignore-scripts left-pad", wantLookupFor: "left-pad"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runPackageGuard(t, tc.tool, tc.input, tc.env...)
			if got := res.decision(); got != tc.want {
				t.Fatalf("decision = %q, want %q (result %+v)", got, tc.want, res)
			}
			if tc.reasonHas != "" {
				reason := res.Stderr
				if res.Output != nil {
					reason += res.Output.PermissionDecisionReason
				}
				if !strings.Contains(reason, tc.reasonHas) {
					t.Errorf("reason %q does not mention %q", reason, tc.reasonHas)
				}
			}
			if tc.rewrittenHas != "" && !strings.Contains(res.rewritten(), tc.rewrittenHas) {
				t.Errorf("rewritten command %q does not contain %q", res.rewritten(), tc.rewrittenHas)
			}
			if tc.keepsDescription != "" && (res.Output == nil || res.Output.UpdatedInput == nil ||
				res.Output.UpdatedInput.Description != tc.keepsDescription) {
				t.Errorf("updatedInput dropped description %q (result %+v)", tc.keepsDescription, res)
			}
			if tc.wantLookupFor != "" {
				found := false
				for _, l := range res.Lookups {
					found = found || l.Name == tc.wantLookupFor
				}
				if !found {
					t.Errorf("no registry lookup for %q: %+v", tc.wantLookupFor, res.Lookups)
				}
			}
		})
	}
}

// TestPackageGuard_RewrittenCommandRuns executes the rewritten command under
// bash with stub package managers on a restricted PATH, proving the safety
// flag lands on the install invocation itself and the command stays valid.
func TestPackageGuard_RewrittenCommandRuns(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("stub managers are POSIX shell scripts")
	}
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	stubs := t.TempDir()
	record := filepath.Join(t.TempDir(), "calls.log")
	for _, mgr := range []string{"npm", "pip"} {
		// Builtins only: nothing else is on the restricted PATH.
		script := "#!" + bashPath + "\nprintf '%s' \"${0##*/}\" >> \"$PG_RECORD\"\nprintf ' [%s]' \"$@\" >> \"$PG_RECORD\"\necho >> \"$PG_RECORD\"\n"
		if err := os.WriteFile(filepath.Join(stubs, mgr), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		command string
		want    string
	}{
		{"npm install left-pad", "npm [install] [--ignore-scripts] [left-pad]"},
		{"pip install requests==2.31.0", "pip [install] [--only-binary] [:all:] [requests==2.31.0]"},
	}
	for _, tc := range cases {
		res := runPackageGuard(t, "Bash", map[string]any{"command": tc.command})
		rewritten := res.rewritten()
		if rewritten == "" {
			t.Fatalf("%q was not rewritten (result %+v)", tc.command, res)
		}
		_ = os.Remove(record)
		cmd := exec.Command(bashPath, "-c", rewritten)
		cmd.Env = []string{"PATH=" + stubs, "PG_RECORD=" + record}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("rewritten command %q failed: %v\n%s", rewritten, err, out)
		}
		got, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(got)) != tc.want {
			t.Errorf("%q ran as %q, want %q", rewritten, strings.TrimSpace(string(got)), tc.want)
		}
	}
}

// TestPackageGuard_BaseBehaviourPinned pins the verdicts the guard gave for
// task-runner wrappers, build scripts and unverifiable managers before the
// U17-WS2 detection work, so each slice of that work can only add detections:
// none of these may change ("allow" means the hook stays silent).
func TestPackageGuard_BaseBehaviourPinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		command      string
		want         string
		reasonHas    string
		rewrittenHas string
	}{
		{command: "mise exec -- npm install left-pad", want: "ask",
			rewrittenHas: "mise exec -- npm install --ignore-scripts left-pad"},
		{command: "asdf exec npm install left-pad", want: "ask",
			rewrittenHas: "asdf exec npm install --ignore-scripts left-pad"},
		{command: "pixi run pnpm add left-pad", want: "allow"},
		{command: "mise run install", want: "allow"},
		{command: "asdf install", want: "allow"},
		{command: "./gradlew build", want: "allow"},
		{command: "sh gradlew build", want: "allow"},
		{command: "./configure --prefix=/usr", want: "allow"},
		{command: "bash x.sh", want: "allow"},
		{command: "./x.sh", want: "allow"},
		{command: "carton exec -- npm i evil-new", want: "allow"},
		{command: "cabal install x", want: "ask", reasonHas: "cannot verify"},
		{command: "conda install x", want: "ask", reasonHas: "cannot verify"},
		{command: "luarocks install x", want: "deny", reasonHas: "`luarocks` installs packages"},
		{command: "dart pub add x", want: "deny", reasonHas: "`dart` installs packages"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()
			res := runPackageGuard(t, "Bash", map[string]any{"command": tc.command})
			if got := res.decision(); got != tc.want {
				t.Fatalf("decision = %q, want %q (result %+v)", got, tc.want, res)
			}
			reason := res.Stderr
			if res.Output != nil {
				reason += res.Output.PermissionDecisionReason
			}
			if !strings.Contains(reason, tc.reasonHas) {
				t.Errorf("reason %q does not mention %q", reason, tc.reasonHas)
			}
			if got := res.rewritten(); tc.rewrittenHas == "" && got != "" {
				t.Errorf("rewritten = %q, want no rewrite", got)
			} else if !strings.Contains(got, tc.rewrittenHas) {
				t.Errorf("rewritten command %q does not contain %q", got, tc.rewrittenHas)
			}
		})
	}
}

// gitBlobID is the object id git gives content as a blob, in a repository
// whose object format uses h (sha1 or sha256).
func gitBlobID(h hash.Hash, content []byte) string {
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// gitLsTreeEnv is the PG_GIT_LSTREE setting that makes the driver's git stub
// report entries (path -> "<mode> blob <id>", "error" or "timeout").
func gitLsTreeEnv(t *testing.T, entries map[string]string) string {
	t.Helper()
	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	return "PG_GIT_LSTREE=" + string(raw)
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPackageGuard_LocalArchives covers U17-10a: an install of a local
// archive the guard cannot tie to an unmodified committed file of the project
// asks, because the archive's contents (and lifecycle scripts) are unchecked.
// An archive git reports as committed with exactly these bytes keeps the base
// behaviour. git is the driver's stub, so no host git runs.
func TestPackageGuard_LocalArchives(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	committed := []byte("committed archive bytes")
	secret := []byte("sk-ant-" + "SECRET")
	writeTestFile(t, filepath.Join(dir, "vendor", "y.tgz"), committed)
	writeTestFile(t, filepath.Join(dir, "x.tgz"), []byte("untracked"))
	writeTestFile(t, filepath.Join(dir, "x.gem"), []byte("untracked"))
	writeTestFile(t, filepath.Join(dir, "x.whl"), []byte("untracked"))
	writeTestFile(t, filepath.Join(dir, "s.tgz"), secret)
	sha1ID := gitBlobID(sha1.New(), committed)
	sha256ID := gitBlobID(sha256.New(), committed)
	const y = "./vendor/y.tgz"

	cases := []struct {
		name         string
		command      string
		tree         map[string]string
		want         string
		reasonHas    string
		reasonLacks  string
		rewrittenHas string
	}{
		{name: "untracked npm archive with --ignore-scripts", command: "npm install --ignore-scripts ./x.tgz",
			want: "ask", reasonHas: "local archive ./x.tgz"},
		{name: "untracked gem", command: "gem install ./x.gem", want: "ask", reasonHas: "local archive ./x.gem"},
		{name: "untracked wheel", command: "pip install --only-binary :all: ./x.whl",
			want: "ask", reasonHas: "local archive ./x.whl"},
		{name: "untracked file: specifier", command: "npm install --ignore-scripts file:./x.tgz",
			want: "ask", reasonHas: "local archive ./x.tgz"},
		{name: "archive outside the repository", command: "npm install --ignore-scripts /tmp/evil.tgz",
			want: "ask", reasonHas: "local archive /tmp/evil.tgz"},
		{name: "committed sha1 archive with --ignore-scripts", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "100644 blob " + sha1ID}, want: "allow"},
		{name: "committed sha256 archive with --ignore-scripts", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "100644 blob " + sha256ID}, want: "allow"},
		{name: "committed executable archive", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "100755 blob " + sha1ID}, want: "allow"},
		{name: "committed archive keeps the base flag rewrite", command: "npm install " + y,
			tree: map[string]string{y: "100644 blob " + sha1ID}, want: "ask", reasonLacks: "local archive",
			rewrittenHas: "npm install --ignore-scripts " + y},
		{name: "modified archive", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "100644 blob " + gitBlobID(sha1.New(), []byte("other"))},
			want: "ask", reasonHas: "local archive " + y},
		{name: "symlink in the tree", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "120000 blob " + sha1ID}, want: "ask", reasonHas: "local archive " + y},
		{name: "git error", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "error"}, want: "ask", reasonHas: "local archive " + y},
		{name: "git timeout", command: "npm install --ignore-scripts " + y,
			tree: map[string]string{y: "timeout"}, want: "ask", reasonHas: "local archive " + y},
		{name: "reason never carries file content", command: "npm install --ignore-scripts ./s.tgz",
			tree: map[string]string{"./s.tgz": "100644 blob " + sha1ID}, want: "ask",
			reasonHas: "local archive ./s.tgz", reasonLacks: "SECRET"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": tc.command}, gitLsTreeEnv(t, tc.tree))
			if got := res.decision(); got != tc.want {
				t.Fatalf("decision = %q, want %q (result %+v)", got, tc.want, res)
			}
			if !strings.Contains(res.reason(), tc.reasonHas) {
				t.Errorf("reason %q does not mention %q", res.reason(), tc.reasonHas)
			}
			if tc.reasonLacks != "" && strings.Contains(res.reason(), tc.reasonLacks) {
				t.Errorf("reason %q mentions %q", res.reason(), tc.reasonLacks)
			}
			if !strings.Contains(res.rewritten(), tc.rewrittenHas) {
				t.Errorf("rewritten command %q does not contain %q", res.rewritten(), tc.rewrittenHas)
			}
			if len(res.Subprocesses) != 1 {
				t.Errorf("subprocesses = %q, want exactly one git ls-tree", res.Subprocesses)
			}
		})
	}
}

// TestPackageGuard_LocalArchivesUncheckable covers the local archives the
// guard cannot tie to one committed file before the command runs: a glob
// (`./dist/*.tgz` after `npm pack`), whose matches only the shell knows; an
// archive the same command line also names elsewhere (literally, quoted in
// pieces, or through a glob or brace pattern), which an earlier segment may
// overwrite; and a relative archive after a directory change, which may be
// another file than the one at that path from the hook's cwd. All ask
// without starting git, even when git would report the path as committed.
func TestPackageGuard_LocalArchivesUncheckable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	committed := []byte("committed archive bytes")
	writeTestFile(t, filepath.Join(dir, "vendor", "y.tgz"), committed)
	writeTestFile(t, filepath.Join(dir, "x.tgz"), committed)
	writeTestFile(t, filepath.Join(dir, "sub", "x.tgz"), []byte("EVIL"))
	writeTestFile(t, filepath.Join(dir, "sub", "z.tgz"), []byte("EVIL"))
	id := "100644 blob " + gitBlobID(sha1.New(), committed)
	tree := gitLsTreeEnv(t, map[string]string{"./vendor/y.tgz": id, "./x.tgz": id, "./z.tgz": id})
	const glob = "local files matched by the glob"
	const elsewhere = "local archive ./vendor/y.tgz, which the same command also names elsewhere"
	const chdir = "after the command changes directory"
	cases := []struct {
		tool      string
		command   string
		reasonHas string
	}{
		{"Bash", "cp /tmp/evil.tgz vendor/y.tg* && npm install --ignore-scripts ./vendor/y.tgz", elsewhere},
		{"Bash", `cp /tmp/evil.tgz "vendor/y.t"gz && npm install --ignore-scripts ./vendor/y.tgz`, elsewhere},
		{"Bash", "cp /tmp/evil.tgz vendor/y.{tgz,bak} && npm install --ignore-scripts ./vendor/y.tgz", elsewhere},
		{"Bash", "cp /tmp/evil.tgz vendor/* && npm install --ignore-scripts ./vendor/y.tgz", elsewhere},
		{"Bash", "cd sub && npm install --ignore-scripts ./x.tgz", chdir + " (cd)"},
		{"Bash", "env -C sub npm install --ignore-scripts ./x.tgz", chdir + " (env)"},
		{"Bash", "env --chdir=sub npm install --ignore-scripts ./x.tgz", chdir},
		{"Bash", "pushd sub; pip install --only-binary :all: ./z.tgz", chdir + " (pushd)"},
		{"Bash", "(cd sub; npm install --ignore-scripts ./x.tgz)", chdir},
		{"Bash", "bash -c 'cd sub && npm install --ignore-scripts ./x.tgz'", chdir},
		{"PowerShell", "Set-Location sub; npm install --ignore-scripts ./x.tgz", chdir + " (Set-Location)"},
		{"PowerShell", "sl sub; npm install --ignore-scripts ./x.tgz", chdir + " (sl)"},
		{"Bash", "npm install --ignore-scripts ./dist/*.tgz", glob + " ./dist/*.tgz"},
		{"Bash", "npm install --ignore-scripts ./dist/evil-?.0.tgz", glob},
		{"Bash", "npm install --ignore-scripts ./dist/evil-1.0.tg[z]", glob},
		{"Bash", "npm install --ignore-scripts ./dist/evil-1.0.t[a-z]z", glob},
		{"Bash", "pip install --only-binary :all: ./dist/*.whl", glob + " ./dist/*.whl"},
		{"Bash", "gem install ./dist/*.gem", glob},
		{"Bash", "cp /tmp/evil.tgz vendor/y.tgz && npm install --ignore-scripts ./vendor/y.tgz", elsewhere},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()
			res := runPackageGuardIn(t, dir, tc.tool, map[string]any{"command": tc.command}, tree)
			if got := res.decision(); got != "ask" || !strings.Contains(res.reason(), tc.reasonHas) {
				t.Fatalf("decision = %q, reason %q; want ask mentioning %q", got, res.reason(), tc.reasonHas)
			}
			if len(res.Subprocesses) != 0 {
				t.Errorf("subprocesses = %q, want none", res.Subprocesses)
			}
		})
	}
	// Without a directory change, quoting or an unrelated glob, the committed
	// archive keeps the base verdict after one git ls-tree.
	for _, command := range []string{
		"npm install --ignore-scripts ./x.tgz",
		"env X=1 npm install --ignore-scripts ./x.tgz",
		"npm install --ignore-scripts './vendor/y.tgz'",
		"ls *.md && npm install --ignore-scripts ./vendor/y.tgz",
	} {
		res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": command}, tree)
		if got := res.decision(); got != "allow" || len(res.Subprocesses) != 1 {
			t.Errorf("%s: decision = %q, subprocesses %q; want allow after one git ls-tree", command, got, res.Subprocesses)
		}
	}
	// A pip extras group is not a glob that can match an archive.
	if res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": "pip install ./pkg[dev]"}); strings.Contains(res.reason(), glob) {
		t.Errorf("pip install ./pkg[dev]: reason %q treats the extras as a glob", res.reason())
	}
}

// TestPackageGuard_ArchiveCheckCallCount guards the cost of the archive check
// on every OS: an install with no local archive starts no process, and one
// archive costs exactly one lock-free git ls-tree.
func TestPackageGuard_ArchiveCheckCallCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": "npm install left-pad"}); len(res.Subprocesses) != 0 {
		t.Errorf("npm install left-pad started %q, want no subprocess", res.Subprocesses)
	}
	res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": "npm install ./x.tgz"})
	if len(res.Subprocesses) != 1 {
		t.Fatalf("npm install ./x.tgz started %q, want exactly one", res.Subprocesses)
	}
	argv := res.Subprocesses[0]
	if !slices.Contains(argv, "--no-optional-locks") || !slices.Contains(argv, "ls-tree") {
		t.Errorf("archive check argv %q lacks --no-optional-locks or ls-tree", argv)
	}
}

// TestPackageGuard_ArchiveFIFO: an archive operand that is a FIFO must not
// hang the hook on open (a hang would end in the watchdog's exit 2, not ask),
// even when git reports a committed file at that path.
func TestPackageGuard_ArchiveFIFO(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are POSIX-only")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping package-guard end-to-end test")
	}
	dir := t.TempDir()
	if out, err := exec.Command(python, "-c", "import os, sys; os.mkfifo(sys.argv[1])",
		filepath.Join(dir, "x.tgz")).CombinedOutput(); err != nil {
		t.Fatalf("mkfifo: %v\n%s", err, out)
	}
	tree := gitLsTreeEnv(t, map[string]string{"./x.tgz": "100644 blob " + gitBlobID(sha1.New(), nil)})
	res := runPackageGuardIn(t, dir, "Bash", map[string]any{"command": "npm install --ignore-scripts ./x.tgz"}, tree)
	if got := res.decision(); got != "ask" || !strings.Contains(res.reason(), "local archive ./x.tgz") {
		t.Fatalf("decision = %q, reason %q; want ask about the local archive", got, res.reason())
	}
}

// psVerdict is one shell-tool command and the verdict the package guard must
// give it with PG_VULN=evil ("allow" means the hook stays silent; "a|b"
// accepts either).
type psVerdict struct {
	tool      string
	command   string
	want      string
	reasonHas string
	// noLookups: decided before any registry lookup.
	noLookups bool
	// parity: the same text handed to pwsh through the Bash tool (as -c and
	// as -EncodedCommand) must get the same verdict.
	parity bool
}

// posixQuote single-quotes s for a POSIX shell.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func checkPSVerdict(t *testing.T, tool, command string, tc psVerdict) {
	t.Helper()
	res := runPackageGuard(t, tool, map[string]any{"command": command}, "PG_VULN=evil")
	if got := res.decision(); !slices.Contains(strings.Split(tc.want, "|"), got) {
		t.Fatalf("[%s] %s: decision = %q, want %q (result %+v)", tool, command, got, tc.want, res)
	}
	if !strings.Contains(res.reason(), tc.reasonHas) {
		t.Errorf("[%s] %s: reason %q does not mention %q", tool, command, res.reason(), tc.reasonHas)
	}
	if tc.noLookups && len(res.Lookups) != 0 {
		t.Errorf("[%s] %s: made registry lookups %+v, want none", tool, command, res.Lookups)
	}
}

func runPSVerdicts(t *testing.T, cases []psVerdict) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.tool+"/"+tc.command, func(t *testing.T) {
			t.Parallel()
			checkPSVerdict(t, tc.tool, tc.command, tc)
			if !tc.parity {
				return
			}
			checkPSVerdict(t, "Bash", "pwsh -c "+posixQuote(tc.command), tc)
			checkPSVerdict(t, "Bash", "pwsh -EncodedCommand "+utf16LEBase64(tc.command), tc)
		})
	}
}

// TestPackageGuard_PowerShellTool covers U17-02 (XS-WS4 D2): a PowerShell
// tool command is read in PowerShell's own dialect, so its installer
// cmdlets, Invoke-Expression strings and Start-Process command lines are
// judged, and a script handed to pwsh through the Bash tool gets the same
// verdict.
func TestPackageGuard_PowerShellTool(t *testing.T) {
	t.Parallel()
	const ghsa = "GHSA-stub-0001"
	const cannotSee = "Invoke-Expression runs text the guard cannot see"
	const unvalidated = "installs packages the package guard cannot validate"
	ps := func(command, want, reasonHas string) psVerdict {
		return psVerdict{tool: "PowerShell", command: command, want: want, reasonHas: reasonHas}
	}
	cmdlet := func(command, name string) psVerdict {
		tc := ps(command, "deny", "`"+name+"` installs packages the package guard cannot validate")
		tc.noLookups = true
		return tc
	}
	parity := func(tc psVerdict) psVerdict {
		tc.parity = true
		return tc
	}
	runPSVerdicts(t, []psVerdict{
		cmdlet("Install-Module -Name EvilMod -Force", "install-module"),
		cmdlet("Install-PSResource EvilMod", "install-psresource"),
		cmdlet(`PowerShellGet\Install-Module Evil`, "install-module"),
		cmdlet("Find-Module X | Install-Module", "install-module"),
		cmdlet("isres Evil", "isres"),
		cmdlet("npm install lodash; Install-Module Evil", "install-module"),
		// A tab after the cmdlet, which the PowerShell permission rules'
		// whole-word match misses (W1N-29): the guard is the backstop.
		cmdlet("Install-Module\tEvil", "install-module"),
		cmdlet("Install-Module\t-Name\tEvil", "install-module"),
		// PowerShellGet's exported aliases and PackageManagement's Save-Package.
		cmdlet("inmo Evil", "inmo"),
		cmdlet("upmo Evil", "upmo"),
		cmdlet("Save-Package Evil", "save-package"),

		// PowerShell reads typographic quotes as quotes and an en dash, em
		// dash or horizontal bar as a parameter's dash.
		parity(ps("iex “npm install evil”", "deny", ghsa)),
		parity(ps("iex ‘npm install evil’", "deny", ghsa)),
		parity(ps("Start-Process npm “install evil”", "deny", ghsa)),
		parity(ps("Start-Process npm -ArgumentList ‘install evil’", "deny", ghsa)),
		parity(ps("Start-Process npm –ArgumentList 'install evil'", "deny", ghsa)),
		parity(ps("Start-Process –FilePath npm –ArgumentList 'install evil'", "deny", ghsa)),
		parity(ps("Start-Process npm —ArgumentList 'install','evil'", "deny", ghsa)),
		parity(ps("Start-Process npm ―ArgumentList 'install evil'", "deny", ghsa)),
		parity(ps("Start-Process pwsh –ArgumentList '-c','npm install evil'", "deny", ghsa)),
		parity(ps("iex „npm install evil”", "deny", ghsa)),
		parity(ps("& ‘Install-Module’ Evil", "deny", unvalidated)),
		// A method call handed an install command line asks.
		parity(ps(`[Diagnostics.Process]::Start('npm', 'install evil')`, "ask", "method call")),
		// A command name followed by a group is a command, not a method call.
		parity(ps(`npm.cmd('install')`, "ask", "computed by the shell")),

		parity(ps(`iex "npm install evil"`, "deny", ghsa)),
		parity(ps(`Invoke-Expression "npm install evil"`, "deny", ghsa)),
		parity(ps(`Invoke-Expression -Command:'npm install evil'`, "deny", ghsa)),
		parity(ps(`Start-Process npm -ArgumentList "install","evil"`, "deny", ghsa)),
		parity(ps(`Start-Process -FilePath npm -ArgumentList 'install evil'`, "deny", ghsa)),
		parity(ps(`Start-Process npm -ArgumentList @('install','evil')`, "deny", ghsa)),
		parity(ps(`Start-Process npm -ArgumentList @('install', 'evil')`, "deny", ghsa)),
		parity(ps(`Start-Process npm @('install','evil')`, "deny", ghsa)),
		parity(ps(`Start-Process npm -ArgumentList ('install','evil')`, "deny", ghsa)),
		parity(ps(`Start-Process npm -ArgumentList 'install', 'evil'`, "deny", ghsa)),
		parity(ps(`saps npm -Args 'install','evil'`, "deny", ghsa)),
		parity(ps(`start -FilePath:npm -ArgumentList:'install evil' -Wait`, "deny", ghsa)),
		parity(ps(`iex $cmd`, "ask", cannotSee)),
		parity(ps(`'npm i x' | iex`, "ask", cannotSee)),
		parity(ps(`Start-Process npm @($pkgs)`, "ask", "Start-Process")),

		// The first positional is -FilePath wherever a named -ArgumentList
		// appears.
		parity(ps(`Start-Process -ArgumentList 'install evil' npm`, "deny", ghsa)),
		parity(ps(`Start-Process -Args 'install','evil' npm`, "deny", ghsa)),
		parity(ps(`Start-Process -NoNewWindow -ArgumentList 'install evil' npm`, "deny", ghsa)),
		parity(ps(`saps -a 'install','evil' npm`, "deny", ghsa)),
		parity(ps(`Start-Process -ArgumentList 'install','evil' 'npm'`, "deny", ghsa)),
		// A program computed by a group expression cannot be read.
		parity(ps(`Start-Process (Get-Command npm).Source 'install evil'`, "ask", "Start-Process")),

		// Assignments, dot-sourcing and script-block arguments run the
		// command they hold.
		parity(ps(`$null = Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`$r=Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`[string]$r = Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`. Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`. iex 'npm install evil'`, "deny", ghsa)),
		parity(ps(`. npm install evil`, "deny", ghsa)),
		parity(ps(`Invoke-Command -ScriptBlock { Install-Module Evil }`, "deny", unvalidated)),
		parity(ps(`1 | % { Install-Module Evil }`, "deny", unvalidated)),
		parity(ps(`ForEach-Object { Install-Module Evil }`, "deny", unvalidated)),
		parity(ps(`. { Install-Module Evil }`, "deny", unvalidated)),
		ps(`Write-Host Install-Module`, "ask", "installer cmdlet Install-Module"),

		// Block comments are blanks, and a backtick escapes a plain character.
		parity(ps(`<# x #> npm install evil`, "deny", ghsa)),
		parity(ps(`<# a #> Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`Write-Host x; <# #> Install-Module Evil`, "deny", unvalidated)),
		parity(ps(`iex <# c #> 'npm install evil'`, "deny", ghsa)),
		parity(ps("In`stall-Module Evil", "deny", unvalidated)),
		ps(`<# unterminated npm install evil`, "deny", "could not be safely analyzed"),

		// The dialect keeps an install in a statement block rewritable.
		ps(`& { npm install left-pad }`, "ask", "safety flags inserted"),
		ps(`if (Test-Path x) { npm install left-pad }`, "ask", "safety flags inserted"),

		{tool: "Bash", command: `pwsh -ExecutionPolicy Bypass -Command "npm install evil"`, want: "deny", reasonHas: ghsa},
		{tool: "Bash", command: `pwsh -ep:Bypass -c "npm install evil"`, want: "deny", reasonHas: ghsa},
		{tool: "Bash", command: `pwsh -NoProfile -WindowStyle Hidden -Command "npm install evil"`, want: "deny", reasonHas: ghsa},
		{tool: "Bash", command: "powershell npm install evil", want: "deny", reasonHas: ghsa},
		{tool: "Bash", command: "powershell -NoProfile -ExecutionPolicy Bypass npm install evil", want: "deny", reasonHas: ghsa},
		{tool: "Bash", command: "pwsh -File build.ps1", want: "allow"},
		{tool: "Bash", command: "pwsh -ExecutionPolicy Bypass build.ps1", want: "allow"},
		{tool: "Bash", command: "powershell -File build.ps1", want: "allow"},
	})
}

// TestPackageGuard_PowerShellBasePinned pins the verdicts the guard gave
// PowerShell-shaped commands before U17-02 read PowerShell in its own
// dialect, so the dialect can only add detections: these must not change.
func TestPackageGuard_PowerShellBasePinned(t *testing.T) {
	t.Parallel()
	b64 := utf16LEBase64("Install-Module Evil")
	const unvalidated = "installs packages the package guard cannot validate"
	ps := func(command, want, reasonHas string) psVerdict {
		return psVerdict{tool: "PowerShell", command: command, want: want, reasonHas: reasonHas}
	}
	runPSVerdicts(t, []psVerdict{
		ps("Start-Process pwsh -e '"+b64+"'", "deny", "`pwsh` "+unvalidated),
		ps("Start-Process pwsh '-e' '"+b64+"'", "deny", "`pwsh` "+unvalidated),
		ps(`Start-Process Rscript -e 'install.packages("evil")'`, "deny", "`rscript` "+unvalidated),
		ps(`Start-Process Rscript '-e' 'install.packages("evil")'`, "deny", "`rscript` "+unvalidated),
		ps(`Start-Process bash -c 'npm install evil'`, "deny", "GHSA-stub-0001"),
		ps(`Start-Process bash '-c' 'npm install evil'`, "deny", "GHSA-stub-0001"),
		ps(`Start-Process pwsh '-c Install-Module Evil'`, "deny", "`pwsh` "+unvalidated),
		ps("& { npm install evil }", "deny", "GHSA-stub-0001"),
		ps("pwsh -c Install-Module X", "deny", "`pwsh` "+unvalidated),
		ps(`$x = "npm install evil"; iex $x`, "ask", "computed by the shell"),
		// Base asked (the embedded-install fallback); the nested command
		// line Start-Process now also yields may only make it stricter.
		ps("Start-Process npm install left-pad", "ask|deny", ""),

		ps("Get-Help Install-Module -Full", "allow", ""),
		ps("Get-Command Install-Module -Syntax", "allow", ""),
		ps("Update-ModuleManifest x.psd1", "allow", ""),
		ps(`Write-Host "npm install evil"`, "allow", ""),
		ps("Start-Process notepad", "allow", ""),
		ps("Start-Process $url", "allow", ""),
		ps("Start-Process notepad @($f)", "allow", ""),
		ps("Get-ChildItem; Write-Host install", "allow", ""),
		// Method calls whose literal arguments are short verb-like words.
		ps(`$x.Replace('a','b')`, "allow", ""),
		ps(`$list.Add('x')`, "allow", ""),
		ps(`[IO.Path]::Combine('a','b')`, "allow", ""),
		ps(`[System.IO.File]::ReadAllText('x')`, "allow", ""),
		ps(`[math]::Round(1.5)`, "allow", ""),
		ps(`(Get-Date).ToString('yyyy')`, "allow", ""),
		ps("Write-Host “it’s fine”", "allow", ""),
		ps("Get-Content inmo.txt", "allow", ""),
		{tool: "Bash", command: "iex -S mix", want: "allow"},
		{tool: "Bash", command: "iex", want: "allow"},
		{tool: "Bash", command: "start foo", want: "allow"},
	})
}

// TestPackageGuard_InlineInterpreterCode covers U17-12a: inline interpreter
// code (python -c, node -e, ruby -e, perl -e, option clusters such as -Ic or
// -wle) that runs a package manager install asks, because the guard cannot
// validate what the program text installs. The check only adds detections:
// a base deny (validated trailing argv, R and CPAN installers) and the
// verdicts for computed command names are unchanged.
func TestPackageGuard_InlineInterpreterCode(t *testing.T) {
	t.Parallel()
	const inline = "inline"
	bash := func(command, want, reasonHas string) psVerdict {
		return psVerdict{tool: "Bash", command: command, want: want, reasonHas: reasonHas}
	}
	runPSVerdicts(t, []psVerdict{
		bash(`python3 -c 'import os; os.system("pip install evil")'`, "ask", "inline python code"),
		bash(`python3 -c 'import subprocess; subprocess.run(["pip","install","x"])'`, "ask", inline),
		bash(`python3 -c 'import pip; pip.main(["install", "x"])'`, "ask", inline),
		bash(`node -e "require('child_process').execSync('npm i evil')"`, "ask", "inline node code"),
		bash(`node --eval="require('child_process').execSync('npm install evil')"`, "ask", inline),
		bash(`ruby -e 'system("gem install x")'`, "ask", "inline ruby code"),
		bash(`perl -wle 'system("cpanm Foo")'`, "ask", "inline perl code"),
		bash(`python3 -Ic 'import os; os.system("npm install x")'`, "ask", inline),
		bash(`deno eval 'new Deno.Command("npm", {args: ["install", "x"]}).outputSync()'`, "ask", inline),
		bash(`bun -e 'Bun.spawnSync(["bun", "add", "x"])'`, "ask", inline),
		bash(`nodejs -e "require('child_process').execSync('npm i evil')"`, "ask", "inline nodejs code"),
		bash(`py -c "import os; os.system('pip install evil')"`, "ask", "inline py code"),
		bash(`php -r "system('composer require evil/evil');"`, "ask", "inline php code"),
		bash(`python3 -c "import os; os.system('npm\tinstall evil')"`, "ask", inline),
		bash(`python3 -c "import subprocess; subprocess.run(['pwsh', '-c', 'Install-Module Evil'])"`, "ask", inline),

		// A verb counts only after its own manager, and only words that
		// name an installer are managers.
		bash(`node -e "const cmd = 'x'; console.log(cmd)"`, "allow", ""),
		bash(`python3 -c "import cmd; c = cmd.Cmd(); c.cmdloop()"`, "allow", ""),
		bash(`node -e "require('child_process').execSync('npm run build')"`, "allow", ""),

		bash(`python3 -c 'print(1)'`, "allow", ""),
		bash(`python3 -c 'print("install")'`, "allow", ""),
		bash(`node -e "const a = require('./npm-utils')"`, "allow", ""),
		bash(`python3 -c 'import pip; print(pip.__version__)'`, "allow", ""),
		bash(`python3 script.py npm install x`, "allow", ""),
		bash("perl -MCPAN::Meta -e 'print CPAN::Meta->load_file(q(META.json))->version'", "allow", ""),

		// Base verdicts, pinned.
		bash(`perl -e 'exec @ARGV' npm install evil`, "deny", "GHSA-stub-0001"),
		bash(`Rscript -e 'install.packages("x")'`, "deny", "`rscript` installs packages"),
		bash("perl -MCPAN -e 'install Foo'", "deny", "installs packages the package guard cannot validate"),
		bash("x=npm; $x install foo", "ask", "computed by the shell"),
		bash("npm${IFS}install${IFS}evil", "ask", "computed by the shell"),
		bash("$(echo npm) $(echo install) evil", "ask", "computed by the shell"),
	})
}

// TestPackageGuard_ParallelArgumentCommands guards W1N-08: GNU parallel with
// no command word runs each ::: argument as a command line, so an install
// handed to it that way is judged like one typed directly. The forms with a
// command word keep their base verdicts.
func TestPackageGuard_ParallelArgumentCommands(t *testing.T) {
	t.Parallel()
	bash := func(command, want, reasonHas string) psVerdict {
		return psVerdict{tool: "Bash", command: command, want: want, reasonHas: reasonHas}
	}
	runPSVerdicts(t, []psVerdict{
		bash(`parallel ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel -j1 ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel --will-cite ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel ::: 'ls' 'npm i evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel ::: 'npm install evil' ::: x`, "deny", "GHSA-stub-0001"),
		bash(`parallel :::+ 'npm install evil'`, "deny", "GHSA-stub-0001"),
		// A line takes one argument of every source; a replacement-string
		// or sh -c template runs the argument; --env takes a value.
		bash(`parallel ::: npm ::: install ::: evil`, "deny", "GHSA-stub-0001"),
		bash(`parallel ::: 'npm' ::: 'i evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel ::: npm :::+ install :::+ evil`, "deny", "GHSA-stub-0001"),
		bash(`parallel {} ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel '{}' ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel sh -c ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel bash -c {} ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel --env X ::: 'npm install evil'`, "deny", "GHSA-stub-0001"),
		bash(`parallel npm install {1} ::: evil ::: x`, "deny", "GHSA-stub-0001"),

		// Base verdicts, pinned.
		bash(`parallel npm install ::: evil`, "deny", "GHSA-stub-0001"),
		bash(`parallel echo ::: a b`, "allow", ""),
		bash(`parallel ::: ls 'echo hi'`, "allow", ""),
		bash(`parallel :::: cmds.txt`, "allow", ""),
		// Jobs read from stdin or a file are not known; as on base, the
		// hook decides without an error.
		bash(`cat jobs.txt | parallel`, "allow", ""),
		bash(`parallel < jobs.txt`, "allow", ""),
		bash(`parallel -j4 < jobs.txt`, "allow", ""),
		bash(`parallel --version`, "allow", ""),
		bash(`sudo parallel`, "allow", ""),
		{tool: "Monitor", command: `cat jobs.txt | parallel`, want: "allow"},
	})
}

// nixRunServers are tool names Claude Code gives the MCP server's nix_run
// tool: the server is named by the client's MCP configuration.
var nixRunServers = []string{"mcp__qsdev__qsdev_nix_run", "mcp__other__qsdev_nix_run"}

// nixRunCase is one well-formed nix_run tool_input and the package-guard
// decision it gets.
type nixRunCase struct {
	name        string
	installable string
	args        []string
	stdin       string
	want        string
}

func (c nixRunCase) input() map[string]any {
	in := map[string]any{"command": c.installable}
	if c.args != nil {
		in["args"] = c.args
	}
	if c.stdin != "" {
		in["stdin"] = c.stdin
	}
	return in
}

// nixRunCases are the well-formed nix_run calls the guard is tested with.
var nixRunCases = []nixRunCase{
	{name: "bash -c pnpm i evil", installable: "nixpkgs#bash", args: []string{"-c", "pnpm i evil"}, want: "deny"},
	{name: "bash -c registry override", installable: "nixpkgs#bash", args: []string{"-c", "NPM_CONFIG_REGISTRY=http://evil npm ci"}, want: "deny"},
	{name: "pnpm add evil", installable: "nixpkgs#pnpm", args: []string{"add", "evil"}, want: "deny"},
	{name: "bash reads an install from stdin", installable: "nixpkgs#bash", stdin: "npm install evil", want: "deny"},
	{name: "busybox sh -c", installable: "nixpkgs#busybox", args: []string{"sh", "-c", "npm i evil"}, want: "deny"},
	{name: "clean package through bash -c", installable: "nixpkgs#bash", args: []string{"-c", "npm install left-pad"}, want: "deny"},
	{name: "clean package run directly", installable: "nixpkgs#nodejs.npm", args: []string{"install", "left-pad"}, want: "deny"},
	{name: "-c after options", installable: "github:x/y#pkgs.bash^out", args: []string{"-o", "errexit", "-c", "--", "npm i evil"}, want: "deny"},
	// An attribute with a version or variant suffix runs the same manager.
	{name: "pnpm_9 add evil", installable: "nixpkgs#pnpm_9", args: []string{"add", "evil"}, want: "deny"},
	{name: "pnpm_8 add evil", installable: "nixpkgs#pnpm_8", args: []string{"add", "evil"}, want: "deny"},
	{name: "yarn-berry add evil", installable: "nixpkgs#yarn-berry", args: []string{"add", "evil"}, want: "deny"},
	{name: "nodejs_22 -e npm i", installable: "nixpkgs#nodejs_22", args: []string{"-e", "require('child_process').execSync('npm i evil')"}, want: "deny"},
	{name: "versioned attr under a set", installable: "nixpkgs#nodePackages.pnpm_9^out", args: []string{"add", "evil"}, want: "deny"},
	{name: "jq filter", installable: "nixpkgs#jq", args: []string{"."}, want: "allow"},
	{name: "versioned tool, no install", installable: "nixpkgs#nodejs_22", args: []string{"--version"}, want: "allow"},
	{name: "jq filter over stdin", installable: "nixpkgs#jq", args: []string{"."}, stdin: `{"a": 1}`, want: "allow"},
	{name: "bash -c ls", installable: "nixpkgs#bash", args: []string{"-c", "ls"}, want: "allow"},
	{name: "hello", installable: "nixpkgs#hello", want: "allow"},
	// parallel reading its jobs from stdin: its args say nothing, stdin is
	// judged as a line.
	{name: "parallel -j4", installable: "nixpkgs#parallel", args: []string{"-j4"}, want: "allow"},
	{name: "parallel with jobs on stdin", installable: "nixpkgs#parallel", stdin: "npm i evil", want: "deny"},
	{name: "parallel with clean stdin", installable: "nixpkgs#parallel", stdin: "ls", want: "allow"},
}

// TestPackageGuard_NixRunTool runs the MCP server's nix_run tool through
// package-guard: an install in the program's args, a -c script or stdin is
// denied with no registry lookup and no updatedInput (a rewrite would
// replace the installable), even for a clean package; a call with no install
// gets no decision; a malformed tool_input fails closed.
func TestPackageGuard_NixRunTool(t *testing.T) {
	t.Parallel()
	for _, tool := range nixRunServers {
		for _, tc := range nixRunCases {
			t.Run(tool+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				res := runPackageGuard(t, tool, tc.input(), "PG_VULN=evil")
				if got := res.decision(); got != tc.want {
					t.Fatalf("decision = %q, want %q (result %+v)", got, tc.want, res)
				}
				if len(res.Lookups) != 0 {
					t.Errorf("registry lookups %+v, want none", res.Lookups)
				}
				if res.Output != nil && res.Output.UpdatedInput != nil {
					t.Errorf("updatedInput %+v, want none", res.Output.UpdatedInput)
				}
				if tc.want == "deny" && !strings.Contains(res.reason(), "cannot be validated or hardened; run them through Bash") {
					t.Errorf("reason %q does not send the install to Bash", res.reason())
				}
			})
		}
		malformed := []struct {
			name  string
			input any
		}{
			{"args a string", map[string]any{"command": "nixpkgs#bash", "args": "-c npm i evil"}},
			{"args holds a number", map[string]any{"command": "nixpkgs#bash", "args": []any{"-c", 5}}},
			{"command a number", map[string]any{"command": 5}},
			{"stdin a list", map[string]any{"command": "nixpkgs#bash", "stdin": []string{"npm i evil"}}},
			{"tool_input a string", "nixpkgs#bash -c 'npm i evil'"},
			// Falsy values of the wrong type are malformed too; only an
			// absent or null field is empty.
			{"args false", map[string]any{"command": "nixpkgs#bash", "args": false}},
			{"args an empty object", map[string]any{"command": "nixpkgs#bash", "args": map[string]any{}}},
			{"stdin 0", map[string]any{"command": "nixpkgs#bash", "args": []string{}, "stdin": 0}},
			{"command false", map[string]any{"command": false}},
		}
		for _, tc := range malformed {
			t.Run(tool+"/malformed/"+tc.name, func(t *testing.T) {
				t.Parallel()
				res := runPackageGuardIn(t, "", tool, tc.input, "PG_VULN=evil")
				if got := res.decision(); got != "deny" {
					t.Fatalf("decision = %q, want deny (result %+v)", got, res)
				}
			})
		}
	}
	// Not the nix_run tool: no decision, whatever it carries.
	for _, tool := range []string{"mcp____qsdev_nix_run", "qsdev_nix_run", "mcp__x__qsdev_nix_runner", "mcp__x__other"} {
		res := runPackageGuard(t, tool, map[string]any{"command": "nixpkgs#pnpm", "args": []string{"add", "evil"}}, "PG_VULN=evil")
		if got := res.decision(); got != "allow" {
			t.Errorf("%s: decision = %q, want no decision (result %+v)", tool, got, res)
		}
	}
}

// TestPackageGuard_NixRunMatchesCommandLines ties the guard's reading of a
// nix_run call to cmdscan.NixRunCommandLines, the Bash lines the server and
// self-protection judge it as: the call is denied exactly when one of those
// lines, other than the literal `nix run` line (whose installable the Bash
// route takes as a nix install), is denied or asked about as a Bash call.
func TestPackageGuard_NixRunMatchesCommandLines(t *testing.T) {
	t.Parallel()
	for _, tc := range nixRunCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lines := cmdscan.NixRunCommandLines(tc.installable, tc.args, tc.stdin)
			if len(lines) == 0 || !strings.HasPrefix(lines[0], "nix run ") {
				t.Fatalf("NixRunCommandLines = %q, want the literal nix run line first", lines)
			}
			refused := false
			for _, line := range lines[1:] {
				switch runPackageGuard(t, "Bash", map[string]any{"command": line}, "PG_VULN=evil").decision() {
				case "deny", "ask":
					refused = true
				}
			}
			got := runPackageGuard(t, nixRunServers[0], tc.input(), "PG_VULN=evil").decision()
			if (got == "deny") != refused {
				t.Errorf("nix_run decision %q, but the Bash lines %q refused = %v", got, lines[1:], refused)
			}
		})
	}
}

// TestPackageGuard_NixRunToolNameMatchesCmdscan keeps the guard's nix_run
// tool name and its recognition of Claude Code's MCP tool names in step with
// cmdscan, which the server and self-protection use.
func TestPackageGuard_NixRunToolNameMatchesCmdscan(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping package-guard nix_run name test")
	}
	path, err := filepath.Abs(filepath.Join("templates", "hooks", "package-guard.py"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{
		"mcp__qsdev__qsdev_nix_run", "mcp__other__qsdev_nix_run", "mcp__my-server__qsdev_nix_run",
		"mcp____qsdev_nix_run", "mcp__x__qsdev_nix_runner", "mcp__x__y__qsdev_nix_run", "mcp__x__",
		"qsdev_nix_run", "mcp_x__qsdev_nix_run", "MCP__x__qsdev_nix_run", "Bash", "",
	}
	namesJSON, err := json.Marshal(names)
	if err != nil {
		t.Fatal(err)
	}
	driver := `import importlib.util, json, os
spec = importlib.util.spec_from_file_location('h', os.environ['HOOK_PATH'])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
names = json.loads(os.environ['NAMES'])
print(json.dumps({'tool': m.NIX_RUN_TOOL, 'is': [m._is_nix_run(n) for n in names]}))`
	cmd := exec.Command(python, "-c", driver)
	cmd.Env = hookEnv(t, "HOOK_PATH="+path, "NAMES="+string(namesJSON))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("driver failed: %v", err)
	}
	var got struct {
		Tool string `json:"tool"`
		Is   []bool `json:"is"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("bad driver output %q: %v", out, err)
	}
	if got.Tool != cmdscan.NixRunTool {
		t.Errorf("NIX_RUN_TOOL = %q, want cmdscan.NixRunTool %q", got.Tool, cmdscan.NixRunTool)
	}
	if len(got.Is) != len(names) {
		t.Fatalf("driver judged %d names, want %d", len(got.Is), len(names))
	}
	for i, name := range names {
		if want := cmdscan.IsNixRunTool(name); got.Is[i] != want {
			t.Errorf("_is_nix_run(%q) = %v, cmdscan.IsNixRunTool = %v", name, got.Is[i], want)
		}
	}
}
