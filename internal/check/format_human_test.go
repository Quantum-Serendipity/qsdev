package check

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatHuman_ContainsCategoryHeaders(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryBinaryCompat, Name: "check1", Status: StatusPass, Severity: SeverityInfo, Message: "ok"},
			{Category: CategoryConfigIntegrity, Name: "check2", Status: StatusPass, Severity: SeverityInfo, Message: "ok"},
		},
		Summary: CheckSummary{Total: 2, Pass: 2},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, false); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Binary Compatibility") {
		t.Error("expected 'Binary Compatibility' header in output")
	}
	if !strings.Contains(output, "Config Integrity") {
		t.Error("expected 'Config Integrity' header in output")
	}
}

func TestFormatHuman_ContainsSummaryLine(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryBinaryCompat, Name: "check1", Status: StatusPass, Severity: SeverityInfo, Message: "ok"},
			{Category: CategoryBinaryCompat, Name: "check2", Status: StatusFail, Severity: SeverityHigh, Message: "bad"},
		},
		Summary: CheckSummary{Total: 2, Pass: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, false); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Summary: 2 checks, 1 passed, 1 failed, 0 warnings") {
		t.Errorf("expected summary line in output; got:\n%s", output)
	}
}

func TestFormatHuman_NoColorMode(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryBinaryCompat, Name: "pass_check", Status: StatusPass, Severity: SeverityInfo, Message: "ok"},
			{Category: CategoryBinaryCompat, Name: "fail_check", Status: StatusFail, Severity: SeverityHigh, Message: "bad", Remediation: "fix it"},
			{Category: CategoryBinaryCompat, Name: "warn_check", Status: StatusWarn, Severity: SeverityMedium, Message: "hmm"},
			{Category: CategoryBinaryCompat, Name: "skip_check", Status: StatusSkip, Severity: SeverityInfo, Message: "n/a"},
		},
		Summary: CheckSummary{Total: 4, Pass: 1, Fail: 1, Warn: 1, Skip: 1},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, false); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// No color mode should use text symbols.
	if !strings.Contains(output, "[PASS]") {
		t.Error("expected [PASS] in non-color output")
	}
	if !strings.Contains(output, "[FAIL]") {
		t.Error("expected [FAIL] in non-color output")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("expected [WARN] in non-color output")
	}
	if !strings.Contains(output, "[SKIP]") {
		t.Error("expected [SKIP] in non-color output")
	}
	// No ANSI codes.
	if strings.Contains(output, "\033[") {
		t.Error("non-color output should not contain ANSI escape codes")
	}
}

func TestFormatHuman_ColorMode(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryBinaryCompat, Name: "pass_check", Status: StatusPass, Severity: SeverityInfo, Message: "ok"},
		},
		Summary: CheckSummary{Total: 1, Pass: 1},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, true); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Color mode should contain ANSI escape codes.
	if !strings.Contains(output, "\033[") {
		t.Error("color output should contain ANSI escape codes")
	}
}

func TestFormatHuman_ShowsRemediation(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Category:    CategoryConfigIntegrity,
				Name:        "fail_check",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     "config error",
				Remediation: "run qsdev init",
			},
		},
		Summary: CheckSummary{Total: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, false); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "run qsdev init") {
		t.Error("expected remediation text in output")
	}
}

func TestFormatHuman_SkippedInSummary(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryFileState, Name: "check", Status: StatusSkip, Severity: SeverityInfo, Message: "n/a"},
		},
		Summary: CheckSummary{Total: 1, Skip: 1},
	}

	var buf bytes.Buffer
	if err := formatHuman(report, &buf, false); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "1 skipped") {
		t.Error("expected '1 skipped' in summary when there are skipped checks")
	}
}

// TestFormatters_RenderEveryCategory guards against a failing result being
// counted in the summary (and exit code) but omitted from the rendered report
// because its category is missing from the display order.
func TestFormatters_RenderEveryCategory(t *testing.T) {
	t.Parallel()

	categories := []CheckCategory{
		CategoryBinaryCompat,
		CategoryConfigIntegrity,
		CategoryRequiredTools,
		CategoryFileState,
		CategorySecurityHarden,
		CategoryDenyConflicts,
		CheckCategory("future_category"),
	}

	formatters := []struct {
		name   string
		format func(*CheckReport, *bytes.Buffer) error
	}{
		{"human", func(r *CheckReport, b *bytes.Buffer) error { return formatHuman(r, b, false) }},
		{"junit", func(r *CheckReport, b *bytes.Buffer) error { return formatJUnit(r, b) }},
	}

	for _, f := range formatters {
		for _, cat := range categories {
			t.Run(f.name+"/"+string(cat), func(t *testing.T) {
				t.Parallel()
				checks := []CheckResult{{
					Category: cat,
					Name:     "result_for_" + string(cat),
					Status:   StatusFail,
					Severity: SeverityHigh,
					Message:  "failure in " + string(cat),
				}}
				report := BuildReport(checks, "1.0.0", "test")

				var buf bytes.Buffer
				if err := f.format(report, &buf); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(buf.String(), "result_for_"+string(cat)) {
					t.Errorf("%s output omits %s result:\n%s", f.name, cat, buf.String())
				}
			})
		}
	}
}

func TestOrderedCategories(t *testing.T) {
	t.Parallel()

	results := []CheckResult{
		{Category: "zeta"},
		{Category: CategoryDenyConflicts},
		{Category: CategoryBinaryCompat},
		{Category: "alpha"},
		{Category: CategoryBinaryCompat},
	}
	got := orderedCategories(results)
	want := []CheckCategory{CategoryBinaryCompat, CategoryDenyConflicts, "zeta", "alpha"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
