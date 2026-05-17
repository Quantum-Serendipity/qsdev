package paths

import "github.com/Quantum-Serendipity/qsdev/pkg/branding"

// QsdevConfigPath returns the project config filename (e.g. ".qsdev.yaml")
// derived from branding rather than a hardcoded constant.
func QsdevConfigPath() string { return branding.Get().ConfigFile }

const (
	ClaudeSettings     = ".claude/settings.json"
	ClaudeDir          = ".claude"
	ClaudeMD           = "CLAUDE.md"
	McpJSON            = ".mcp.json"
	DevenvNix          = "devenv.nix"
	DevenvYAML         = "devenv.yaml"
	Envrc              = ".envrc"
	Npmrc              = ".npmrc"
	GitLabCI           = ".gitlab-ci.yml"
	GitHubWorkflowsDir = ".github/workflows"
)
