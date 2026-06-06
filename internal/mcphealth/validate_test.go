package mcphealth

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		servers      map[string]ServerConfig
		wantCount    int
		wantSeverity string
		wantContains string
	}{
		{
			name: "valid config produces no warnings",
			servers: map[string]ServerConfig{
				"test": {
					Name:    "test",
					Command: "bash",
				},
			},
			wantCount: 0,
		},
		{
			name: "missing binary produces error warning",
			servers: map[string]ServerConfig{
				"broken": {
					Name:    "broken",
					Command: "this-binary-does-not-exist-xyz-999",
				},
			},
			wantCount:    1,
			wantSeverity: "error",
			wantContains: "not found on PATH",
		},
		{
			name: "empty command produces error warning",
			servers: map[string]ServerConfig{
				"empty-cmd": {
					Name:    "empty-cmd",
					Command: "",
				},
			},
			wantCount:    1,
			wantSeverity: "error",
			wantContains: "command is empty",
		},
		{
			name: "unset required env produces warning",
			servers: map[string]ServerConfig{
				"needs-env": {
					Name:        "needs-env",
					Command:     "bash",
					RequiredEnv: []string{"QSDEV_TEST_MISSING_VAR_XYZ_12345"},
				},
			},
			wantCount:    1,
			wantSeverity: "warning",
			wantContains: "QSDEV_TEST_MISSING_VAR_XYZ_12345",
		},
		{
			name: "unset env reference produces warning",
			servers: map[string]ServerConfig{
				"ref-env": {
					Name:    "ref-env",
					Command: "bash",
					Env:     map[string]string{"TOKEN": "${QSDEV_TEST_MISSING_REF_ABC_67890}"},
				},
			},
			wantCount:    1,
			wantSeverity: "warning",
			wantContains: "QSDEV_TEST_MISSING_REF_ABC_67890",
		},
		{
			name: "set env reference produces no warning",
			servers: map[string]ServerConfig{
				"ok-env": {
					Name:    "ok-env",
					Command: "bash",
					Env:     map[string]string{"H": "${HOME}"},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			warnings := ValidateConfig(tt.servers)

			if len(warnings) != tt.wantCount {
				t.Fatalf("got %d warnings, want %d: %+v", len(warnings), tt.wantCount, warnings)
			}

			if tt.wantCount > 0 {
				w := warnings[0]
				if w.Severity != tt.wantSeverity {
					t.Errorf("severity = %q, want %q", w.Severity, tt.wantSeverity)
				}
				if tt.wantContains != "" && !strings.Contains(w.Message, tt.wantContains) {
					t.Errorf("message %q does not contain %q", w.Message, tt.wantContains)
				}
			}
		})
	}
}

func TestValidateConfig_EmptyServerName(t *testing.T) {
	t.Parallel()

	servers := map[string]ServerConfig{
		"": {
			Name:    "",
			Command: "bash",
		},
	}

	warnings := ValidateConfig(servers)

	found := false
	for _, w := range warnings {
		if w.Severity == "error" && strings.Contains(w.Message, "name is empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about empty server name, got %+v", warnings)
	}
}
