package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
	"gopkg.in/yaml.v3"
)

// LocalConfig represents the .gdev.local.yaml file, which contains
// per-developer overrides. It omits project-level fields (Version,
// GdevVersion, Profile, Client, Infrastructure) that only belong in
// the shared .gdev.yaml.
type LocalConfig struct {
	Languages     []types.LanguageConfig `yaml:"languages,omitempty"`
	Services      []types.ServiceConfig  `yaml:"services,omitempty"`
	Security      types.SecurityConfig   `yaml:"security,omitempty"`
	Tools         types.ToolsConfig      `yaml:"tools,omitempty"`
	ClaudeCode    types.ClaudeCodeConfig `yaml:"claude_code,omitempty"`
	ExtraPackages []string               `yaml:"extra_packages,omitempty"`
}

// ParseLocalConfig reads and parses a .gdev.local.yaml file.
// Returns (nil, nil) if the file does not exist — this is not an error
// since the local config file is optional. Only parse failures produce errors.
func ParseLocalConfig(path string) (*LocalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading local config %s: %w", path, err)
	}

	var local LocalConfig
	if err := yaml.Unmarshal(data, &local); err != nil {
		return nil, fmt.Errorf("parsing local config %s: %w", path, err)
	}

	return &local, nil
}

// localToGdevConfig converts a LocalConfig to a GdevConfig for use in the
// merge chain. Fields that exist only in GdevConfig (Version, GdevVersion,
// Profile, Client, Infrastructure) are left at zero values.
func localToGdevConfig(local *LocalConfig) *types.GdevConfig {
	if local == nil {
		return nil
	}

	return &types.GdevConfig{
		Languages:  local.Languages,
		Services:   local.Services,
		Security:   local.Security,
		Tools:      local.Tools,
		ClaudeCode: local.ClaudeCode,
	}
}

// GenerateLocalTemplate writes a .gdev.local.yaml template file with
// commented-out examples. It only creates the file if it doesn't already
// exist (idempotent). The template content is context-sensitive: it includes
// language version overrides if the resolved config contains languages.
func GenerateLocalTemplate(projectRoot string, resolved *types.GdevConfig) error {
	path := filepath.Join(projectRoot, ".gdev.local.yaml")

	// Don't overwrite an existing file.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	var b strings.Builder
	b.WriteString("# .gdev.local.yaml — Local developer overrides (gitignored)\n")
	b.WriteString("# These settings override .gdev.yaml but cannot lower security settings.\n")
	b.WriteString("#\n")
	b.WriteString("# extra_packages:\n")
	b.WriteString("#   - neovim\n")
	b.WriteString("#   - lazygit\n")
	b.WriteString("#\n")

	// Include language version overrides if resolved config has languages.
	if resolved != nil && len(resolved.Languages) > 0 {
		b.WriteString("# languages:\n")
		for _, lang := range resolved.Languages {
			version := lang.Version
			if version == "" {
				version = "latest"
			}
			fmt.Fprintf(&b, "#   - name: %s\n", lang.Name)
			fmt.Fprintf(&b, "#     version: %q\n", version)
		}
		b.WriteString("#\n")
	}

	// Include Claude Code section if enabled.
	if resolved != nil && resolved.ClaudeCode.Enabled != nil && *resolved.ClaudeCode.Enabled {
		b.WriteString("# claude_code:\n")
		b.WriteString("#   permission_level: permissive\n")
		b.WriteString("#\n")
	}

	b.WriteString("# tools:\n")
	b.WriteString("#   enabled:\n")
	b.WriteString("#     - changelog\n")

	return fileutil.WriteFileAtomic(path, []byte(b.String()), 0644)
}

// EnsureGitignoreEntry reads the .gitignore file in projectRoot, checks for
// an exact line match of entry, and appends it with a section comment if
// missing. Creates the .gitignore file if it doesn't exist. The write is atomic.
func EnsureGitignoreEntry(projectRoot, entry string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	var lines []string
	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	if err == nil {
		content := string(data)
		lines = strings.Split(content, "\n")

		// Check for exact line match.
		for _, line := range lines {
			if strings.TrimSpace(line) == entry {
				return nil // Already present.
			}
		}
	}

	// Append the entry with a section comment.
	var b strings.Builder
	if len(data) > 0 {
		b.Write(data)
		// Ensure trailing newline before our section.
		if !strings.HasSuffix(string(data), "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("# gdev local configuration\n")
	b.WriteString(entry)
	b.WriteString("\n")

	return fileutil.WriteFileAtomic(gitignorePath, []byte(b.String()), 0644)
}
