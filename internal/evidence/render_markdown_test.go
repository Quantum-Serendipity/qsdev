package evidence

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
)

func TestRenderMarkdown_ContainsFrameworkName(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Test Framework") {
		t.Error("output should contain the framework name")
	}
}

func TestRenderMarkdown_ContainsProjectName(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "test-project") {
		t.Error("output should contain the project name")
	}
}

func TestRenderMarkdown_ContainsDisclaimer(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "DISCLAIMER") {
		t.Error("output should contain the disclaimer")
	}
}

func TestRenderMarkdown_ContainsSummaryTable(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "## Summary") {
		t.Error("output should contain Summary section")
	}
	if !strings.Contains(output, "Total Controls") {
		t.Error("output should contain Total Controls in summary")
	}
	if !strings.Contains(output, "Coverage") {
		t.Error("output should contain Coverage in summary")
	}
}

func TestRenderMarkdown_ContainsControlMappingsTable(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "## Control Mappings") {
		t.Error("output should contain Control Mappings section")
	}
	if !strings.Contains(output, "Control ID") {
		t.Error("output should contain Control ID column header")
	}
}

func TestRenderMarkdown_ContainsControlDetails(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "## Control Details") {
		t.Error("output should contain Control Details section")
	}
	if !strings.Contains(output, "TEST-1") {
		t.Error("output should contain control ID TEST-1")
	}
}

func TestRenderMarkdown_ContainsPostureReportJSON(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Full Posture Report") {
		t.Error("output should contain Full Posture Report section")
	}
	if !strings.Contains(output, "<details>") {
		t.Error("output should contain collapsible details element")
	}
	if !strings.Contains(output, "schemaVersion") {
		t.Error("output should contain PostureReport JSON with schemaVersion")
	}
}

func TestRenderMarkdown_NilReport_ReturnsError(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(nil, &buf)
	if err == nil {
		t.Error("expected error for nil report")
	}
}

func TestRenderMarkdown_LayerTable(t *testing.T) {
	report := testEvidenceReport()
	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Defense Layers") {
		t.Error("output should contain Defense Layers section for controls with layers")
	}
	if !strings.Contains(output, "sast") {
		t.Error("output should contain layer name 'sast'")
	}
}

func TestRenderMarkdown_NotesRendered(t *testing.T) {
	report := testEvidenceReport()
	// Add a control with notes.
	report.Controls = append(report.Controls, ControlMapping{
		ControlID:   "TEST-NA",
		ControlName: "N/A Control",
		ControlDesc: "Not applicable",
		Category:    "Test",
		Status:      StatusNotApplicable,
		GdevLayers:  []LayerEvidence{},
		Artifacts:   []EvidenceArtifact{},
		Notes:       "This is a test note for auditors",
	})

	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "This is a test note for auditors") {
		t.Error("output should contain the notes text")
	}
}

func TestRenderMarkdown_StatusLabels(t *testing.T) {
	report := testEvidenceReport()
	report.Controls = []ControlMapping{
		{ControlID: "A", ControlName: "Addressed", ControlDesc: "d", Category: "c",
			Status: StatusAddressed, GdevLayers: []LayerEvidence{}, Artifacts: []EvidenceArtifact{}},
		{ControlID: "P", ControlName: "Partial", ControlDesc: "d", Category: "c",
			Status: StatusPartial, GdevLayers: []LayerEvidence{}, Artifacts: []EvidenceArtifact{}},
		{ControlID: "N", ControlName: "Not Addressed", ControlDesc: "d", Category: "c",
			Status: StatusNotAddressed, GdevLayers: []LayerEvidence{}, Artifacts: []EvidenceArtifact{}},
		{ControlID: "NA", ControlName: "N/A", ControlDesc: "d", Category: "c",
			Status: StatusNotApplicable, GdevLayers: []LayerEvidence{}, Artifacts: []EvidenceArtifact{}},
	}

	var buf bytes.Buffer
	if err := RenderMarkdown(report, &buf); err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Addressed") {
		t.Error("output should contain 'Addressed' status")
	}
	if !strings.Contains(output, "Partial") {
		t.Error("output should contain 'Partial' status")
	}
	if !strings.Contains(output, "Not Addressed") {
		t.Error("output should contain 'Not Addressed' status")
	}
	if !strings.Contains(output, "N/A") {
		t.Error("output should contain 'N/A' status")
	}
}

func testEvidenceReport() *EvidenceReport {
	return &EvidenceReport{
		SchemaVersion: "1.0.0",
		GeneratedAt:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		GdevVersion:   "0.1.0",
		ProjectName:   "test-project",
		Framework:     "Test Framework",
		FrameworkVer:  "1.0",
		Disclaimer:    Disclaimer,
		Summary: EvidenceSummary{
			TotalControls:    2,
			AddressedFully:   1,
			AddressedPartial: 0,
			NotAddressed:     1,
			NotApplicable:    0,
			CoveragePercent:  50.0,
		},
		Controls: []ControlMapping{
			{
				ControlID:   "TEST-1",
				ControlName: "Test Control 1",
				ControlDesc: "First test control",
				Category:    "Test Category",
				Status:      StatusAddressed,
				GdevLayers: []LayerEvidence{
					{
						LayerName:   "sast",
						Status:      "enabled",
						Relevance:   "primary",
						Description: "SAST provides code analysis",
					},
				},
				Artifacts: []EvidenceArtifact{},
			},
			{
				ControlID:   "TEST-2",
				ControlName: "Test Control 2",
				ControlDesc: "Second test control",
				Category:    "Test Category",
				Status:      StatusNotAddressed,
				GdevLayers:  []LayerEvidence{},
				Artifacts:   []EvidenceArtifact{},
			},
		},
		Posture: &posture.PostureReport{
			SchemaVersion: "1.0.0",
			GeneratedAt:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			GdevVersion:   "0.1.0",
			ProjectName:   "test-project",
		},
	}
}
