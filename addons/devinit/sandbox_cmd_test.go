package devinit

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestUnenforceableLayers verifies the status-honesty helper against both
// build shapes: a plain `go test` build carries neither ll-restrict nor the
// seccomp filter, while a Nix build (or a test run with the nix-sandbox job's
// -ldflags) carries both. Tiers that do not advertise a layer must never list
// it, whatever the build carries.
func TestUnenforceableLayers(t *testing.T) {
	t.Parallel()

	const (
		llBin   = "/nix/store/x-ll-restrict/bin/ll-restrict"
		bpfFile = "/nix/store/x-hook-seccomp-filter/hook-blocklist.bpf"
	)
	tests := []struct {
		name        string
		tier        sandbox.DegradationTier
		llBin       string
		seccompFile string
		want        []string
	}{
		{"full without tools", sandbox.TierFull, "", "", []string{"Landlock", "seccomp"}},
		{"full without seccomp filter", sandbox.TierFull, llBin, "", []string{"seccomp"}},
		{"full without ll-restrict", sandbox.TierFull, "", bpfFile, []string{"Landlock"}},
		{"full with both tools", sandbox.TierFull, llBin, bpfFile, nil},
		{"bwrap-without-landlock without tools", sandbox.TierBwrapWithoutLandlock, "", "", []string{"seccomp"}},
		{"bwrap-without-landlock with filter", sandbox.TierBwrapWithoutLandlock, "", bpfFile, nil},
		{"bwrap-without-seccomp without tools", sandbox.TierBwrapWithoutSeccomp, "", "", []string{"Landlock"}},
		{"bwrap-without-seccomp with ll-restrict", sandbox.TierBwrapWithoutSeccomp, llBin, "", nil},
		{"systemd-run advertises no LSM layer", sandbox.TierSystemdRun, "", "", nil},
		{"unsandboxed advertises no LSM layer", sandbox.TierUnsandboxed, "", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := layersWithoutTools(tt.tier, tt.llBin, tt.seccompFile)
			if !slices.Equal(got, tt.want) {
				t.Errorf("layersWithoutTools(%v, %q, %q) = %v, want %v",
					tt.tier, tt.llBin, tt.seccompFile, got, tt.want)
			}
		})
	}

	// The production wrapper reads this build's tool paths, so whatever
	// -ldflags the run carries it must agree with the seam.
	t.Run("wrapper uses this build's tools", func(t *testing.T) {
		t.Parallel()
		for _, tier := range []sandbox.DegradationTier{sandbox.TierFull, sandbox.TierBwrapWithoutLandlock, sandbox.TierBwrapWithoutSeccomp} {
			want := layersWithoutTools(tier, sandbox.LLRestrictBin(), sandbox.SeccompFilterFile())
			if got := unenforceableLayers(tier); !slices.Equal(got, want) {
				t.Errorf("unenforceableLayers(%v) = %v, want %v", tier, got, want)
			}
		}
	})
}
