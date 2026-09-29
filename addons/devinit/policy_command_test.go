package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
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

	data, err := renderPolicySARIF(posture, 2)
	if err != nil {
		t.Fatalf("renderPolicySARIF: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
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
	// Every rule enabled but in monitor mode: nothing is enforced (F054).
	monitorOnly := &sarif.PolicyPosture{RulesActive: 1, RulesTotal: 1, MonitorModeCount: 1}

	tests := []struct {
		name       string
		posture    *sarif.PolicyPosture
		enforcing  int
		sarifFlag  bool
		auditLevel string
		wantExit   bool
	}{
		// The flag default is "any"; zero active rules must fail on both paths.
		{"sarif zero rules default audit", zeroRules, 0, true, "any", true},
		{"text zero rules default audit", zeroRules, 0, false, "any", true},
		// A healthy posture passes on both paths.
		{"sarif healthy default audit", healthy, 3, true, "any", false},
		{"text healthy default audit", healthy, 3, false, "any", false},
		// A monitor-only policy enforces nothing and fails on both paths.
		{"sarif monitor-only default audit", monitorOnly, 0, true, "any", true},
		{"text monitor-only default audit", monitorOnly, 0, false, "any", true},
		// The --audit-level none escape disables the gate on both paths.
		{"sarif zero rules audit none escape", zeroRules, 0, true, "none", false},
		{"text zero rules audit none escape", zeroRules, 0, false, "none", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&buf)

			err := evaluatePolicyPosture(cmd, tt.posture, tt.enforcing,
				policyCheckOptions{sarif: tt.sarifFlag, auditLevel: tt.auditLevel})

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

func TestEnforcingRuleCount(t *testing.T) {
	t.Parallel()
	disabled := false
	rules := []policy.CompiledRule{
		{Rule: policy.PolicyRule{ID: "enforcing"}},
		{Rule: policy.PolicyRule{ID: "monitor", MonitorMode: true}},
		{Rule: policy.PolicyRule{ID: "disabled", Enabled: &disabled}},
	}
	if got := enforcingRuleCount(rules); got != 1 {
		t.Errorf("enforcingRuleCount = %d, want 1", got)
	}
}

// TestPolicyCheck_OutputAndAuditLevel is the regression test for --output being
// ignored without --sarif and --audit-level accepting any value.
func TestPolicyCheck_OutputAndAuditLevel(t *testing.T) {
	t.Parallel()

	t.Run("text output is written to --output", func(t *testing.T) {
		t.Parallel()
		out := filepath.Join(t.TempDir(), "policy.txt")
		posture := &sarif.PolicyPosture{RulesActive: 2, RulesTotal: 2}
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		if err := evaluatePolicyPosture(cmd, posture, 2, policyCheckOptions{auditLevel: "any", output: out}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("--output file not written: %v", err)
		}
		if !strings.Contains(string(data), "Policy Posture Summary") {
			t.Errorf("--output file content = %q", data)
		}
		if buf.Len() != 0 {
			t.Errorf("stdout should be empty when --output is set, got %q", buf.String())
		}
	})

	t.Run("unknown audit level is rejected", func(t *testing.T) {
		t.Parallel()
		cmd := policyCheckCmd()
		cmd.SetArgs([]string{"--audit-level", "bogus"})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid --audit-level") {
			t.Errorf("err = %v, want an invalid --audit-level error", err)
		}
	})
}

// TestRenderPolicyText_Deterministic verifies map-backed sections are sorted.
func TestRenderPolicyText_Deterministic(t *testing.T) {
	t.Parallel()
	posture := &sarif.PolicyPosture{
		RulesActive:       3,
		RulesTotal:        3,
		BypassTierSummary: map[string]int{"tier-c": 1, "tier-a": 1, "tier-b": 1},
		CategoryCoverage:  map[string]bool{"zeta": true, "alpha": true, "mid": true},
	}
	text := string(renderPolicyText(posture, 3))
	for _, pair := range [][2]string{{"tier-a", "tier-b"}, {"tier-b", "tier-c"}, {"alpha", "mid"}, {"mid", "zeta"}} {
		if strings.Index(text, pair[0]) > strings.Index(text, pair[1]) {
			t.Errorf("%q should be listed before %q:\n%s", pair[0], pair[1], text)
		}
	}
}
