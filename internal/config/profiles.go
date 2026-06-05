package config

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProfileSummary describes a built-in infrastructure profile.
type ProfileSummary struct {
	Name        string
	Description string
}

// GetBuiltInProfile returns the QsdevConfig for a named built-in profile.
// Legacy names (startup-fast, consulting-default) are resolved via aliases
// with a deprecation warning. Returns an error listing available profiles
// if the name is not found.
func GetBuiltInProfile(name string) (*types.QsdevConfig, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}
	aliases := cat.ProfileAliases()

	if alias, ok := aliases[name]; ok {
		log.Printf("warning: profile %q is deprecated, use %q instead", name, alias)
		name = alias
	}

	def, ok := cat.Profile(name)
	if !ok {
		profiles := cat.Profiles()
		available := make([]string, 0, len(profiles))
		for k := range profiles {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown profile %q; available profiles: %s", name, strings.Join(available, ", "))
	}

	return profileDefToConfig(def), nil
}

// ListBuiltInProfiles returns summaries of all built-in profiles, sorted by name.
func ListBuiltInProfiles() []ProfileSummary {
	profiles := catalog.MustDefault().Profiles()
	var result []ProfileSummary
	for name, def := range profiles {
		result = append(result, ProfileSummary{
			Name:        name,
			Description: def.Description,
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
	aliases := catalog.MustDefault().ProfileAliases()
	if alias, ok := aliases[name]; ok {
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

func profileDefToConfig(def catalog.ProfileDef) *types.QsdevConfig {
	cfg := &types.QsdevConfig{
		Tier: def.Tier,
	}

	if def.Security != nil {
		cfg.Security.Level = def.Security.Level
		cfg.Security.AgeGating = def.Security.AgeGating
		cfg.Security.ScriptBlocking = def.Security.ScriptBlocking
		cfg.Security.LockEnforcement = def.Security.LockEnforcement
		cfg.Security.VulnScanning = def.Security.VulnScanning
	}

	if def.Tools != nil {
		cfg.Tools.Enabled = def.Tools.Enabled
	}

	if def.ClaudeCode != nil {
		cfg.ClaudeCode.Enabled = def.ClaudeCode.Enabled
		cfg.ClaudeCode.PermissionLevel = def.ClaudeCode.PermissionLevel
		cfg.ClaudeCode.MCPServers = def.ClaudeCode.MCPServers
	}

	return cfg
}
