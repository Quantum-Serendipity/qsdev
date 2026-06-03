package policy

// PolicySpec mirrors the JSON output of .qsdev/policy.nix. It captures
// filesystem isolation, network egress rules, resource limits, and per-category
// sandbox profiles for hook execution.
type PolicySpec struct {
	Filesystem     FilesystemPolicy          `json:"filesystem"`
	Network        NetworkPolicySpec         `json:"network"`
	Resources      ResourceSpec              `json:"resources"`
	HookCategories map[string]CategoryPolicy `json:"hookCategories"`
	HookOverrides  map[string]HookOverride   `json:"hookOverrides,omitempty"`
	Backend        string                    `json:"backend"`
}

// FilesystemPolicy declares path-level access for sandboxed processes.
type FilesystemPolicy struct {
	AllowRead  []string `json:"allowRead"`
	AllowWrite []string `json:"allowWrite"`
	Deny       []string `json:"deny"`
}

// NetworkPolicySpec controls outbound network access.
type NetworkPolicySpec struct {
	Mode        string       `json:"mode"` // "deny", "allow", "filtered"
	EgressRules []EgressRule `json:"egressRules,omitempty"`
	DenyLAN     bool         `json:"denyLAN"`
}

// EgressRule allows a specific outbound connection to a host:port pair.
type EgressRule struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ResourceSpec bounds resource consumption of sandboxed processes.
type ResourceSpec struct {
	MemoryBytes     int64 `json:"memoryBytes"`
	MaxPIDs         int   `json:"maxPIDs"`
	CPUQuotaPercent int   `json:"cpuQuotaPercent"`
}

// CategoryPolicy defines the sandbox profile for a hook category.
type CategoryPolicy struct {
	WorktreeAccess string      `json:"worktreeAccess"` // "ro" or "rw"
	Network        string      `json:"network"`        // "deny", "allow", "filtered"
	ExtraMounts    []MountDecl `json:"extraMounts,omitempty"`
}

// MountDecl describes an additional bind mount to inject into the sandbox.
type MountDecl struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// HookOverride provides per-hook customisation that layers on top of the
// category profile.
type HookOverride struct {
	Category        string      `json:"category,omitempty"`
	ExtraMounts     []MountDecl `json:"extraMounts,omitempty"`
	NetworkOverride string      `json:"networkOverride,omitempty"`
}
