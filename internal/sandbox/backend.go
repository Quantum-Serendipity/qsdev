package sandbox

import "context"

// SandboxBackend abstracts a sandbox runtime. Backends are registered with
// BackendRegistry and selected by the degradation engine based on available
// kernel capabilities.
type SandboxBackend interface {
	// Name returns a human-readable identifier for the backend.
	Name() string

	// Available probes whether this backend can operate on the current system.
	// Returns nil when available, or an error explaining why not.
	Available() error

	// Tier returns the degradation tier this backend provides.
	Tier() DegradationTier

	// RunHook creates a sandbox, executes the hook command, collects results,
	// and tears down. This is the single-call path for hook execution.
	RunHook(ctx context.Context, cfg *SandboxConfig) (*SandboxResult, error)
}

// UnsandboxedBackend is the fallback backend that runs commands without any
// isolation. It is always available and provides TierUnsandboxed.
type UnsandboxedBackend struct{}

func (u *UnsandboxedBackend) Name() string          { return "unsandboxed" }
func (u *UnsandboxedBackend) Available() error      { return nil }
func (u *UnsandboxedBackend) Tier() DegradationTier { return TierUnsandboxed }

// RunHook executes the hook command without sandbox isolation.
func (u *UnsandboxedBackend) RunHook(ctx context.Context, cfg *SandboxConfig) (*SandboxResult, error) {
	return runUnsandboxed(ctx, cfg)
}
