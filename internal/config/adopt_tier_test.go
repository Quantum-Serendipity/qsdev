package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestAdoptCommittedTier verifies regeneration gives legacy answers (no tier) the
// tier committed in .qsdev.yaml rather than an inferred one, and falls back to
// inference only when the config records no valid tier.
func TestAdoptCommittedTier(t *testing.T) {
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
			if tt.config != "" {
				path := filepath.Join(root, branding.Get().ConfigFile)
				if err := os.WriteFile(path, []byte(tt.config), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			a := tt.answers
			AdoptCommittedTier(root, &a)
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
