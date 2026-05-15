package config

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestMigrateConfig_NoMigrationNeeded(t *testing.T) {
	raw := map[string]any{
		"version": types.ConfigVersionCurrent,
		"languages": []any{
			map[string]any{"name": "go"},
		},
	}

	result, err := MigrateConfig(raw, types.ConfigVersionCurrent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be unchanged.
	if result["version"] != types.ConfigVersionCurrent {
		t.Errorf("version = %v, want %d", result["version"], types.ConfigVersionCurrent)
	}
}

func TestMigrateConfig_VersionTooHigh(t *testing.T) {
	raw := map[string]any{"version": 999}

	_, err := MigrateConfig(raw, 999)
	if err == nil {
		t.Fatal("expected error for version too high")
	}
	if !strings.Contains(err.Error(), "newer than") {
		t.Errorf("error = %q, want 'newer than' message", err.Error())
	}
}

func TestMigrateConfig_VersionTooLow(t *testing.T) {
	raw := map[string]any{"version": 0}

	_, err := MigrateConfig(raw, 0)
	if err == nil {
		t.Fatal("expected error for version too low")
	}
	if !strings.Contains(err.Error(), "too old") {
		t.Errorf("error = %q, want 'too old' message", err.Error())
	}
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		version int
		want    bool
	}{
		{0, false},
		{types.ConfigVersionCurrent, false},
		{types.ConfigVersionCurrent + 1, false},
		// Since ConfigVersionMin == ConfigVersionCurrent == 1, there's
		// no version that would trigger migration. But if ConfigVersionCurrent
		// were higher, versions below it would need migration.
	}

	for _, tt := range tests {
		got := NeedsMigration(tt.version)
		if got != tt.want {
			t.Errorf("NeedsMigration(%d) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestNeedsMigration_FutureVersions(t *testing.T) {
	// NeedsMigration should return false for versions at or above current.
	if NeedsMigration(types.ConfigVersionCurrent) {
		t.Error("NeedsMigration should return false for current version")
	}
	if NeedsMigration(types.ConfigVersionCurrent + 1) {
		t.Error("NeedsMigration should return false for future version")
	}
}
