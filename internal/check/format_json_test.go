package check

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestFormatJSON_ValidOutput(t *testing.T) {
	report := &CheckReport{
		Version:   "1.0.0",
		Project:   "test",
		Timestamp: "2024-01-01T00:00:00Z",
		Checks: []CheckResult{
			{
				Category: CategoryBinaryCompat,
				Name:     "test_check",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "all good",
			},
		},
		Summary: CheckSummary{Total: 1, Pass: 1},
	}

	var buf bytes.Buffer
	err := formatJSON(report, &buf)
	if err != nil {
		t.Fatalf("formatJSON error: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check key fields.
	if parsed["version"] != "1.0.0" {
		t.Errorf("version = %v, want %q", parsed["version"], "1.0.0")
	}
	if parsed["project"] != "test" {
		t.Errorf("project = %v, want %q", parsed["project"], "test")
	}
}

func TestFormatJSON_Roundtrip(t *testing.T) {
	original := &CheckReport{
		Version:   "2.0.0",
		Project:   "roundtrip-test",
		Timestamp: "2024-06-15T12:00:00Z",
		Checks: []CheckResult{
			{
				Category:    CategoryConfigIntegrity,
				Name:        "config_valid",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     "config error",
				Remediation: "fix it",
				FilePath:    ".qsdev.yaml",
			},
			{
				Category: CategorySecurityHarden,
				Name:     "lock_check",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "ok",
			},
		},
		Summary: CheckSummary{Total: 2, Pass: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatJSON(original, &buf); err != nil {
		t.Fatalf("formatJSON error: %v", err)
	}

	var decoded CheckReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version = %q, want %q", decoded.Version, original.Version)
	}
	if decoded.Project != original.Project {
		t.Errorf("Project = %q, want %q", decoded.Project, original.Project)
	}
	if len(decoded.Checks) != len(original.Checks) {
		t.Fatalf("len(Checks) = %d, want %d", len(decoded.Checks), len(original.Checks))
	}
	if decoded.Checks[0].Name != "config_valid" {
		t.Errorf("Checks[0].Name = %q, want %q", decoded.Checks[0].Name, "config_valid")
	}
	if decoded.Checks[0].Remediation != "fix it" {
		t.Errorf("Checks[0].Remediation = %q, want %q", decoded.Checks[0].Remediation, "fix it")
	}
	if decoded.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want 2", decoded.Summary.Total)
	}
}

func TestFormatJSON_TrailingNewline(t *testing.T) {
	report := &CheckReport{
		Version:   "1.0.0",
		Project:   "test",
		Timestamp: "2024-01-01T00:00:00Z",
		Summary:   CheckSummary{},
	}

	var buf bytes.Buffer
	if err := formatJSON(report, &buf); err != nil {
		t.Fatalf("formatJSON error: %v", err)
	}

	data := buf.Bytes()
	if len(data) == 0 {
		t.Fatal("empty output")
	}
	if data[len(data)-1] != '\n' {
		t.Error("output should end with trailing newline")
	}
}
