package trust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScoreServer(t *testing.T) {
	t.Parallel()

	engine := NewMcpTrustEngine(filepath.Join(t.TempDir(), "nonexistent.yaml"))

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

	engine := NewMcpTrustEngine(filepath.Join(t.TempDir(), "nonexistent.yaml"))

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

	engine := NewMcpTrustEngine(configPath)

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
	engine := NewMcpTrustEngine("/nonexistent/path/trust.yaml")
	if engine == nil {
		t.Fatal("engine should not be nil")
		return
	}
	if engine.config == nil {
		t.Fatal("config should not be nil")
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
