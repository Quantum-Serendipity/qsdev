package backendselect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// writeStubBinary creates a stat-able file in dir and returns its path. The
// resolver's backends only os.Stat their binary path in Available(), so a plain
// file is sufficient to make a backend selectable in these tests.
func writeStubBinary(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing stub binary %s: %v", path, err)
	}
	return path
}

// withUserBus makes the systemd-run backend's user-session check pass, so the
// tests do not depend on whether the host running them has a systemd user bus.
// It uses t.Setenv, so callers must not be parallel.
func withUserBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/qsdev-test-bus")
}

// withoutUserBus simulates a host without a systemd user session (SSH without
// lingering, containers, CI). It uses t.Setenv, so callers must not be parallel.
func withoutUserBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
}

func TestResolveBackend(t *testing.T) {
	withUserBus(t)

	tmp := t.TempDir()
	realBwrap := writeStubBinary(t, tmp, "bwrap")
	realSystemd := writeStubBinary(t, tmp, "systemd-run")

	fullCaps := func(bwrapPath, systemdPath string) sandbox.SystemCapabilities {
		return sandbox.SystemCapabilities{
			HasBwrap:       bwrapPath != "",
			BwrapPath:      bwrapPath,
			HasUserNS:      true,
			LandlockABI:    1,
			HasSeccomp:     true,
			HasCgroupV2:    true,
			HasSystemdRun:  systemdPath != "",
			SystemdRunPath: systemdPath,
		}
	}

	tests := []struct {
		name          string
		caps          sandbox.SystemCapabilities
		wantName      string
		wantTier      sandbox.DegradationTier
		wantSandboxed bool
	}{
		{
			name:          "full caps but bwrap binary absent falls back to unsandboxed",
			caps:          fullCaps(filepath.Join(tmp, "missing-bwrap"), ""),
			wantName:      "unsandboxed",
			wantTier:      sandbox.TierUnsandboxed,
			wantSandboxed: false,
		},
		{
			name:          "stat-able bwrap path selects the bubblewrap backend",
			caps:          fullCaps(realBwrap, ""),
			wantName:      "bubblewrap",
			wantTier:      sandbox.TierFull,
			wantSandboxed: true,
		},
		{
			name:          "only systemd-run available selects the systemd-run backend",
			caps:          sandbox.SystemCapabilities{HasSystemdRun: true, SystemdRunPath: realSystemd},
			wantName:      "systemd-run",
			wantTier:      sandbox.TierSystemdRun,
			wantSandboxed: true,
		},
		{
			name:          "no capabilities returns unsandboxed",
			caps:          sandbox.SystemCapabilities{},
			wantName:      "unsandboxed",
			wantTier:      sandbox.TierUnsandboxed,
			wantSandboxed: false,
		},
		{
			// Regression: a build without ll-restrict or a seccomp filter
			// probes no LSM layer. bwrap used to be registered at the
			// systemd-run (or unsandboxed) tier and mis-reported as such.
			name: "bwrap without LSM layers reports bwrap-only, not systemd-run",
			caps: sandbox.SystemCapabilities{
				HasBwrap: true, BwrapPath: realBwrap, HasUserNS: true,
				HasSystemdRun: true, SystemdRunPath: realSystemd,
			},
			wantName:      "bubblewrap",
			wantTier:      sandbox.TierBwrapOnly,
			wantSandboxed: true,
		},
		{
			name:          "bwrap without LSM layers or systemd-run is not reported unsandboxed",
			caps:          sandbox.SystemCapabilities{HasBwrap: true, BwrapPath: realBwrap, HasUserNS: true},
			wantName:      "bubblewrap",
			wantTier:      sandbox.TierBwrapOnly,
			wantSandboxed: true,
		},
		{
			name:          "bubblewrap preferred over systemd-run when both available",
			caps:          fullCaps(realBwrap, realSystemd),
			wantName:      "bubblewrap",
			wantTier:      sandbox.TierFull,
			wantSandboxed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, tier := ResolveBackend(tt.caps)

			if backend.Name() != tt.wantName {
				t.Errorf("backend.Name() = %q, want %q", backend.Name(), tt.wantName)
			}
			if tier != tt.wantTier {
				t.Errorf("tier = %v, want %v", tier, tt.wantTier)
			}
			// The returned tier must be the selected backend's own tier (the
			// effective tier), never a separately-computed probed tier.
			if backend.Tier() != tier {
				t.Errorf("backend.Tier() = %v, but resolver returned tier %v", backend.Tier(), tier)
			}
			if got := tier != sandbox.TierUnsandboxed; got != tt.wantSandboxed {
				t.Errorf("isSandboxed = %v, want %v", got, tt.wantSandboxed)
			}
		})
	}
}

// TestResolveBackend_ClosesP0 is a focused regression for BL-P0-1: when a usable
// isolating backend is available, the resolver MUST NOT fall back to the
// unsandboxed backend. The pre-fix runSandboxed hard-coded UnsandboxedBackend on
// every branch and could never satisfy this.
func TestResolveBackend_ClosesP0(t *testing.T) {
	t.Parallel()

	realBwrap := writeStubBinary(t, t.TempDir(), "bwrap")

	caps := sandbox.SystemCapabilities{
		HasBwrap:    true,
		BwrapPath:   realBwrap,
		HasUserNS:   true,
		LandlockABI: 1,
		HasSeccomp:  true,
	}

	backend, tier := ResolveBackend(caps)
	if backend.Name() == "unsandboxed" || tier == sandbox.TierUnsandboxed {
		t.Fatalf("resolver returned unsandboxed backend despite an available bubblewrap backend: name=%q tier=%v",
			backend.Name(), tier)
	}
}

// TestResolveBackend_BwrapWithoutUserNSPrefersSystemdRun is a regression for
// #S3: when a bwrap binary is present but unprivileged user namespaces are
// disabled, DetermineTier demotes bwrap to TierSystemdRun — the SAME tier as the
// systemd-run fallback. Because bwrap always requests --unshare-user it would
// fail every exec, yet a non-stable tier sort could still pick it over the
// working systemd-run backend. The fix makes the bwrap backend report itself
// unavailable (no user namespaces) and makes same-tier ordering stable, so the
// resolver deterministically selects the functional systemd-run backend.
func TestResolveBackend_BwrapWithoutUserNSPrefersSystemdRun(t *testing.T) {
	withUserBus(t)

	tmp := t.TempDir()
	realBwrap := writeStubBinary(t, tmp, "bwrap")
	realSystemd := writeStubBinary(t, tmp, "systemd-run")

	caps := sandbox.SystemCapabilities{
		HasBwrap:       true,
		BwrapPath:      realBwrap,
		HasUserNS:      false, // user namespaces disabled -> bwrap cannot run
		HasSystemdRun:  true,
		SystemdRunPath: realSystemd,
	}

	// Run repeatedly: a non-stable sort of equal-tier backends is a source of
	// flakiness, so a single pass could pass by luck. The selection must be the
	// working systemd-run backend on every iteration.
	for i := 0; i < 64; i++ {
		backend, tier := ResolveBackend(caps)
		if backend.Name() != "systemd-run" {
			t.Fatalf("iteration %d: selected backend = %q, want %q (bwrap with no user namespaces must not be chosen)",
				i, backend.Name(), "systemd-run")
		}
		if backend.Name() == "bubblewrap" {
			t.Fatalf("iteration %d: broken bubblewrap backend selected despite disabled user namespaces", i)
		}
		if tier != sandbox.TierSystemdRun {
			t.Fatalf("iteration %d: tier = %v, want %v", i, tier, sandbox.TierSystemdRun)
		}
	}
}

// TestResolveBackend_SystemdRunWithoutUserSession pins that a systemd-run binary
// alone does not make the systemd-run backend selectable: without a user bus
// every `systemd-run --user` fails, so selecting it would turn every hook into
// a spurious exit 1 instead of running it.
func TestResolveBackend_SystemdRunWithoutUserSession(t *testing.T) {
	withoutUserBus(t)

	caps := sandbox.SystemCapabilities{
		HasSystemdRun:  true,
		SystemdRunPath: writeStubBinary(t, t.TempDir(), "systemd-run"),
	}

	backend, tier := ResolveBackend(caps)
	if backend.Name() != "unsandboxed" || tier != sandbox.TierUnsandboxed {
		t.Errorf("ResolveBackend() = (%q, %v), want (unsandboxed, %v)", backend.Name(), tier, sandbox.TierUnsandboxed)
	}
}

func TestResolvePreferredBackend(t *testing.T) {
	withUserBus(t)

	tmp := t.TempDir()
	caps := sandbox.SystemCapabilities{
		HasBwrap:       true,
		BwrapPath:      writeStubBinary(t, tmp, "bwrap"),
		HasUserNS:      true,
		LandlockABI:    1,
		HasSeccomp:     true,
		HasSystemdRun:  true,
		SystemdRunPath: writeStubBinary(t, tmp, "systemd-run"),
	}
	noBwrap := sandbox.SystemCapabilities{
		HasSystemdRun:  true,
		SystemdRunPath: caps.SystemdRunPath,
	}

	tests := []struct {
		name       string
		caps       sandbox.SystemCapabilities
		preference string
		wantName   string
		wantTier   sandbox.DegradationTier
		wantErr    bool
	}{
		{name: "empty means auto", caps: caps, preference: "", wantName: "bubblewrap", wantTier: sandbox.TierFull},
		{name: "auto selects strongest", caps: caps, preference: BackendAuto, wantName: "bubblewrap", wantTier: sandbox.TierFull},
		{name: "explicit weaker backend honoured", caps: caps, preference: "systemd-run", wantName: "systemd-run", wantTier: sandbox.TierSystemdRun},
		{name: "explicit unsandboxed honoured", caps: caps, preference: "unsandboxed", wantName: "unsandboxed", wantTier: sandbox.TierUnsandboxed},
		{name: "required backend missing fails closed", caps: noBwrap, preference: "bubblewrap", wantErr: true},
		{name: "unknown backend fails closed", caps: caps, preference: "firejail", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, tier, err := ResolvePreferredBackend(tt.caps, tt.preference)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolvePreferredBackend(%q) = %q, want error", tt.preference, backend.Name())
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePreferredBackend(%q) error: %v", tt.preference, err)
			}
			if backend.Name() != tt.wantName || tier != tt.wantTier {
				t.Errorf("ResolvePreferredBackend(%q) = (%q, %v), want (%q, %v)",
					tt.preference, backend.Name(), tier, tt.wantName, tt.wantTier)
			}
		})
	}
}
