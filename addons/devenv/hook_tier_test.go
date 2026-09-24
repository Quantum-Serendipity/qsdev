package devenv_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// securityHookIDs are the security hooks that must run at every security
// level: only non-security hooks are tiered.
var securityHookIDs = []string{
	"ripsecrets", "gitleaks", "semgrep", "opengrep", "nix-secrets-check",
	"lock-file-audit", "shellcheck", "govulncheck", "bandit", "tfsec",
}

func TestFilterHooksByTier(t *testing.T) {
	t.Parallel()
	hooks := []string{
		"check-merge-conflicts", "ripsecrets", "gofmt", "gitleaks", "semgrep",
		"shellcheck", "statix", "nix-secrets-check", "lock-file-audit",
		"govulncheck", "clippy", "uncatalogued-hook",
	}
	security := []string{
		"check-merge-conflicts", "ripsecrets", "gitleaks", "semgrep",
		"shellcheck", "nix-secrets-check", "lock-file-audit", "govulncheck",
		"uncatalogued-hook",
	}
	tests := []struct {
		name string
		tier string
		want []string
	}{
		{name: "empty tier keeps every hook", tier: "", want: hooks},
		{name: "baseline keeps security and uncatalogued hooks in order", tier: "baseline", want: security},
		{name: "enhanced adds language hooks", tier: "enhanced", want: hooks},
		{name: "strict keeps every hook", tier: "strict", want: hooks},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mustFilterHooksByTier(t, hooks, tt.tier)
			if !slices.Equal(got, tt.want) {
				t.Errorf("FilterHooksByTier(%q) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestFilterHooksByTier_SecurityHooksAtEveryLevel(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	for _, level := range cat.SecurityLevels() {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			got := mustFilterHooksByTier(t, securityHookIDs, level)
			if !slices.Equal(got, securityHookIDs) {
				t.Errorf("FilterHooksByTier(%q) dropped security hooks: got %v, want %v", level, got, securityHookIDs)
			}
		})
	}
}

func TestFilterHooksByTier_UnknownTier(t *testing.T) {
	t.Parallel()
	for _, tier := range []string{"full", "specialized", "Baseline"} {
		t.Run(tier, func(t *testing.T) {
			t.Parallel()
			got, err := devenv.FilterHooksByTier([]string{"ripsecrets"}, tier)
			if err == nil || !strings.Contains(err.Error(), "unknown hook tier") {
				t.Errorf("FilterHooksByTier(%q) = %v, %v; want an unknown hook tier error", tier, got, err)
			}
		})
	}
}

func TestFilterHooksByTier_EmptyHooks(t *testing.T) {
	t.Parallel()
	if got := mustFilterHooksByTier(t, nil, "baseline"); got != nil {
		t.Errorf("expected nil for empty hooks, got %v", got)
	}
}

// TestHookTiersNameRealHooks guards the catalog's hook tiers against drifting
// from the hooks qsdev generates: every tier entry must be a real hook ID (an
// always-on hook, a custom hook, an ecosystem module hook, or a catalog tool
// that ships its own hook), and every hook the generator can render must be
// tiered so that a security level's tier is complete.
func TestHookTiersNameRealHooks(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}

	generated := make(map[string]bool)
	for _, id := range cat.SecurityHooks() {
		generated[id] = true
	}
	for _, h := range cat.CustomHooks() {
		generated[h.ID] = true
	}
	// Extras that switch on each module's optional hooks.
	optional := ecosystem.ModuleConfig{Extras: map[string]string{
		"eslint": ".", "prettier": ".", "mypy": "true", "kotlin": "true",
	}}
	for _, mod := range ecosystem.DefaultRegistry().All() {
		for _, cfg := range []ecosystem.ModuleConfig{{}, optional} {
			for _, h := range mod.PreCommitHooks(cfg) {
				generated[h.ID] = true
			}
		}
	}

	tiered := make(map[string]bool)
	for tier, hooks := range cat.HookTiers() {
		for _, id := range hooks {
			tiered[id] = true
			if _, isTool := cat.Tool(id); !generated[id] && !isTool {
				t.Errorf("hook tier %q lists %q, which is not a hook qsdev generates or a catalog tool", tier, id)
			}
		}
	}
	for id := range generated {
		if !tiered[id] {
			t.Errorf("hook %q is not in any catalog hook tier", id)
		}
	}
}

// mustFilterHooksByTier calls FilterHooksByTier and fails the test on error.
func mustFilterHooksByTier(t *testing.T, hooks []string, tier string) []string {
	t.Helper()
	result, err := devenv.FilterHooksByTier(hooks, tier)
	if err != nil {
		t.Fatalf("FilterHooksByTier(%v, %q): %v", hooks, tier, err)
	}
	return result
}
