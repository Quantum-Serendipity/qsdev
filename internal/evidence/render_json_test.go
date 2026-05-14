package evidence

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRenderJSON_NilReport_Error(t *testing.T) {
	var buf bytes.Buffer
	err := RenderJSON(nil, &buf)
	if err == nil {
		t.Error("expected error for nil report")
	}
}

func TestRenderJSON_ProducesValidJSON(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderJSON(report, &buf); err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var decoded EvidenceReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if decoded.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q, want %q", decoded.ProjectName, "test-project")
	}
	if decoded.Framework != "Test Framework" {
		t.Errorf("Framework = %q, want %q", decoded.Framework, "Test Framework")
	}
	if len(decoded.Controls) != 2 {
		t.Errorf("expected 2 controls, got %d", len(decoded.Controls))
	}
}

func TestRenderJSON_ContainsAllFields(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderJSON(report, &buf); err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	output := buf.String()
	requiredFields := []string{
		"schemaVersion", "generatedAt", "gdevVersion",
		"projectName", "framework", "frameworkVersion",
		"disclaimer", "summary", "controls", "posture",
	}
	for _, field := range requiredFields {
		if !bytes.Contains(buf.Bytes(), []byte(field)) {
			t.Errorf("output missing field %q. output:\n%s", field, output)
		}
	}
}
