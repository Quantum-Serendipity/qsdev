package devinit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

// bypassEnforcePolicy has a session-tier rule blocking Bash and a
// command-tier rule blocking Write.
const bypassEnforcePolicy = `apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: bypass-test
rules:
  - id: S-BASH
    category: supply-chain
    name: Bash gated per session
    severity: high
    bypass_tier: session
    conditions:
      type: tool_match
      tool_name: Bash
    action:
      type: block
      message: "Bash is gated"
  - id: C-WRITE
    category: supply-chain
    name: Write gated per call
    severity: high
    bypass_tier: command
    conditions:
      type: tool_match
      tool_name: Write
    action:
      type: block
      message: "Write is gated"
`

func sessionPayload(tool, sessionID string) string {
	input := map[string]string{"command": "ls"}
	if tool == "Write" {
		input = map[string]string{"file_path": "notes.txt", "content": "x"}
	}
	data, _ := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"tool_name":       tool,
		"tool_input":      input,
	})
	return string(data)
}

// TestRunEnforce_BypassGrantScope is the F193 end-to-end regression through
// the hook: a grant lifts its rule only in the project and Claude Code session
// it names and only until it expires; a command-tier grant lifts one call; an
// unscoped override left by an older release lifts nothing.
func TestRunEnforce_BypassGrantScope(t *testing.T) {
	type call struct {
		tool, session string
		wantCode      int
		wantMsg       string
	}
	tests := []struct {
		name string
		// grants are issued at grantAt for the test project unless otherScope.
		grants     map[string]policy.BypassTier
		grantAt    time.Duration // offset from now
		otherScope bool
		legacy     bool
		calls      []call
	}{
		{
			name: "no grant blocks and names the bypass command",
			calls: []call{
				{tool: "Bash", session: "sess-a", wantCode: 2, wantMsg: "session allow S-BASH --session sess-a"},
				{tool: "Write", session: "sess-a", wantCode: 2, wantMsg: "for one call"},
			},
		},
		{
			name:   "session grant lifts the rule for its session only",
			grants: map[string]policy.BypassTier{"S-BASH": policy.Session},
			calls: []call{
				{tool: "Bash", session: "sess-a", wantCode: 0},
				{tool: "Bash", session: "sess-a", wantCode: 0},
				{tool: "Bash", session: "sess-b", wantCode: 2},
				{tool: "Bash", session: "", wantCode: 2},
			},
		},
		{
			name:       "grant for another project does not apply",
			grants:     map[string]policy.BypassTier{"S-BASH": policy.Session},
			otherScope: true,
			calls:      []call{{tool: "Bash", session: "sess-a", wantCode: 2}},
		},
		{
			name:    "expired grant does not apply",
			grants:  map[string]policy.BypassTier{"S-BASH": policy.Session},
			grantAt: -9 * time.Hour,
			calls:   []call{{tool: "Bash", session: "sess-a", wantCode: 2}},
		},
		{
			name:   "command token lifts exactly one call",
			grants: map[string]policy.BypassTier{"C-WRITE": policy.Command},
			calls: []call{
				{tool: "Write", session: "sess-b", wantCode: 2},
				{tool: "Write", session: "sess-a", wantCode: 0},
				{tool: "Write", session: "sess-a", wantCode: 2},
			},
		},
		{
			name:   "legacy unscoped override lifts nothing",
			legacy: true,
			calls:  []call{{tool: "Bash", session: "sess-a", wantCode: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnforceEnv(t)
			e.writePolicy(t, bypassEnforcePolicy)
			t.Setenv(envClaudeProjectDir, e.project)
			t.Chdir(e.project)

			statePath := filepath.Join(e.home, policyDirName, "session-state.json")
			scope := policy.BypassScope{ProjectRoot: canonicalProjectRoot(e.project), SessionID: "sess-a"}
			if tt.otherScope {
				scope.ProjectRoot = canonicalProjectRoot(t.TempDir())
			}
			grantAt := time.Now().Add(tt.grantAt)
			var grants []policy.BypassGrant
			for id, tier := range tt.grants {
				g, err := policy.NewBypassGrant(id, tier, scope, 0, grantAt)
				if err != nil {
					t.Fatal(err)
				}
				grants = append(grants, g)
			}
			if len(grants) > 0 {
				if err := policy.NewFileSessionStateStore(statePath).AddGrants(grants, grantAt); err != nil {
					t.Fatal(err)
				}
			}
			if tt.legacy {
				if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(statePath, []byte(`{"sessionBypassOverrides":["S-BASH","C-WRITE"]}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			for i, c := range tt.calls {
				_, _, err := executeEnforce(t, hookEventPreToolUse, sessionPayload(c.tool, c.session))
				if got := exitCodeOf(err); got != c.wantCode {
					t.Fatalf("call %d (%s, %q): exit code = %d, want %d (err: %v)", i+1, c.tool, c.session, got, c.wantCode, err)
				}
				if c.wantMsg != "" && !strings.Contains(err.Error(), c.wantMsg) {
					t.Errorf("call %d: error %q does not mention %q", i+1, err.Error(), c.wantMsg)
				}
			}
		})
	}
}
