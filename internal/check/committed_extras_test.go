package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDetectedModuleConfig_CommittedExtrasWin checks the checks see the
// ModuleConfig generation uses: detected extras completed by, and overridden
// with, the extras .qsdev.yaml declares.
func TestDetectedModuleConfig_CommittedExtrasWin(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	goMod := "module example.com/m\n\ngo 1.26.0\n\ntoolchain go1.26.8\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	mod, ok := ecosystem.DefaultRegistry().ByName(ecosystem.NameGo)
	if !ok {
		t.Fatal("go module not registered")
	}
	tests := []struct {
		name          string
		extras        []string
		wantSource    string
		wantToolchain string
	}{
		{"detected only", nil, "", "1.26.8"},
		{"committed added", []string{"go_source=nixpkgs"}, "nixpkgs", "1.26.8"},
		{"committed overrides detected", []string{"toolchain=1.26.9"}, "", "1.26.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lang := types.LanguageConfig{Name: ecosystem.NameGo, Extras: tt.extras}
			cfg := detectedModuleConfig(root, mod, lang, 0, types.InfraConfig{})
			if got := cfg.Extra(types.SettingGoSource, ""); got != tt.wantSource {
				t.Errorf("go_source = %q, want %q", got, tt.wantSource)
			}
			if got := cfg.Extra("toolchain", ""); got != tt.wantToolchain {
				t.Errorf("toolchain = %q, want %q", got, tt.wantToolchain)
			}
		})
	}
}
