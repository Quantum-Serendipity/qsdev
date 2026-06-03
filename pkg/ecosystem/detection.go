package ecosystem

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DetectionSummary bundles per-module detection results with an aggregated
// DetectedProject that the wizard and generators consume.
type DetectionSummary struct {
	Project types.DetectedProject      `yaml:"project" json:"project"`
	Results map[string]DetectionResult `yaml:"results" json:"results"`
}

// EnvironmentState captures existing configuration and tooling state
// found in a project directory.
type EnvironmentState struct {
	HasDevenvNix      bool
	HasDevenvYaml     bool
	HasClaudeDir      bool
	HasClaudeMd       bool
	HasClaudeSettings bool
	HasEnvrc          bool
	HasMcpJson        bool
	IsGitRepo         bool
	HasGitHooks       bool
	RemoteURL         string
}

// DetectEnvironment scans the project root for configuration files,
// tooling state, and git metadata.
func DetectEnvironment(root string) EnvironmentState {
	s := EnvironmentState{
		HasDevenvNix:      fileutil.FileExists(root, "devenv.nix"),
		HasDevenvYaml:     fileutil.FileExists(root, "devenv.yaml"),
		HasClaudeDir:      fileutil.DirExists(root, ".claude"),
		HasClaudeMd:       fileutil.FileExists(root, "CLAUDE.md"),
		HasClaudeSettings: fileutil.FileExists(root, ".claude", "settings.json"),
		HasEnvrc:          fileutil.FileExists(root, ".envrc"),
		HasMcpJson:        fileutil.FileExists(root, ".mcp.json"),
		IsGitRepo:         isGitRepository(root),
	}

	if s.IsGitRepo {
		gitDir := resolveGitDir(root)
		s.HasGitHooks = hasExecutableHooks(filepath.Join(gitDir, "hooks"))
		s.RemoteURL = parseOriginURL(filepath.Join(gitDir, "config"))
	}

	return s
}

// isGitRepository returns true if root contains a .git directory (normal repo)
// or a .git file (worktree). Both are valid git working trees.
func isGitRepository(root string) bool {
	info, err := os.Stat(filepath.Join(root, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

// resolveGitDir returns the actual .git directory path. For normal repos this
// is root/.git. For worktrees, the .git file contains "gitdir: <path>" pointing
// to the real git directory; we follow it to find the common dir for config access.
func resolveGitDir(root string) string {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return gitPath
	}
	if info.IsDir() {
		return gitPath
	}
	// Worktree: .git is a file with "gitdir: <path>"
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return gitPath
	}
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "gitdir: ") {
		return gitPath
	}
	gitDir := strings.TrimPrefix(content, "gitdir: ")
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	// The gitdir points to .git/worktrees/<name>. The config is in the
	// common directory (parent of worktrees/). Check for a commondir file.
	commonDirPath := filepath.Join(gitDir, "commondir")
	if commonData, err := os.ReadFile(commonDirPath); err == nil {
		commonDir := strings.TrimSpace(string(commonData))
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
		return commonDir
	}
	return gitDir
}

// hasExecutableHooks returns true if the hooks directory contains at least
// one file with the executable bit set.
func hasExecutableHooks(hooksDir string) bool {
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Skip .sample files shipped by git init
		if strings.HasSuffix(e.Name(), ".sample") {
			continue
		}
		// On Windows, file mode bits are not meaningful; treat any
		// non-sample hook file as executable.
		if runtime.GOOS == "windows" {
			return true
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 != 0 {
			return true
		}
	}
	return false
}

// parseOriginURL extracts the URL for the remote named "origin" from a
// git config file. It uses a simple line scanner rather than a full INI parser.
func parseOriginURL(configPath string) string {
	f, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	inOrigin := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect section headers.
		if strings.HasPrefix(line, "[") {
			inOrigin = line == `[remote "origin"]`
			continue
		}

		if inOrigin && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// applyEnvironment copies EnvironmentState fields into a DetectedProject.
func applyEnvironment(p *types.DetectedProject, env EnvironmentState) {
	p.HasDevenvNix = env.HasDevenvNix
	p.HasDevenvYaml = env.HasDevenvYaml
	p.HasClaudeDir = env.HasClaudeDir
	p.HasClaudeMd = env.HasClaudeMd
	p.HasClaudeSettings = env.HasClaudeSettings
	p.HasEnvrc = env.HasEnvrc
	p.HasMcpJson = env.HasMcpJson
	p.IsGitRepo = env.IsGitRepo
	p.HasGitHooks = env.HasGitHooks
	p.RemoteURL = env.RemoteURL
}

// aggregateDetections maps individual module DetectionResults into the
// well-known fields of types.DetectedProject. Modules whose names do not
// correspond to a dedicated field are recorded in the Ecosystems map.
func aggregateDetections(results map[string]DetectionResult) types.DetectedProject {
	p := types.DetectedProject{
		Ecosystems: make(map[string]bool),
	}

	for name, dr := range results {
		if !dr.Detected {
			continue
		}

		// Record every detected ecosystem in the extensible map.
		p.Ecosystems[name] = true

		// Populate well-known fields for modules that have dedicated struct fields.
		switch name {
		case NameGo:
			p.HasGoMod = true
			if dr.SuggestedConfig.Version != "" {
				p.GoVersion = dr.SuggestedConfig.Version
			}

		case NameJavaScript:
			p.Ecosystems[NameNode] = true
			p.HasPackageJSON = true
			if dr.SuggestedConfig.Version != "" {
				p.NodeVersion = dr.SuggestedConfig.Version
			}
			if dr.SuggestedConfig.PackageManager != "" {
				p.PackageManager = dr.SuggestedConfig.PackageManager
			}

		case NamePython:
			p.HasPyProject = true
			if dr.SuggestedConfig.Version != "" {
				p.PythonVersion = dr.SuggestedConfig.Version
			}

		case NameRust:
			p.HasCargoToml = true

		case NameJava:
			if dr.SuggestedConfig.Extras != nil {
				if _, ok := dr.SuggestedConfig.Extras["build_tool"]; ok {
					switch dr.SuggestedConfig.Extras["build_tool"] {
					case "maven":
						p.HasPomXML = true
					case "gradle":
						p.HasBuildGradle = true
					case "both":
						p.HasPomXML = true
						p.HasBuildGradle = true
					}
				}
			}

		case NameDotnet:
			p.HasCsproj = true

		case NameContainer, "docker":
			p.HasDockerfile = true

		case NameTerraform:
			p.HasTerraform = true
		}
	}

	return p
}
