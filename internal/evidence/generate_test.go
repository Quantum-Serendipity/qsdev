package evidence

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

func TestGenerate_NilFramework_Error(t *testing.T) {
	report := &posture.PostureReport{}
	_, err := Generate(nil, report, "test")
	if err == nil {
		t.Fatal("expected error for nil framework")
	}
}

func TestGenerate_NilReport_Error(t *testing.T) {
	fw := SOC2Framework()
	_, err := Generate(&fw, nil, "test")
	if err == nil {
		t.Fatal("expected error for nil report")
	}
}

func TestGenerate_SOC2_ProducesValidReport(t *testing.T) {
	fw := SOC2Framework()
	pr := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "pretooluse-hooks", Status: posture.LayerEnabled, Weight: posture.WeightCritical},
				{Name: "nix-hardening", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "install-script-blocking", Status: posture.LayerEnabled, Weight: posture.WeightHigh},
				{Name: "age-gating", Status: posture.LayerEnabled, Weight: posture.WeightHigh},
				{Name: "secrets-scanning", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "vulnerability-scanning", Status: posture.LayerEnabled, Weight: posture.WeightHigh},
				{Name: "lock-file-enforcement", Status: posture.LayerEnabled, Weight: posture.WeightHigh},
			},
		},
	}

	result, err := Generate(&fw, pr, "my-project")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.ProjectName != "my-project" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "my-project")
	}
	if result.Framework != "SOC 2 Type II" {
		t.Errorf("Framework = %q, want %q", result.Framework, "SOC 2 Type II")
	}
	if result.Disclaimer == "" {
		t.Error("Disclaimer should not be empty")
	}
	if result.SchemaVersion != posture.SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", result.SchemaVersion, posture.SchemaVersion)
	}
	if len(result.Controls) != 8 {
		t.Errorf("expected 8 controls, got %d", len(result.Controls))
	}
	if result.Summary.TotalControls != 8 {
		t.Errorf("TotalControls = %d, want 8", result.Summary.TotalControls)
	}
	if result.Posture != pr {
		t.Error("Posture should reference the input report")
	}
}

func TestGenerate_AllLayersDisabled(t *testing.T) {
	fw := SOC2Framework()
	pr := &posture.PostureReport{
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "pretooluse-hooks", Status: posture.LayerDisabled},
				{Name: "nix-hardening", Status: posture.LayerDisabled},
				{Name: "install-script-blocking", Status: posture.LayerDisabled},
				{Name: "age-gating", Status: posture.LayerDisabled},
				{Name: "secrets-scanning", Status: posture.LayerDisabled},
				{Name: "sast", Status: posture.LayerDisabled},
				{Name: "vulnerability-scanning", Status: posture.LayerDisabled},
				{Name: "lock-file-enforcement", Status: posture.LayerDisabled},
			},
		},
	}

	result, err := Generate(&fw, pr, "empty-project")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// All controls with primary layers should be not-addressed.
	for _, cm := range result.Controls {
		hasPrimary := false
		for _, l := range cm.GdevLayers {
			if l.Relevance == "primary" {
				hasPrimary = true
				break
			}
		}
		if hasPrimary && cm.Status != StatusNotAddressed {
			t.Errorf("control %s: expected not-addressed when all layers disabled, got %q",
				cm.ControlID, cm.Status)
		}
	}
}

func TestGenerate_SummaryCalculation(t *testing.T) {
	fw := Framework{
		ID:      "test-summary",
		Name:    "Test Summary",
		Version: "1.0",
		Controls: func() []ControlDefinition {
			return []ControlDefinition{
				{
					ID: "A", Name: "Addressed", Desc: "d", Category: "c",
					Layers: []LayerMapping{
						{LayerName: "sast", Relevance: "primary", Description: "d"},
					},
				},
				{
					ID: "B", Name: "Not Addressed", Desc: "d", Category: "c",
					Layers: []LayerMapping{
						{LayerName: "vulnerability-scanning", Relevance: "primary", Description: "d"},
					},
				},
				{
					ID: "C", Name: "N/A", Desc: "d", Category: "c",
					NotApplicableReason: "outside scope",
				},
			}
		},
	}

	pr := &posture.PostureReport{
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "sast", Status: posture.LayerEnabled},
				{Name: "vulnerability-scanning", Status: posture.LayerDisabled},
			},
		},
	}

	result, err := Generate(&fw, pr, "test")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	s := result.Summary
	if s.TotalControls != 3 {
		t.Errorf("TotalControls = %d, want 3", s.TotalControls)
	}
	if s.AddressedFully != 1 {
		t.Errorf("AddressedFully = %d, want 1", s.AddressedFully)
	}
	if s.NotAddressed != 1 {
		t.Errorf("NotAddressed = %d, want 1", s.NotAddressed)
	}
	if s.NotApplicable != 1 {
		t.Errorf("NotApplicable = %d, want 1", s.NotApplicable)
	}
	// Coverage = 1/(3-1) * 100 = 50%
	if s.CoveragePercent < 49.9 || s.CoveragePercent > 50.1 {
		t.Errorf("CoveragePercent = %.1f, want ~50.0", s.CoveragePercent)
	}
}

func TestGenerate_AllNA_Coverage100(t *testing.T) {
	fw := Framework{
		ID:      "all-na",
		Name:    "All N/A",
		Version: "1.0",
		Controls: func() []ControlDefinition {
			return []ControlDefinition{
				{ID: "1", Name: "N/A 1", Desc: "d", Category: "c", NotApplicableReason: "n/a"},
				{ID: "2", Name: "N/A 2", Desc: "d", Category: "c", NotApplicableReason: "n/a"},
			}
		},
	}

	pr := &posture.PostureReport{}

	result, err := Generate(&fw, pr, "test")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Summary.CoveragePercent != 100.0 {
		t.Errorf("CoveragePercent = %.1f, want 100.0 when all N/A", result.Summary.CoveragePercent)
	}
}
