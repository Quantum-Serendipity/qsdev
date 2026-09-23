package profile

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// generateRenovateJSON produces a renovate.json GeneratedFile from the
// profile's update configuration.
func (p *InfraProfile) generateRenovateJSON() types.GeneratedFile {
	type packageRule struct {
		MatchUpdateTypes  []string `json:"matchUpdateTypes,omitempty"`
		MinimumReleaseAge string   `json:"minimumReleaseAge,omitempty"`
		AutomergeType     string   `json:"automergeType,omitempty"`
		Automerge         bool     `json:"automerge,omitempty"`
		MatchManagers     []string `json:"matchManagers,omitempty"`
		MatchDepTypes     []string `json:"matchDepTypes,omitempty"`
	}

	// vulnerabilityAlerts configures Renovate's security-fix updates. An
	// explicit null minimumReleaseAge exempts them from the age gate below.
	// (matchCategories selects language/manager categories such as "js" or
	// "golang"; there is no "vulnerability" category, so a packageRule cannot
	// target security fixes.)
	type vulnerabilityAlerts struct {
		Labels            []string `json:"labels"`
		MinimumReleaseAge *string  `json:"minimumReleaseAge"`
	}

	type renovateConfig struct {
		Schema              string              `json:"$schema"`
		Extends             []string            `json:"extends"`
		VulnerabilityAlerts vulnerabilityAlerts `json:"vulnerabilityAlerts"`
		PackageRules        []packageRule       `json:"packageRules,omitempty"`
	}

	cfg := renovateConfig{
		Schema:  "https://docs.renovatebot.com/renovate-schema.json",
		Extends: []string{"config:recommended"},
		// Vulnerability alerts bypass age-gating and are labelled.
		VulnerabilityAlerts: vulnerabilityAlerts{Labels: []string{"security"}},
	}

	// Default age-gating rule.
	if p.Updates.AgeGatingDays > 0 {
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MinimumReleaseAge: fmt.Sprintf("%d days", p.Updates.AgeGatingDays),
		})
	}

	// Automerge patches if enabled.
	if p.Updates.AutomergePatches {
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MatchUpdateTypes: []string{"patch"},
			Automerge:        true,
			AutomergeType:    "pr",
		})
	}

	// Ecosystem-specific overrides, in sorted order so the generated file is
	// byte-for-byte stable across runs.
	for _, eco := range slices.Sorted(maps.Keys(p.Updates.EcosystemOverrides)) {
		manager := ecosystemToRenovateManager(eco)
		if manager == "" {
			continue
		}
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MatchManagers:     []string{manager},
			MinimumReleaseAge: fmt.Sprintf("%d days", p.Updates.EcosystemOverrides[eco]),
		})
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	data = append(data, '\n')

	return types.GeneratedFile{
		Path:     "renovate.json",
		Content:  data,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}
}

// ecosystemToRenovateManager maps ecosystem names to Renovate manager names.
func ecosystemToRenovateManager(eco string) string {
	switch eco {
	case "npm":
		return "npm"
	case "pypi":
		return "pip_requirements"
	case "go":
		return "gomod"
	case "cargo":
		return "cargo"
	case "maven":
		return "maven"
	case "nuget":
		return "nuget"
	case "container":
		return "dockerfile"
	case "terraform":
		return "terraform"
	default:
		return ""
	}
}
