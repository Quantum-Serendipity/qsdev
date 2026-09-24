package projectctx

import "testing"

func TestGenericToolsTierSpread(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	tiers := map[int]bool{}
	for _, reg := range pc.Tools() {
		tiers[reg.Tier] = true
		if reg.Category == "" {
			t.Errorf("tool %q has no category; rate-limiter binding requires one", reg.Name)
		}
	}
	if !tiers[int(TierCritical)] {
		t.Errorf("expected at least one critical-tier generic tool")
	}
}
