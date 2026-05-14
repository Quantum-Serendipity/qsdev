package teamreport

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	report := makeTeamReport()

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON.
	var parsed TeamReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.SchemaVersion != "1.0.0" {
		t.Errorf("expected schemaVersion '1.0.0', got %q", parsed.SchemaVersion)
	}

	if parsed.Summary.ProjectCount != 3 {
		t.Errorf("expected 3 projects, got %d", parsed.Summary.ProjectCount)
	}

	if len(parsed.Projects) != 3 {
		t.Errorf("expected 3 project summaries, got %d", len(parsed.Projects))
	}
}

func TestRenderJSONTrailingNewline(t *testing.T) {
	report := makeTeamReport()
	data, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(string(data), "\n") {
		t.Error("expected trailing newline in JSON output")
	}
}

func TestRenderJSONNilSlices(t *testing.T) {
	report := &TeamReport{
		SchemaVersion: "1.0.0",
		Summary:       TeamSummary{},
		// Deliberately nil slices.
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil slices should be serialized as [] not null.
	if strings.Contains(string(data), "null") {
		t.Error("expected nil slices to be serialized as empty arrays, not null")
	}
}
