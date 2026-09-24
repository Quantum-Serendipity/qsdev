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

// Regression: a boolean needs-migration check returned false for too-new and
// invalid versions, which `config migrate` reported as "already current". CheckMigration must
// distinguish current, migratable and unsupported versions.
func TestCheckMigration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		version    int
		wantNeeded bool
		wantErr    string
	}{
		{"current", types.ConfigVersionCurrent, false, ""},
		{"newer than supported", types.ConfigVersionCurrent + 4, false, "newer than"},
		{"zero", 0, false, "too old"},
		{"negative", -1, false, "too old"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			needed, err := CheckMigration(tt.version)
			if needed != tt.wantNeeded {
				t.Errorf("CheckMigration(%d) needed = %v, want %v", tt.version, needed, tt.wantNeeded)
			}
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("CheckMigration(%d) error = %v, want nil", tt.version, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("CheckMigration(%d) error = %v, want it to contain %q", tt.version, err, tt.wantErr)
			}
		})
	}
}

// toInt reads the YAML-decoded "version" value; only whole numbers count.
func TestToInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     any
		want   int
		wantOK bool
	}{
		{"int", 1, 1, true},
		{"int64", int64(2), 2, true},
		{"whole float", float64(3), 3, true},
		{"fractional float", 1.5, 0, false},
		{"string", "1", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := toInt(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("toInt(%v) = (%d, %v), want (%d, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
