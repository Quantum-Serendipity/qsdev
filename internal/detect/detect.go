// Package detect scans a project directory to identify programming languages,
// build systems, and existing configuration state. The results are returned
// as a [types.DetectedProject] that drives the rest of the init wizard.
package detect

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Detect scans projectRoot for language markers, lockfiles, configuration
// files, and git metadata, returning a fully populated DetectedProject.
func Detect(projectRoot string) types.DetectedProject {
	dp := types.DetectedProject{
		Ecosystems: make(map[string]bool),
	}

	// --- Languages / ecosystems ---

	if detected, version := detectGo(projectRoot); detected {
		dp.HasGoMod = true
		dp.GoVersion = version
		dp.Ecosystems["go"] = true
	}

	if detected, version, pm := detectNode(projectRoot); detected {
		dp.HasPackageJSON = true
		dp.NodeVersion = version
		dp.PackageManager = pm
		dp.Ecosystems["node"] = true
	}

	if detected, version, _ := detectPython(projectRoot); detected {
		dp.HasPyProject = true
		dp.PythonVersion = version
		dp.Ecosystems["python"] = true
	}

	if detectRust(projectRoot) {
		dp.HasCargoToml = true
		dp.Ecosystems["rust"] = true
	}

	hasMaven, hasGradle := detectJava(projectRoot)
	if hasMaven {
		dp.HasPomXML = true
		dp.Ecosystems["java"] = true
	}
	if hasGradle {
		dp.HasBuildGradle = true
		dp.Ecosystems["java"] = true
	}

	if detectDotNet(projectRoot) {
		dp.HasCsproj = true
		dp.Ecosystems["dotnet"] = true
	}

	if detectDocker(projectRoot) {
		dp.HasDockerfile = true
		dp.Ecosystems["docker"] = true
	}

	if detectTerraform(projectRoot) {
		dp.HasTerraform = true
		dp.Ecosystems["terraform"] = true
	}

	// --- Environment state ---

	env := detectEnvironment(projectRoot)
	dp.HasDevenvNix = env.HasDevenvNix
	dp.HasDevenvYaml = env.HasDevenvYaml
	dp.HasClaudeDir = env.HasClaudeDir
	dp.HasClaudeMd = env.HasClaudeMd
	dp.HasClaudeSettings = env.HasClaudeSettings
	dp.HasEnvrc = env.HasEnvrc
	dp.HasMcpJson = env.HasMcpJson
	dp.IsGitRepo = env.IsGitRepo
	dp.HasGitHooks = env.HasGitHooks
	dp.RemoteURL = env.RemoteURL

	return dp
}
