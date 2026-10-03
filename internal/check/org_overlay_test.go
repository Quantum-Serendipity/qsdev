package check

import (
	"strings"
	"testing"
)

// TestCheckOrgOverlay pins that check reports an org overlay that runs no
// human starts ignore (U18-WS1 round 3), and passes when there is none.
func TestCheckOrgOverlay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		drift  string
		status CheckStatus
	}{
		{"pinned", "", StatusPass},
		{"pinned with a source", "", StatusPass},
		{"drifted", "the org overlay resolves to /x, not /y (pinned for the project)", StatusWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := ""
			if tt.name == "pinned with a source" {
				source = "/etc/org/defaults.yaml (pinned for the project)"
			}
			got := CheckOrgOverlay(CheckContext{OrgConfigDrift: tt.drift, OrgConfigSource: source})
			// Round 4: a pass names what is read, not a record that may not exist.
			if source != "" && !strings.Contains(got.Message, source) {
				t.Errorf("CheckOrgOverlay message %q, want the overlay read named", got.Message)
			}
			if tt.drift == "" && strings.Contains(got.Message, "recorded") {
				t.Errorf("CheckOrgOverlay message %q claims a record", got.Message)
			}
			if got.Status != tt.status || got.Name != "org_overlay_pinned" {
				t.Errorf("CheckOrgOverlay = %+v, want status %s", got, tt.status)
			}
			if tt.drift != "" && (!strings.Contains(got.Message, tt.drift) || got.Remediation == "") {
				t.Errorf("CheckOrgOverlay message %q / remediation %q, want the drift and a remediation", got.Message, got.Remediation)
			}
		})
	}
	report := RunAllChecks(CheckContext{OrgConfigDrift: "drift"})
	found := false
	for _, r := range report.Checks {
		found = found || (r.Name == "org_overlay_pinned" && r.Status == StatusWarn)
	}
	if !found {
		t.Error("RunAllChecks does not report the org overlay drift")
	}
}
