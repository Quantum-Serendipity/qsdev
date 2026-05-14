package check

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestFormatSARIF_ValidStructure(t *testing.T) {
	report := &CheckReport{
		Version:   "1.0.0",
		Project:   "test",
		Timestamp: "2024-01-01T00:00:00Z",
		Checks: []CheckResult{
			{
				Category: CategoryBinaryCompat,
				Name:     "version_check",
				Status:   StatusFail,
				Severity: SeverityCritical,
				Message:  "version mismatch",
				FilePath: ".gdev.yaml",
			},
			{
				Category: CategoryConfigIntegrity,
				Name:     "config_valid",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "all good",
			},
		},
		Summary: CheckSummary{Total: 2, Pass: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatSARIF(report, &buf); err != nil {
		t.Fatalf("formatSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want %q", parsed.Version, "2.1.0")
	}
	if parsed.Schema == "" {
		t.Error("SARIF $schema should not be empty")
	}
	if len(parsed.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(parsed.Runs))
	}
	if parsed.Runs[0].Tool.Driver.Name != "gdev" {
		t.Errorf("tool name = %q, want %q", parsed.Runs[0].Tool.Driver.Name, "gdev")
	}
	if parsed.Runs[0].Tool.Driver.Version != "1.0.0" {
		t.Errorf("tool version = %q, want %q", parsed.Runs[0].Tool.Driver.Version, "1.0.0")
	}
}

func TestFormatSARIF_OnlyFailedChecks(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{Name: "pass1", Status: StatusPass, Severity: SeverityInfo},
			{Name: "fail1", Status: StatusFail, Severity: SeverityHigh, Message: "bad"},
			{Name: "warn1", Status: StatusWarn, Severity: SeverityMedium},
			{Name: "skip1", Status: StatusSkip, Severity: SeverityInfo},
			{Name: "fail2", Status: StatusFail, Severity: SeverityCritical, Message: "worse"},
		},
		Summary: CheckSummary{Total: 5, Pass: 1, Fail: 2, Warn: 1, Skip: 1},
	}

	var buf bytes.Buffer
	if err := formatSARIF(report, &buf); err != nil {
		t.Fatalf("formatSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results := parsed.Runs[0].Results
	if len(results) != 2 {
		t.Fatalf("expected 2 SARIF results (only failures), got %d", len(results))
	}
}

func TestFormatSARIF_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity CheckSeverity
		expected string
	}{
		{SeverityCritical, "error"},
		{SeverityHigh, "error"},
		{SeverityMedium, "warning"},
		{SeverityLow, "note"},
		{SeverityInfo, "note"},
	}

	for _, tt := range tests {
		got := sarifSeverity(tt.severity)
		if got != tt.expected {
			t.Errorf("sarifSeverity(%s) = %q, want %q", tt.severity, got, tt.expected)
		}
	}
}

func TestFormatSARIF_FilePathAsURI(t *testing.T) {
	report := &CheckReport{
		Version: "1.0.0",
		Project: "test",
		Checks: []CheckResult{
			{
				Name:     "file_check",
				Status:   StatusFail,
				Severity: SeverityMedium,
				Message:  "modified",
				FilePath: ".claude/settings.json",
			},
		},
		Summary: CheckSummary{Total: 1, Fail: 1},
	}

	var buf bytes.Buffer
	if err := formatSARIF(report, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed sarifReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	if len(parsed.Runs[0].Results) != 1 {
		t.Fatal("expected 1 result")
	}
	result := parsed.Runs[0].Results[0]
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result.Locations))
	}
	uri := result.Locations[0].PhysicalLocation.ArtifactLocation.URI
	if uri != ".claude/settings.json" {
		t.Errorf("URI = %q, want %q", uri, ".claude/settings.json")
	}
}
