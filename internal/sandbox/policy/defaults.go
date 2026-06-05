package policy

import (
	"log/slog"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// DefaultPolicy returns a PolicySpec with sensible security defaults suitable
// for most projects. Credential directories are denied, network is blocked,
// and the five standard hook categories are pre-configured.
func DefaultPolicy() *PolicySpec {
	return &PolicySpec{
		Filesystem: FilesystemPolicy{
			Deny: denylist.AllDenyPaths(),
		},
		Network: NetworkPolicySpec{
			Mode:    "deny",
			DenyLAN: true,
		},
		Resources: ResourceSpec{
			MemoryBytes:     2 * 1024 * 1024 * 1024, // 2 GB
			MaxPIDs:         4096,
			CPUQuotaPercent: 200, // 2 cores
		},
		HookCategories: map[string]CategoryPolicy{
			"linter":         {WorktreeAccess: "ro", Network: "deny"},
			"formatter":      {WorktreeAccess: "rw", Network: "deny"},
			"network-linter": {WorktreeAccess: "ro", Network: "filtered"},
			"generator":      {WorktreeAccess: "rw", Network: "deny"},
			"test-runner":    {WorktreeAccess: "rw", Network: "filtered"},
		},
		Backend: "auto",
	}
}

// ToSandboxConfig merges the policy spec with a category profile and optional
// per-hook overrides into a concrete sandbox.SandboxConfig. The merge proceeds
// in three layers:
//  1. Base policy (filesystem deny list, network mode, resource limits)
//  2. Category profile (worktree access, network mode, extra mounts)
//  3. Per-hook override (extra mounts, network override, category reassignment)
func ToSandboxConfig(spec *PolicySpec, category sandbox.HookCategory, hookName string) *sandbox.SandboxConfig {
	cfg := &sandbox.SandboxConfig{
		HookCategory: category,
		Resources: sandbox.ResourceLimits{
			MemoryBytes:     spec.Resources.MemoryBytes,
			MaxPIDs:         spec.Resources.MaxPIDs,
			CPUQuotaPercent: spec.Resources.CPUQuotaPercent,
		},
		Network: sandbox.NetworkPolicy{
			Mode:    spec.Network.Mode,
			DenyLAN: spec.Network.DenyLAN,
		},
	}

	// Copy base egress rules.
	for _, r := range spec.Network.EgressRules {
		cfg.Network.EgressRules = append(cfg.Network.EgressRules, sandbox.EgressRule{
			Host: r.Host,
			Port: r.Port,
		})
	}

	// Copy filesystem deny paths as read-only deny mounts.
	for _, p := range spec.Filesystem.Deny {
		cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{
			Source:   p,
			Target:   p,
			ReadOnly: true,
		})
	}

	// Determine the effective category name, which a hook override may replace.
	effectiveCategory := category.String()

	// Check for per-hook override first to resolve category reassignment.
	override, hasOverride := spec.HookOverrides[hookName]
	if hasOverride && override.Category != "" {
		effectiveCategory = override.Category
		cfg.HookCategory = sandbox.ParseHookCategory(effectiveCategory)
	}

	// Apply the category profile.
	if catPolicy, ok := spec.HookCategories[effectiveCategory]; ok {
		cfg.Network.Mode = catPolicy.Network

		for _, m := range catPolicy.ExtraMounts {
			if err := ValidateMountDecl(m); err != nil {
				slog.Warn("skipping invalid category mount", "category", effectiveCategory, "error", err)
				continue
			}
			cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{
				Source:   m.Source,
				Target:   m.Target,
				ReadOnly: m.ReadOnly,
			})
		}
	}

	// Apply per-hook overrides on top of category.
	if hasOverride {
		if override.NetworkOverride != "" {
			cfg.Network.Mode = override.NetworkOverride
		}

		for _, m := range override.ExtraMounts {
			if err := ValidateMountDecl(m); err != nil {
				slog.Warn("skipping invalid override mount", "hook", hookName, "error", err)
				continue
			}
			cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{
				Source:   m.Source,
				Target:   m.Target,
				ReadOnly: m.ReadOnly,
			})
		}
	}

	return cfg
}
