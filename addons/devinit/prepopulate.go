// Package devinit provides the orchestration addon for gdev init.
package devinit

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: "docker",
		})
	}

	if detected.HasTerraform {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: "terraform",
		})
	}

	// Tier 2-4 ecosystems use the forward-compatible Ecosystems map.
	ecosystemLanguages := []string{
		"php", "ruby", "scala", "cpp", "helm", "ansible", "shell",
		"elixir", "dart", "swift", "haskell", "clojure", "bazel", "nix",
		"perl", "r", "lua", "zig", "powershell",
	}
	for _, name := range ecosystemLanguages {
		if detected.Ecosystems[name] {
			answers.Languages = append(answers.Languages, types.LanguageChoice{
				Name: name,
			})
		}
	}

	// --- Scalar field mappings ---

	if detected.HasEnvrc {
		answers.Direnv = true
	}

	if detected.HasClaudeDir || detected.HasClaudeMd || detected.HasClaudeSettings {
		answers.ClaudeCode = true
	}

	return answers
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
	// Remove trailing .git suffix
	url = strings.TrimSuffix(url, ".git")
	// Remove trailing slash
	url = strings.TrimRight(url, "/")

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
