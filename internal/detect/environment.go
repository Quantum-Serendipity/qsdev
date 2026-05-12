package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

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

// detectEnvironment scans the project root for configuration files,
// tooling state, and git metadata.
func detectEnvironment(root string) EnvironmentState {
	s := EnvironmentState{
		HasDevenvNix:      fileExists(root, "devenv.nix"),
		HasDevenvYaml:     fileExists(root, "devenv.yaml"),
		HasClaudeDir:      dirExists(root, ".claude"),
		HasClaudeMd:       fileExists(root, "CLAUDE.md"),
		HasClaudeSettings: fileExists(root, ".claude", "settings.json"),
		HasEnvrc:          fileExists(root, ".envrc"),
		HasMcpJson:        fileExists(root, ".mcp.json"),
		IsGitRepo:         dirExists(root, ".git"),
	}

	if s.IsGitRepo {
		s.HasGitHooks = hasExecutableHooks(filepath.Join(root, ".git", "hooks"))
		s.RemoteURL = parseOriginURL(filepath.Join(root, ".git", "config"))
	}

	return s
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
