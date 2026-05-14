package check

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitGitHubAnnotations_FailedChecks(t *testing.T) {
	results := []CheckResult{
		{
			Name:     "fail_with_file",
			Status:   StatusFail,
			Severity: SeverityHigh,
			Message:  "something broke",
			FilePath: "path/to/file.yaml",
		},
		{
			Name:     "fail_without_file",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Message:  "something else broke",
		},
		{
			Name:     "passing_check",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "ok",
		},
		{
			Name:     "warn_check",
			Status:   StatusWarn,
			Severity: SeverityMedium,
			Message:  "might be bad",
			FilePath: "warn.txt",
		},
	}

	var buf bytes.Buffer
	EmitGitHubAnnotations(results, &buf)

	output := buf.String()

	if !strings.Contains(output, "::error file=path/to/file.yaml::fail_with_file: something broke") {
		t.Errorf("expected error annotation with file; got:\n%s", output)
	}
	if !strings.Contains(output, "::error::fail_without_file: something else broke") {
		t.Errorf("expected error annotation without file; got:\n%s", output)
	}
	if strings.Contains(output, "passing_check") {
		t.Error("passing checks should not produce annotations")
	}
	if !strings.Contains(output, "::warning file=warn.txt::warn_check: might be bad") {
		t.Errorf("expected warning annotation; got:\n%s", output)
	}
}

func TestEmitGitHubAnnotations_Empty(t *testing.T) {
	var buf bytes.Buffer
	EmitGitHubAnnotations(nil, &buf)

	if buf.Len() != 0 {
		t.Errorf("expected no output for empty results; got %q", buf.String())
	}
}
