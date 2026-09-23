package profile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// generateDependabotYML produces a .github/dependabot.yml GeneratedFile.
//
// Update entries come from the ecosystems the project actually uses, not from
// p.Registry.Ecosystems: that list names what the organization's package
// proxy serves, and entries for manifests the repository does not have make
// Dependabot fail while the project's real ecosystems go unwatched. The
// profile's age gate is emitted as Dependabot's cooldown.
func (p *InfraProfile) generateDependabotYML(in ProjectInputs) (types.GeneratedFile, error) {
	type schedule struct {
		Interval string `yaml:"interval"`
	}

	type cooldown struct {
		DefaultDays int `yaml:"default-days"`
	}

	type updateEntry struct {
		PackageEcosystem      string    `yaml:"package-ecosystem"`
		Directory             string    `yaml:"directory"`
		Schedule              schedule  `yaml:"schedule"`
		Cooldown              *cooldown `yaml:"cooldown,omitempty"`
		OpenPullRequestsLimit int       `yaml:"open-pull-requests-limit"`
	}

	type dependabotConfig struct {
		Version int           `yaml:"version"`
		Updates []updateEntry `yaml:"updates"`
	}

	var cd *cooldown
	if p.Updates.AgeGatingDays > 0 {
		cd = &cooldown{DefaultDays: p.Updates.AgeGatingDays}
	}

	var depEcos []string
	for _, eco := range in.Ecosystems {
		depEcos = append(depEcos, ecosystemToDependabotEcosystem(eco))
	}
	// The profile's own security-scan workflow pins actions by SHA; keep
	// those pins current through the github-actions ecosystem.
	if p.generatesSecurityScanWorkflow() {
		depEcos = append(depEcos, "github-actions")
	}

	cfg := dependabotConfig{Version: 2}
	seen := make(map[string]bool)
	for _, depEco := range depEcos {
		if depEco == "" || seen[depEco] {
			continue
		}
		seen[depEco] = true
		cfg.Updates = append(cfg.Updates, updateEntry{
			PackageEcosystem:      depEco,
			Directory:             "/",
			Schedule:              schedule{Interval: "weekly"},
			Cooldown:              cd,
			OpenPullRequestsLimit: 10,
		})
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return types.GeneratedFile{}, fmt.Errorf("marshaling dependabot config: %w", err)
	}

	// Prepend a comment header.
	var buf strings.Builder
	buf.WriteString("# Managed by qsdev. Do not edit.\n")
	buf.Write(data)

	return types.GeneratedFile{
		Path:     ".github/dependabot.yml",
		Content:  []byte(buf.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}, nil
}

// ecosystemToDependabotEcosystem maps ecosystem names to Dependabot
// package-ecosystem values.
func ecosystemToDependabotEcosystem(eco string) string {
	switch eco {
	case "npm":
		return "npm"
	case "pypi":
		return "pip"
	case "go":
		return "gomod"
	case "cargo":
		return "cargo"
	case "maven":
		return "maven"
	case "gradle":
		return "gradle"
	case "nuget":
		return "nuget"
	case "composer":
		return "composer"
	case "container":
		return "docker"
	case "terraform":
		return "terraform"
	default:
		return ""
	}
}
