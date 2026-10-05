package profile

import (
	"slices"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProjectInputs carries the facts about the project being generated that
// profile-driven config files depend on. The profile itself only describes
// organization-wide tooling choices; which package ecosystems a repository
// uses is a property of the project.
type ProjectInputs struct {
	// Ecosystems are the package ecosystems the project uses, named with the
	// same keys as RegistryConfig.Ecosystems ("npm", "pypi", "go", "cargo",
	// "maven", "gradle", "nuget", "composer", "container", "terraform").
	Ecosystems []string
	// Infrastructure is the project's effective infrastructure settings
	// (after an explicit infra profile was resolved), which the security
	// overview describes.
	Infrastructure types.InfraConfig
	// CI are the project's ecosystem CI commands (frozen/locked installs,
	// tests, audits) grouped by phase, from each selected module's
	// CICommands. The security-scan workflow runs them in the project's
	// devenv shell. The caller that holds the ecosystem registry fills this
	// in (see ecosystem.AggregateCICommands); ProjectInputsFromAnswers
	// leaves it empty.
	CI []ecosystem.CIPhaseGroup
	// MinReleaseAge is the release-age window of the project's compliance
	// level (catalog age_gating_threshold_hours). Update tools delay PRs by
	// at least this long; zero means unknown and keeps the profile's delay.
	MinReleaseAge time.Duration
}

// updateAgeDays returns the update-PR delay in days for a profile (or
// ecosystem override) delay of days: the larger of it and the compliance
// window, so the tier never loosens a profile and a profile never loosens
// the tier. An unknown window keeps days.
func (in ProjectInputs) updateAgeDays(days int) int {
	if in.MinReleaseAge <= 0 {
		return days
	}
	return max(days, ecosystem.ReleaseAgeDays(in.MinReleaseAge))
}

// ProjectInputsFromAnswers derives ProjectInputs from the wizard answers: the
// selected languages plus the manifests detected in the repository, and the
// compliance level's release-age window.
func ProjectInputsFromAnswers(answers types.WizardAnswers) ProjectInputs {
	var ecos []string
	for _, lang := range answers.Languages {
		if key := languageEcosystem(lang); key != "" {
			ecos = append(ecos, key)
		}
	}

	d := answers.Detected
	for _, m := range []struct {
		present bool
		key     string
	}{
		{d.HasGoMod, "go"},
		{d.HasPackageJSON, "npm"},
		{d.HasPyProject, "pypi"},
		{d.HasCargoToml, "cargo"},
		{d.HasPomXML, "maven"},
		{d.HasBuildGradle, "gradle"},
		{d.HasCsproj, "nuget"},
		{d.HasDockerfile, "container"},
		{d.HasTerraform, "terraform"},
	} {
		if m.present {
			ecos = append(ecos, m.key)
		}
	}

	slices.Sort(ecos)
	return ProjectInputs{
		Ecosystems:     slices.Compact(ecos),
		Infrastructure: answers.Infrastructure,
		MinReleaseAge:  catalog.EffectiveAgeGate(answers.ComplianceLevel),
	}
}

// languageEcosystem maps a language selection to its package ecosystem key:
// the registry proxy key its module routes through (see
// ecosystem.ProxyKeyProvider), or the language name for the container and
// terraform ecosystems.
func languageEcosystem(lang types.LanguageChoice) string {
	if key := ecosystem.ProxyKeyForLanguage(lang); key != "" {
		return key
	}
	switch name := lang.Name; name {
	case ecosystem.NameContainer, ecosystem.NameTerraform:
		return name
	default:
		return ""
	}
}
