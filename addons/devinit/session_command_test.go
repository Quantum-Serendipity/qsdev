package devinit

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

const sessionTestPolicy = `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: session-test
rules:
  - id: SESSION-OK
    category: supply-chain
    name: Session bypassable
    severity: medium
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: warn
  - id: ALWAYS-ON
    category: self-protection
    name: Never bypassable
    severity: critical
    bypass_tier: enforce_always
    conditions:
      type: tool_match
      tool_name: Edit
    action:
      type: block
`

// setupSessionTest isolates HOME and the project directory, writes a policy
// with one session-tier and one enforce_always rule, and returns the session
// state path.
func setupSessionTest(t *testing.T, interactive, inAgent bool) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentMarker := ""
	if inAgent {
		agentMarker = "1"
	}
	// The suite itself may run inside an agent session; pin the marker.
	t.Setenv("CLAUDECODE", agentMarker)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".qsdev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".qsdev", "policy.yaml"), []byte(sessionTestPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	orig := sessionAllowInteractive
	t.Cleanup(func() { sessionAllowInteractive = orig })
	sessionAllowInteractive = func(io.Reader) bool { return interactive }

	return filepath.Join(home, ".qsdev", "session-state.json")
}

func runSessionAllowCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := sessionCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"allow"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// TestSessionAllow is the F033 regression: an agent (no terminal) must not be
// able to grant itself a persistent, machine-wide policy bypass, and only
// confirmed, existing, session-tier rules may be bypassed.
func TestSessionAllow(t *testing.T) {
	tests := []struct {
		name        string
		interactive bool
		inAgent     bool
		stdin       string
		args        []string
		wantErr     string
		wantGranted []string
	}{
		{
			name:        "non-interactive caller is refused",
			interactive: false,
			stdin:       "y\n",
			args:        []string{"SESSION-OK"},
			wantErr:     "requires an interactive terminal",
		},
		{
			name:        "caller inside an agent session is refused even on a terminal",
			interactive: true,
			inAgent:     true,
			stdin:       "y\n",
			args:        []string{"SESSION-OK"},
			wantErr:     "agent session",
		},
		{
			name:        "unknown rule is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        []string{"NO-SUCH-RULE"},
			wantErr:     "no such rule",
		},
		{
			name:        "enforce_always rule is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        []string{"ALWAYS-ON"},
			wantErr:     "cannot be bypassed",
		},
		{
			name:        "declined confirmation grants nothing",
			interactive: true,
			stdin:       "n\n",
			args:        []string{"SESSION-OK"},
		},
		{
			name:        "empty confirmation grants nothing",
			interactive: true,
			stdin:       "",
			args:        []string{"SESSION-OK"},
		},
		{
			name:        "confirmed session-tier rule is granted",
			interactive: true,
			stdin:       "yes\n",
			args:        []string{"SESSION-OK"},
			wantGranted: []string{"SESSION-OK"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := setupSessionTest(t, tt.interactive, tt.inAgent)
			out, err := runSessionAllowCmd(t, tt.stdin, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v\n%s", tt.wantErr, err, out)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			got := policy.NewFileSessionStateReader(statePath).SessionOverrides()
			if strings.Join(got, ",") != strings.Join(tt.wantGranted, ",") {
				t.Errorf("granted overrides = %v, want %v", got, tt.wantGranted)
			}
		})
	}
}
