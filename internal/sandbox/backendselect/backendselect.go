// Package backendselect wires the concrete sandbox backends (bubblewrap,
// systemd-run) into the tier-ordered BackendRegistry and resolves the strongest
// backend that is actually available on the host.
//
// It lives in its own leaf package because the bwrap and cgroup backends import
// the sandbox package; a resolver that constructs them cannot live in package
// sandbox without creating an import cycle.
package backendselect

import (
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
	reg := sandbox.NewRegistry()

	// A bubblewrap backend is a candidate only when a bwrap binary was probed.
	// It is constructed with the tier the probed capabilities imply; the
	// registry still skips it if the binary is not stat-able at Select() time.
	if caps.BwrapPath != "" {
		reg.Register(bwrap.NewBubblewrapBackend(sandbox.DetermineTier(&caps), caps.BwrapPath))
	}

	// A systemd-run backend is a candidate only when systemd-run was probed.
	if caps.SystemdRunPath != "" {
		reg.Register(cgroup.NewSystemdRunBackend(caps.SystemdRunPath))
	}

	// The unsandboxed backend is always available as the last-resort fallback.
	reg.Register(&sandbox.UnsandboxedBackend{})

	// Select() cannot error (it always returns at least the UnsandboxedBackend).
	backend, _ := reg.Select()
	return backend, backend.Tier()
}
