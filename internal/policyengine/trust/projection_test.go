package trust

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// toolsWhere returns the sorted crossToolEquivalence tool names whose server
// and first-party tool satisfy keep.
func toolsWhere(t *testing.T, keep func(server, firstParty string) bool) []string {
	t.Helper()
	var tools []string
	for _, name := range slices.Sorted(maps.Keys(crossToolEquivalence)) {
		server, ok := serverForTool(name)
		if !ok {
			t.Fatalf("equivalence tool %q has no server segment", name)
		}
		if keep(server, crossToolEquivalence[name].FirstPartyTool) {
			tools = append(tools, name)
		}
	}
	return tools
}

func tiers(m map[string]TrustTier) func(string) TrustTier {
	return func(server string) TrustTier {
		if tier, ok := m[server]; ok {
			return tier
		}
		return Tier3Fallback
	}
}

func TestGenerateDenyRuleProjections(t *testing.T) {
	t.Parallel()

	readAndEdit := []string{"Read(./.env)", "Edit(.git/**)", "Bash(rm -rf *)"}
	all := func(string, string) bool { return true }

	tests := []struct {
		name   string
		deny   []string
		tierOf func(string) TrustTier
		want   []string
	}{
		{
			name:   "every tier-3 server gets whole-tool denies",
			deny:   readAndEdit,
			tierOf: tiers(nil),
			want:   toolsWhere(t, all),
		},
		{
			name:   "nil tier function treats every server as tier 3",
			deny:   readAndEdit,
			tierOf: nil,
			want:   toolsWhere(t, all),
		},
		{
			name:   "tier 1 and tier 2 servers are left to the hook",
			deny:   readAndEdit,
			tierOf: tiers(map[string]TrustTier{"filesystem": Tier1Local, "github": Tier2Enterprise}),
			want:   nil,
		},
		{
			name:   "only the tier-3 server is projected",
			deny:   readAndEdit,
			tierOf: tiers(map[string]TrustTier{"github": Tier1Local}),
			want:   toolsWhere(t, func(server, _ string) bool { return server == "filesystem" }),
		},
		{
			name:   "read rules project only read tools",
			deny:   []string{"Read(~/.ssh/**)"},
			tierOf: tiers(nil),
			want:   toolsWhere(t, func(_, fp string) bool { return fp == "Read" }),
		},
		{
			name:   "write rules project edit tools",
			deny:   []string{"Write(/etc/**)"},
			tierOf: tiers(nil),
			want:   toolsWhere(t, func(_, fp string) bool { return fp == "Edit" }),
		},
		{
			name:   "no path rules projects nothing",
			deny:   []string{"Bash(curl *)", "Read", "Edit()", "WebFetch(domain:example.com)"},
			tierOf: tiers(nil),
			want:   nil,
		},
		{
			name:   "empty deny list projects nothing",
			tierOf: tiers(nil),
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GenerateDenyRuleProjections(tt.deny, tt.tierOf)
			if !slices.Equal(got, tt.want) {
				t.Errorf("GenerateDenyRuleProjections() =\n  %v\nwant\n  %v", got, tt.want)
			}
		})
	}
}

// TestGenerateDenyRuleProjectionsShape checks the regression the projection
// used to have: every entry is a bare MCP tool name (no path pattern, no
// first-party rule), each appears once, and the order is stable.
func TestGenerateDenyRuleProjectionsShape(t *testing.T) {
	t.Parallel()

	deny := []string{"Read(./.env)", "Read(./secrets/**)", "Edit(.git/**)", "Edit(flake.nix)"}
	got := GenerateDenyRuleProjections(deny, nil)
	if len(got) != len(crossToolEquivalence) {
		t.Fatalf("got %d entries, want one per path-bearing tool (%d): %v", len(got), len(crossToolEquivalence), got)
	}
	if !slices.IsSorted(got) {
		t.Errorf("entries are not sorted: %v", got)
	}
	if len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Errorf("entries contain duplicates: %v", got)
	}
	for _, entry := range got {
		if !strings.HasPrefix(entry, "mcp__") || strings.ContainsAny(entry, "()*") {
			t.Errorf("entry %q is not a whole-tool MCP deny", entry)
		}
	}
	for range 5 {
		if again := GenerateDenyRuleProjections(deny, nil); !slices.Equal(again, got) {
			t.Fatalf("projection is not deterministic: %v vs %v", again, got)
		}
	}
}

func TestServerForTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tool   string
		want   string
		wantOK bool
	}{
		{"mcp__filesystem__read_file", "filesystem", true},
		{"mcp__qsdev__qsdev_cc_config_render", "qsdev", true},
		{"mcp__noTool", "", false},
		{"mcp____read_file", "", false},
		{"Read", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			got, ok := serverForTool(tt.tool)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("serverForTool(%q) = %q, %v; want %q, %v", tt.tool, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
