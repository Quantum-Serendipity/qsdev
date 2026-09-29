package render

import (
	"encoding/json"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

// TestRenderSARIF_DriftCategoryMapping is the F340 regression test: state
// files that failed to load and version drift reach SARIF, and a
// marker-integrity finding keeps its own severity.
func TestRenderSARIF_DriftCategoryMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		category  string
		severity  drift.Severity
		subject   string
		wantRule  string
		wantLevel string
	}{
		{
			name:      "corrupt state file",
			category:  posture.StateFilesCategory,
			severity:  drift.Warning,
			subject:   ".devinit/.qsdev-init-state.yaml",
			wantRule:  ruleID("state-corrupt"),
			wantLevel: "error",
		},
		{
			name:      "version drift",
			category:  "Version Drift",
			severity:  drift.Info,
			subject:   "qsdev version",
			wantRule:  ruleID("version-drift"),
			wantLevel: "note",
		},
		{
			name:      "missing CLAUDE.md",
			category:  categoryMarkerIntegrity,
			severity:  drift.Error,
			subject:   "CLAUDE.md",
			wantRule:  ruleID("markers-broken"),
			wantLevel: "error",
		},
		{
			name:      "unpaired marker",
			category:  categoryMarkerIntegrity,
			severity:  drift.Warning,
			subject:   "marker:gitleaks",
			wantRule:  ruleID("markers-broken"),
			wantLevel: "warning",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := &posture.PostureReport{
				Drift: drift.Report{
					Categories: []drift.Category{{
						Name: tt.category,
						Findings: []drift.Finding{{
							Category:    tt.category,
							Severity:    tt.severity,
							Subject:     tt.subject,
							Description: tt.name,
						}},
					}},
				},
			}
			data, err := RenderSARIF(report)
			if err != nil {
				t.Fatalf("RenderSARIF: %v", err)
			}
			var log sarifLog
			if err := json.Unmarshal(data, &log); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			results := log.Runs[0].Results
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1: %+v", len(results), results)
			}
			if results[0].RuleID != tt.wantRule || results[0].Level != tt.wantLevel {
				t.Errorf("result = %s/%s, want %s/%s", results[0].RuleID, results[0].Level, tt.wantRule, tt.wantLevel)
			}

			var ruleDefined bool
			for _, r := range log.Runs[0].Tool.Driver.Rules {
				if r.ID == tt.wantRule {
					ruleDefined = true
				}
			}
			if !ruleDefined {
				t.Errorf("rule %s is not defined in the driver", tt.wantRule)
			}
		})
	}
}
