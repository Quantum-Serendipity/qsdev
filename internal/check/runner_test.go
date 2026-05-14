package check

import (
	"testing"

)

func TestShouldFail_AuditLevelNone(t *testing.T) {
	results := []CheckResult{
		{Status: StatusFail, Severity: SeverityCritical},
	}
	if ShouldFail(results, AuditLevelNone) {
		t.Error("ShouldFail with AuditLevelNone should always return false")
	}
}

func TestShouldFail_AuditLevelCritical_MediumOnly(t *testing.T) {
	results := []CheckResult{
		{Status: StatusFail, Severity: SeverityMedium},
		{Status: StatusFail, Severity: SeverityLow},
	}
	if ShouldFail(results, AuditLevelCritical) {
		t.Error("ShouldFail with AuditLevelCritical should return false for medium-only failures")
	}
}

func TestShouldFail_AuditLevelMedium_MediumFails(t *testing.T) {
	results := []CheckResult{
		{Status: StatusFail, Severity: SeverityMedium},
	}
	if !ShouldFail(results, AuditLevelMedium) {
		t.Error("ShouldFail with AuditLevelMedium should return true for medium failures")
	}
}

func TestShouldFail_AuditLevelMedium_LowOnly(t *testing.T) {
	results := []CheckResult{
		{Status: StatusFail, Severity: SeverityLow},
	}
	if ShouldFail(results, AuditLevelMedium) {
		t.Error("ShouldFail with AuditLevelMedium should return false for low-only failures")
	}
}

func TestShouldFail_PassOnly(t *testing.T) {
	results := []CheckResult{
		{Status: StatusPass, Severity: SeverityCritical},
		{Status: StatusSkip, Severity: SeverityHigh},
	}
	if ShouldFail(results, AuditLevelLow) {
		t.Error("ShouldFail should return false when no checks have StatusFail")
	}
}

func TestBuildReport_TalliesCorrectly(t *testing.T) {
	results := []CheckResult{
		{Status: StatusPass},
		{Status: StatusPass},
		{Status: StatusFail},
		{Status: StatusWarn},
		{Status: StatusSkip},
	}

	report := BuildReport(results, "1.0.0", "test-project")

	if report.Summary.Total != 5 {
		t.Errorf("Total = %d, want 5", report.Summary.Total)
	}
	if report.Summary.Pass != 2 {
		t.Errorf("Pass = %d, want 2", report.Summary.Pass)
	}
	if report.Summary.Fail != 1 {
		t.Errorf("Fail = %d, want 1", report.Summary.Fail)
	}
	if report.Summary.Warn != 1 {
		t.Errorf("Warn = %d, want 1", report.Summary.Warn)
	}
	if report.Summary.Skip != 1 {
		t.Errorf("Skip = %d, want 1", report.Summary.Skip)
	}
	if report.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", report.Version, "1.0.0")
	}
	if report.Project != "test-project" {
		t.Errorf("Project = %q, want %q", report.Project, "test-project")
	}
	if report.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestFailCount(t *testing.T) {
	results := []CheckResult{
		{Status: StatusFail, Severity: SeverityCritical},
		{Status: StatusFail, Severity: SeverityMedium},
		{Status: StatusFail, Severity: SeverityLow},
		{Status: StatusPass, Severity: SeverityCritical},
	}

	if got := FailCount(results, AuditLevelNone); got != 0 {
		t.Errorf("FailCount(AuditLevelNone) = %d, want 0", got)
	}
	if got := FailCount(results, AuditLevelCritical); got != 1 {
		t.Errorf("FailCount(AuditLevelCritical) = %d, want 1", got)
	}
	if got := FailCount(results, AuditLevelMedium); got != 2 {
		t.Errorf("FailCount(AuditLevelMedium) = %d, want 2", got)
	}
	if got := FailCount(results, AuditLevelLow); got != 3 {
		t.Errorf("FailCount(AuditLevelLow) = %d, want 3", got)
	}
}

func TestProjectName_FromPath(t *testing.T) {
	ctx := CheckContext{
		ProjectRoot: "/some/path/myproject",
	}
	if got := projectName(ctx); got != "myproject" {
		t.Errorf("projectName = %q, want %q", got, "myproject")
	}
}

func TestCheckFailedError_Error(t *testing.T) {
	e := &CheckFailedError{FailCount: 3, Level: AuditLevelMedium}
	expected := "3 check(s) failed at audit level medium"
	if got := e.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}
