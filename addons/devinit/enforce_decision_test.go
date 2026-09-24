package devinit

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

func TestPolicyBlockMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decision policy.PolicyDecision
		want     string
	}{
		{
			name:     "rule and authored message",
			decision: policy.PolicyDecision{RuleID: "SP-001", Message: "Cannot edit .claude/settings.json: protected configuration file"},
			want:     "qsdev-policy: SP-001 \u2014 Cannot edit .claude/settings.json: protected configuration file",
		},
		{
			name:     "missing rule and message",
			decision: policy.PolicyDecision{},
			want:     "qsdev-policy: policy \u2014 tool call blocked by security policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := policyBlockMessage(tt.decision); got != tt.want {
				t.Errorf("policyBlockMessage = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWritePolicyFindings(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writePolicyFindings(&buf, []policy.Finding{
		{RuleID: "CG-002", Message: "Write to package manager config detected"},
		{RuleID: "AUD-001", Message: "audited", Monitor: true},
	})

	want := "qsdev-policy: warning: CG-002 \u2014 Write to package manager config detected\n" +
		"qsdev-policy: audit: AUD-001 \u2014 audited\n"
	if got := buf.String(); got != want {
		t.Errorf("findings output = %q, want %q", got, want)
	}
}

const enforceTestRule = `  - id: ENF-001
    category: integrity
    name: block curl
    severity: high
    bypass_tier: enforce_always
    conditions:
      type: command_match
      pattern: curl
    action:
      type: block
      message: "curl is not allowed"
`

func buildEnforceTestPolicy(settings, rules string) string {
	content := "apiVersion: qsdev/v1\nkind: SecurityPolicy\nmetadata:\n  name: test\n"
	if settings != "" {
		content += "settings:\n" + settings
	}
	return content + "rules:\n" + rules
}

// TestRunEnforce_PolicyOutcome drives runEnforce end to end (F177, F188): a
// policy that fails to load fails closed for PreToolUse unless the base policy
// opts into fail_open, and a block reports the rule's authored message.
// It changes the working directory, HOME and os.Stdin, so it cannot run in
// parallel.
func TestRunEnforce_PolicyOutcome(t *testing.T) {
	tests := []struct {
		name        string
		hook        string
		project     string
		user        string
		command     string
		noHome      bool
		wantCode    int
		wantMessage string
	}{
		{
			name:        "valid policy block reports rule message",
			hook:        "PreToolUse",
			project:     buildEnforceTestPolicy("", enforceTestRule),
			command:     "curl https://example.com",
			wantCode:    2,
			wantMessage: "ENF-001 \u2014 curl is not allowed",
		},
		{
			name:     "valid policy allow",
			hook:     "PreToolUse",
			project:  buildEnforceTestPolicy("", enforceTestRule),
			command:  "ls",
			wantCode: 0,
		},
		{
			name:        "malformed project policy fails closed",
			hook:        "PreToolUse",
			project:     buildEnforceTestPolicy("", enforceTestRule+"    severity_typo: crit\n"),
			command:     "ls",
			wantCode:    2,
			wantMessage: "failed to load",
		},
		{
			name:    "user overlay floor violation fails closed",
			hook:    "PreToolUse",
			project: buildEnforceTestPolicy("", enforceTestRule),
			user: buildEnforceTestPolicy("", strings.Replace(enforceTestRule,
				"bypass_tier: enforce_always", "bypass_tier: session", 1)),
			command:     "curl https://example.com",
			wantCode:    2,
			wantMessage: "security floor violation",
		},
		{
			// Every policy file must opt in (devinit-commands-1); the base
			// must load cleanly (F177). A broken overlay rule is then
			// tolerated.
			name:     "base fail_open tolerates broken overlay that also opts in",
			hook:     "PreToolUse",
			project:  buildEnforceTestPolicy("  fail_mode: fail_open\n", enforceTestRule),
			user:     buildEnforceTestPolicy("  fail_mode: fail_open\n", enforceTestRule+"    severity_typo: crit\n"),
			command:  "ls",
			wantCode: 0,
		},
		{
			name:        "base fail_open with an overlay that does not opt in fails closed",
			hook:        "PreToolUse",
			project:     buildEnforceTestPolicy("  fail_mode: fail_open\n", enforceTestRule),
			user:        "not: [valid",
			command:     "ls",
			wantCode:    2,
			wantMessage: "failed to load",
		},
		{
			name:        "overlay alone cannot opt into fail_open",
			hook:        "PreToolUse",
			project:     buildEnforceTestPolicy("", enforceTestRule+"    severity_typo: crit\n"),
			user:        buildEnforceTestPolicy("  fail_mode: fail_open\n", enforceTestRule),
			command:     "ls",
			wantCode:    2,
			wantMessage: "failed to load",
		},
		{
			name:        "unresolvable home still enforces project policy",
			hook:        "PreToolUse",
			project:     buildEnforceTestPolicy("", enforceTestRule),
			command:     "curl https://example.com",
			noHome:      true,
			wantCode:    2,
			wantMessage: "ENF-001",
		},
		{
			name:     "PostToolUse never blocks on load failure",
			hook:     "PostToolUse",
			project:  "not: [valid",
			command:  "ls",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := t.TempDir()
			homeDir := t.TempDir()
			t.Chdir(projectDir)
			if tt.noHome {
				homeDir = ""
			}
			t.Setenv("HOME", homeDir)
			t.Setenv("USERPROFILE", homeDir) // os.UserHomeDir on Windows

			writeEnforceTestFile(t, filepath.Join(projectDir, ".qsdev", "policy.yaml"), tt.project)
			if tt.user != "" {
				writeEnforceTestFile(t, filepath.Join(homeDir, ".qsdev", "policy.yaml"), tt.user)
			}

			stdinPath := filepath.Join(t.TempDir(), "stdin.json")
			writeEnforceTestFile(t, stdinPath, `{"tool_name":"Bash","tool_input":{"command":"`+tt.command+`"}}`)
			stdin, err := os.Open(stdinPath)
			if err != nil {
				t.Fatalf("opening stdin fixture: %v", err)
			}
			t.Cleanup(func() { _ = stdin.Close() })
			origStdin := os.Stdin
			os.Stdin = stdin
			t.Cleanup(func() { os.Stdin = origStdin })

			cmd := &cobra.Command{}
			var stderr, stdout bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.SetOut(&stdout)

			err = runEnforce(cmd, tt.hook)
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("runEnforce: unexpected error: %v", err)
				}
				return
			}

			var exitErr *exitcode.Error
			if !errors.As(err, &exitErr) {
				t.Fatalf("runEnforce error = %v, want exit code %d", err, tt.wantCode)
			}
			if exitErr.Code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", exitErr.Code, tt.wantCode)
			}
			if !strings.Contains(exitErr.Message, tt.wantMessage) {
				t.Errorf("message = %q, want it to contain %q", exitErr.Message, tt.wantMessage)
			}
		})
	}
}

// TestConfiguredMcpServers pins the production source of MCP trust scoring
// input (F189): server definitions come from the project's .mcp.json.
func TestConfiguredMcpServers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mcpJSON string
		wantErr bool
		want    map[string]string // server name -> command
	}{
		{name: "missing file", want: map[string]string{}},
		{
			name:    "servers",
			mcpJSON: `{"mcpServers":{"man-pages":{"command":"qsdev","args":["mcp","man-pages"]},"fetch":{"command":"uvx","args":["mcp-server-fetch"]}}}`,
			want:    map[string]string{"man-pages": "qsdev", "fetch": "uvx"},
		},
		{name: "malformed", mcpJSON: `{"mcpServers":`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tt.mcpJSON != "" {
				writeEnforceTestFile(t, filepath.Join(dir, ".mcp.json"), tt.mcpJSON)
			}

			servers, err := configuredMcpServers(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for a malformed .mcp.json")
				}
				return
			}
			if err != nil {
				t.Fatalf("configuredMcpServers: %v", err)
			}
			if len(servers) != len(tt.want) {
				t.Fatalf("got %d servers, want %d", len(servers), len(tt.want))
			}
			for name, command := range tt.want {
				got, ok := servers[name]
				if !ok || got.Name != name || got.Command != command {
					t.Errorf("server %q = %+v, want command %q", name, got, command)
				}
			}
		})
	}
}

func writeEnforceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
