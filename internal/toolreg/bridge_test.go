package toolreg

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// buildFromOverlay loads the embedded catalog with overlay as an org file
// and builds a registry from it, returning the first error from either step.
func buildFromOverlay(t *testing.T, overlay string) (*Registry, error) {
	t.Helper()
	f := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(f, []byte(overlay), 0o600); err != nil {
		t.Fatalf("writing overlay: %v", err)
	}
	cat, err := catalog.Load(catalog.WithOrgConfigFile(f))
	if err != nil {
		return nil, err
	}
	return buildRegistryFromCatalog(cat)
}

// TestBuildRegistryFromCatalog_RejectsInvalidToolFields checks that an
// invalid overlay tool never yields a registry: catalog validation rejects
// catalog-level mistakes and the bridge rejects values only it interprets
// (toggle_field).
func TestBuildRegistryFromCatalog_RejectsInvalidToolFields(t *testing.T) {
	t.Parallel()
	const header = `
tools:
  custom-tool:
    display_name: Custom
    category: devex
    description: A custom tool
`
	tests := []struct {
		name    string
		fields  string
		wantErr string
	}{
		{
			// A mis-cased "shared" must not become an exclusive file that
			// disable would delete wholesale.
			name: "unknown ownership",
			fields: `    default_policy: opt-in
    owned_files:
      - path: devenv.nix
        ownership: Shared
        section_id: custom-tool
`,
			wantErr: `unknown ownership "Shared"`,
		},
		{
			name:    "unknown default policy",
			fields:  "    default_policy: always_on\n",
			wantErr: `unknown default_policy "always_on"`,
		},
		{
			name: "unknown toggle field",
			fields: `    default_policy: opt-in
    toggle_field: hooks.safetyblock
`,
			wantErr: `unknown toggle_field "hooks.safetyblock"`,
		},
		{
			name: "path escaping the project",
			fields: `    default_policy: opt-in
    owned_files:
      - path: ../outside.txt
        ownership: exclusive
`,
			wantErr: "must be a relative path inside the project",
		},
		{
			name: "absolute path",
			fields: `    default_policy: opt-in
    owned_files:
      - path: /etc/passwd
        ownership: exclusive
`,
			wantErr: "must be a relative path inside the project",
		},
		{
			name: "shared file without section",
			fields: `    default_policy: opt-in
    owned_files:
      - path: CLAUDE.md
        ownership: shared
`,
			wantErr: "has no section_id",
		},
		{
			name: "unknown prerequisite",
			fields: `    default_policy: opt-in
    prerequisites: [no-such-tool]
`,
			wantErr: `references unknown tool "no-such-tool"`,
		},
		{
			name: "unknown conflict",
			fields: `    default_policy: opt-in
    conflicts: [no-such-tool]
`,
			wantErr: `references unknown tool "no-such-tool"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildFromOverlay(t, header+tt.fields)
			if err == nil {
				t.Fatalf("invalid tool accepted; want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), "custom-tool") {
				t.Errorf("error = %v, want it to name custom-tool and contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildRegistryFromCatalog_AcceptsValidOverlayTool(t *testing.T) {
	t.Parallel()
	reg, err := buildFromOverlay(t, `
tools:
  custom-tool:
    display_name: Custom
    category: devex
    description: A custom tool
    default_policy: opt-in
    prerequisites: [gitleaks]
    owned_files:
      - path: .custom/config.yml
        ownership: exclusive
      - path: CLAUDE.md
        ownership: shared
        section_id: custom-tool
        section_content: "- Custom tool is active."
`)
	if err != nil {
		t.Fatalf("building registry from a valid overlay: %v", err)
	}
	tool, ok := reg.ByName("custom-tool")
	if !ok {
		t.Fatal("custom-tool not registered")
	}
	if got := tool.SharedFiles(); len(got) != 1 || got[0].Path != "CLAUDE.md" {
		t.Errorf("SharedFiles = %+v, want CLAUDE.md", got)
	}
	if got := tool.ExclusiveFiles(); len(got) != 1 || got[0].Path != ".custom/config.yml" {
		t.Errorf("ExclusiveFiles = %+v, want .custom/config.yml", got)
	}
	if _, ok := tool.SharedContent[SharedSection{Path: "CLAUDE.md", SectionID: "custom-tool"}]; !ok {
		t.Error("section_content was not turned into SharedContent")
	}
}

// TestBridge_MultipleBehaviorSourcesAllApply covers a tool that declares
// both a toggle_field and an mcp_server_name: enabling it must set the
// toggle and add the MCP server, and disabling must undo both.
func TestBridge_MultipleBehaviorSourcesAllApply(t *testing.T) {
	t.Parallel()
	reg, err := BuildFromCatalogE()
	if err != nil {
		t.Fatalf("BuildFromCatalogE: %v", err)
	}
	tool, ok := reg.ByName(ToolAgentPostmortem)
	if !ok {
		t.Fatalf("%s not in registry", ToolAgentPostmortem)
	}

	var a types.WizardAnswers
	tool.EnableFunc(&a)
	if !a.AgentTools.PostmortemEnabled {
		t.Error("enable did not set agent_tools.postmortem_enabled")
	}
	if !slices.Contains(a.MCPServers, "agent-postmortem") {
		t.Errorf("enable did not add the MCP server; MCPServers = %v", a.MCPServers)
	}

	tool.DisableFunc(&a)
	if a.AgentTools.PostmortemEnabled {
		t.Error("disable did not clear agent_tools.postmortem_enabled")
	}
	if slices.Contains(a.MCPServers, "agent-postmortem") {
		t.Errorf("disable did not remove the MCP server; MCPServers = %v", a.MCPServers)
	}
}

func TestSetToggle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		field string
		get   func(types.WizardAnswers) bool
	}{
		{"hooks.safety_block", func(a types.WizardAnswers) bool { return a.Hooks.SafetyBlock }},
		{"agent_tools.postmortem_enabled", func(a types.WizardAnswers) bool { return a.AgentTools.PostmortemEnabled }},
		{"agent_tools.version_sentinel", func(a types.WizardAnswers) bool { return a.AgentTools.VersionSentinel }},
		{"agent_tools.semble_enabled", func(a types.WizardAnswers) bool { return a.AgentTools.SembleEnabled }},
	}
	if len(tests) != len(toggleFields) {
		t.Fatalf("test covers %d toggle fields, toggleFields has %d", len(tests), len(toggleFields))
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()
			var a types.WizardAnswers
			setToggle(&a, tt.field, true)
			if !tt.get(a) {
				t.Errorf("setToggle(%q, true) did not set the field", tt.field)
			}
			setToggle(&a, tt.field, false)
			if tt.get(a) {
				t.Errorf("setToggle(%q, false) did not clear the field", tt.field)
			}
		})
	}
}

// resetDefaultsForTest isolates a test that rebuilds the default registry:
// it clears the cached catalog and registry now and again afterwards (after
// any t.Setenv registered later has restored the environment), and restores
// the registered behavior providers.
func resetDefaultsForTest(t *testing.T) {
	t.Helper()
	saved := slices.Clone(behaviorProviders)
	reset := func() {
		catalog.ResetDefault()
		ResetDefaultRegistry()
	}
	t.Cleanup(func() {
		defaultMu.Lock()
		behaviorProviders = saved
		defaultMu.Unlock()
		reset()
	})
	reset()
}

// TestDefault_BadOrgConfigFallsBackAndReports: a malformed user-level org
// overlay must neither panic nor break the registry. catalog.Default falls
// back to the built-in catalog and records the overlay's error
// (config-profile-catalog-1 F301), which names the bad file.
func TestDefault_BadOrgConfigFallsBackAndReports(t *testing.T) {
	resetDefaultsForTest(t)
	bad := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(bad, []byte("tools: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QSDEV_ORG_CONFIG", bad)

	reg, err := Default()
	if err != nil {
		t.Fatalf("Default() = %v, want the built-in catalog as fallback", err)
	}
	if reg == nil {
		t.Fatal("Default() returned no registry")
	}
	overlayErr := catalog.OrgOverlayError()
	if overlayErr == nil {
		t.Fatal("the malformed org overlay's error was not recorded")
	}
	if !strings.Contains(overlayErr.Error(), bad) {
		t.Errorf("overlay error does not name the bad file: %v", overlayErr)
	}
}

func TestRegisterBehaviors(t *testing.T) {
	resetDefaultsForTest(t)

	calls := 0
	RegisterBehaviors(func(r *Registry) {
		calls++
		r.AttachBehavior("changelog", ToolBehavior{
			DetectFunc: func(types.DetectedProject) bool { return true },
		})
	})
	if calls != 0 {
		t.Fatalf("provider ran %d times before the registry was built; registering must not build it", calls)
	}

	tool, ok := DefaultRegistry().ByName("changelog")
	if !ok || tool.DetectFunc == nil {
		t.Fatal("provider behavior not attached when the registry was built")
	}
	if calls != 1 {
		t.Errorf("provider ran %d times, want 1", calls)
	}

	// A rebuild re-applies registered providers.
	ResetDefaultRegistry()
	if tool, _ := DefaultRegistry().ByName("changelog"); tool.DetectFunc == nil {
		t.Error("provider behavior lost after ResetDefaultRegistry")
	}

	// A provider registered after the build is applied immediately.
	late := false
	RegisterBehaviors(func(*Registry) { late = true })
	if !late {
		t.Error("provider registered after build was not applied")
	}
}
