package sandbox

import (
	"io"
	"time"
)

// DegradationTier represents the level of sandbox isolation available on the
// current system. The degradation engine selects the strongest tier supported.
type DegradationTier int

const (
	TierFull                 DegradationTier = iota // bwrap + Landlock + seccomp + cgroups
	TierBwrapWithoutLandlock                        // bwrap + seccomp + cgroups (no Landlock)
	TierBwrapWithoutSeccomp                         // bwrap + Landlock + cgroups (no seccomp)
	TierSystemdRun                                  // systemd-run scope only (no namespaces)
	TierUnsandboxed                                 // no isolation
)

func (t DegradationTier) String() string {
	switch t {
	case TierFull:
		return "full"
	case TierBwrapWithoutLandlock:
		return "bwrap-without-landlock"
	case TierBwrapWithoutSeccomp:
		return "bwrap-without-seccomp"
	case TierSystemdRun:
		return "systemd-run"
	case TierUnsandboxed:
		return "unsandboxed"
	default:
		return "unknown"
	}
}

// HookCategory determines the sandbox permission profile applied to a hook.
type HookCategory int

const (
	CategoryLinter        HookCategory = iota // ro worktree, no network
	CategoryFormatter                         // rw worktree, no network
	CategoryNetworkLinter                     // ro worktree, filtered network
	CategoryGenerator                         // rw worktree, no network
	CategoryTestRunner                        // rw worktree, filtered network
)

func (c HookCategory) String() string {
	switch c {
	case CategoryLinter:
		return "linter"
	case CategoryFormatter:
		return "formatter"
	case CategoryNetworkLinter:
		return "network-linter"
	case CategoryGenerator:
		return "generator"
	case CategoryTestRunner:
		return "test-runner"
	default:
		return "unknown"
	}
}

// ParseHookCategory converts a string to a HookCategory.
// Returns CategoryLinter for unrecognized values.
func ParseHookCategory(s string) HookCategory {
	switch s {
	case "linter":
		return CategoryLinter
	case "formatter":
		return CategoryFormatter
	case "network-linter":
		return CategoryNetworkLinter
	case "generator":
		return CategoryGenerator
	case "test-runner":
		return CategoryTestRunner
	default:
		return CategoryLinter
	}
}

// WorktreeReadOnly reports whether this category's worktree mount is read-only.
func (c HookCategory) WorktreeReadOnly() bool {
	return c == CategoryLinter || c == CategoryNetworkLinter
}

// NetworkAllowed reports whether this category permits network access.
func (c HookCategory) NetworkAllowed() bool {
	return c == CategoryNetworkLinter || c == CategoryTestRunner
}

// SandboxConfig is the configuration for a single sandboxed hook execution.
type SandboxConfig struct {
	ProjectDir        string
	HookCommand       []string
	HookCategory      HookCategory
	Environment       map[string]string
	NixStorePaths     []string
	Mounts            []MountSpec
	Resources         ResourceLimits
	Network           NetworkPolicy
	SeccompFilterPath string
	PolicyPath        string
}

// MountSpec describes a single bind mount in the sandbox.
type MountSpec struct {
	Source   string
	Target   string
	ReadOnly bool
}

// NetworkPolicy controls network access within the sandbox.
type NetworkPolicy struct {
	Mode        string // "deny", "allow", "filtered"
	EgressRules []EgressRule
	DenyLAN     bool
}

// EgressRule allows a specific outbound connection.
type EgressRule struct {
	Host string
	Port int
}

// ResourceLimits bounds resource consumption of a sandboxed process.
type ResourceLimits struct {
	MemoryBytes     int64
	MaxPIDs         int
	CPUQuotaPercent int
}

// DefaultResourceLimits returns the default resource limits for hook execution.
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MemoryBytes:     2 * 1024 * 1024 * 1024, // 2 GB
		MaxPIDs:         4096,
		CPUQuotaPercent: 200, // 2 cores
	}
}

// SandboxResult captures the outcome of sandboxed execution.
type SandboxResult struct {
	ExitCode        int
	Stdout          []byte
	Stderr          []byte
	Duration        time.Duration
	SandboxOverhead time.Duration
	Tier            DegradationTier
}

// ExecOpts controls how a command is executed inside the sandbox.
type ExecOpts struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// SystemCapabilities describes what sandbox features the host supports.
type SystemCapabilities struct {
	HasBwrap       bool
	BwrapPath      string
	HasUserNS      bool
	LandlockABI    int // 0 = unsupported
	HasSeccomp     bool
	HasCgroupV2    bool
	HasCgroupDeleg bool
	HasSystemdRun  bool
	SystemdRunPath string
	KernelVersion  string
}
