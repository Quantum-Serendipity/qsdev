package render

import (
	"encoding/json"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

// TestRenderSARIF_ConfigFileStates guards the config-health mapping: a file
// that cannot be read must surface, and a deleted file reported by both config
// health and drift detection must yield a single config-missing result.
func TestRenderSARIF_ConfigFileStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		files    []posture.ConfigFileInfo
		findings []drift.Finding
		want     map[string]int // rule ID -> result count
	}{
		{
			name:  "corrupt file",
			files: []posture.ConfigFileInfo{{Path: ".claude/settings.json", State: "corrupt"}},
			want:  map[string]int{"qsdev/config-corrupt": 1},
		},
		{
			name:  "deleted file reported twice",
			files: []posture.ConfigFileInfo{{Path: ".claude/settings.json", State: "missing"}},
			findings: []drift.Finding{{
				Severity:    drift.Error,
				Subject:     ".claude/settings.json",
				Description: "Generated file \".claude/settings.json\" has been deleted",
			}},
			want: map[string]int{"qsdev/config-missing": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := &posture.PostureReport{
				Config: posture.ConfigHealth{Files: tt.files},
				Drift: drift.Report{
					Categories: []drift.Category{{Name: categoryFileModification, Findings: tt.findings}},
					BySeverity: map[drift.Severity]int{},
				},
			}

			data, err := RenderSARIF(report)
			if err != nil {
				t.Fatal(err)
			}
			var log sarifLog
			if err := json.Unmarshal(data, &log); err != nil {
				t.Fatal(err)
			}

			got := make(map[string]int)
			for _, r := range log.Runs[0].Results {
				got[r.RuleID]++
			}
			for id, n := range tt.want {
				if got[id] != n {
					t.Errorf("rule %s: %d results, want %d (all: %v)", id, got[id], n, got)
				}
			}
			for _, r := range log.Runs[0].Results {
				if !ruleDefined(log, r.RuleID) {
					t.Errorf("result uses undefined rule %s", r.RuleID)
				}
			}
		})
	}
}

func ruleDefined(log sarifLog, id string) bool {
	for _, rule := range log.Runs[0].Tool.Driver.Rules {
		if rule.ID == id {
			return true
		}
	}
	return false
}
