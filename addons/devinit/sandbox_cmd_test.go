package devinit

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestUnenforceableLayers verifies the status-honesty helper. SeccompFilterFile
// is populated only via -ldflags at Nix build time, so under `go test` it is
// always empty; therefore any tier that advertises seccomp must list it as
// unenforceable. Tiers that do not advertise a layer must never list it.
func TestUnenforceableLayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tier        sandbox.DegradationTier
		mustContain []string
		mustOmit    []string
	}{
		{
			name:        "full advertises both LSM layers; seccomp unenforceable in test build",
			tier:        sandbox.TierFull,
			mustContain: []string{"seccomp"},
			mustOmit:    nil,
		},
		{
			name:        "bwrap-without-landlock advertises seccomp only",
			tier:        sandbox.TierBwrapWithoutLandlock,
			mustContain: []string{"seccomp"},
			mustOmit:    []string{"Landlock"},
		},
		{
			name:        "bwrap-without-seccomp never lists seccomp",
			tier:        sandbox.TierBwrapWithoutSeccomp,
			mustContain: nil,
			mustOmit:    []string{"seccomp"},
		},
		{
			name:        "systemd-run advertises no LSM layer",
			tier:        sandbox.TierSystemdRun,
			mustContain: nil,
			mustOmit:    []string{"Landlock", "seccomp"},
		},
		{
			name:        "unsandboxed advertises no LSM layer",
			tier:        sandbox.TierUnsandboxed,
			mustContain: nil,
			mustOmit:    []string{"Landlock", "seccomp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unenforceableLayers(tt.tier)
			for _, want := range tt.mustContain {
				if !slices.Contains(got, want) {
					t.Errorf("expected %q in unenforceable layers, got %v", want, got)
				}
			}
			for _, unwant := range tt.mustOmit {
				if slices.Contains(got, unwant) {
					t.Errorf("did not expect %q in unenforceable layers, got %v", unwant, got)
				}
			}
		})
	}
}
