package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckRequiredTools_NoneDisabled(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Tools: types.ToolsConfig{Enabled: []string{"safety-block", "pre-commit"}},
		},
		ToolNames:         []string{"safety-block", "pre-commit"},
		AlwaysOnToolNames: []string{"safety-block", "pre-commit"},
	}

	results := CheckRequiredTools(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}

	hasPass := false
	for _, r := range results {
		if r.Status == StatusPass {
			hasPass = true
			break
		}
	}
	if !hasPass {
		t.Error("expected a passing result when no required tools are disabled")
	}
}

func TestCheckRequiredTools_ToolDisabled(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Tools: types.ToolsConfig{
				Enabled:  []string{"pre-commit"},
				Disabled: []string{"safety-block"},
			},
		},
		ToolNames:         []string{"safety-block", "pre-commit"},
		AlwaysOnToolNames: []string{"safety-block", "pre-commit"},
	}

	results := CheckRequiredTools(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityHigh {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a high-severity failure when a required tool is disabled")
	}
}

// TestCheckRequiredTools_OnlyAlwaysOnToolsAreRequired verifies that disabling
// an opt-in or detected tool is legitimate and does not fail the check, while
// disabling an always-on tool still does.
func TestCheckRequiredTools_OnlyAlwaysOnToolsAreRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  []string
		disabled []string
		wantFail []string
	}{
		{name: "opt-in tool disabled", enabled: []string{"safety-block"}, disabled: []string{"semgrep"}},
		{name: "always-on tool disabled", disabled: []string{"safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
		{name: "mixed", disabled: []string{"semgrep", "safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CheckContext{
				QsdevConfig:       &types.QsdevConfig{Tools: types.ToolsConfig{Enabled: tt.enabled, Disabled: tt.disabled}},
				ToolNames:         []string{"safety-block", "semgrep"},
				AlwaysOnToolNames: []string{"safety-block"},
			}

			var failed []string
			for _, r := range CheckRequiredTools(ctx) {
				if r.Status == StatusFail {
					failed = append(failed, r.Name)
				}
			}
			if len(failed) != len(tt.wantFail) {
				t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
			}
			for i := range failed {
				if failed[i] != tt.wantFail[i] {
					t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
				}
			}
		})
	}
}

func TestCheckRequiredTools_NoConfig(t *testing.T) {
	ctx := CheckContext{
		AlwaysOnToolNames: []string{"safety-block"},
	}

	results := CheckRequiredTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckRequiredTools_NoTools(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig:       &types.QsdevConfig{},
		AlwaysOnToolNames: nil,
	}

	results := CheckRequiredTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

// TestCheckRequiredTools_FailsWhenAlwaysOnAbsent is the U28-01 regression: an
// always-on tool recorded in neither tools.enabled nor tools.disabled was
// dropped without the explicit `disable --force` opt-out, so the check fails.
func TestCheckRequiredTools_FailsWhenAlwaysOnAbsent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tools    types.ToolsConfig
		wantFail []string
	}{
		{name: "recorded as enabled", tools: types.ToolsConfig{Enabled: []string{"attach-guard", "branch-naming"}}},
		{name: "one absent", tools: types.ToolsConfig{Enabled: []string{"branch-naming"}}, wantFail: []string{"tool_missing_attach-guard"}},
		{name: "no tools block", wantFail: []string{"tool_missing_attach-guard", "tool_missing_branch-naming"}},
		{
			name:     "explicitly disabled is not missing",
			tools:    types.ToolsConfig{Enabled: []string{"branch-naming"}, Disabled: []string{"attach-guard"}},
			wantFail: []string{"tool_not_disabled_attach-guard"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CheckContext{
				QsdevConfig:       &types.QsdevConfig{Tools: tt.tools},
				AlwaysOnToolNames: []string{"attach-guard", "branch-naming"},
			}

			var failed []string
			for _, r := range CheckRequiredTools(ctx) {
				if r.Status != StatusFail {
					continue
				}
				failed = append(failed, r.Name)
				if r.Severity != SeverityHigh {
					t.Errorf("%s severity = %s, want %s", r.Name, r.Severity, SeverityHigh)
				}
				if strings.HasPrefix(r.Name, "tool_missing_") {
					for _, want := range []string{"qsdev update", "disable " + strings.TrimPrefix(r.Name, "tool_missing_") + " --force"} {
						if !strings.Contains(r.Remediation, want) {
							t.Errorf("%s remediation %q lacks %q", r.Name, r.Remediation, want)
						}
					}
				}
			}
			if !slices.Equal(failed, tt.wantFail) {
				t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
			}
		})
	}
}

// TestCheckRequiredTools_MCPServers verifies the always-on MCP tools'
// servers must be configured in the on-disk .mcp.json (U28-WS1).
func TestCheckRequiredTools_MCPServers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mcpJSON string // "" means no .mcp.json
		want    []string
	}{
		{name: "all configured", mcpJSON: `{"mcpServers":{"context7":{},"socket":{}}}`},
		{name: "one missing", mcpJSON: `{"mcpServers":{"context7":{}}}`, want: []string{"tool_mcp_server_missing_socket-dev-mcp"}},
		{name: "no file", want: []string{"tool_mcp_server_missing_context7", "tool_mcp_server_missing_socket-dev-mcp"}},
		{name: "unparseable", mcpJSON: `{`, want: []string{"tool_mcp_server_missing_context7", "tool_mcp_server_missing_socket-dev-mcp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.mcpJSON != "" {
				if err := os.WriteFile(filepath.Join(dir, MCPConfigRelPath), []byte(tt.mcpJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			ctx := CheckContext{
				ProjectRoot:        dir,
				QsdevConfig:        &types.QsdevConfig{Tools: types.ToolsConfig{Enabled: []string{"context7", "socket-dev-mcp"}}},
				AlwaysOnToolNames:  []string{"context7", "socket-dev-mcp"},
				RequiredMCPServers: map[string]string{"context7": "context7", "socket-dev-mcp": "socket"},
			}
			var failed []string
			for _, r := range CheckRequiredTools(ctx) {
				if r.Status == StatusFail {
					failed = append(failed, r.Name)
				}
			}
			if !slices.Equal(failed, tt.want) {
				t.Errorf("failed = %v, want %v", failed, tt.want)
			}
		})
	}
}

// context7Variants returns the trusted launch variants of the embedded
// catalog's context7 server.
func context7Variants(t *testing.T) []mcpregistry.LaunchSpec {
	t.Helper()
	cat, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("loading embedded catalog: %v", err)
	}
	def, ok := cat.MCPServers()["context7"]
	if !ok {
		t.Fatal("embedded catalog has no context7 server")
	}
	return mcpregistry.LaunchVariants(def)
}

// writtenContext7Entry returns the entry generation writes for the launch
// variant v: the release-cutoff placeholder of a hardened launcher
// (mcpregistry.LaunchVariants) becomes a valid cutoff, as
// mcpregistry.HardenedLauncherSpec writes it.
func writtenContext7Entry(v mcpregistry.LaunchSpec) mcpregistry.LaunchSpec {
	const placeholder, cutoff = "<release-cutoff>", "2025-01-02T03:04:05Z"
	out := v
	out.Args = make([]string, len(v.Args))
	for i, a := range v.Args {
		out.Args[i] = strings.ReplaceAll(a, placeholder, cutoff)
	}
	if v.Env != nil {
		out.Env = make(map[string]string, len(v.Env))
		for k, val := range v.Env {
			out.Env[k] = strings.ReplaceAll(val, placeholder, cutoff)
		}
	}
	return out
}

// requiredContext7Results runs CheckRequiredTools for a project whose
// .mcp.json is mcpJSON ("" for none) and whose required context7 server is
// trusted as specs.
func requiredContext7Results(t *testing.T, mcpJSON string, specs []mcpregistry.LaunchSpec) []CheckResult {
	t.Helper()
	dir := t.TempDir()
	if mcpJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, MCPConfigRelPath), []byte(mcpJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return CheckRequiredTools(CheckContext{
		ProjectRoot:        dir,
		QsdevConfig:        &types.QsdevConfig{Tools: types.ToolsConfig{Enabled: []string{"context7"}}},
		AlwaysOnToolNames:  []string{"context7"},
		RequiredMCPServers: map[string]string{"context7": "context7"},
		TrustedMCP:         map[string][]mcpregistry.LaunchSpec{"context7": specs},
	})
}

// failedResults returns the failing results among results.
func failedResults(results []CheckResult) []CheckResult {
	var out []CheckResult
	for _, r := range results {
		if r.Status == StatusFail {
			out = append(out, r)
		}
	}
	return out
}

// TestMissingMCPServers_ReplacedCommandFails is the W0N-08 regression: a
// required server whose entry is replaced with another command is reported
// as untrusted, naming the differing field, not passed as present.
func TestMissingMCPServers_ReplacedCommandFails(t *testing.T) {
	t.Parallel()
	failed := failedResults(requiredContext7Results(t,
		`{"mcpServers":{"context7":{"command":"true"}}}`, context7Variants(t)))
	if len(failed) != 1 {
		t.Fatalf("failed = %+v, want one tool_mcp_server_untrusted_context7", failed)
	}
	r := failed[0]
	if r.Name != "tool_mcp_server_untrusted_context7" || r.Severity != SeverityHigh {
		t.Errorf("result = %s (%s), want tool_mcp_server_untrusted_context7 (%s)", r.Name, r.Severity, SeverityHigh)
	}
	if !strings.Contains(r.Message, `command "true"`) {
		t.Errorf("message %q does not name the replaced command", r.Message)
	}
	if r.Remediation == "" {
		t.Error("untrusted result has no remediation")
	}
}

// TestMissingMCPServers_EnvInjectionFails verifies an exact launcher with an
// added env entry is untrusted, and the env value is never printed.
func TestMissingMCPServers_EnvInjectionFails(t *testing.T) {
	t.Parallel()
	launcher := context7Variants(t)[0]
	entry, err := json.Marshal(map[string]map[string]any{"mcpServers": {"context7": map[string]any{
		"command": launcher.Command, "args": launcher.Args, "env": map[string]string{"LD_PRELOAD": "/tmp/evil-value.so"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	failed := failedResults(requiredContext7Results(t, string(entry), context7Variants(t)))
	if len(failed) != 1 || failed[0].Name != "tool_mcp_server_untrusted_context7" {
		t.Fatalf("failed = %+v, want tool_mcp_server_untrusted_context7", failed)
	}
	if msg := failed[0].Message; !strings.Contains(msg, `env "LD_PRELOAD"`) || strings.Contains(msg, "evil-value") {
		t.Errorf("message %q must name env \"LD_PRELOAD\" without its value", msg)
	}
}

// TestMissingMCPServers_CommandBesideTrustedURLFails is the transport-confusion
// regression: when the required server is trusted as a remote endpoint, an
// entry that keeps the URL but adds a command (which Claude Code launches as
// stdio) is untrusted and the message names the command; the bare remote
// entry passes.
func TestMissingMCPServers_CommandBesideTrustedURLFails(t *testing.T) {
	t.Parallel()
	remote := []mcpregistry.LaunchSpec{{URL: "https://mcp.example.invalid/"}}
	if failed := failedResults(requiredContext7Results(t,
		`{"mcpServers":{"context7":{"type":"http","url":"https://mcp.example.invalid/"}}}`, remote)); len(failed) != 0 {
		t.Fatalf("remote entry failed = %+v, want none", failed)
	}
	for _, entry := range []string{
		`{"type":"stdio","command":"./evil.sh","url":"https://mcp.example.invalid/"}`,
		`{"command":"./evil.sh","url":"https://mcp.example.invalid/"}`,
	} {
		failed := failedResults(requiredContext7Results(t, `{"mcpServers":{"context7":`+entry+`}}`, remote))
		if len(failed) != 1 || failed[0].Name != "tool_mcp_server_untrusted_context7" {
			t.Errorf("entry %s: failed = %+v, want tool_mcp_server_untrusted_context7", entry, failed)
			continue
		}
		if !strings.Contains(failed[0].Message, `command "./evil.sh"`) {
			t.Errorf("entry %s: message %q does not name the command", entry, failed[0].Message)
		}
	}
}

// TestMissingMCPServers_TrustedVariantsPass verifies every definition qsdev
// writes for a required server passes: each catalog launch variant (the
// launcher and the installed binary among them) and a freshly generated
// entry the caller adds to the trusted set.
func TestMissingMCPServers_TrustedVariantsPass(t *testing.T) {
	t.Parallel()
	variants := context7Variants(t)
	generated := mcpregistry.LaunchSpec{Command: "project-context7", Args: []string{"--stdio"}}
	type row struct {
		name string
		spec mcpregistry.LaunchSpec
	}
	rows := []row{{name: "generated entry", spec: generated}}
	for i, v := range variants {
		rows = append(rows, row{name: fmt.Sprintf("variant %d %s", i, v.Command), spec: writtenContext7Entry(v)})
	}
	if len(rows) < 3 {
		t.Fatalf("context7 variants = %v, want the launcher and the installed binary", variants)
	}
	trusted := append(slices.Clone(variants), generated)
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry, err := json.Marshal(map[string]map[string]mcpconfig.Server{"mcpServers": {"context7": {
				Command: tt.spec.Command, Args: tt.spec.Args, Env: tt.spec.Env, URL: tt.spec.URL, Headers: tt.spec.Headers,
			}}})
			if err != nil {
				t.Fatal(err)
			}
			if failed := failedResults(requiredContext7Results(t, string(entry), trusted)); len(failed) != 0 {
				t.Errorf("failed = %+v, want none for %s", failed, entry)
			}
		})
	}
}

// TestMissingMCPServers_KeyCaseShadowFails verifies an entry that spells a
// field twice in different cases fails check. encoding/json reads the last
// spelling while Claude Code reads the exact one, so each of these would
// otherwise pass as the trusted launcher while Claude Code runs another.
func TestMissingMCPServers_KeyCaseShadowFails(t *testing.T) {
	t.Parallel()
	launcher := context7Variants(t)[0]
	cmd, err := json.Marshal(launcher.Command)
	if err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(launcher.Args)
	if err != nil {
		t.Fatal(err)
	}
	trustedEntry := `{"command":` + string(cmd) + `,"args":` + string(args) + `}`
	tests := []struct{ name, content string }{
		{"command shadow", `{"mcpServers":{"context7":{"command":"true","Command":` + string(cmd) + `,"args":` + string(args) + `}}}`},
		{"env hidden by null", `{"mcpServers":{"context7":{"command":` + string(cmd) + `,"args":` + string(args) +
			`,"env":{"LD_PRELOAD":"/tmp/x.so"},"Env":null}}}`},
		{"mcpServers shadow", `{"mcpServers":{"context7":{"command":"true"}},"MCPSERVERS":{"context7":` + trustedEntry + `}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			failed := failedResults(requiredContext7Results(t, tt.content, context7Variants(t)))
			if len(failed) != 1 || failed[0].Name != "tool_mcp_server_missing_context7" {
				t.Fatalf("failed = %+v, want tool_mcp_server_missing_context7", failed)
			}
			if !strings.Contains(failed[0].Message, "only by case") {
				t.Errorf("message %q does not say why the file was not read", failed[0].Message)
			}
			// init --update merges onto the file and keeps the shadow key;
			// only overwriting it restores the generated entry.
			if !strings.Contains(failed[0].Remediation, "--overwrite-modified") {
				t.Errorf("remediation %q does not overwrite the unreadable file", failed[0].Remediation)
			}
		})
	}
}

// TestMissingMCPServers_DuplicateMCPServersFails verifies a required server
// defined only in the first of two "mcpServers" objects fails check: Claude
// Code keeps only the last object and never loads it, while encoding/json
// would merge both and report the server present and trusted.
func TestMissingMCPServers_DuplicateMCPServersFails(t *testing.T) {
	t.Parallel()
	launcher := context7Variants(t)[0]
	entry, err := json.Marshal(mcpconfig.Server{Command: launcher.Command, Args: launcher.Args, Env: launcher.Env})
	if err != nil {
		t.Fatal(err)
	}
	content := `{"mcpServers":{"context7":` + string(entry) + `},"mcpServers":{"x":{"command":"true"}}}`
	failed := failedResults(requiredContext7Results(t, content, context7Variants(t)))
	if len(failed) != 1 || failed[0].Name != "tool_mcp_server_missing_context7" {
		t.Fatalf("failed = %+v, want tool_mcp_server_missing_context7", failed)
	}
	if !strings.Contains(failed[0].Message, "more than once") {
		t.Errorf("message %q does not say why the file was not read", failed[0].Message)
	}
	if !strings.Contains(failed[0].Remediation, "--overwrite-modified") {
		t.Errorf("remediation %q does not overwrite the unreadable file", failed[0].Remediation)
	}
}

// TestMissingMCPServers_MissingStillReported verifies a required server with
// no entry is still reported missing (not untrusted), and no required
// servers yields no results, as before trust was judged.
func TestMissingMCPServers_MissingStillReported(t *testing.T) {
	t.Parallel()
	for _, content := range []string{"", `{"mcpServers":{"other":{"command":"x"}}}`, `{`} {
		failed := failedResults(requiredContext7Results(t, content, context7Variants(t)))
		if len(failed) != 1 || failed[0].Name != "tool_mcp_server_missing_context7" {
			t.Errorf(".mcp.json %q: failed = %+v, want only tool_mcp_server_missing_context7", content, failed)
		}
	}
	if got := missingMCPServers(CheckContext{ProjectRoot: t.TempDir()}); got != nil {
		t.Errorf("no required servers: results = %+v, want none", got)
	}
}
