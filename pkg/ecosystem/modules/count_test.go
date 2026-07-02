package modules

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestEcosystemModuleCount pins the number of ecosystem modules that register
// themselves with the DefaultRegistry when this package is imported for its
// side effects. It exists to prevent silent drift between the code and the
// user-facing docs that quote this number.
//
// If this count changes, update the ecosystem count in ALL of these places:
//   - README.md ("Detects your stack across N ecosystems" and the
//     "Supported Ecosystems" table)
//   - CLAUDE.md ("The system covers N language/platform ecosystems")
//   - internal-docs/implementation-plan/plan.md (intro + Phase 1 summary)
//   - internal-docs/profile-comparison.md ("Ecosystem modules" row)
func TestEcosystemModuleCount(t *testing.T) {
	t.Parallel()

	const wantModules = 30

	got := len(ecosystem.DefaultRegistry().All())
	if got != wantModules {
		t.Errorf("registered ecosystem modules = %d, want %d; if this change is intentional, update the count in README.md, CLAUDE.md, internal-docs/implementation-plan/plan.md, and internal-docs/profile-comparison.md",
			got, wantModules)
	}
}

// TestEcosystemModuleTierBreakdown pins the per-tier module counts. The tier-1
// ("core") count in particular is quoted in docs, so guard it here.
//
// If any tier count changes, update internal-docs/profile-comparison.md
// ("Ecosystem modules" row notes: "N core (tier 1), M extended").
func TestEcosystemModuleTierBreakdown(t *testing.T) {
	t.Parallel()

	want := map[int]int{1: 8, 2: 10, 3: 7, 4: 5}

	got := map[int]int{}
	for _, m := range ecosystem.DefaultRegistry().All() {
		got[m.Tier()]++
	}

	for tier, wantN := range want {
		if got[tier] != wantN {
			t.Errorf("tier %d modules = %d, want %d; if intentional, update internal-docs/profile-comparison.md",
				tier, got[tier], wantN)
		}
	}
}
