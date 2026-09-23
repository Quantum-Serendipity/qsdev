// Package devinit provides the orchestration addon for qsdev init.
package devinit

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// MapDetectionToDefaults converts detection results into pre-populated
// WizardAnswers defaults. It maps detected languages and configuration
// state into the initial values that the wizard will present to the user.
func MapDetectionToDefaults(detected types.DetectedProject, projectRoot string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectName: projectNameFromDetection(detected, projectRoot),
		ProjectRoot: projectRoot,
		Detected:    detected,
	}

	// --- Language mappings ---

	if detected.HasGoMod {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:    "go",
			Version: detected.GoVersion,
		})
	}

	// CRITICAL: detection sets Ecosystems["node"] but the canonical name is "javascript"
	if detected.HasPackageJSON {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:           "javascript",
			Version:        detected.NodeVersion,
			PackageManager: detected.PackageManager,
		})
	}

	if detected.HasPyProject {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name:    "python",
			Version: detected.PythonVersion,
		})
	}

	if detected.HasCargoToml {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: "rust",
		})
	}

	if detected.HasPomXML || detected.HasBuildGradle {
		jc := types.LanguageChoice{Name: "java"}
		switch {
		case detected.HasPomXML && detected.HasBuildGradle:
			jc.Extras = []string{"build_tool=both"}
		case detected.HasPomXML:
			jc.Extras = []string{"build_tool=maven"}
		case detected.HasBuildGradle:
			jc.Extras = []string{"build_tool=gradle"}
		}
		answers.Languages = append(answers.Languages, jc)
	}

	if detected.HasCsproj {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: "dotnet",
		})
	}

	if detected.HasDockerfile {
		dc := types.LanguageChoice{Name: "container"}
		if detected.ContainerRuntime != "" {
			dc.Extras = append(dc.Extras, "container_runtime="+detected.ContainerRuntime)
		}
		if detected.OSFamily != "" {
			dc.Extras = append(dc.Extras, "os_family="+detected.OSFamily)
		}
		answers.Languages = append(answers.Languages, dc)
	}

	if detected.HasTerraform {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: "terraform",
		})
	}

	answers.Languages = appendDetectedEcosystems(answers.Languages, detected)

	// --- Scalar field mappings ---

	if detected.HasEnvrc {
		answers.Direnv = true
	}

	if detected.HasClaudeDir || detected.HasClaudeMd || detected.HasClaudeSettings {
		answers.ClaudeCode = true
	}

	return answers
}

// appendDetectedEcosystems adds every other detected ecosystem (tiers 2-4)
// that is a registered ecosystem module and not already in langs, in name
// order. Deriving the list from the registry means a new module needs no change
// here; detection-only aliases such as "node" are not modules and are skipped.
func appendDetectedEcosystems(langs []types.LanguageChoice, detected types.DetectedProject) []types.LanguageChoice {
	registry := ecosystem.DefaultRegistry()
	for _, name := range slices.Sorted(maps.Keys(detected.Ecosystems)) {
		if !detected.Ecosystems[name] {
			continue
		}
		if slices.ContainsFunc(langs, func(l types.LanguageChoice) bool { return l.Name == name }) {
			continue
		}
		if _, ok := registry.ByName(name); !ok {
			continue
		}
		langs = append(langs, types.LanguageChoice{Name: name})
	}
	return langs
}

// projectNameFromDetection derives a project name from the detection results.
// It prefers extracting the repository name from the remote URL, falling back
// to the base directory name.
func projectNameFromDetection(detected types.DetectedProject, projectRoot string) string {
	if detected.RemoteURL != "" {
		if name := extractRepoName(detected.RemoteURL); name != "" {
			return name
		}
	}
	return filepath.Base(projectRoot)
}

// extractRepoName extracts the repository name from a git remote URL.
// It handles HTTPS URLs (https://github.com/org/repo.git), SSH URLs
// (git@github.com:org/repo.git), and plain paths.
func extractRepoName(url string) string {
	// Remove trailing slashes first so "repo.git/" still loses its .git suffix.
	url = strings.TrimRight(url, "/")
	url = strings.TrimSuffix(url, ".git")

	if url == "" {
		return ""
	}

	// SSH-style: git@host:org/repo
	if idx := strings.LastIndex(url, ":"); idx != -1 && !strings.Contains(url, "://") {
		path := url[idx+1:]
		if slashIdx := strings.LastIndex(path, "/"); slashIdx != -1 {
			return path[slashIdx+1:]
		}
		return path
	}

	// HTTPS or path-style: take the last path segment
	if slashIdx := strings.LastIndex(url, "/"); slashIdx != -1 {
		return url[slashIdx+1:]
	}

	return url
}
