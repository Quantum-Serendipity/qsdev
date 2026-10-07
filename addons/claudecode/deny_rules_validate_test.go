package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestEveryDenyRuleValidates generates settings at every catalog permission
// preset with every registered ecosystem enabled and checks each deny, ask
// and allow rule against denyutil.Validate: what reaches settings.json must
// be well formed and free of the legacy ":*" suffix, which Claude Code reads
// as a trailing " *".
func TestEveryDenyRuleValidates(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	reg := ecosystem.DefaultRegistry()
	var langs []types.LanguageChoice
	for _, mod := range reg.All() {
		langs = append(langs, types.LanguageChoice{Name: mod.Name()})
	}
	for _, preset := range cat.PermissionPresets() {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{PermissionLevel: preset, Languages: langs}
			s := mustUnmarshalSettings(t, mustGenerateSettings(t, answers, reg))
			if len(s.Permissions.Deny) == 0 {
				t.Fatal("preset generated no deny rules")
			}
			for kind, rules := range map[string][]string{
				"deny": s.Permissions.Deny, "ask": s.Permissions.Ask, "allow": s.Permissions.Allow,
			} {
				for _, rule := range rules {
					if err := denyutil.Validate(rule); err != nil {
						t.Errorf("%s: %v", kind, err)
					}
				}
			}
		})
	}
}
