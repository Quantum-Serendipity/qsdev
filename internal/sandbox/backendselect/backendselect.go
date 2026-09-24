// Package backendselect wires the concrete sandbox backends (bubblewrap,
// systemd-run) into the tier-ordered BackendRegistry and resolves the strongest
// backend that is actually available on the host.
//
// It lives in its own leaf package because the bwrap and cgroup backends import
// the sandbox package; a resolver that constructs them cannot live in package
// sandbox without creating an import cycle.
package backendselect

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/cgroup"
)

// ResolveBackend builds the candidate sandbox backends implied by the probed
// system capabilities, registers them in a tier-ordered registry, and returns
// the strongest one that reports itself Available together with the tier that
// backend actually enforces (the EFFECTIVE tier).
//
// The effective tier is the selected backend's own Tier(), NOT the probed
// DetermineTier(caps). This is deliberate: DetermineTier reports what the kernel
// could support, but a backend whose binary is missing (Available() != nil) is
// skipped, so the returned tier reflects what will actually run. On a host with
// no usable backend this returns the UnsandboxedBackend and TierUnsandboxed
// rather than overstating isolation the tool cannot deliver.
func ResolveBackend(caps sandbox.SystemCapabilities) (sandbox.SandboxBackend, sandbox.DegradationTier) {
	// Select() cannot error (it always returns at least the UnsandboxedBackend).
	backend, _ := candidates(caps).Select()
	return backend, backend.Tier()
}

// BackendAuto is the backend preference that selects the strongest available
// backend. An empty preference means the same.
const BackendAuto = "auto"

// ResolvePreferredBackend resolves the backend a sandbox policy asks for.
// BackendAuto (or "") behaves exactly like ResolveBackend. Any other value
// names a backend ("bubblewrap", "systemd-run", "unsandboxed"), which must be
// a candidate for the probed capabilities and report itself available; if it
// is not, an error is returned rather than silently running the hook under a
// different backend than the policy requires.
func ResolvePreferredBackend(caps sandbox.SystemCapabilities, preference string) (sandbox.SandboxBackend, sandbox.DegradationTier, error) {
	if preference == "" || preference == BackendAuto {
		backend, tier := ResolveBackend(caps)
		return backend, tier, nil
	}
	backend, err := candidates(caps).SelectByName(preference)
	if err != nil {
		return nil, sandbox.TierUnsandboxed, fmt.Errorf("resolving policy backend: %w", err)
	}
	return backend, backend.Tier(), nil
}

// candidates registers the sandbox backends implied by the probed capabilities
// in a tier-ordered registry.
func candidates(caps sandbox.SystemCapabilities) *sandbox.BackendRegistry {
	reg := sandbox.NewRegistry()

	// A bubblewrap backend is a candidate only when a bwrap binary was probed.
	// It is constructed with the tier the probed capabilities imply; the
	// registry still skips it at Select() time if the binary is not stat-able
	// OR if unprivileged user namespaces are unavailable (bwrap always requests
	// one, so it would otherwise register at the same demoted tier as the
	// systemd-run fallback yet fail every exec). When systemd-run was probed,
	// the backend also applies the policy's cgroup resource limits through it.
	if caps.BwrapPath != "" {
		reg.Register(bwrap.NewBubblewrapBackend(sandbox.DetermineTier(&caps), caps.BwrapPath, caps.HasUserNS,
			bwrap.WithSystemdRun(caps.SystemdRunPath)))
	}

	// A systemd-run backend is a candidate only when systemd-run was probed.
	if caps.SystemdRunPath != "" {
		reg.Register(cgroup.NewSystemdRunBackend(caps.SystemdRunPath))
	}

	// The unsandboxed backend is always available as the last-resort fallback.
	reg.Register(&sandbox.UnsandboxedBackend{})
	return reg
}
