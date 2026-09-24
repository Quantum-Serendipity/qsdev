package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

func TestEvaluate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		reqs       []Requirement
		wantPass   bool
		wantChecks []posture.ConformanceCheck
	}{
		{
			name: "all pass",
			reqs: []Requirement{
				{Name: "sast", Check: "tools.semgrep.enabled == true"},
				{Name: "score", Check: " score.total >= 80 "},
			},
			wantPass: true,
			wantChecks: []posture.ConformanceCheck{
				{Name: "sast", Pass: true, Status: posture.CheckPass, Reason: "tools.semgrep.enabled == true"},
				{Name: "score", Pass: true, Status: posture.CheckPass, Reason: "score.total >= 80"},
			},
		},
		{
			name: "failure reports the actual value",
			reqs: []Requirement{
				{Name: "sast", Check: "tools.semgrep.enabled == true"},
				{Name: "no high", Check: "dependencies.totals.high == 0"},
			},
			wantPass: false,
			wantChecks: []posture.ConformanceCheck{
				{Name: "sast", Pass: true, Status: posture.CheckPass, Reason: "tools.semgrep.enabled == true"},
				{Name: "no high", Status: posture.CheckFail, Reason: "dependencies.totals.high == 0: actual 2"},
			},
		},
		{
			name:     "invalid expression fails",
			reqs:     []Requirement{{Name: "typo", Check: "defense.sast.status = enabled"}},
			wantPass: false,
			wantChecks: []posture.ConformanceCheck{{
				Name:   "typo",
				Status: posture.CheckFail,
				Reason: `defense.sast.status = enabled: no supported operator found in expression: "defense.sast.status = enabled"`,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Evaluate(&Custom{Requirements: tt.reqs}, makeReport())
			if got.Pass != tt.wantPass {
				t.Errorf("Pass = %v, want %v", got.Pass, tt.wantPass)
			}
			if len(got.Checks) != len(tt.wantChecks) {
				t.Fatalf("checks = %+v, want %+v", got.Checks, tt.wantChecks)
			}
			for i, want := range tt.wantChecks {
				if got.Checks[i] != want {
					t.Errorf("check %d = %+v, want %+v", i, got.Checks[i], want)
				}
			}
		})
	}
}

// TestEvaluate_UnscannedDependencies is the regression test for a custom
// "no critical vulns" requirement passing although no scan ran.
func TestEvaluate_UnscannedDependencies(t *testing.T) {
	t.Parallel()
	report := makeReport()
	report.Dependencies = posture.DependencyHealth{}
	got := Evaluate(&Custom{Requirements: []Requirement{
		{Name: "no critical", Check: "dependencies.totals.critical == 0"},
	}}, report)
	if got.Pass || got.Checks[0].Pass {
		t.Fatalf("unscanned dependencies passed: %+v", got)
	}
	if !strings.Contains(got.Checks[0].Reason, "not scanned") {
		t.Errorf("reason %q does not say the dependencies were not scanned", got.Checks[0].Reason)
	}
}

func TestApply(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// policy is written to the policy file; "" leaves it absent.
		policy     string
		wantCustom bool
		wantPass   bool
		wantCheck  posture.CheckName
		wantReason string
	}{
		{name: "no policy file"},
		{name: "policy without custom section", policy: "conformance: {}\n"},
		{
			name: "passing policy",
			policy: "conformance:\n  custom:\n    name: strict\n    requirements:\n" +
				"      - name: sast\n        check: tools.semgrep.enabled == true\n",
			wantCustom: true,
			wantPass:   true,
			wantCheck:  "sast",
		},
		{
			name: "failing policy",
			policy: "conformance:\n  custom:\n    requirements:\n" +
				"      - name: nix\n        check: defense.nix-hardening.status == enabled\n",
			wantCustom: true,
			wantCheck:  "nix",
			wantReason: "actual disabled",
		},
		{
			name:       "malformed policy fails closed",
			policy:     "conformance:\n  custom:\n    requirement: []\n",
			wantCustom: true,
			wantCheck:  PolicyFileCheck,
			wantReason: "requirement",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.policy != "" {
				path := filepath.Join(dir, PolicyFileName())
				if err := os.WriteFile(path, []byte(tt.policy), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			report := makeReport()
			Apply(dir, report)

			custom := report.Conformance.Custom
			if !tt.wantCustom {
				if custom != nil {
					t.Fatalf("Custom = %+v, want nil", custom)
				}
				return
			}
			if custom == nil {
				t.Fatal("Custom = nil, want an evaluated level")
			}
			if custom.Pass != tt.wantPass {
				t.Errorf("Pass = %v, want %v", custom.Pass, tt.wantPass)
			}
			if len(custom.Checks) != 1 || custom.Checks[0].Name != tt.wantCheck {
				t.Fatalf("checks = %+v, want one named %q", custom.Checks, tt.wantCheck)
			}
			if !strings.Contains(custom.Checks[0].Reason, tt.wantReason) {
				t.Errorf("reason %q does not contain %q", custom.Checks[0].Reason, tt.wantReason)
			}
		})
	}
}

func TestPolicyFileName(t *testing.T) {
	t.Parallel()
	if got := PolicyFileName(); got != ".qsdev-policy.yaml" {
		t.Errorf("PolicyFileName() = %q, want .qsdev-policy.yaml", got)
	}
}
