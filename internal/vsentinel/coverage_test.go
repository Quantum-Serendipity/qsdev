package vsentinel

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCoverage_MatchesCheckers proves Coverage reports "diffed" exactly for a
// manifest DetectDrift can version-diff — a specificCheckers ecosystem's
// manifest paired with its primary lockfile (or the default) — and
// "presence-only" for every other catalog manifest/lockfile combination, and
// "uncovered" for a manifest outside the catalog.
func TestCoverage_MatchesCheckers(t *testing.T) {
	t.Parallel()

	for _, pair := range driftPairs() {
		locks := append([]string{""}, ecosystem.LockFilesByEcosystem[pair.eco]...)
		for _, lock := range locks {
			want := VerificationPresenceOnly
			if pair.checker != nil && (lock == "" || lock == pair.primaryLock) {
				want = VerificationDiffed
			}
			t.Run(pair.eco+"/"+pair.manifest+"/"+lock, func(t *testing.T) {
				t.Parallel()
				if got := Coverage(pair.manifest, lock); got != want {
					t.Errorf("Coverage(%q, %q) = %q, want %q", pair.manifest, lock, got, want)
				}
			})
		}
	}

	tests := []struct {
		name, manifest, lock string
		want                 CoverageLevel
	}{
		{"manifest in a subdirectory path", "services/api/go.mod", "services/api/go.sum", VerificationDiffed},
		{"concrete csproj has no checker", "App.csproj", "", VerificationPresenceOnly},
		{"non-catalog manifest is uncovered", "mix.exs", "mix.lock", VerificationUncovered},
		{"non-catalog glob is uncovered", "*.tf", "", VerificationUncovered},
		{"empty manifest", "", "", VerificationUncovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Coverage(tt.manifest, tt.lock); got != tt.want {
				t.Errorf("Coverage(%q, %q) = %q, want %q", tt.manifest, tt.lock, got, tt.want)
			}
		})
	}
}
