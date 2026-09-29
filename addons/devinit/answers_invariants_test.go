package devinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestEnforceAnswerInvariants_RecordsTier verifies answers saved before the
// tier was always persisted get their legacy-inferred tier recorded (so update
// writes it back), and that default MCP servers never make that tier full.
func TestEnforceAnswerInvariants_RecordsTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    string
	}{
		{"explicit tier kept", types.WizardAnswers{Tier: "full"}, "full"},
		{"legacy default MCP servers", types.WizardAnswers{
			ClaudeCode: true, PermissionLevel: "standard",
			MCPServers: []string{"context7", "github", "socket", "semble"},
		}, "standard"},
		{"legacy supply-chain-only", types.WizardAnswers{ClaudeCode: true, PermissionLevel: "supply-chain-only"}, "supply-chain-only"},
		{"legacy non-default MCP server", types.WizardAnswers{
			ClaudeCode: true, PermissionLevel: "standard", MCPServers: []string{"custom-db"},
		}, "full"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := tt.answers
			enforceAnswerInvariants(&a)
			if a.Tier != tt.want {
				t.Errorf("Tier = %q, want %q", a.Tier, tt.want)
			}
		})
	}
}

// TestAdoptCommittedTier verifies update gives legacy answers (no tier) the
// tier committed in .qsdev.yaml rather than an inferred one, and falls back to
// inference only when the config records no valid tier.
func TestAdoptCommittedTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		config    string // .qsdev.yaml content; empty means no file
		answers   types.WizardAnswers
		wantTier  string
		wantFinal string // after enforceAnswerInvariants
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
			if tt.config != "" {
				path := filepath.Join(root, branding.Get().ConfigFile)
				if err := os.WriteFile(path, []byte(tt.config), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			a := tt.answers
			adoptCommittedTier(root, &a)
			if a.Tier != tt.wantTier {
				t.Errorf("after adopt Tier = %q, want %q", a.Tier, tt.wantTier)
			}
			enforceAnswerInvariants(&a)
			if a.Tier != tt.wantFinal {
				t.Errorf("final Tier = %q, want %q", a.Tier, tt.wantFinal)
			}
		})
	}
}
