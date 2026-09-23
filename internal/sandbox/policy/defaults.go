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
	limits := sandbox.DefaultResourceLimits()
	return &PolicySpec{
		Filesystem: FilesystemPolicy{
			Deny: denylist.AllDenyPaths(),
		},
		Network: NetworkPolicySpec{
			Mode:    "deny",
			DenyLAN: true,
		},
		Resources: ResourceSpec{
			MemoryBytes:     limits.MemoryBytes,
			MaxPIDs:         limits.MaxPIDs,
			CPUQuotaPercent: limits.CPUQuotaPercent,
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
//  1. Base policy (filesystem deny list and allowRead/allowWrite binds, network
//     mode, resource limits, backend)
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
		Backend: spec.Backend,
	}

	// Copy base egress rules.
	for _, r := range spec.Network.EgressRules {
		cfg.Network.EgressRules = append(cfg.Network.EgressRules, sandbox.EgressRule{
			Host: r.Host,
			Port: r.Port,
		})
	}

	// Carry filesystem deny paths as explicit deny intent. They must never be
	// encoded as mounts: a mount exposes its source, a deny entry hides it.
	cfg.Deny = append(cfg.Deny, spec.Filesystem.Deny...)

	// Explicitly allowed paths are bound at the same location inside the
	// sandbox: allowRead read-only, allowWrite read-write. They pass the same
	// validation as extra mounts, so they can never re-expose a deny path.
	appendMounts(cfg, pathMounts(spec.Filesystem.AllowRead, true), "filesystem.allowRead")
	appendMounts(cfg, pathMounts(spec.Filesystem.AllowWrite, false), "filesystem.allowWrite")

	// Determine the effective category name, which a hook override may replace.
	effectiveCategory := category.String()

	// Check for per-hook override first to resolve category reassignment.
	override, hasOverride := spec.HookOverrides[hookName]
	if hasOverride && override.Category != "" {
		effectiveCategory = override.Category
		cfg.HookCategory = sandbox.ParseHookCategory(effectiveCategory)
	}

	// Apply the category profile.
	// An empty network setting at any layer inherits the layer below; when
	// every layer is empty, sandbox.SandboxConfig.EffectiveNetworkMode falls
	// back to the category default.
	if catPolicy, ok := spec.HookCategories[effectiveCategory]; ok {
		if catPolicy.Network != "" {
			cfg.Network.Mode = catPolicy.Network
		}
		cfg.WorktreeAccess = worktreeAccess(catPolicy.WorktreeAccess, effectiveCategory)
		appendMounts(cfg, catPolicy.ExtraMounts, "hookCategories."+effectiveCategory)
	}

	// Apply per-hook overrides on top of category.
	if hasOverride {
		if override.NetworkOverride != "" {
			cfg.Network.Mode = override.NetworkOverride
		}

		appendMounts(cfg, override.ExtraMounts, "hookOverrides."+hookName)
	}

	return cfg
}

// worktreeAccess validates a category's worktreeAccess value. An empty value
// leaves the HookCategory default in force. An unrecognised value is reported
// and mapped to read-only, so a typo can never widen access to the worktree.
func worktreeAccess(access, category string) string {
	switch access {
	case "", sandbox.WorktreeAccessReadOnly, sandbox.WorktreeAccessReadWrite:
		return access
	default:
		slog.Warn("invalid worktreeAccess in sandbox policy; using read-only",
			"category", category, "worktreeAccess", access)
		return sandbox.WorktreeAccessReadOnly
	}
}

// pathMounts turns a list of paths into same-path bind declarations.
func pathMounts(paths []string, readOnly bool) []MountDecl {
	decls := make([]MountDecl, 0, len(paths))
	for _, p := range paths {
		decls = append(decls, MountDecl{Source: p, Target: p, ReadOnly: readOnly})
	}
	return decls
}

// appendMounts validates each declaration and appends the valid ones to
// cfg.Mounts. Invalid ones are skipped with a warning naming the policy field
// they came from.
func appendMounts(cfg *sandbox.SandboxConfig, decls []MountDecl, field string) {
	for _, m := range decls {
		if err := ValidateMountDecl(m); err != nil {
			slog.Warn("skipping invalid sandbox policy mount", "field", field, "error", err)
			continue
		}
		cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}
}
