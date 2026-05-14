package check

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestFormatJUnit_ValidXML(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Category: CategoryBinaryCompat,
				Name:     "version_ok",
				Status:   StatusPass,
				Severity: SeverityInfo,
			},
			{
				Category: CategoryConfigIntegrity,
				Name:     "config_valid",
				Status:   StatusFail,
				Severity: SeverityHigh,
				Message:  "parse error",
			},
		},
		Summary: CheckSummary{Total: 2, Pass: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatJUnit(report, &buf); err != nil {
		t.Fatalf("formatJUnit error: %v", err)
	}

	output := buf.String()

	// Check XML header.
	if !strings.HasPrefix(output, xml.Header) {
		t.Error("output should start with XML header")
	}

	// Verify it parses as valid XML.
	var parsed junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid XML: %v\n%s", err, output)
	}
}

func TestFormatJUnit_CategoriesAsTestSuites(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Category: CategoryBinaryCompat, Name: "check1", Status: StatusPass},
			{Category: CategoryBinaryCompat, Name: "check2", Status: StatusFail, Severity: SeverityHigh, Message: "bad"},
			{Category: CategoryConfigIntegrity, Name: "check3", Status: StatusPass},
		},
		Summary: CheckSummary{Total: 3, Pass: 2, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatJUnit(report, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	if len(parsed.TestSuites) != 2 {
		t.Fatalf("expected 2 test suites, got %d", len(parsed.TestSuites))
	}

	// First suite should be binary_compatibility with 2 tests.
	s1 := parsed.TestSuites[0]
	if s1.Name != string(CategoryBinaryCompat) {
		t.Errorf("suite[0].Name = %q, want %q", s1.Name, string(CategoryBinaryCompat))
	}
	if s1.Tests != 2 {
		t.Errorf("suite[0].Tests = %d, want 2", s1.Tests)
	}
	if s1.Failures != 1 {
		t.Errorf("suite[0].Failures = %d, want 1", s1.Failures)
	}
}

func TestFormatJUnit_FailuresHaveFailureElement(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Category:    CategoryBinaryCompat,
				Name:        "fail_check",
				Status:      StatusFail,
				Severity:    SeverityCritical,
				Message:     "version mismatch",
				Remediation: "update gdev",
			},
		},
		Summary: CheckSummary{Total: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatJUnit(report, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	tc := parsed.TestSuites[0].Cases[0]
	if tc.Failure == nil {
		t.Fatal("expected <failure> element for failed test case")
	}
	if tc.Failure.Type != string(SeverityCritical) {
		t.Errorf("failure type = %q, want %q", tc.Failure.Type, string(SeverityCritical))
	}
	if tc.Failure.Message != "version mismatch" {
		t.Errorf("failure message = %q, want %q", tc.Failure.Message, "version mismatch")
	}
}

func TestFormatJUnit_SkippedElement(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Category: CategoryFileState,
				Name:     "skip_check",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "no state file",
			},
		},
		Summary: CheckSummary{Total: 1, Skip: 1},
	}

	var buf bytes.Buffer
	if err := formatJUnit(report, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	tc := parsed.TestSuites[0].Cases[0]
	if tc.Skipped == nil {
		t.Fatal("expected <skipped> element for skipped test case")
	}
}

func TestFormatJUnit_ClassnameIsCategory(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "lockfile_go",
				Status:   StatusPass,
			},
		},
		Summary: CheckSummary{Total: 1, Pass: 1},
	}

	var buf bytes.Buffer
	if err := formatJUnit(report, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	tc := parsed.TestSuites[0].Cases[0]
	if tc.ClassName != string(CategorySecurityHarden) {
		t.Errorf("classname = %q, want %q", tc.ClassName, string(CategorySecurityHarden))
	}
}
