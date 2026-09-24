package haskell_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/haskell"
)

// newModule returns a fresh Module for testing.
func newModule() *haskell.Module {
	return &haskell.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*haskell.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "haskell", "Haskell", 3)
}

// --- Detection tests ---

func TestDetect_CabalFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.cabal"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !containsSubstr(r.Evidence, "*.cabal") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "*.cabal")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_NonEmpty(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
}

func TestDetect_StackSnapshotGHC(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		stack       string
		wantVersion string
	}{
		{name: "LTS snapshot", stack: "resolver: lts-22.44\n", wantVersion: "9.6.7"},
		{name: "compiler override", stack: "snapshot: lts-22.44\ncompiler: ghc-9.6.6\n", wantVersion: "9.6.6"},
		{name: "unknown snapshot", stack: "resolver: nightly-2025-01-01\n", wantVersion: ""},
		{name: "no snapshot", stack: "packages: [.]\n", wantVersion: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "stack.yaml"), []byte(tt.stack), 0o644); err != nil {
				t.Fatal(err)
			}
			r := newModule().Detect(dir)
			if got := r.SuggestedConfig.Version; got != tt.wantVersion {
				t.Errorf("SuggestedConfig.Version = %q, want %q", got, tt.wantVersion)
			}
			if got := r.SuggestedConfig.Extra("build_tool", ""); got != "stack" {
				t.Errorf("build_tool = %q, want stack", got)
			}
			if tt.wantVersion != "" && !containsSubstr(r.Evidence, "uses GHC "+tt.wantVersion) {
				t.Errorf("Evidence = %q, want the snapshot's GHC", r.Evidence)
			}
		})
	}
}

func TestDetect_CabalHasNoVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.cabal"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := newModule().Detect(dir).SuggestedConfig.Version; v != "" {
		t.Errorf("SuggestedConfig.Version = %q, want empty for a Cabal project", v)
	}
}

func TestDevenvNixFragment_GHC(t *testing.T) {
	t.Parallel()
	stack := map[string]string{"build_tool": "stack"}
	tests := []struct {
		name    string
		config  ecosystem.ModuleConfig
		want    []string
		notWant []string
	}{
		{
			name:   "stack with snapshot GHC",
			config: ecosystem.ModuleConfig{Version: "9.6.7", Extras: stack},
			want: []string{
				"languages.haskell.stack.enable = true;",
				"builtins.tryEval (pkgs.haskell.compiler.ghc967 or null)",
				`ghc.value.version == "9.6.7"`,
				"then ghc.value\n    else pkgs.ghc;",
				"languages.haskell.stack.args = lib.mkIf\n    (!(config.languages.haskell.package.version == \"9.6.7\"))",
				`[ "--no-nix" ]`,
				"so Stack installs GHC 9.6.7 itself, outside Nix",
			},
		},
		{
			name:    "stack with unknown GHC",
			config:  ecosystem.ModuleConfig{Extras: stack},
			want:    []string{"languages.haskell.stack.enable = true;", `languages.haskell.stack.args = [ "--no-nix" ];`},
			notWant: []string{"languages.haskell.package", "lib.mkIf"},
		},
		{
			name:    "cabal with a series version",
			config:  ecosystem.ModuleConfig{Version: "9.8"},
			want:    []string{"pkgs.haskell.compiler.ghc98 or null", `lib.versions.majorMinor ghc.value.version == "9.8"`, `else lib.warn "GHC 9.8 is configured`},
			notWant: []string{"stack"},
		},
		{
			name:    "cabal without version",
			config:  ecosystem.ModuleConfig{},
			notWant: []string{"languages.haskell.package", "stack"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(tt.config)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			for _, sub := range tt.want {
				if !strings.Contains(frag, sub) {
					t.Errorf("fragment does not contain %q:\n%s", sub, frag)
				}
			}
			for _, sub := range tt.notWant {
				if strings.Contains(frag, sub) {
					t.Errorf("fragment contains %q:\n%s", sub, frag)
				}
			}
		})
	}
}

func TestDevenvNixFragment_InvalidVersion(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"9", "9.6.7; evil", `9.6" + x`, "latest"} {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if _, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Version: v}); err == nil {
				t.Errorf("DevenvNixFragment(Version %q): want an error", v)
			}
		})
	}
}

// --- helpers ---

func containsSubstr(ss []string, substr string) bool {
	for _, s := range ss {
		if len(s) >= len(substr) && searchString(s, substr) {
			return true
		}
	}
	return false
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
