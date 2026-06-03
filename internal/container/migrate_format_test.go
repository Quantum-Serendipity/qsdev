package container

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleReport() *MigrationReport {
	return &MigrationReport{
		Timestamp:     "2026-01-01T00:00:00Z",
		ProjectRoot:   "/tmp/test",
		SourceRuntime: RuntimeDocker,
		TargetRuntime: RuntimePodmanRootless,
		ComposeFiles:  []string{"/tmp/test/docker-compose.yml"},
		RuntimeInfo:   &RuntimeInfo{Active: RuntimePodmanRootless, Version: "4.9.3"},
		Issues: []MigrationIssue{
			{
				Category:    CategoryImageName,
				Severity:    SeverityInfo,
				File:        "/tmp/test/docker-compose.yml",
				Service:     "web",
				Description: "unqualified image",
				AutoFixable: true,
			},
			{
				Category:    CategoryPrivileged,
				Severity:    SeverityCritical,
				File:        "/tmp/test/docker-compose.yml",
				Service:     "infra",
				Description: "privileged mode",
				AutoFixable: false,
			},
			{
				Category:    CategoryPrivPorts,
				Severity:    SeverityWarning,
				File:        "/tmp/test/docker-compose.yml",
				Service:     "web",
				Description: "port 80",
				AutoFixable: true,
			},
		},
		Summary: MigrationSummary{
			Total:       3,
			Critical:    1,
			Warning:     1,
			Info:        1,
			AutoFixable: 2,
			ManualOnly:  1,
		},
	}
}

func TestFormatMigrationReport_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	report := sampleReport()

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatJSON, &buf, false); err != nil {
		t.Fatalf("FormatMigrationReport(JSON) error = %v", err)
	}

	var decoded MigrationReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON unmarshal error = %v", err)
	}

	if decoded.Summary.Total != 3 {
		t.Errorf("decoded Summary.Total = %d, want 3", decoded.Summary.Total)
	}
	if decoded.Summary.Critical != 1 {
		t.Errorf("decoded Summary.Critical = %d, want 1", decoded.Summary.Critical)
	}
	if len(decoded.Issues) != 3 {
		t.Errorf("decoded Issues = %d, want 3", len(decoded.Issues))
	}
	if decoded.SourceRuntime != RuntimeDocker {
		t.Errorf("decoded SourceRuntime = %q, want %q", decoded.SourceRuntime, RuntimeDocker)
	}
}

func TestFormatMigrationReport_TextContainsIssues(t *testing.T) {
	t.Parallel()
	report := sampleReport()

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatText, &buf, false); err != nil {
		t.Fatalf("FormatMigrationReport(text) error = %v", err)
	}

	text := buf.String()
	for _, want := range []string{
		"unqualified image",
		"privileged mode",
		"port 80",
		"3 issue(s)",
		"Auto-fixable: 2",
		"Manual: 1",
		"[CRITICAL]",
		"[WARNING]",
		"[INFO]",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q:\n%s", want, text)
		}
	}
}

func TestFormatMigrationReport_ColorCodesPresent(t *testing.T) {
	t.Parallel()
	report := sampleReport()

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatText, &buf, true); err != nil {
		t.Fatalf("FormatMigrationReport(color) error = %v", err)
	}

	text := buf.String()
	// ANSI escape codes should be present.
	if !strings.Contains(text, "\033[31m") {
		t.Error("expected red ANSI code for critical severity")
	}
	if !strings.Contains(text, "\033[33m") {
		t.Error("expected yellow ANSI code for warning severity")
	}
	if !strings.Contains(text, "\033[36m") {
		t.Error("expected cyan ANSI code for info severity")
	}
}

func TestFormatMigrationReport_ColorCodesAbsent(t *testing.T) {
	t.Parallel()
	report := sampleReport()

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatText, &buf, false); err != nil {
		t.Fatalf("FormatMigrationReport(no color) error = %v", err)
	}

	text := buf.String()
	if strings.Contains(text, "\033[") {
		t.Error("found ANSI escape codes when useColor=false")
	}
}

func TestFormatMigrationReport_EmptyReport(t *testing.T) {
	t.Parallel()
	report := &MigrationReport{
		SourceRuntime: RuntimeDocker,
		TargetRuntime: RuntimePodmanRootless,
		RuntimeInfo:   &RuntimeInfo{Active: RuntimeNone},
	}

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatText, &buf, false); err != nil {
		t.Fatalf("FormatMigrationReport(empty) error = %v", err)
	}
	text := buf.String()
	if !strings.Contains(text, "No compose files found") {
		t.Errorf("expected 'No compose files found' in output:\n%s", text)
	}
}

func TestFormatMigrationReport_JSONNilRuntimeInfo(t *testing.T) {
	t.Parallel()
	report := &MigrationReport{
		SourceRuntime: RuntimeDocker,
		TargetRuntime: RuntimePodmanRootless,
		ComposeFiles:  []string{"compose.yml"},
	}

	var buf bytes.Buffer
	if err := FormatMigrationReport(report, FormatJSON, &buf, false); err != nil {
		t.Fatalf("FormatMigrationReport(nil runtime) error = %v", err)
	}

	var decoded MigrationReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if decoded.RuntimeInfo != nil {
		t.Error("expected nil RuntimeInfo in decoded report")
	}
}
