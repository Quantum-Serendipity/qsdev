package haskell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

var stackConfig = ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": "stack"}}

func TestToolchainWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		stack    string // stack.yaml content; "" for none
		config   ecosystem.ModuleConfig
		shellGHC string
		probeErr error
		want     []string // substrings of the single warning; nil for none
	}{
		{
			name: "snapshot GHC differs from PATH", stack: "resolver: lts-22.44\n", config: stackConfig, shellGHC: "9.10.3",
			want: []string{`snapshot "lts-22.44" needs GHC 9.6.7`, "ghc on PATH is 9.10.3", "No compiler found", "haskell.compiler.ghc967"},
		},
		{name: "snapshot GHC matches PATH", stack: "resolver: lts-22.44\n", config: stackConfig, shellGHC: "9.6.7"},
		{name: "cabal project", stack: "resolver: lts-22.44\n", config: ecosystem.ModuleConfig{}, shellGHC: "9.10.3"},
		{name: "unknown snapshot GHC", stack: "resolver: nightly-2025-01-01\n", config: stackConfig, shellGHC: "9.10.3"},
		{name: "no stack.yaml", config: stackConfig, shellGHC: "9.10.3"},
		{name: "no ghc on PATH", stack: "resolver: lts-22.44\n", config: stackConfig, probeErr: errors.New("not found")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.stack != "" {
				dir = writeStackYAML(t, tt.stack)
			}
			m := &Module{ghcVersion: func(context.Context) (string, error) { return tt.shellGHC, tt.probeErr }}
			got := m.ToolchainWarnings(context.Background(), dir, tt.config)
			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("ToolchainWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("ToolchainWarnings() = %q, want one warning", got)
			}
			for _, sub := range tt.want {
				if !strings.Contains(got[0], sub) {
					t.Errorf("warning %q does not contain %q", got[0], sub)
				}
			}
		})
	}
}

func TestSetupWarnings(t *testing.T) {
	t.Parallel()
	withVersion := func(v string) ecosystem.ModuleConfig {
		return ecosystem.ModuleConfig{Version: v, Extras: stackConfig.Extras}
	}
	tests := []struct {
		name   string
		stack  string
		config ecosystem.ModuleConfig
		want   []string
	}{
		{name: "detected version matches", stack: "resolver: lts-22.44\n", config: withVersion("9.6.7")},
		{
			name: "stale version after a snapshot bump", stack: "resolver: lts-23.0\n", config: withVersion("9.6.7"),
			want: []string{"configured GHC 9.6.7", "GHC 9.8.4", `snapshot "lts-23.0"`, "qsdev init --update"},
		},
		{
			name: "series version is not exact", stack: "resolver: lts-22.44\n", config: withVersion("9.6"),
			want: []string{"configured GHC 9.6 does not match GHC 9.6.7"},
		},
		{name: "configured version, unknown snapshot", stack: "resolver: nightly-2025-01-01\n", config: withVersion("9.8.4")},
		{
			name: "unknown snapshot", stack: "resolver: nightly-2025-01-01\n", config: stackConfig,
			want: []string{`snapshot "nightly-2025-01-01"`, "Stack install GHC itself"},
		},
		{
			name: "no snapshot", stack: "packages: [.]\n", config: stackConfig,
			want: []string{"no snapshot or resolver", "Stack install GHC itself"},
		},
		{
			name: "version unset in an older .qsdev.yaml", stack: "resolver: lts-22.44\n", config: stackConfig,
			want: []string{"version in .qsdev.yaml is not set", "GHC 9.6.7", "haskell.compiler.ghc967", "qsdev init --update"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := (&Module{}).SetupWarnings(writeStackYAML(t, tt.stack), tt.config)
			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("SetupWarnings() = %q, want one warning", got)
			}
			for _, sub := range tt.want {
				if !strings.Contains(got[0], sub) {
					t.Errorf("warning %q does not contain %q", got[0], sub)
				}
			}
		})
	}
}

// TestSetupWarnings_Cabal checks a Cabal project is warned when its
// dependencies are not pinned: no cabal.project.freeze (the CI install step
// then fails), or a freeze file without an index-state line, so `cabal
// update` lets the Hackage index, and with it revisions, drift (U10-11).
func TestSetupWarnings_Cabal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		config ecosystem.ModuleConfig
		want   []string
	}{
		{
			name: "freeze file missing", files: map[string]string{"x.cabal": ""},
			want: []string{"cabal.project.freeze", "cabal freeze"},
		},
		{
			name: "freeze file without index-state", files: map[string]string{"x.cabal": "", "cabal.project.freeze": "constraints: any.base ==4.19.1.0\n"},
			want: []string{"index-state"},
		},
		{
			name: "freeze file with index-state", config: ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": "cabal"}},
			files: map[string]string{"x.cabal": "", "cabal.project.freeze": "active-repositories: hackage.haskell.org\r\nindex-state: hackage.haskell.org 2026-01-01T00:00:00Z\r\n"},
		},
		{
			name: "stack project", config: ecosystem.ModuleConfig{Version: "9.6.7", Extras: stackConfig.Extras},
			files: map[string]string{"stack.yaml": "resolver: lts-22.44\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := (&Module{}).SetupWarnings(dir, tt.config)
			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("SetupWarnings() = %q, want one warning", got)
			}
			for _, sub := range tt.want {
				if !strings.Contains(got[0], sub) {
					t.Errorf("warning %q does not contain %q", got[0], sub)
				}
			}
		})
	}
}

func TestPathGHCVersion_NotOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := pathGHCVersion(context.Background()); err == nil {
		t.Error("pathGHCVersion() with no ghc on PATH: want an error")
	}
}
