package check

import "testing"

func TestCheckCustomConformance(t *testing.T) {
	t.Parallel()
	custom := &CustomConformance{
		PolicyFile: ".qsdev-policy.yaml",
		Requirements: []PolicyRequirement{
			{Name: "sast", Pass: true, Reason: "tools.semgrep.enabled == true"},
			{Name: "no critical", Reason: "dependencies.totals.critical == 0: actual 2"},
		},
	}
	tests := []struct {
		name       string
		custom     *CustomConformance
		want       []CheckResult
		failMedium bool
	}{
		{name: "no policy", custom: nil, want: nil},
		{
			name:   "requirements map to checks",
			custom: custom,
			want: []CheckResult{
				{
					Category: CategoryCustomConformance, Name: "sast", Status: StatusPass,
					Severity: SeverityInfo, Message: "tools.semgrep.enabled == true",
					FilePath: ".qsdev-policy.yaml",
				},
				{
					Category: CategoryCustomConformance, Name: "no critical", Status: StatusFail,
					Severity: SeverityHigh, Message: "dependencies.totals.critical == 0: actual 2",
					FilePath: ".qsdev-policy.yaml",
					Remediation: `Bring the project in line with requirement "no critical", ` +
						"or change it in .qsdev-policy.yaml",
				},
			},
			failMedium: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CheckCustomConformance(CheckContext{CustomConformance: tt.custom})
			if len(got) != len(tt.want) {
				t.Fatalf("results = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i].Category != tt.want[i].Category || got[i].Name != tt.want[i].Name ||
					got[i].Status != tt.want[i].Status || got[i].Severity != tt.want[i].Severity ||
					got[i].Message != tt.want[i].Message || got[i].FilePath != tt.want[i].FilePath ||
					got[i].Remediation != tt.want[i].Remediation {
					t.Errorf("result %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
			if fail := ShouldFail(got, AuditLevelMedium); fail != tt.failMedium {
				t.Errorf("ShouldFail(medium) = %v, want %v", fail, tt.failMedium)
			}
		})
	}
}

// TestRunAllChecks_IncludesCustomConformance: the runner reports the custom
// policy's requirements, so a failing one reaches the exit code.
func TestRunAllChecks_IncludesCustomConformance(t *testing.T) {
	t.Parallel()
	report := RunAllChecks(CheckContext{
		ProjectRoot: t.TempDir(),
		CustomConformance: &CustomConformance{
			PolicyFile:   ".qsdev-policy.yaml",
			Requirements: []PolicyRequirement{{Name: "strict", Reason: "score.total >= 90: actual 50"}},
		},
	})
	for _, r := range report.Checks {
		if r.Category == CategoryCustomConformance && r.Name == "strict" && r.Status == StatusFail {
			return
		}
	}
	t.Errorf("custom requirement missing from report: %+v", report.Checks)
}

func TestCategoryCustomConformance_DisplayName(t *testing.T) {
	t.Parallel()
	if got := categoryDisplayName(CategoryCustomConformance); got != "Custom Conformance" {
		t.Errorf("categoryDisplayName = %q, want %q", got, "Custom Conformance")
	}
}
