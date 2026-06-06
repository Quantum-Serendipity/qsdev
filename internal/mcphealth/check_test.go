package mcphealth

import (
	"testing"
	"time"
)

func TestCheckServer_MissingBinary(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Name:    "nonexistent",
		Command: "this-binary-does-not-exist-xyz-999",
		Args:    []string{"--stdio"},
	}

	h := CheckServer(cfg, 5*time.Second)

	if h.Name != "nonexistent" {
		t.Errorf("name = %q, want %q", h.Name, "nonexistent")
	}
	if h.Status != StatusUnreachable {
		t.Errorf("status = %q, want %q", h.Status, StatusUnreachable)
	}
	if h.Error == "" {
		t.Error("expected non-empty error for missing binary")
	}
}

func TestCheckServer_Prerequisites(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Name:        "needs-secret",
		Command:     "bash",
		Args:        []string{"-c", "echo hello"},
		RequiredEnv: []string{"QSDEV_TEST_MISSING_SECRET_XYZ_999"},
	}

	h := CheckServer(cfg, 5*time.Second)

	if h.Status != StatusDegraded {
		t.Errorf("status = %q, want %q", h.Status, StatusDegraded)
	}

	if len(h.Prerequisites) == 0 {
		t.Fatal("expected at least one prerequisite status")
	}

	p := h.Prerequisites[0]
	if p.Met {
		t.Error("expected prerequisite to be unmet")
	}
	if p.Name != "QSDEV_TEST_MISSING_SECRET_XYZ_999" {
		t.Errorf("prereq name = %q, want QSDEV_TEST_MISSING_SECRET_XYZ_999", p.Name)
	}
}

func TestCheckAll_EmptyServers(t *testing.T) {
	t.Parallel()

	report := CheckAll(map[string]ServerConfig{}, 5*time.Second)

	if report.TotalCount != 0 {
		t.Errorf("total = %d, want 0", report.TotalCount)
	}
	if report.HealthyCount != 0 {
		t.Errorf("healthy = %d, want 0", report.HealthyCount)
	}
	if len(report.Servers) != 0 {
		t.Errorf("servers = %d, want 0", len(report.Servers))
	}
	if report.CheckedAt.IsZero() {
		t.Error("expected CheckedAt to be set")
	}
}

func TestCheckAll_MixedResults(t *testing.T) {
	t.Parallel()

	servers := map[string]ServerConfig{
		"missing-binary": {
			Name:    "missing-binary",
			Command: "this-binary-does-not-exist-xyz-999",
		},
		"missing-prereq": {
			Name:        "missing-prereq",
			Command:     "bash",
			RequiredEnv: []string{"QSDEV_TEST_MISSING_MIX_12345"},
		},
	}

	report := CheckAll(servers, 5*time.Second)

	if report.TotalCount != 2 {
		t.Errorf("total = %d, want 2", report.TotalCount)
	}
	if report.HealthyCount != 0 {
		t.Errorf("healthy = %d, want 0", report.HealthyCount)
	}
	if len(report.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(report.Servers))
	}

	statuses := map[string]string{}
	for _, s := range report.Servers {
		statuses[s.Name] = s.Status
	}

	if statuses["missing-binary"] != StatusUnreachable {
		t.Errorf("missing-binary status = %q, want %q", statuses["missing-binary"], StatusUnreachable)
	}
	if statuses["missing-prereq"] != StatusDegraded {
		t.Errorf("missing-prereq status = %q, want %q", statuses["missing-prereq"], StatusDegraded)
	}
}

func TestCountTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "two tools",
			input: `{"tools":[{"name":"a"},{"name":"b"}]}`,
			want:  2,
		},
		{
			name:  "empty tools list",
			input: `{"tools":[]}`,
			want:  0,
		},
		{
			name:  "invalid json",
			input: `not json`,
			want:  0,
		},
		{
			name:  "missing tools key",
			input: `{"other":"value"}`,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := countTools([]byte(tt.input))
			if got != tt.want {
				t.Errorf("countTools(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
