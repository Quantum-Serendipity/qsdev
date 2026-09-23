package claudecode_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
// PG_URL_ERROR, PG_INTERNAL_ERROR).
func runPackageGuard(t *testing.T, tool string, input map[string]any, env ...string) pgResult {
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
	envelope, err := json.Marshal(map[string]any{"tool_name": tool, "tool_input": input})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, driver)
	cmd.Env = append(os.Environ(),
		"PYTHONDONTWRITEBYTECODE=1",
		"PG_PATH="+hook,
		"PG_ENVELOPE="+string(envelope),
		"CLAUDE_PROJECT_DIR="+t.TempDir(),
		"CLAUDE_AUDIT_DIR="+t.TempDir(),
	)
	cmd.Env = append(cmd.Env, env...)
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
			want: "allow", rewrittenHas: "npm install left-pad --ignore-scripts", wantLookupFor: "left-pad"},
		{name: "vulnerable package", tool: "Bash", input: bash("npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "freshly published package", tool: "Bash", input: bash("pip install fresh"), env: []string{"PG_FRESH=fresh"},
			want: "deny", reasonHas: "days ago"},
		{name: "pinned version is checked", tool: "Bash", input: bash("pip install requests==2.31.0"),
			want: "allow", rewrittenHas: "--only-binary :all:", wantLookupFor: "requests"},
		{name: "network failure fails closed", tool: "Bash", input: bash("npm install left-pad"), env: []string{"PG_URL_ERROR=1"},
			want: "deny", reasonHas: "Failing closed"},
		{name: "internal error fails closed", tool: "Bash", input: bash("npm install left-pad"), env: []string{"PG_INTERNAL_ERROR=1"},
			want: "deny"},
		{name: "imperative nix install", tool: "Bash", input: bash("nix-env -iA nixpkgs.hello"), want: "deny", reasonHas: "nix-env"},
		{name: "not an install", tool: "Bash", input: bash("go test ./..."), want: "allow"},
		{name: "compound command rewrites the install segment", tool: "Bash", input: bash("cd web && npm install left-pad && npm test"),
			want: "allow", rewrittenHas: "npm install left-pad --ignore-scripts &&"},
		// W033: the same commands through PowerShell and Monitor.
		{name: "powershell vulnerable package", tool: "PowerShell", input: bash("npm install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor vulnerable package", tool: "Monitor", input: map[string]any{"command": "npm install evil", "description": "d"},
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor rewrite keeps the other input fields", tool: "Monitor",
			input: map[string]any{"command": "npm install left-pad", "description": "watch install"},
			want:  "allow", rewrittenHas: "--ignore-scripts", keepsDescription: "watch install"},
		{name: "powershell npm.cmd launcher", tool: "PowerShell", input: bash("npm.cmd install evil"), env: []string{"PG_VULN=evil"},
			want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "powershell windows path to pip.exe", tool: "PowerShell", input: bash(`C:\Python312\Scripts\pip.exe install evil`),
			env: []string{"PG_VULN=evil"}, want: "deny", reasonHas: "GHSA-stub-0001"},
		{name: "monitor websocket source", tool: "Monitor", input: map[string]any{"ws": map[string]any{"url": "wss://x.example"}, "description": "d"},
			want: "allow"},
		{name: "non-shell tool", tool: "Write", input: map[string]any{"file_path": "x", "content": "npm install evil"},
			env: []string{"PG_VULN=evil"}, want: "allow"},
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
		{"npm install left-pad", "npm [install] [left-pad] [--ignore-scripts]"},
		{"pip install requests==2.31.0", "pip [install] [requests==2.31.0] [--only-binary] [:all:]"},
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
