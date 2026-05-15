package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProfileSummary describes a built-in infrastructure profile.
type ProfileSummary struct {
	Name        string
	Description string
}

// builtInProfiles contains the pre-defined infrastructure profiles.
var builtInProfiles = map[string]*types.QsdevConfig{
	"consulting-default": consultingDefaultProfile(),
	"startup-fast":       startupFastProfile(),
	"enterprise":         enterpriseProfile(),
}

// builtInDescriptions provides human-readable descriptions for built-in profiles.
var builtInDescriptions = map[string]string{
	"consulting-default": "Enhanced security with semgrep, gitleaks, secretspec; standard Claude Code; context7 + github MCP servers",
	"startup-fast":       "Baseline security with gitleaks only; standard Claude Code; minimal overhead",
	"enterprise":         "Strict security with all security tools; restricted Claude Code; audit logging enabled",
}

func consultingDefaultProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:          "enhanced",
			AgeGating:      &t,
			ScriptBlocking: &t,
			LockEnforce:    &t,
			VulnScanning:   &t,
		},
		Tools: types.ToolsConfig{
			Enabled: []string{"semgrep", "gitleaks", "secretspec"},
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "standard",
			MCPServers:      []string{"context7", "github"},
		},
	}
}

func startupFastProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:          "baseline",
			AgeGating:      &t,
			ScriptBlocking: &t,
			LockEnforce:    &t,
			VulnScanning:   &t,
		},
		Tools: types.ToolsConfig{
			Enabled: []string{"gitleaks"},
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "standard",
		},
	}
}

func enterpriseProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:          "strict",
			AgeGating:      &t,
			ScriptBlocking: &t,
			LockEnforce:    &t,
			VulnScanning:   &t,
		},
		Tools: types.ToolsConfig{
			Enabled: []string{"semgrep", "gitleaks", "secretspec", "license-compliance"},
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "restricted",
		},
	}
}

// GetBuiltInProfile returns the QsdevConfig for a named built-in profile.
// Returns an error listing available profiles if the name is not found.
func GetBuiltInProfile(name string) (*types.QsdevConfig, error) {
	cfg, ok := builtInProfiles[name]
	if !ok {
		available := make([]string, 0, len(builtInProfiles))
		for k := range builtInProfiles {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown profile %q; available profiles: %s", name, strings.Join(available, ", "))
	}
	return cfg, nil
}

// ListBuiltInProfiles returns summaries of all built-in profiles, sorted by name.
func ListBuiltInProfiles() []ProfileSummary {
	var result []ProfileSummary
	for name := range builtInProfiles {
		result = append(result, ProfileSummary{
			Name:        name,
			Description: builtInDescriptions[name],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// OrgDefaults returns the organization-wide default QsdevConfig.
// This is a convenience re-export of DefaultQsdevConfig for clarity in
// the resolution chain.
func OrgDefaults() *types.QsdevConfig {
	return DefaultQsdevConfig()
}
