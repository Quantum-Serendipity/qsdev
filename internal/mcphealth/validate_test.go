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
			name: "env reference with default produces no warning",
			servers: map[string]ServerConfig{
				"default-env": {
					Name:    "default-env",
					Command: "bash",
					Env:     map[string]string{"P": "${QSDEV_TEST_MISSING_REF_DEF_111:-x}"},
				},
			},
			wantCount: 0,
		},
		{
			// F276: a remote server has a URL and no command; it used to be
			// reported as "command is empty".
			name: "https server without command produces no warning",
			servers: map[string]ServerConfig{
				"socket": {Name: "socket", URL: "https://mcp.socket.dev/"},
			},
			wantCount: 0,
		},
		{
			name: "unset reference in args produces warning",
			servers: map[string]ServerConfig{
				"ref-arg": {
					Name:    "ref-arg",
					Command: "bash",
					Args:    []string{"--db", "${QSDEV_TEST_MISSING_ARG_222}"},
				},
			},
			wantCount:    1,
			wantSeverity: "warning",
			wantContains: "QSDEV_TEST_MISSING_ARG_222",
		},
		{
			name: "unset reference in header produces warning",
			servers: map[string]ServerConfig{
				"ref-header": {
					Name:    "ref-header",
					Command: "bash",
					Headers: map[string]string{"Authorization": "Bearer ${QSDEV_TEST_MISSING_HDR_333}"},
				},
			},
			wantCount:    1,
			wantSeverity: "warning",
			wantContains: "QSDEV_TEST_MISSING_HDR_333",
		},
		{
			name: "http localhost server produces no warning",
			servers: map[string]ServerConfig{
				"local": {Name: "local", URL: "http://127.0.0.1:8080/mcp"},
			},
			wantCount: 0,
		},
		{
			name: "plain http to a remote host produces error",
			servers: map[string]ServerConfig{
				"remote": {Name: "remote", URL: "http://mcp.example.com/mcp"},
			},
			wantCount:    1,
			wantSeverity: "error",
			wantContains: "plain http",
		},
		{
			name: "malformed url produces error",
			servers: map[string]ServerConfig{
				"bad": {Name: "bad", URL: "ftp://mcp.example.com"},
			},
			wantCount:    1,
			wantSeverity: "error",
			wantContains: "not a valid http(s) URL",
		},
		{
			name: "set env reference produces no warning",
			servers: map[string]ServerConfig{
				"ok-env": {
					Name:    "ok-env",
					Command: "bash",
					Env:     map[string]string{"P": "${PATH}"},
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
