package trust

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestScoreServer(t *testing.T) {
	t.Parallel()

	engine := mustTrustEngine(t, filepath.Join(t.TempDir(), "nonexistent.yaml"))

	tests := []struct {
		name        string
		info        McpServerInfo
		wantTier    TrustTier
		minScore    int
		maxScore    int
		wantCeiling string
	}{
		{
			name: "man-pages scores high tier 1",
			info: McpServerInfo{
				Name:                  "man-pages",
				Command:               "qsdev",
				IsLocalBinary:         true,
				OfflineCapable:        true,
				ControlledUpdates:     true,
				VerifiedInstallSource: true,
				PinnedVersion:         true,
			},
			wantTier: Tier1Local,
			minScore: 75,
			maxScore: 100,
		},
		{
			name: "context7 scores low tier 3",
			info: McpServerInfo{
				Name:                    "context7",
				Command:                 "npx",
				ServesCommunityCContent: true,
			},
			wantTier: Tier3Fallback,
			maxScore: 44,
		},
		{
			name: "semble scores high tier 1",
			info: McpServerInfo{
				Name:                  "semble",
				Command:               "qsdev",
				IsLocalBinary:         true,
				ControlledUpdates:     true,
				VerifiedInstallSource: true,
				PinnedVersion:         true,
				HasUserAttestation:    true,
			},
			wantTier: Tier1Local,
			minScore: 75,
			maxScore: 100,
		},
		{
			name: "worst characteristics tier 3",
			info: McpServerInfo{
				Name:                    "evil-server",
				ServesCommunityCContent: true,
				HasKnownVulnerabilities: true,
			},
			wantTier:    Tier3Fallback,
			maxScore:    33,
			wantCeiling: "known-vulnerability",
		},
		{
			name: "known vulnerability caps at 33",
			info: McpServerInfo{
				Name:                    "vuln-server",
				IsLocalBinary:           true,
				OfflineCapable:          true,
				ControlledUpdates:       true,
				VerifiedInstallSource:   true,
				PinnedVersion:           true,
				HasKnownVulnerabilities: true,
			},
			wantTier:    Tier3Fallback,
			maxScore:    33,
			wantCeiling: "known-vulnerability",
		},
		{
			name: "community content caps at 45",
			info: McpServerInfo{
				Name:                    "community-server",
				IsLocalBinary:           true,
				OfflineCapable:          true,
				ControlledUpdates:       true,
				VerifiedInstallSource:   true,
				PinnedVersion:           true,
				HasUserAttestation:      true,
				ServesCommunityCContent: true,
			},
			wantTier:    Tier2Enterprise,
			maxScore:    45,
			wantCeiling: "community-content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := engine.ScoreServer(&tt.info)

			if result.Tier != tt.wantTier {
				t.Errorf("tier = %v, want %v (score=%d)", result.Tier, tt.wantTier, result.Score)
			}

			if result.Score < tt.minScore {
				t.Errorf("score = %d, want >= %d", result.Score, tt.minScore)
			}

			if result.Score > tt.maxScore {
				t.Errorf("score = %d, want <= %d", result.Score, tt.maxScore)
			}

			if tt.wantCeiling != "" && result.CeilingApplied != tt.wantCeiling {
				t.Errorf("ceiling = %q, want %q", result.CeilingApplied, tt.wantCeiling)
			}
		})
	}
}

func TestScoreAll(t *testing.T) {
	t.Parallel()

	engine := mustTrustEngine(t, filepath.Join(t.TempDir(), "nonexistent.yaml"))

	servers := []McpServerInfo{
		{Name: "server-a", IsLocalBinary: true, OfflineCapable: true},
		{Name: "server-b", ServesCommunityCContent: true},
	}

	results := engine.ScoreAll(servers)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	if _, ok := results["server-a"]; !ok {
		t.Error("missing result for server-a")
	}
	if _, ok := results["server-b"]; !ok {
		t.Error("missing result for server-b")
	}
}

func TestManualOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "trust.yaml")

	cfg := &TrustConfig{
		Servers: map[string]TrustServerEntry{
			"overridden": {
				Tier:           Tier1Local,
				Score:          100,
				ManualOverride: true,
			},
		},
	}

	if err := SaveTrustConfig(configPath, cfg); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	engine := mustTrustEngine(t, configPath)

	result := engine.ScoreServer(&McpServerInfo{
		Name:                    "overridden",
		ServesCommunityCContent: true,
	})

	if result.Tier != Tier1Local {
		t.Errorf("tier = %v, want %v (manual override should apply)", result.Tier, Tier1Local)
	}
}

func TestLoadSaveTrustConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "trust.yaml")

	cfg := &TrustConfig{
		Servers: map[string]TrustServerEntry{
			"test-server": {
				Tier:  Tier2Enterprise,
				Score: 55,
			},
		},
	}

	if err := SaveTrustConfig(path, cfg); err != nil {
		t.Fatalf("saving: %v", err)
	}

	loaded, err := LoadTrustConfig(path)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	entry, ok := loaded.Servers["test-server"]
	if !ok {
		t.Fatal("missing test-server entry")
	}

	if entry.Tier != Tier2Enterprise {
		t.Errorf("tier = %v, want %v", entry.Tier, Tier2Enterprise)
	}
	if entry.Score != 55 {
		t.Errorf("score = %d, want 55", entry.Score)
	}
}

func TestLoadTrustConfigMissing(t *testing.T) {
	t.Parallel()

	_, err := LoadTrustConfig(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestKnownServerInfo(t *testing.T) {
	t.Parallel()

	info, ok := KnownServerInfo("man-pages")
	if !ok {
		t.Fatal("man-pages should be known")
	}
	if !info.IsLocalBinary {
		t.Error("man-pages should be local binary")
	}

	_, ok = KnownServerInfo("unknown-server")
	if ok {
		t.Error("unknown-server should not be known")
	}
}

func TestTrustTierString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier TrustTier
		want string
	}{
		{Tier1Local, "tier-1-local"},
		{Tier2Enterprise, "tier-2-enterprise"},
		{Tier3Fallback, "tier-3-fallback"},
		{TrustTier(99), "TrustTier(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.tier.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMcpTrustEngineWithBadPath(t *testing.T) {
	t.Parallel()

	// Should not panic, should create engine with empty config
	engine := mustTrustEngine(t, "/nonexistent/path/trust.yaml")
	if engine == nil {
		t.Fatal("engine should not be nil")
		return
	}
	if engine.config == nil {
		t.Fatal("config should not be nil")
	}
}

func mustTrustEngine(t *testing.T, path string) *McpTrustEngine {
	t.Helper()
	engine, err := NewMcpTrustEngine(path)
	if err != nil {
		t.Fatalf("NewMcpTrustEngine(%q): %v", path, err)
	}
	return engine
}

// TestNewMcpTrustEngine_ConfigErrors guards F199: a trust config that exists
// but fails to parse (a YAML error or a misspelled key) used to be swallowed,
// silently dropping every manual override. It must now be reported, and the
// engine must fall back to the strictest tier rather than computed tiers.
func TestNewMcpTrustEngine_ConfigErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "yaml syntax error", content: "servers:\n  local: [unterminated\n", wantErr: "parsing trust config"},
		{name: "misspelled override key", content: "servers:\n  local:\n    tier: 3\n    manual_overide: true\n", wantErr: "manual_overide"},
		{name: "unknown top-level key", content: "server:\n  local:\n    tier: 3\n", wantErr: "server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "trust.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("writing config: %v", err)
			}

			engine, err := NewMcpTrustEngine(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewMcpTrustEngine error = %v, want it to mention %q", err, tt.wantErr)
			}
			if engine == nil {
				t.Fatal("engine must stay usable after a config error")
			}

			local, _ := KnownServerInfo("man-pages")
			if got := engine.ScoreServer(&local).Tier; got != Tier3Fallback {
				t.Errorf("tier after config error = %v, want %v", got, Tier3Fallback)
			}
		})
	}
}

func TestNewMcpTrustEngine_EmptyAndMissingConfig(t *testing.T) {
	t.Parallel()

	empty := filepath.Join(t.TempDir(), "trust.yaml")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing.yaml"), empty} {
		engine := mustTrustEngine(t, path)
		local, _ := KnownServerInfo("man-pages")
		if got := engine.ScoreServer(&local).Tier; got != Tier1Local {
			t.Errorf("path %q: tier = %v, want %v (no overrides, computed tier)", path, got, Tier1Local)
		}
	}
}

// TestScoreServerDeterministicCategories guards F197 for trust scoring.
func TestScoreServerDeterministicCategories(t *testing.T) {
	t.Parallel()

	engine := mustTrustEngine(t, "")
	info := McpServerInfo{Name: "s", IsLocalBinary: true, PinnedVersion: true}

	names := func() []string {
		var out []string
		for _, c := range engine.ScoreServer(&info).Categories {
			out = append(out, c.Name)
		}
		return out
	}

	want := names()
	if len(want) < 2 || !slices.IsSorted(want) {
		t.Fatalf("categories %v are not in a fixed (sorted) order", want)
	}
	for range 50 {
		if got := names(); !slices.Equal(got, want) {
			t.Fatalf("category order changed between runs: %v vs %v", got, want)
		}
	}
}

// TestResolveServerInfo guards F189: a known server's trust signals apply only
// when the configured definition runs the known command, so a server cannot
// claim a known server's tier by reusing its name.
func TestResolveServerInfo(t *testing.T) {
	t.Parallel()

	known, _ := KnownServerInfo("man-pages")
	engine := mustTrustEngine(t, "")

	tests := []struct {
		name       string
		server     string
		configured *McpServerInfo
		wantTier   TrustTier
	}{
		{
			name:       "known server with matching command",
			server:     "man-pages",
			configured: &McpServerInfo{Command: known.Command, Args: []string{"mcp", "man-pages"}},
			wantTier:   Tier1Local,
		},
		{
			name:       "known name with a different command",
			server:     "man-pages",
			configured: &McpServerInfo{Command: "npx", Args: []string{"-y", "evil-man-pages"}},
			wantTier:   Tier3Fallback,
		},
		{name: "known name not configured", server: "man-pages", wantTier: Tier3Fallback},
		{
			name:       "unknown configured server",
			server:     "custom",
			configured: &McpServerInfo{Command: "npx"},
			wantTier:   Tier3Fallback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := ResolveServerInfo(tt.server, tt.configured)
			if info.Name != tt.server {
				t.Errorf("Name = %q, want %q", info.Name, tt.server)
			}
			if tt.configured != nil && !slices.Equal(info.Args, tt.configured.Args) {
				t.Errorf("Args = %v, want configured %v", info.Args, tt.configured.Args)
			}
			if got := engine.ScoreServer(&info).Tier; got != tt.wantTier {
				t.Errorf("tier = %v, want %v", got, tt.wantTier)
			}
		})
	}
}

func TestSaveTrustConfigCreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "new-trust.yaml")

	cfg := &TrustConfig{
		Servers: map[string]TrustServerEntry{},
	}

	if err := SaveTrustConfig(path, cfg); err != nil {
		t.Fatalf("saving: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}
