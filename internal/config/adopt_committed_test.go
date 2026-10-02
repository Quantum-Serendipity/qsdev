package config

import (
	"os"
	"path/filepath"
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
