// Package devinit provides the orchestration addon for qsdev init.
package devinit

import (
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

	// Languages come from the same detection mapping FillDefaults uses, so
	// the wizard's defaults match what the quick/--yes path generates. Only
	// registered ecosystem modules are offered (detection-only aliases are
	// not modules).
	answers.Languages = registeredLanguages(detected.LanguageChoices())

	// --- Scalar field mappings ---

	if detected.HasEnvrc {
		answers.Direnv = true
	}

	if detected.HasClaudeDir || detected.HasClaudeMd || detected.HasClaudeSettings {
		answers.ClaudeCode = true
	}

	return answers
}

// registeredLanguages keeps the language choices that name a registered
// ecosystem module, in order. Deriving the check from the registry means a new
// module needs no change here.
func registeredLanguages(langs []types.LanguageChoice) []types.LanguageChoice {
	registry := ecosystem.DefaultRegistry()
	return slices.DeleteFunc(langs, func(l types.LanguageChoice) bool {
		_, ok := registry.ByName(l.Name)
		return !ok
	})
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
