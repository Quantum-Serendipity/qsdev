package sandbox

import (
	"io"
	"time"
)

// DegradationTier represents the level of sandbox isolation available on the
// current system. The degradation engine selects the strongest tier supported.
type DegradationTier int

// Degradation tiers, strongest first. Every bwrap tier additionally applies
// cgroup resource limits by running the sandbox inside a transient systemd
// --user scope when one is usable; the bwrap backend warns when it cannot.
const (
	TierFull                 DegradationTier = iota // bwrap + Landlock + seccomp (+ cgroup limits)
	TierBwrapWithoutLandlock                        // bwrap + seccomp (+ cgroup limits), no Landlock
	TierBwrapWithoutSeccomp                         // bwrap + Landlock (+ cgroup limits), no seccomp
	TierBwrapOnly                                   // bwrap namespaces (+ cgroup limits), no Landlock, no seccomp
	TierSystemdRun                                  // systemd-run scope resource limits only (no namespaces)
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
	case TierBwrapOnly:
		return "bwrap-only"
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

// Worktree access modes accepted by SandboxConfig.WorktreeAccess and the
// policy's per-category worktreeAccess field.
const (
	WorktreeAccessReadOnly  = "ro"
	WorktreeAccessReadWrite = "rw"
)

// SandboxConfig is the configuration for a single sandboxed hook execution.
type SandboxConfig struct {
	ProjectDir   string
	HookCommand  []string
	HookCategory HookCategory
	// WorktreeAccess overrides the category's default worktree mount mode
	// ("ro" or "rw"). Empty means the HookCategory default; see
	// WorktreeReadOnly.
	WorktreeAccess string
	Environment    map[string]string
	NixStorePaths  []string
	Mounts         []MountSpec
	// Deny lists absolute paths that must be neither readable nor writable
	// inside the sandbox (the policy's filesystem.deny). Backends mask every
	// entry, whether or not it is on the built-in credential deny list, and
	// never grant one to an inner restriction layer. A Mount always exposes its
	// Source; a Deny entry always hides its path.
	Deny      []string
	Resources ResourceLimits
	Network   NetworkPolicy
	// Backend names the sandbox backend the policy requires ("bubblewrap",
	// "systemd-run", "unsandboxed"). Empty or "auto" selects the strongest
	// available one.
	Backend string

	// ExecOpts carries the hook's stdio. Stdin is forwarded to the hook (Claude
	// Code delivers the tool-call payload there, so a wrapper such as `sandbox
	// exec` must forward it); a nil output writer means that stream is captured
	// into the SandboxResult instead of streamed.
	ExecOpts
}

// WorktreeReadOnly reports whether the project directory is mounted read-only
// for this execution. An explicit WorktreeAccess wins, and any value other than
// "rw" counts as read-only so a malformed setting fails closed. Without one,
// the HookCategory default applies.
func (c *SandboxConfig) WorktreeReadOnly() bool {
	if c.WorktreeAccess == "" {
		return c.HookCategory.WorktreeReadOnly()
	}
	return c.WorktreeAccess != WorktreeAccessReadWrite
}

// MountSpec describes a single bind mount in the sandbox.
type MountSpec struct {
	Source   string
	Target   string
	ReadOnly bool
}

// Network modes accepted in NetworkPolicy.Mode.
const (
	NetworkModeDeny     = "deny"
	NetworkModeAllow    = "allow"
	NetworkModeFiltered = "filtered"
)

// NetworkPolicy controls network access within the sandbox.
type NetworkPolicy struct {
	Mode string // "deny", "allow", "filtered"; empty means the category default
	// EgressRules and DenyLAN describe the egress filter intended for
	// "filtered" mode. No backend enforces them yet, so "filtered" currently
	// shares the host network like "allow" (see UnenforcedNetworkControls).
	EgressRules []EgressRule
	DenyLAN     bool
}

// DefaultNetworkMode returns the network mode a category gets when the policy
// sets none: "filtered" for categories that need the network, "deny" otherwise.
func (c HookCategory) DefaultNetworkMode() string {
	if c.NetworkAllowed() {
		return NetworkModeFiltered
	}
	return NetworkModeDeny
}

// EffectiveNetworkMode resolves the network mode the sandbox must enforce: the
// configured Network.Mode, or the category default when it is empty. Every
// isolation layer derives its network decision from this one value, so an
// explicit "deny" is honoured even for categories that default to network.
func (c *SandboxConfig) EffectiveNetworkMode() string {
	if c.Network.Mode != "" {
		return c.Network.Mode
	}
	return c.HookCategory.DefaultNetworkMode()
}

// NetworkIsolated reports whether the hook must be cut off from the host
// network. Only "allow" and "filtered" share it; "deny" and any unrecognised
// mode fail closed.
func (c *SandboxConfig) NetworkIsolated() bool {
	switch c.EffectiveNetworkMode() {
	case NetworkModeAllow, NetworkModeFiltered:
		return false
	default:
		return true
	}
}

// FilteredNetworkNotice states, for status and doctor output, that the
// "filtered" network mode has no egress filter yet. The default policy gives
// it to the network-linter and test-runner categories, so it applies to every
// install whatever its tier.
const FilteredNetworkNotice = `Network mode "filtered" (the network-linter and test-runner default) is not ` +
	`enforced: those hooks share the host network, and egressRules/denyLAN are not applied.`

// UnenforcedNetworkControls describes configured network controls that no
// backend can enforce yet. "filtered" has no egress filter implementation, so
// the hook shares the host network and any egress allowlist or LAN denial is
// not applied. The result is empty when every network control is enforced.
func (c *SandboxConfig) UnenforcedNetworkControls() []string {
	if c.EffectiveNetworkMode() != NetworkModeFiltered {
		return nil
	}
	controls := []string{`network mode "filtered" is not enforced: the hook shares the host network`}
	if len(c.Network.EgressRules) > 0 {
		controls = append(controls, "network egress allowlist is not enforced")
	}
	if c.Network.DenyLAN {
		controls = append(controls, "network denyLAN is not enforced")
	}
	return controls
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

// Any reports whether at least one limit is set.
func (r ResourceLimits) Any() bool {
	return r.MemoryBytes > 0 || r.MaxPIDs > 0 || r.CPUQuotaPercent > 0
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

// ExecOpts carries the stdio streams of a sandboxed command.
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
