package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

func TestCheckAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.env")
	if err := os.WriteFile(secretFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []policy.DenyRule{
		{Pattern: tmpDir + "/*", Type: "path"},
	}

	tests := []struct {
		name      string
		tool      string
		args      map[string]string
		rules     []policy.DenyRule
		wantBlock bool
	}{
		{
			name:      "known tool accessing denied path",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: true,
		},
		{
			name:      "unknown mcp tool not blocked",
			tool:      "mcp__custom__do_stuff",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "non-mcp tool not checked",
			tool:      "Read",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "known tool accessing allowed path",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": "/tmp/allowed.txt"},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "empty deny rules",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": secretFile},
			rules:     nil,
			wantBlock: false,
		},
		{
			name:      "github create_or_update_file denied",
			tool:      "mcp__github__create_or_update_file",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			argsJSON, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("marshaling args: %v", err)
			}

			blocked, reason := CheckAccess(tt.tool, argsJSON, tt.rules)

			if blocked != tt.wantBlock {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.wantBlock, reason)
			}

			if blocked && reason == "" {
				t.Error("blocked but no reason provided")
			}
		})
	}
}

func TestCheckAccessPathCanonicalization(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("creating nested dir: %v", err)
	}
	testFile := filepath.Join(nestedDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []policy.DenyRule{
		{Pattern: tmpDir + "/*", Type: "path"},
	}

	// Use a relative-style path with ..
	relativePath := filepath.Join(nestedDir, "..", "..", "a", "b", "file.txt")

	args, _ := json.Marshal(map[string]string{"path": relativePath})

	blocked, _ := CheckAccess("mcp__filesystem__read_file", args, denyRules)
	if !blocked {
		t.Error("relative path should resolve and be blocked")
	}
}

func TestCheckAccessEmptyArgs(t *testing.T) {
	t.Parallel()

	denyRules := []policy.DenyRule{{Pattern: "/secret/*", Type: "path"}}

	blocked, _ := CheckAccess("mcp__filesystem__read_file", nil, denyRules)
	if blocked {
		t.Error("nil args should not block")
	}

	blocked, _ = CheckAccess("mcp__filesystem__read_file", json.RawMessage("{}"), denyRules)
	if blocked {
		t.Error("empty object args should not block")
	}
}

func TestCheckAccessExactPathMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	exactFile := filepath.Join(tmpDir, "exact.txt")
	if err := os.WriteFile(exactFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []policy.DenyRule{
		{Pattern: exactFile, Type: "path"},
	}

	args, _ := json.Marshal(map[string]string{"path": exactFile})

	blocked, _ := CheckAccess("mcp__filesystem__read_file", args, denyRules)
	if !blocked {
		t.Error("exact path match should block")
	}
}

// TestCheckAccess_GlobDenyPatterns is the regression guard for the deputy
// re-implementing pattern matching with filepath.Match: relative and `**`
// patterns (the style policies are written in) must mean the same thing here as
// in the policy engine's path_glob / denied_path_check conditions.
func TestCheckAccess_GlobDenyPatterns(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}

	tests := []struct {
		name      string
		pattern   string
		path      string
		wantBlock bool
	}{
		{"double-star dir any depth", "**/.ssh/*", "/home/u/.ssh/id_rsa", true},
		{"double-star dir nested", "**/.ssh/*", "/home/u/.ssh/keys/deploy", true},
		{"double-star dir unrelated", "**/.ssh/*", "/home/u/project/main.go", false},
		{"double-star file", "**/.env", "/srv/app/.env", true},
		{"double-star file dot segments", "**/.env", "/srv/app/./config/../.env", true},
		{"double-star file other", "**/.env", "/srv/app/.env.example", false},
		{"tilde pattern", "~/.ssh/*", filepath.Join(home, ".ssh", "id_ed25519"), true},
		{"tilde path", "**/.ssh/*", "~/.ssh/id_rsa", true},
		{"absolute prefix", "/etc/secret/*", "/etc/secret/token", true},
		{"absolute prefix sibling", "/etc/secret/*", "/etc/secretive/token", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args, err := json.Marshal(map[string]string{"path": tt.path})
			if err != nil {
				t.Fatalf("marshaling args: %v", err)
			}
			rules := []policy.DenyRule{{Pattern: tt.pattern, Type: "path"}}
			blocked, reason := CheckAccess("mcp__filesystem__read_file", args, rules)
			if blocked != tt.wantBlock {
				t.Errorf("CheckAccess(%q vs %q) blocked = %v, want %v (reason: %s)", tt.path, tt.pattern, blocked, tt.wantBlock, reason)
			}
		})
	}
}

// TestCheckAccess_ToolScopedRuleCoversMCPTool reproduces the path-laundering
// attack end to end: a deny rule compiled from a policy scoped to the
// first-party Read tool must still stop an MCP filesystem read of the same path.
func TestCheckAccess_ToolScopedRuleCoversMCPTool(t *testing.T) {
	t.Parallel()

	set, err := policy.Compile(&policy.SecurityPolicy{
		APIVersion: "qsdev/v1",
		Kind:       "SecurityPolicy",
		Metadata:   policy.PolicyMetadata{Name: "deputy"},
		Rules: []policy.PolicyRule{{
			ID: "CG-SSH", Category: "config-guard", Name: "no ssh reads",
			Severity: policy.Critical, BypassTier: policy.EnforceAlways,
			Conditions: policy.Condition{Type: policy.All, Conditions: []policy.Condition{
				{Type: policy.ToolMatch, ToolName: "Read"},
				{Type: policy.PathGlob, Pattern: "**/.ssh/*"},
			}},
			Action: policy.Action{Type: policy.Block, Message: "denied"},
		}},
	})
	if err != nil {
		t.Fatalf("compiling policy: %v", err)
	}

	// Normalize the compiled types the same way PolicyEngine.FilePathDenyRules does.
	rules := make([]policy.DenyRule, len(set.DenyRules))
	for i, r := range set.DenyRules {
		rules[i] = policy.DenyRule{Pattern: r.Pattern, Type: "path"}
	}
	if len(rules) == 0 {
		t.Fatal("expected the path_glob condition to produce a deny rule")
	}

	args := json.RawMessage(`{"path":"/home/u/.ssh/id_rsa"}`)
	if blocked, _ := CheckAccess("mcp__filesystem__read_file", args, rules); !blocked {
		t.Error("MCP read of ~/.ssh/id_rsa must be blocked by the Read-scoped **/.ssh/* rule")
	}
}

// TestCheckAccess_PathBearingTools is the regression guard for the deputy only
// knowing single-`path` tools and failing open on unreadable arguments: every
// path of a multi-path or move tool is checked, the current filesystem server's
// tool names are covered, remote-repository paths are not treated as local, and
// arguments that cannot be read block the call.
func TestCheckAccess_PathBearingTools(t *testing.T) {
	t.Parallel()

	rules := []policy.DenyRule{{Pattern: "**/.ssh/*", Type: "path"}}
	key := "/home/u/.ssh/id_rsa"

	tests := []struct {
		name      string
		tool      string
		args      string
		wantBlock bool
	}{
		{"read_text_file", "mcp__filesystem__read_text_file", `{"path":"` + key + `"}`, true},
		{"read_media_file", "mcp__filesystem__read_media_file", `{"path":"` + key + `"}`, true},
		{"read_multiple_files", "mcp__filesystem__read_multiple_files", `{"paths":["/tmp/a","` + key + `"]}`, true},
		{"read_multiple_files harmless", "mcp__filesystem__read_multiple_files", `{"paths":["/tmp/a","/tmp/b"]}`, false},
		{"move_file source", "mcp__filesystem__move_file", `{"source":"` + key + `","destination":"/tmp/k"}`, true},
		{"move_file destination", "mcp__filesystem__move_file", `{"source":"/tmp/k","destination":"` + key + `"}`, true},
		{"get_file_info", "mcp__filesystem__get_file_info", `{"path":"` + key + `"}`, true},
		{"github remote path not local", "mcp__github__get_file_contents", `{"owner":"o","repo":"r","path":".ssh/id_rsa"}`, false},
		{"malformed args fail closed", "mcp__filesystem__read_file", `{"path":`, true},
		{"non-string path fails closed", "mcp__filesystem__read_file", `{"path":42}`, true},
		{"mixed-type paths fail closed", "mcp__filesystem__read_multiple_files", `{"paths":["/tmp/a",{"p":"x"}]}`, true},
		{"null path allowed", "mcp__filesystem__read_file", `{"path":null}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := CheckAccess(tt.tool, json.RawMessage(tt.args), rules)
			if blocked != tt.wantBlock {
				t.Errorf("CheckAccess(%s, %s) blocked = %v, want %v (reason: %s)", tt.tool, tt.args, blocked, tt.wantBlock, reason)
			}
		})
	}
}

// TestCheckAccess_SymlinkAlias guards the canonical-path form: a path reaching a
// denied directory through a symlink must be blocked.
func TestCheckAccess_SymlinkAlias(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretDir := filepath.Join(tmpDir, "secret")
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatalf("creating secret dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "key"), []byte("k"), 0o644); err != nil {
		t.Fatalf("creating key: %v", err)
	}
	link := filepath.Join(tmpDir, "innocent")
	if err := os.Symlink(secretDir, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	rules := []policy.DenyRule{{Pattern: filepath.ToSlash(secretDir) + "/*", Type: "path"}}
	args, err := json.Marshal(map[string]string{"path": filepath.Join(link, "key")})
	if err != nil {
		t.Fatalf("marshaling args: %v", err)
	}
	if blocked, _ := CheckAccess("mcp__filesystem__read_file", args, rules); !blocked {
		t.Error("read through a symlink into a denied directory must be blocked")
	}
}

// TestCheckAccess_QsdevConfigRender guards F224: a qsdev_cc_config_render call
// asking to write materializes the agent's own .claude/settings.json and
// .mcp.json, so it is checked as an Edit of both files whatever name the qsdev
// server is registered under. A dry-run call writes nothing and is not blocked.
func TestCheckAccess_QsdevConfigRender(t *testing.T) {
	t.Parallel()

	settingsRule := []policy.DenyRule{{Pattern: "**/.claude/settings.json", Type: "path"}}
	mcpRule := []policy.DenyRule{{Pattern: "**/.mcp.json", Type: "path"}}

	tests := []struct {
		name      string
		tool      string
		args      string
		rules     []policy.DenyRule
		wantBlock bool
	}{
		{"write=true hits settings.json", "mcp__qsdev__qsdev_cc_config_render", `{"write":true}`, settingsRule, true},
		{"write=true hits .mcp.json", "mcp__qsdev__qsdev_cc_config_render", `{"write":true}`, mcpRule, true},
		{"any server name", "mcp__qsdev-universal__qsdev_cc_config_render", `{"write":true}`, settingsRule, true},
		{"non-boolean write fails closed", "mcp__qsdev__qsdev_cc_config_render", `{"write":"yes"}`, settingsRule, true},
		{"malformed args fail closed", "mcp__qsdev__qsdev_cc_config_render", `{"write":`, settingsRule, true},
		{"dry-run empty args", "mcp__qsdev__qsdev_cc_config_render", `{}`, settingsRule, false},
		{"dry-run no args", "mcp__qsdev__qsdev_cc_config_render", ``, settingsRule, false},
		{"write=false", "mcp__qsdev__qsdev_cc_config_render", `{"write":false}`, settingsRule, false},
		{"write=null", "mcp__qsdev__qsdev_cc_config_render", `{"write":null}`, settingsRule, false},
		{"write=true, unrelated rule", "mcp__qsdev__qsdev_cc_config_render", `{"write":true}`, []policy.DenyRule{{Pattern: "**/.ssh/*", Type: "path"}}, false},
		{"other qsdev tool unaffected", "mcp__qsdev__qsdev_cc_permissions", `{"write":true}`, settingsRule, false},
		{"bare tool name is not an mcp call", "qsdev_cc_config_render", `{"write":true}`, settingsRule, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := CheckAccess(tt.tool, json.RawMessage(tt.args), tt.rules)
			if blocked != tt.wantBlock {
				t.Errorf("CheckAccess(%s, %s) blocked = %v, want %v (reason: %s)", tt.tool, tt.args, blocked, tt.wantBlock, reason)
			}
		})
	}
}
