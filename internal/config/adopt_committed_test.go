package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeCommittedConfig writes content as the project's .qsdev.yaml in root;
// empty content writes nothing.
func writeCommittedConfig(t *testing.T, root, content string) {
	t.Helper()
	if content == "" {
		return
	}
	path := filepath.Join(root, branding.Get().ConfigFile)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAdoptCommitted_Tier verifies regeneration gives legacy answers (no tier)
// the tier committed in .qsdev.yaml rather than an inferred one, and falls back
// to inference only when the config records no valid tier.
func TestAdoptCommitted_Tier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		config    string // .qsdev.yaml content; empty means no file
		answers   types.WizardAnswers
		wantTier  string
		wantFinal string // after answers.EnforceInvariants
	}{
		{"committed tier adopted", "version: 1\ntier: full\n",
			types.WizardAnswers{MCPServers: []string{"context7"}}, "full", "full"},
		{"answers tier kept", "version: 1\ntier: full\n",
			types.WizardAnswers{Tier: "supply-chain-only"}, "supply-chain-only", "supply-chain-only"},
		{"no committed tier infers", "version: 1\n",
			types.WizardAnswers{MCPServers: []string{"context7"}}, "", "standard"},
		{"invalid committed tier infers", "version: 1\ntier: ful\n",
			types.WizardAnswers{MCPServers: []string{"custom-db"}}, "", "full"},
		{"no config infers", "",
			types.WizardAnswers{PermissionLevel: "supply-chain-only"}, "", "supply-chain-only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCommittedConfig(t, root, tt.config)
			a := tt.answers
			AdoptCommitted(root, &a)
			if a.Tier != tt.wantTier {
				t.Errorf("after adopt Tier = %q, want %q", a.Tier, tt.wantTier)
			}
			answers.EnforceInvariants(&a)
			if a.Tier != tt.wantFinal {
				t.Errorf("final Tier = %q, want %q", a.Tier, tt.wantFinal)
			}
		})
	}
}

// TestAdoptCommitted_ClaudeCode verifies the agent-writable answers file
// cannot turn Claude Code off while the committed .qsdev.yaml enables it
// (U18-01): an update would otherwise stop generating, and remove, the
// settings that register the self-protection hook. Only a committed
// claude_code.enabled: false leaves the answers' choice alone.
func TestAdoptCommitted_ClaudeCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  string // .qsdev.yaml content; empty means no file
		answers bool
		want    bool
	}{
		{"committed on overrides answers off", "version: 2\nclaude_code:\n    enabled: true\n", false, true},
		{"absent key is on (legacy default)", "version: 2\n", false, true},
		{"committed off keeps answers off", "version: 2\nclaude_code:\n    enabled: false\n", false, false},
		{"committed off keeps answers on", "version: 2\nclaude_code:\n    enabled: false\n", true, true},
		{"no config keeps answers off", "", false, false},
		{"unparseable config keeps answers off", "version: [\n", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCommittedConfig(t, root, tt.config)
			a := types.WizardAnswers{ClaudeCode: tt.answers}
			AdoptCommitted(root, &a)
			if a.ClaudeCode != tt.want {
				t.Errorf("ClaudeCode = %v, want %v", a.ClaudeCode, tt.want)
			}
		})
	}
}

// TestAdoptCommitted_ClientMCPPolicy verifies answers take the client MCP
// policy from the committed client block, not from the local answers file,
// and drop every server it blocks: a stale or missing local mcp_policy must
// not let an always-on tool record a blocked server in the committed
// mcp_servers (U28-WS1).
func TestAdoptCommitted_ClientMCPPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		config      string // .qsdev.yaml content; empty means no file
		answers     types.WizardAnswers
		wantBlocked []string
		wantServers []string
	}{
		{
			name:        "committed block adopted when local policy is missing",
			config:      "version: 2\nclient:\n    name: acme\n    blocked_mcp_servers: [context7]\n",
			answers:     types.WizardAnswers{MCPServers: []string{"context7", "github"}},
			wantBlocked: []string{"context7"},
			wantServers: []string{"github"},
		},
		{
			name:   "stale local policy replaced by committed none",
			config: "version: 2\n",
			answers: types.WizardAnswers{
				MCPPolicy:  types.MCPPolicy{Blocked: []string{"github"}},
				MCPServers: []string{"github"},
			},
			wantServers: []string{"github"},
		},
		{
			name: "no config keeps answers' policy",
			answers: types.WizardAnswers{
				MCPPolicy:  types.MCPPolicy{Blocked: []string{"github"}},
				MCPServers: []string{"context7"},
			},
			wantBlocked: []string{"github"},
			wantServers: []string{"context7"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCommittedConfig(t, root, tt.config)
			a := tt.answers
			AdoptCommitted(root, &a)
			if !slices.Equal(a.MCPPolicy.Blocked, tt.wantBlocked) {
				t.Errorf("MCPPolicy.Blocked = %v, want %v", a.MCPPolicy.Blocked, tt.wantBlocked)
			}
			if !slices.Equal(a.MCPServers, tt.wantServers) {
				t.Errorf("MCPServers = %v, want %v", a.MCPServers, tt.wantServers)
			}
		})
	}
}

// TestAdoptCommitted_PolicyBlocks verifies the hooks block and
// claude_code.permissions always come from the committed .qsdev.yaml: the
// agent-writable answers file can neither keep a rule the committed file
// lacks nor survive the committed file's removal.
func TestAdoptCommitted_PolicyBlocks(t *testing.T) {
	t.Parallel()
	stale := types.WizardAnswers{
		HookPolicy:        types.HooksConfig{ToolGates: types.ToolGatesConfig{Allowed: []string{"*"}}},
		ClaudePermissions: types.ClaudePermissionsConfig{Allow: []string{"Bash(curl *)"}},
	}
	tests := []struct {
		name      string
		config    string // .qsdev.yaml content; empty means no file
		wantGates []string
		wantAllow []string
	}{
		{"committed blocks adopted",
			"version: 1\nhooks:\n  tool_gates:\n    denied: [WebFetch]\nclaude_code:\n  permissions:\n    allow: [\"Bash(make lint)\"]\n",
			nil, []string{"Bash(make lint)"}},
		{"committed file without blocks clears them", "version: 1\n", nil, nil},
		{"no committed file clears them", "", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCommittedConfig(t, root, tt.config)
			a := stale
			a.HookPolicy = stale.HookPolicy.Clone()
			a.ClaudePermissions = stale.ClaudePermissions.Clone()
			AdoptCommitted(root, &a)
			if !slices.Equal(a.HookPolicy.ToolGates.Allowed, tt.wantGates) {
				t.Errorf("HookPolicy.ToolGates.Allowed = %v, want %v", a.HookPolicy.ToolGates.Allowed, tt.wantGates)
			}
			if !slices.Equal(a.ClaudePermissions.Allow, tt.wantAllow) {
				t.Errorf("ClaudePermissions.Allow = %v, want %v", a.ClaudePermissions.Allow, tt.wantAllow)
			}
		})
	}
}

// TestAdoptCommitted_LanguageVersions verifies a regeneration from stale saved
// answers keeps the language version .qsdev.yaml pins (the self-regeneration
// that rewrote this repository's Go 1.26.7 pin to 1.26.3), raised to what the
// project's go.mod requires, and that only the committed file lowers a pin.
func TestAdoptCommitted_LanguageVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   string // .qsdev.yaml content; empty means no file
		answers  string // saved answers' go version
		detected string // go.mod go directive
		want     string
	}{
		{"committed newer pin kept over stale answers",
			"version: 2\nlanguages:\n  - name: go\n    version: 1.26.7\n", "1.26.3", "1.26.3", "1.26.7"},
		{"committed lower pin is the team's choice",
			"version: 2\nlanguages:\n  - name: go\n    version: 1.25.4\n", "1.26.7", "1.24", "1.25.4"},
		{"committed pin raised to go.mod",
			"version: 2\nlanguages:\n  - name: go\n    version: 1.25.4\n", "1.25.4", "1.26.7", "1.26.7"},
		{"unpinned committed language keeps answers",
			"version: 2\nlanguages:\n  - name: go\n", "1.26.3", "1.24", "1.26.3"},
		{"no config still raised to go.mod", "", "1.26.3", "1.26.7", "1.26.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCommittedConfig(t, root, tt.config)
			a := types.WizardAnswers{
				Detected:  types.DetectedProject{HasGoMod: true, GoVersion: tt.detected},
				Languages: []types.LanguageChoice{{Name: "go", Version: tt.answers}},
			}
			AdoptCommitted(root, &a)
			if got := a.Languages[0].Version; got != tt.want {
				t.Errorf("go version = %q, want %q", got, tt.want)
			}
		})
	}
}
