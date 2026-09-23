package devinit

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func denySetRules(t *testing.T, set string) []string {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	rules := cat.PermissionDenyRules(set)
	if len(rules) == 0 {
		t.Fatalf("catalog deny set %q is empty", set)
	}
	return rules
}

// TestRequiredDenyRules_CoverPresetDenySets is the regression test for
// `qsdev check` enforcing only two deny sets: removing a sudo_prefix or
// sandbox_config_edits rule from settings.json went unnoticed.
func TestRequiredDenyRules_CoverPresetDenySets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		answers     types.WizardAnswers
		cfg         *types.QsdevConfig
		mustInclude []string
		mustExclude []string
	}{
		{
			name:        "standard preset requires every standard deny set",
			answers:     types.WizardAnswers{PermissionLevel: "standard"},
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits", "shell_wrapping", "destructive_ops", "pipe_to_shell"},
		},
		{
			name:        "no answers or config falls back to standard",
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits"},
		},
		{
			name:        "supply-chain-only preset does not require sets it never generates",
			answers:     types.WizardAnswers{PermissionLevel: "supply-chain-only"},
			mustInclude: []string{"sudo_prefix", "pipe_to_shell"},
			mustExclude: []string{"destructive_ops", "sandbox_config_edits"},
		},
		{
			name:        "tier from project config selects its default preset",
			cfg:         &types.QsdevConfig{Tier: "supply-chain-only"},
			mustInclude: []string{"pipe_to_shell"},
			mustExclude: []string{"sandbox_config_edits"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requiredDenyRules(tt.answers, tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			for _, set := range tt.mustInclude {
				for _, rule := range denySetRules(t, set) {
					if !slices.Contains(got, rule) {
						t.Errorf("required rules missing %q from deny set %s", rule, set)
					}
				}
			}
			for _, set := range tt.mustExclude {
				for _, rule := range denySetRules(t, set) {
					if slices.Contains(got, rule) {
						t.Errorf("required rules include %q from deny set %s, which this preset does not generate", rule, set)
					}
				}
			}
		})
	}
}

// TestRequiredDenyRules_MatchGeneratedSettings verifies that check never
// requires a rule the generator does not write, so a freshly generated
// project passes deny_rules_present for every permission preset.
func TestRequiredDenyRules_MatchGeneratedSettings(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, preset := range cat.PermissionPresets() {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{ClaudeCode: true, PermissionLevel: preset}
			file, err := claudecode.GenerateSettings(answers, ecosystem.DefaultRegistry(), claudecode.Config{})
			if err != nil {
				t.Fatal(err)
			}
			var settings struct {
				Permissions struct {
					Deny []string `json:"deny"`
				} `json:"permissions"`
			}
			if err := json.Unmarshal(file.Content, &settings); err != nil {
				t.Fatal(err)
			}
			required, err := requiredDenyRules(answers, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, rule := range required {
				if !slices.Contains(settings.Permissions.Deny, rule) {
					t.Errorf("check requires %q but generated settings.json does not contain it", rule)
				}
			}
		})
	}
}
