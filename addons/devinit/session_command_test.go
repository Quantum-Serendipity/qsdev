package devinit

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
  - id: COMMAND-OK
    category: supply-chain
    name: Command bypassable
    severity: medium
    bypass_tier: command
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
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

// testSessionID is a Claude Code session ID in the usual UUID form.
const testSessionID = "0f8e2c1a-5b7d-4e3f-9a6b-1c2d3e4f5a6b"

// setupSessionTest isolates HOME and the project directory, writes a policy
// with a session-tier, a command-tier and an enforce_always rule, chdirs into
// the project and returns the session state path and the canonical project
// root.
func setupSessionTest(t *testing.T, interactive, inAgent bool) (statePath, project string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentMarker := ""
	if inAgent {
		agentMarker = "1"
	}
	// The suite itself may run inside an agent session; pin the marker.
	t.Setenv("CLAUDECODE", agentMarker)
	project = canonicalProjectRoot(t.TempDir())
	if err := os.MkdirAll(filepath.Join(project, ".qsdev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".qsdev", "policy.yaml"), []byte(sessionTestPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	orig := humanAtTerminal
	t.Cleanup(func() { humanAtTerminal = orig })
	humanAtTerminal = func(io.Reader) bool { return interactive }

	return filepath.Join(home, ".qsdev", "session-state.json"), project
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

// TestSessionAllow is the F033 and F193 regression: an agent (no terminal)
// must not be able to grant itself a policy bypass; only confirmed, existing,
// bypassable rules may be bypassed; and every grant is bound to one project
// and Claude Code session, with a command-tier rule getting a one-shot token.
func TestSessionAllow(t *testing.T) {
	session := []string{"--session", testSessionID}
	tests := []struct {
		name        string
		interactive bool
		inAgent     bool
		stdin       string
		args        []string
		wantErr     string
		// wantGranted lists "RULE:tier" for every stored grant.
		wantGranted []string
		wantTTL     time.Duration
	}{
		{
			name:        "non-interactive caller is refused",
			interactive: false,
			stdin:       "y\n",
			args:        append([]string{"SESSION-OK"}, session...),
			wantErr:     "requires an interactive terminal",
		},
		{
			name:        "caller inside an agent session is refused even on a terminal",
			interactive: true,
			inAgent:     true,
			stdin:       "y\n",
			args:        append([]string{"SESSION-OK"}, session...),
			wantErr:     "agent session",
		},
		{
			name:        "missing session ID is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        []string{"SESSION-OK"},
			wantErr:     "session",
		},
		{
			name:        "malformed session ID is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        []string{"SESSION-OK", "--session", "abc\u200bdef"},
			wantErr:     "invalid character",
		},
		{
			name:        "unknown rule is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        append([]string{"NO-SUCH-RULE"}, session...),
			wantErr:     "no such rule",
		},
		{
			name:        "enforce_always rule is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        append([]string{"ALWAYS-ON"}, session...),
			wantErr:     "cannot be bypassed",
		},
		{
			name:        "lifetime over the maximum is rejected",
			interactive: true,
			stdin:       "y\n",
			args:        append([]string{"SESSION-OK", "--ttl", "25h"}, session...),
			wantErr:     "at most",
		},
		{
			name:        "declined confirmation grants nothing",
			interactive: true,
			stdin:       "n\n",
			args:        append([]string{"SESSION-OK"}, session...),
		},
		{
			name:        "empty confirmation grants nothing",
			interactive: true,
			stdin:       "",
			args:        append([]string{"SESSION-OK"}, session...),
		},
		{
			name:        "confirmed session-tier rule is granted for the session",
			interactive: true,
			stdin:       "yes\n",
			args:        append([]string{"SESSION-OK"}, session...),
			wantGranted: []string{"SESSION-OK:session"},
			wantTTL:     policy.DefaultSessionGrantTTL,
		},
		{
			name:        "confirmed command-tier rule gets a one-shot token",
			interactive: true,
			stdin:       "yes\n",
			args:        append([]string{"COMMAND-OK", "--ttl", "10m"}, session...),
			wantGranted: []string{"COMMAND-OK:command"},
			wantTTL:     10 * time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath, project := setupSessionTest(t, tt.interactive, tt.inAgent)
			out, err := runSessionAllowCmd(t, tt.stdin, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v\n%s", tt.wantErr, err, out)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}

			grants, _, err := policy.NewFileSessionStateStore(statePath).Grants(time.Now())
			if err != nil {
				t.Fatalf("reading grants: %v", err)
			}
			var got []string
			for _, g := range grants {
				got = append(got, g.RuleID+":"+g.Tier)
				if g.ProjectRoot != project || g.SessionID != testSessionID {
					t.Errorf("grant %s scoped to (%s, %s), want (%s, %s)", g.RuleID, g.ProjectRoot, g.SessionID, project, testSessionID)
				}
				if ttl := g.ExpiresAt.Sub(g.GrantedAt); ttl != tt.wantTTL {
					t.Errorf("grant %s lifetime = %s, want %s", g.RuleID, ttl, tt.wantTTL)
				}
			}
			if strings.Join(got, ",") != strings.Join(tt.wantGranted, ",") {
				t.Errorf("granted = %v, want %v", got, tt.wantGranted)
			}
		})
	}
}

// TestSessionAllow_ProjectFlag binds the grant to the project named by
// --project, resolved to its qsdev project root, instead of the current
// directory's project.
func TestSessionAllow_ProjectFlag(t *testing.T) {
	statePath, project := setupSessionTest(t, true, false)
	sub := filepath.Join(project, "internal", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	out, err := runSessionAllowCmd(t, "y\n", "SESSION-OK", "--session", testSessionID, "--project", sub)
	if err != nil {
		t.Fatalf("session allow: %v\n%s", err, out)
	}
	grants, _, err := policy.NewFileSessionStateStore(statePath).Grants(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].ProjectRoot != project {
		t.Errorf("grants = %+v, want one grant for %s", grants, project)
	}
}

func TestWriteSessionGrants(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	scope := policy.BypassScope{ProjectRoot: "/work/p", SessionID: testSessionID}

	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
		want  []string
	}{
		{
			name: "no state",
			want: []string{"No active session bypasses"},
		},
		{
			name: "scoped grants",
			setup: func(t *testing.T, path string) {
				var grants []policy.BypassGrant
				for id, tier := range map[string]policy.BypassTier{"S-1": policy.Session, "C-1": policy.Command} {
					g, err := policy.NewBypassGrant(id, tier, scope, time.Hour, now)
					if err != nil {
						t.Fatal(err)
					}
					grants = append(grants, g)
				}
				if err := policy.NewFileSessionStateStore(path).AddGrants(grants, now); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{"S-1 (session) project /work/p, session " + testSessionID, "C-1 (one-shot)"},
		},
		{
			name: "legacy unscoped overrides are reported as ignored",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte(`{"sessionBypassOverrides":["CG-001"]}`), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{"no longer apply): CG-001", "No active session bypasses"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session-state.json")
			if tt.setup != nil {
				tt.setup(t, path)
			}
			var out bytes.Buffer
			if err := writeSessionGrants(&out, policy.NewFileSessionStateStore(path), now); err != nil {
				t.Fatalf("writeSessionGrants: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output %q does not contain %q", out.String(), want)
				}
			}
		})
	}
}
