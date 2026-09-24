package profile

import (
	"slices"

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
}

// ProjectInputsFromAnswers derives ProjectInputs from the wizard answers: the
// selected languages plus the manifests detected in the repository.
func ProjectInputsFromAnswers(answers types.WizardAnswers) ProjectInputs {
	var ecos []string
	for _, lang := range answers.Languages {
		if key := languageEcosystem(lang.Name, lang.PackageManager); key != "" {
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
	return ProjectInputs{Ecosystems: slices.Compact(ecos), Infrastructure: answers.Infrastructure}
}

// languageEcosystem maps a language selection to its package ecosystem key.
func languageEcosystem(name, packageManager string) string {
	if key := ecosystem.ProxyKeyForLanguage(name, packageManager); key != "" {
		return key
	}
	switch name {
	case ecosystem.NameContainer, ecosystem.NameTerraform:
		return name
	default:
		return ""
	}
}
