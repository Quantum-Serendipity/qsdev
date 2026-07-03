package devinit

import "testing"

// TestVulnAuditLevelNeedsScan pins the honest-reporting behavior (M1): every
// gating level — including the DEFAULT "high" and "info" — must surface the
// "dependencies not scanned" note when --scan is omitted, so a zero vulnerability
// count is never presented as a clean gate. Only "none" (which never gates) is
// exempt. The earlier enumeration omitted "high" and "info", suppressing the note
// for the most common invocation (`qsdev status`).
func TestVulnAuditLevelNeedsScan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level string
		want  bool
	}{
		{"none", false},
		{"info", true},
		{"low", true},
		{"moderate", true},
		{"high", true}, // the default level — regression guard for M1
		{"critical", true},
	}
	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			t.Parallel()
			if got := vulnAuditLevelNeedsScan(tc.level); got != tc.want {
				t.Errorf("vulnAuditLevelNeedsScan(%q) = %v, want %v", tc.level, got, tc.want)
			}
		})
	}
}
