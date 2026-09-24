package devenv

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// renderedHookIDs returns every git-hooks attribute BuildDevenvNixData
// enables or scopes.
func renderedHookIDs(data *DevenvNixTemplateData) map[string]bool {
	ids := make(map[string]bool)
	for _, id := range data.SecurityHooks {
		ids[id] = true
	}
	for _, h := range data.BuiltInHooks {
		ids[h.ID] = true
	}
	for _, h := range data.CustomHooks {
		ids[h.ID] = true
	}
	for _, h := range data.HookOverrides {
		ids[h.ID] = true
	}
	return ids
}

func TestBuildDevenvNixData_HookTier(t *testing.T) {
	t.Parallel()
	langs := []types.LanguageChoice{{Name: "go"}, {Name: "python"}, {Name: "shell"}, {Name: "terraform"}, {Name: "nix"}}
	// Security hooks BuildDevenvNixData renders for these languages; they must
	// survive at every level.
	security := []string{"ripsecrets", "shellcheck", "nix-secrets-check", "lock-file-audit", "govulncheck", "bandit", "tfsec"}
	hygiene := []string{"check-added-large-files", "no-commit-to-branch", "check-merge-conflicts"}
	language := []string{"gofmt", "govet", "staticcheck", "ruff", "shfmt", "tflint", "statix", "deadnix"}

	tests := []struct {
		name            string
		hookTier        string
		complianceLevel string
		wantLanguage    bool
	}{
		{name: "unset keeps every hook", wantLanguage: true},
		{name: "baseline drops language hooks", hookTier: "baseline", complianceLevel: "baseline"},
		{name: "enhanced keeps language hooks", hookTier: "enhanced", complianceLevel: "enhanced", wantLanguage: true},
		{name: "strict keeps language hooks", hookTier: "strict", complianceLevel: "strict", wantLanguage: true},
		{name: "compliance level alone selects the tier", complianceLevel: "baseline"},
		{name: "hook tier alone selects the tier", hookTier: "baseline"},
		{name: "stricter compliance level wins", hookTier: "baseline", complianceLevel: "enhanced", wantLanguage: true},
		{name: "stricter hook tier wins", hookTier: "strict", complianceLevel: "baseline", wantLanguage: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName:     "tiers",
				Languages:       langs,
				HookTier:        tt.hookTier,
				ComplianceLevel: tt.complianceLevel,
			}
			data, err := BuildDevenvNixData(answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}
			ids := renderedHookIDs(data)
			for _, id := range slices.Concat(security, hygiene) {
				if !ids[id] {
					t.Errorf("hook %q missing, want it at every level", id)
				}
			}
			for _, id := range language {
				if ids[id] != tt.wantLanguage {
					t.Errorf("hook %q rendered = %v, want %v", id, ids[id], tt.wantLanguage)
				}
			}
		})
	}
}

func TestBuildDevenvNixData_HookTierDropsHookPackages(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go"}}, HookTier: "baseline"}
	data, err := BuildDevenvNixData(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("BuildDevenvNixData: %v", err)
	}
	// staticcheck is tiered out, so its package is not pulled in for it;
	// govulncheck is a security hook and keeps its package.
	if slices.Contains(data.Packages, "go-tools") {
		t.Errorf("packages %v contain go-tools for the tiered-out staticcheck hook", data.Packages)
	}
	if !slices.Contains(data.Packages, "govulncheck") {
		t.Errorf("packages %v lack govulncheck for the govulncheck security hook", data.Packages)
	}
}

func TestBuildDevenvNixData_UnknownHookTier(t *testing.T) {
	t.Parallel()
	_, err := BuildDevenvNixData(types.WizardAnswers{HookTier: "maximum"}, ecosystem.DefaultRegistry())
	if err == nil || !strings.Contains(err.Error(), "unknown hook tier") {
		t.Fatalf("BuildDevenvNixData with an unknown hook tier: err = %v, want an unknown hook tier error", err)
	}
}
