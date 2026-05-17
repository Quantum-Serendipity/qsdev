package profile

import (
	"os"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

// generateDependabotYML produces a .github/dependabot.yml GeneratedFile from
// the profile's update configuration.
func (p *InfraProfile) generateDependabotYML() types.GeneratedFile {
	type schedule struct {
		Interval string `yaml:"interval"`
	}

	type updateEntry struct {
		PackageEcosystem      string   `yaml:"package-ecosystem"`
		Directory             string   `yaml:"directory"`
		Schedule              schedule `yaml:"schedule"`
		OpenPullRequestsLimit int      `yaml:"open-pull-requests-limit"`
	}

	type dependabotConfig struct {
		Version int           `yaml:"version"`
		Updates []updateEntry `yaml:"updates"`
	}

	cfg := dependabotConfig{
		Version: 2,
	}

	for _, eco := range p.Registry.Ecosystems {
		depEco := ecosystemToDependabotEcosystem(eco)
		if depEco == "" {
			continue
		}
		cfg.Updates = append(cfg.Updates, updateEntry{
			PackageEcosystem:      depEco,
			Directory:             "/",
			Schedule:              schedule{Interval: "weekly"},
			OpenPullRequestsLimit: 10,
		})
	}

	// Always include docker and terraform if not already covered by registry
	// ecosystems, since Dependabot supports them natively.
	seen := make(map[string]bool)
	for _, u := range cfg.Updates {
		seen[u.PackageEcosystem] = true
	}
	for _, extra := range []string{"docker", "terraform"} {
		depEco := ecosystemToDependabotEcosystem(extra)
		if depEco != "" && !seen[depEco] {
			cfg.Updates = append(cfg.Updates, updateEntry{
				PackageEcosystem:      depEco,
				Directory:             "/",
				Schedule:              schedule{Interval: "weekly"},
				OpenPullRequestsLimit: 10,
			})
		}
	}

	data, _ := yaml.Marshal(cfg)

	// Prepend a comment header.
	var buf strings.Builder
	buf.WriteString("# Managed by qsdev. Do not edit.\n")
	buf.Write(data)

	return types.GeneratedFile{
		Path:     ".github/dependabot.yml",
		Content:  []byte(buf.String()),
		Mode:     os.FileMode(0o644),
		Strategy: types.Overwrite,
	}
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
	case "nuget":
		return "nuget"
	case "docker":
		return "docker"
	case "terraform":
		return "terraform"
	default:
		return ""
	}
}
