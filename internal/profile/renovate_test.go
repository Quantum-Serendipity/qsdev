package profile

import (
	"strings"
	"testing"
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
		got := string(p.generateRenovateJSON().Content)
		if got != want {
			t.Fatalf("renovate.json mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	}
}

func TestGenerateRenovateJSON_NoVulnerabilityCategory(t *testing.T) {
	t.Parallel()

	got := string((&InfraProfile{}).generateRenovateJSON().Content)
	if strings.Contains(got, "matchCategories") {
		t.Errorf("renovate.json must not use matchCategories for security fixes (no such category):\n%s", got)
	}
	if !strings.Contains(got, `"vulnerabilityAlerts"`) {
		t.Errorf("renovate.json missing vulnerabilityAlerts:\n%s", got)
	}
}
