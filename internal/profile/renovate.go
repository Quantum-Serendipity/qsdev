package profile

import (
	"encoding/json"
	"fmt"
	"os"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// generateRenovateJSON produces a renovate.json GeneratedFile from the
// profile's update configuration.
func (p *InfraProfile) generateRenovateJSON() types.GeneratedFile {
	type packageRule struct {
		MatchUpdateTypes    []string `json:"matchUpdateTypes,omitempty"`
		MinimumReleaseAge   string   `json:"minimumReleaseAge,omitempty"`
		AutomergeType       string   `json:"automergeType,omitempty"`
		Automerge           bool     `json:"automerge,omitempty"`
		MatchManagers       []string `json:"matchManagers,omitempty"`
		MatchCategories     []string `json:"matchCategories,omitempty"`
		Labels              []string `json:"labels,omitempty"`
		MatchDepTypes       []string `json:"matchDepTypes,omitempty"`
	}

	type renovateConfig struct {
		Schema      string        `json:"$schema"`
		Extends     []string      `json:"extends"`
		PackageRules []packageRule `json:"packageRules,omitempty"`
	}

	cfg := renovateConfig{
		Schema:  "https://docs.renovatebot.com/renovate-schema.json",
		Extends: []string{"config:recommended"},
	}

	// Default age-gating rule.
	if p.Updates.AgeGatingDays > 0 {
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MinimumReleaseAge: fmt.Sprintf("%d days", p.Updates.AgeGatingDays),
		})
	}

	// Vulnerability alerts bypass age-gating.
	cfg.PackageRules = append(cfg.PackageRules, packageRule{
		MatchCategories:   []string{"vulnerability"},
		MinimumReleaseAge: "0 days",
		Labels:            []string{"security"},
	})

	// Automerge patches if enabled.
	if p.Updates.AutomergePatches {
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MatchUpdateTypes: []string{"patch"},
			Automerge:        true,
			AutomergeType:    "pr",
		})
	}

	// Ecosystem-specific overrides.
	for eco, days := range p.Updates.EcosystemOverrides {
		manager := ecosystemToRenovateManager(eco)
		if manager == "" {
			continue
		}
		cfg.PackageRules = append(cfg.PackageRules, packageRule{
			MatchManagers:     []string{manager},
			MinimumReleaseAge: fmt.Sprintf("%d days", days),
		})
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	data = append(data, '\n')

	return types.GeneratedFile{
		Path:     "renovate.json",
		Content:  data,
		Mode:     os.FileMode(0644),
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
	case "docker":
		return "dockerfile"
	case "terraform":
		return "terraform"
	default:
		return ""
	}
}
