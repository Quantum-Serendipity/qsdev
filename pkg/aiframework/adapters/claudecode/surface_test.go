package claudecode_test

import (
	"context"
	"encoding/json"
	"testing"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	refcc "github.com/Quantum-Serendipity/qsdev/pkg/aiframework/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestAdapterAdvertisesOnlyImplementedInterfaces pins the adapter's surface to
// the interfaces it genuinely implements. Claude Code has no hook-undeploy,
// metrics, MCP-registry or state backend, so the adapter must not satisfy
// those interfaces with success-reporting stubs that a caller could wire in
// and trust (permanently "healthy" metrics, silent no-op undeploys).
func TestAdapterAdvertisesOnlyImplementedInterfaces(t *testing.T) {
	t.Parallel()

	var a any = newAdapter()
	tests := []struct {
		name string
		ok   bool
		want bool
	}{
		{"DetectionAdapter", implements[aiframework.DetectionAdapter](a), true},
		{"ConfigRenderer", implements[aiframework.ConfigRenderer](a), true},
		{"ToolAdapter", implements[aiframework.ToolAdapter](a), true},
		{"HookDeployer", implements[aiframework.HookDeployer](a), false},
		{"RegistryClient", implements[aiframework.RegistryClient](a), false},
		{"MetricsProvider", implements[aiframework.MetricsProvider](a), false},
		{"StateBackend", implements[aiframework.StateBackend](a), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.ok != tt.want {
				t.Errorf("adapter implements %s = %v, want %v", tt.name, tt.ok, tt.want)
			}
		})
	}
}

func implements[I any](v any) bool {
	_, ok := v.(I)
	return ok
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      []types.GeneratedFile
		wantIssues int
	}{
		{"valid json", []types.GeneratedFile{{Path: "a.json", Content: []byte(`{"k":"v"}`)}}, 0},
		{"invalid json", []types.GeneratedFile{{Path: "broken.json", Content: []byte(`{invalid`)}}, 1},
		{"non-json file ignored", []types.GeneratedFile{{Path: "hook.py", Content: []byte(`{invalid`)}}, 0},
		{"no files", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			issues := newAdapter().Validate(context.Background(), tt.files)
			if len(issues) != tt.wantIssues {
				t.Fatalf("Validate() = %d issues (%v), want %d", len(issues), issues, tt.wantIssues)
			}
			for _, is := range issues {
				if is.Severity != aiframework.SeverityError {
					t.Errorf("issue severity = %v, want SeverityError", is.Severity)
				}
			}
		})
	}
}

func TestDetect_ConfidenceAndConfigPaths(t *testing.T) {
	t.Parallel()

	det, err := newAdapter().Detect(presentRoot(t))
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}
	if det.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain (.claude/ present)", det.Confidence)
	}
	if len(det.ConfigPaths) < 2 {
		t.Errorf("ConfigPaths = %v, want .claude and CLAUDE.md", det.ConfigPaths)
	}
}

// TestRender_ConfiguredMCPServersNotAliased guards against a policy's
// command-based MCP servers being appended into the configured MCPServers
// backing array: with spare capacity, an unguarded append would let one
// Render call's servers leak into the next call's output.
func TestRender_ConfiguredMCPServersNotAliased(t *testing.T) {
	t.Parallel()

	servers := make([]ccaddon.MCPServerConfig, 1, 8)
	servers[0] = ccaddon.MCPServerConfig{Name: "base", Command: "base-cmd"}
	a := refcc.New(ccaddon.Config{
		DefaultPermissions: ccaddon.PermissionPresetStandard,
		MCPServers:         servers,
	}, nil)

	render := func(name string) map[string]ccaddon.MCPServerEntry {
		t.Helper()
		files, err := a.Render(context.Background(), &aiframework.PolicyInput{
			ProjectRoot: t.TempDir(),
			MCPServers:  []aiframework.MCPServerSpec{{Name: name, Command: name + "-cmd"}},
		})
		if err != nil {
			t.Fatalf("Render(%s) error: %v", name, err)
		}
		for _, f := range files {
			if f.Path == ".mcp.json" {
				var doc ccaddon.McpJSON
				if err := json.Unmarshal(f.Content, &doc); err != nil {
					t.Fatalf("invalid .mcp.json: %v", err)
				}
				return doc.MCPServers
			}
		}
		t.Fatalf("Render(%s) produced no .mcp.json", name)
		return nil
	}

	render("first")
	second := render("second")
	if _, leaked := second["first"]; leaked {
		t.Errorf("second render contains the first call's server: %v", second)
	}
	if _, ok := second["base"]; !ok {
		t.Errorf("second render lost the configured server: %v", second)
	}
	if extra := servers[:2][1]; extra.Name != "" {
		t.Errorf("Render wrote %q into the configured MCPServers backing array", extra.Name)
	}
}
