package profile

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateRenovateJSON_Golden(t *testing.T) {
	t.Parallel()

	p := &InfraProfile{Updates: UpdateConfig{
		AgeGatingDays:    3,
		AutomergePatches: true,
		EcosystemOverrides: map[string]int{
			"npm": 7, "go": 1, "cargo": 5, "unknown-eco": 9, "pypi": 2,
		},
	}}

	const want = `{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    "config:recommended"
  ],
  "vulnerabilityAlerts": {
    "labels": [
      "security"
    ],
    "minimumReleaseAge": null
  },
  "packageRules": [
    {
      "minimumReleaseAge": "3 days"
    },
    {
      "matchUpdateTypes": [
        "patch"
      ],
      "automergeType": "pr",
      "automerge": true
    },
    {
      "minimumReleaseAge": "5 days",
      "matchManagers": [
        "cargo"
      ]
    },
    {
      "minimumReleaseAge": "1 days",
      "matchManagers": [
        "gomod"
      ]
    },
    {
      "minimumReleaseAge": "7 days",
      "matchManagers": [
        "npm"
      ]
    },
    {
      "minimumReleaseAge": "2 days",
      "matchManagers": [
        "pip_requirements"
      ]
    }
  ]
}
`
	// Generate repeatedly: map iteration order must not leak into the output.
	for range 20 {
		got := string(p.generateRenovateJSON(ProjectInputs{}).Content)
		if got != want {
			t.Fatalf("renovate.json mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	}
}

func TestGenerateRenovateJSON_NoVulnerabilityCategory(t *testing.T) {
	t.Parallel()

	got := string((&InfraProfile{}).generateRenovateJSON(ProjectInputs{}).Content)
	if strings.Contains(got, "matchCategories") {
		t.Errorf("renovate.json must not use matchCategories for security fixes (no such category):\n%s", got)
	}
	if !strings.Contains(got, `"vulnerabilityAlerts"`) {
		t.Errorf("renovate.json missing vulnerabilityAlerts:\n%s", got)
	}
}

type renovateFile struct {
	PackageRules []struct {
		MinimumReleaseAge string   `json:"minimumReleaseAge"`
		MatchManagers     []string `json:"matchManagers"`
	} `json:"packageRules"`
}

// renovateAges maps each age-gated packageRule to its minimumReleaseAge:
// "" for the default rule, otherwise the rule's first manager.
func renovateAges(t *testing.T, f types.GeneratedFile) map[string]string {
	t.Helper()
	var cfg renovateFile
	if err := json.Unmarshal(f.Content, &cfg); err != nil {
		t.Fatalf("renovate.json is not valid JSON: %v\n%s", err, f.Content)
	}
	ages := make(map[string]string)
	for _, r := range cfg.PackageRules {
		if r.MinimumReleaseAge == "" {
			continue
		}
		key := ""
		if len(r.MatchManagers) > 0 {
			key = r.MatchManagers[0]
		}
		ages[key] = r.MinimumReleaseAge
	}
	return ages
}

// TestRenovate_MinimumReleaseAgeByComplianceLevel pins the D18 rule for the
// Renovate update PR delay: the larger of the profile's (or an ecosystem
// override's) days and the compliance level's window, never looser than
// either.
func TestRenovate_MinimumReleaseAgeByComplianceLevel(t *testing.T) {
	t.Parallel()
	overrides := &InfraProfile{Updates: UpdateConfig{
		Type:               UpdateToolRenovate,
		AgeGatingDays:      3,
		EcosystemOverrides: map[string]int{"go": 1, "npm": 30},
	}}
	tests := []struct {
		name  string
		p     *InfraProfile
		level string
		want  map[string]string
	}{
		{"consulting-default baseline", ConsultingDefault, "baseline", map[string]string{"": "3 days"}},
		{"consulting-default enhanced", ConsultingDefault, "enhanced", map[string]string{"": "7 days"}},
		{"consulting-default strict", ConsultingDefault, "strict", map[string]string{"": "14 days"}},
		{"enterprise baseline never looser", Enterprise, "baseline", map[string]string{"": "7 days"}},
		{"enterprise strict", Enterprise, "strict", map[string]string{"": "14 days"}},
		{
			"override below tier raised, above kept", overrides, "enhanced",
			map[string]string{"": "7 days", "gomod": "7 days", "npm": "30 days"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := ProjectInputsFromAnswers(types.WizardAnswers{ComplianceLevel: tt.level})
			got := renovateAges(t, mustFile(t, tt.p, in, "renovate.json"))
			if !maps.Equal(got, tt.want) {
				t.Errorf("renovate minimumReleaseAge = %v, want %v", got, tt.want)
			}
		})
	}
}
