package config

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProfileSummary describes a built-in infrastructure profile.
type ProfileSummary struct {
	Name        string
	Description string
}

// builtInProfiles contains the tier-based infrastructure profiles.
var builtInProfiles = map[string]*types.QsdevConfig{
	"supply-chain-only": supplyChainOnlyProfile(),
	"standard":          standardProfile(),
	"full":              fullProfile(),
}

// builtInDescriptions provides human-readable descriptions for built-in profiles.
var builtInDescriptions = map[string]string{
	"supply-chain-only": "Package supply chain security + devenv sandbox; no Claude Code restrictions",
	"standard":          "Supply chain deny rules + Claude Code governance + CLAUDE.md + gitleaks",
	"full":              "Full tooling: MCP servers, agent tools, consulting workflows, AlwaysOn tools",
}

// profileAliases maps legacy profile names to their tier-based replacements.
var profileAliases = map[string]string{
	"startup-fast":       "standard",
	"consulting-default": "full",
}

func supplyChainOnlyProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Tier: "supply-chain-only",
		Security: types.SecurityConfig{
			Level:          "baseline",
			AgeGating:      &t,
			ScriptBlocking: &t,
			LockEnforce:    &t,
			VulnScanning:   &t,
		},
		Tools: types.ToolsConfig{
			Enabled: []string{},
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "supply-chain-only",
		},
	}
}

func standardProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Tier: "standard",
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

func fullProfile() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Tier: "full",
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

// GetBuiltInProfile returns the QsdevConfig for a named built-in profile.
// Legacy names (startup-fast, consulting-default) are resolved via aliases
// with a deprecation warning. Returns an error listing available profiles
// if the name is not found.
func GetBuiltInProfile(name string) (*types.QsdevConfig, error) {
	if alias, ok := profileAliases[name]; ok {
		log.Printf("warning: profile %q is deprecated, use %q instead", name, alias)
		name = alias
	}
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

// ResolveProfileAlias returns the canonical profile name, resolving any legacy
// alias. The second return value is true if the name was an alias.
func ResolveProfileAlias(name string) (string, bool) {
	if alias, ok := profileAliases[name]; ok {
		return alias, true
	}
	return name, false
}

// OrgDefaults returns the organization-wide default QsdevConfig.
// This is a convenience re-export of DefaultQsdevConfig for clarity in
// the resolution chain.
func OrgDefaults() *types.QsdevConfig {
	return DefaultQsdevConfig()
}
