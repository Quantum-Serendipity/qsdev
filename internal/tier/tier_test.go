package tier

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

func TestParseTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  Tier
	}{
		{"supply-chain-only", SupplyChainOnly},
		{"standard", Standard},
		{"full", Full},
	}
	for _, tc := range tests {
		got, err := ParseTier(tc.input)
		if err != nil {
			t.Errorf("ParseTier(%q): unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("ParseTier(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseTier_Invalid(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "bogus", "STANDARD", "enterprise"} {
		_, err := ParseTier(input)
		if err == nil {
			t.Errorf("ParseTier(%q): expected error, got nil", input)
		}
	}
}

func TestTierString_Roundtrip(t *testing.T) {
	t.Parallel()
	for _, name := range catalog.MustDefault().TierOrder() {
		tier, err := ParseTier(name)
		if err != nil {
			t.Fatal(err)
		}
		if tier.String() != name {
			t.Errorf("Tier(%d).String() = %q, want %q", tier, tier.String(), name)
		}
	}
}

func TestSuperset(t *testing.T) {
	t.Parallel()
	if Standard <= SupplyChainOnly {
		t.Error("Standard should be > SupplyChainOnly")
	}
	if Full <= Standard {
		t.Error("Full should be > Standard")
	}
	if Full <= SupplyChainOnly {
		t.Error("Full should be > SupplyChainOnly")
	}
}

func TestDefaultPermissionPreset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier Tier
		want string
	}{
		{SupplyChainOnly, "supply-chain-only"},
		{Standard, "standard"},
		{Full, "standard"},
	}
	for _, tc := range tests {
		got := tc.tier.DefaultPermissionPreset()
		if got != tc.want {
			t.Errorf("%v.DefaultPermissionPreset() = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestNextTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		current  string
		wantNext string
		wantOK   bool
	}{
		{"supply-chain-only", "standard", true},
		{"standard", "full", true},
		{"full", "", false},
		{"bogus", "", false},
	}
	for _, tc := range tests {
		next, ok := NextTier(tc.current)
		if ok != tc.wantOK || next != tc.wantNext {
			t.Errorf("NextTier(%q) = (%q, %v), want (%q, %v)",
				tc.current, next, ok, tc.wantNext, tc.wantOK)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tierStr   string
		permLevel string
		mcp       []string
		want      Tier
	}{
		{"supply-chain-only", "standard", nil, SupplyChainOnly},
		{"full", "standard", nil, Full},
		{"", "standard", nil, Standard},
		{"", "supply-chain-only", nil, SupplyChainOnly},
		{"", "standard", []string{"github"}, Full},
		{"bogus", "standard", nil, Standard},
		{"", "", nil, Standard},
	}
	for _, tc := range tests {
		got := Resolve(tc.tierStr, tc.permLevel, tc.mcp)
		if got != tc.want {
			t.Errorf("Resolve(%q, %q, %v) = %v, want %v",
				tc.tierStr, tc.permLevel, tc.mcp, got, tc.want)
		}
	}
}

func TestInfer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		permLevel  string
		mcpServers []string
		want       Tier
	}{
		{"supply-chain-only", nil, SupplyChainOnly},
		{"standard", nil, Standard},
		{"standard", []string{"github"}, Full},
		{"permissive", []string{"context7", "github"}, Full},
		{"minimal", nil, Standard},
		{"", nil, Standard},
	}
	for _, tc := range tests {
		got := Infer(tc.permLevel, tc.mcpServers)
		if got != tc.want {
			t.Errorf("Infer(%q, %v) = %v, want %v",
				tc.permLevel, tc.mcpServers, got, tc.want)
		}
	}
}

func TestAllTiers_Order(t *testing.T) {
	t.Parallel()
	tiers := AllTiers()
	if len(tiers) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(tiers))
	}
	for i := 1; i < len(tiers); i++ {
		if tiers[i].Level <= tiers[i-1].Level {
			t.Errorf("tiers not in ascending order: %v after %v", tiers[i].Level, tiers[i-1].Level)
		}
	}
	for _, ti := range tiers {
		if ti.Description == "" {
			t.Errorf("tier %q has empty description", ti.Name)
		}
	}
}
