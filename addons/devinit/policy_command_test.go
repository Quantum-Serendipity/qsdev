package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
)

// TestRenderPolicySARIFEmitsSARIF is the red→green guard for F-CAP-21.3-1 at the
// command surface: `policy check --sarif` must emit a SARIF 2.1.0 document, not
// the bare PolicyPosture JSON it used to marshal.
func TestRenderPolicySARIFEmitsSARIF(t *testing.T) {
	t.Parallel()

	posture := &sarif.PolicyPosture{
		RulesActive:      3,
		RulesTotal:       3,
		MonitorModeCount: 1,
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	if err := renderPolicySARIF(cmd, posture, ""); err != nil {
		t.Fatalf("renderPolicySARIF: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if parsed["$schema"] != sarif.SchemaURL {
		t.Errorf("$schema = %v, want %q", parsed["$schema"], sarif.SchemaURL)
	}
	if parsed["version"] != "2.1.0" {
		t.Errorf("version = %v, want \"2.1.0\"", parsed["version"])
	}
	runs, ok := parsed["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("expected one run, got %v", parsed["runs"])
	}
	run := runs[0].(map[string]any)
	tool, ok := run["tool"].(map[string]any)
	if !ok {
		t.Fatalf("run missing tool")
	}
	if _, ok := tool["driver"].(map[string]any); !ok {
		t.Fatalf("tool missing driver")
	}
	if _, ok := run["results"].([]any); !ok {
		t.Fatalf("run missing results array")
	}
}

// TestEvaluatePolicyPostureExitGateParity guards BUG #4: the --sarif render path
// used to return before the exit gate, so `policy check --sarif` always exited 0
// even with zero enforced policy. Both render paths must now reach the same gate,
// so a zero-rules posture at the default audit level fails identically whether the
// output is SARIF or text, while a healthy posture passes on both paths. The
// `--audit-level none` escape must remain intact on both paths.
func TestEvaluatePolicyPostureExitGateParity(t *testing.T) {
	t.Parallel()

	zeroRules := &sarif.PolicyPosture{RulesActive: 0, RulesTotal: 5}
	healthy := &sarif.PolicyPosture{RulesActive: 3, RulesTotal: 3}

	tests := []struct {
		name       string
		posture    *sarif.PolicyPosture
		sarifFlag  bool
		auditLevel string
		wantExit   bool
	}{
		// The flag default is "any"; zero active rules must fail on both paths.
		{"sarif zero rules default audit", zeroRules, true, "any", true},
		{"text zero rules default audit", zeroRules, false, "any", true},
		// A healthy posture passes on both paths.
		{"sarif healthy default audit", healthy, true, "any", false},
		{"text healthy default audit", healthy, false, "any", false},
		// The --audit-level none escape disables the gate on both paths.
		{"sarif zero rules audit none escape", zeroRules, true, "none", false},
		{"text zero rules audit none escape", zeroRules, false, "none", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&buf)

			err := evaluatePolicyPosture(cmd, tt.posture, tt.sarifFlag, tt.auditLevel, "")

			if !tt.wantExit {
				if err != nil {
					t.Fatalf("evaluatePolicyPosture() = %v, want nil", err)
				}
				return
			}

			var exitErr *ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("evaluatePolicyPosture() = %v, want *ExitError", err)
			}
			if exitErr.Code != 1 {
				t.Errorf("ExitError.Code = %d, want 1", exitErr.Code)
			}
		})
	}
}
