package evidence

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

func TestDeriveControlMapping_AllPrimaryEnabled_Addressed(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-1",
		Name:     "Test Control",
		Desc:     "A test control",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "sast", Relevance: "primary", Description: "SAST checks"},
			{LayerName: "secrets-scanning", Relevance: "primary", Description: "Secrets scanning"},
			{LayerName: "nix-hardening", Relevance: "supporting", Description: "Nix hardening"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
		{Name: "secrets-scanning", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
		{Name: "nix-hardening", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusAddressed {
		t.Errorf("expected %q, got %q", StatusAddressed, cm.Status)
	}
	if cm.ControlID != "TEST-1" {
		t.Errorf("expected ControlID %q, got %q", "TEST-1", cm.ControlID)
	}
	if len(cm.GdevLayers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(cm.GdevLayers))
	}
}

func TestDeriveControlMapping_AllPrimaryDisabled_NotAddressed(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-2",
		Name:     "Test Control 2",
		Desc:     "A test control",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "sast", Relevance: "primary", Description: "SAST checks"},
			{LayerName: "vulnerability-scanning", Relevance: "primary", Description: "Vuln scan"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "sast", Status: posture.LayerDisabled, Weight: posture.WeightMedium},
		{Name: "vulnerability-scanning", Status: posture.LayerDisabled, Weight: posture.WeightHigh},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusNotAddressed {
		t.Errorf("expected %q, got %q", StatusNotAddressed, cm.Status)
	}
}

func TestDeriveControlMapping_MixedPrimary_Partial(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-3",
		Name:     "Test Control 3",
		Desc:     "A test control",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "sast", Relevance: "primary", Description: "SAST checks"},
			{LayerName: "vulnerability-scanning", Relevance: "primary", Description: "Vuln scan"},
			{LayerName: "nix-hardening", Relevance: "supporting", Description: "Supporting"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
		{Name: "vulnerability-scanning", Status: posture.LayerDisabled, Weight: posture.WeightHigh},
		{Name: "nix-hardening", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusPartial {
		t.Errorf("expected %q, got %q", StatusPartial, cm.Status)
	}
}

func TestDeriveControlMapping_NotApplicable(t *testing.T) {
	def := ControlDefinition{
		ID:                  "TEST-NA",
		Name:                "Not Applicable Control",
		Desc:                "This control is N/A",
		Category:            "Test",
		Layers:              []LayerMapping{},
		NotApplicableReason: "This is outside the tool's scope",
	}

	layers := []posture.DefenseLayer{
		{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusNotApplicable {
		t.Errorf("expected %q, got %q", StatusNotApplicable, cm.Status)
	}
	if cm.Notes != "This is outside the tool's scope" {
		t.Errorf("Notes should contain NotApplicableReason, got %q", cm.Notes)
	}
}

func TestDeriveControlMapping_NoLayers_NoReason_NotAddressed(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-EMPTY",
		Name:     "Empty Control",
		Desc:     "No layers, no reason",
		Category: "Test",
		Layers:   []LayerMapping{},
	}

	layers := []posture.DefenseLayer{}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusNotAddressed {
		t.Errorf("expected %q, got %q", StatusNotAddressed, cm.Status)
	}
	if cm.Notes == "" {
		t.Error("expected Notes to contain guidance for unmapped controls")
	}
}

func TestDeriveControlMapping_PartialPrimaryLayer_Partial(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-PARTIAL",
		Name:     "Partial Layer Control",
		Desc:     "Has a partial primary layer",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "secrets-scanning", Relevance: "primary", Description: "Secrets"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "secrets-scanning", Status: posture.LayerPartial, Weight: posture.WeightMedium, Score: 5},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusPartial {
		t.Errorf("expected %q, got %q", StatusPartial, cm.Status)
	}
}

func TestDeriveControlMapping_OnlySupportingLayersEnabled_Partial(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-SUPPORT",
		Name:     "Supporting Only Control",
		Desc:     "Has only supporting layers",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "nix-hardening", Relevance: "supporting", Description: "Supporting layer"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "nix-hardening", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusPartial {
		t.Errorf("expected %q for supporting-only enabled, got %q", StatusPartial, cm.Status)
	}
}

func TestDeriveControlMapping_OnlySupportingLayersDisabled_NotAddressed(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-SUPPORT-OFF",
		Name:     "Supporting Only Disabled",
		Desc:     "Supporting layers all disabled",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "nix-hardening", Relevance: "supporting", Description: "Supporting layer"},
		},
	}

	layers := []posture.DefenseLayer{
		{Name: "nix-hardening", Status: posture.LayerDisabled, Weight: posture.WeightMedium},
	}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusNotAddressed {
		t.Errorf("expected %q for supporting-only disabled, got %q", StatusNotAddressed, cm.Status)
	}
}

func TestDeriveControlMapping_LayerNotInReport_TreatedAsDisabled(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-MISSING",
		Name:     "Missing Layer",
		Desc:     "References a layer not in the report",
		Category: "Test",
		Layers: []LayerMapping{
			{LayerName: "sast", Relevance: "primary", Description: "Not in report"},
		},
	}

	// Empty layers — sast is not present.
	layers := []posture.DefenseLayer{}

	cm := DeriveControlMapping(def, layers)
	if cm.Status != StatusNotAddressed {
		t.Errorf("expected %q when layer is missing from report, got %q", StatusNotAddressed, cm.Status)
	}
	if len(cm.GdevLayers) != 1 {
		t.Fatalf("expected 1 layer evidence, got %d", len(cm.GdevLayers))
	}
	if cm.GdevLayers[0].Status != string(posture.LayerDisabled) {
		t.Errorf("missing layer should show as disabled, got %q", cm.GdevLayers[0].Status)
	}
}

func TestDeriveControlMapping_ArtifactsInitialized(t *testing.T) {
	def := ControlDefinition{
		ID:       "TEST-ART",
		Name:     "Artifacts Test",
		Desc:     "Test",
		Category: "Test",
		Layers:   []LayerMapping{},
	}

	cm := DeriveControlMapping(def, nil)
	if cm.Artifacts == nil {
		t.Error("Artifacts should be initialized to empty slice, not nil")
	}
	if len(cm.Artifacts) != 0 {
		t.Errorf("Artifacts should be empty, got %d", len(cm.Artifacts))
	}
}

func TestDeriveControlMapping_CoverageCalculation(t *testing.T) {
	// Build a framework with known controls and test the summary calculation
	// through the Generate function.
	fw := Framework{
		ID:      "test-coverage",
		Name:    "Test Coverage",
		Version: "1.0",
		Controls: func() []ControlDefinition {
			return []ControlDefinition{
				{
					ID: "C1", Name: "Addressed", Desc: "d", Category: "c",
					Layers: []LayerMapping{
						{LayerName: "sast", Relevance: "primary", Description: "d"},
					},
				},
				{
					ID: "C2", Name: "Not Addressed", Desc: "d", Category: "c",
					Layers: []LayerMapping{
						{LayerName: "vulnerability-scanning", Relevance: "primary", Description: "d"},
					},
				},
				{
					ID: "C3", Name: "N/A", Desc: "d", Category: "c",
					NotApplicableReason: "not applicable",
				},
				{
					ID: "C4", Name: "Partial", Desc: "d", Category: "c",
					Layers: []LayerMapping{
						{LayerName: "sast", Relevance: "primary", Description: "d"},
						{LayerName: "vulnerability-scanning", Relevance: "primary", Description: "d"},
					},
				},
			}
		},
	}

	report := &posture.PostureReport{
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "vulnerability-scanning", Status: posture.LayerDisabled, Weight: posture.WeightHigh},
			},
		},
	}

	evidenceReport, err := Generate(&fw, report, "test-project")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	s := evidenceReport.Summary
	if s.TotalControls != 4 {
		t.Errorf("TotalControls = %d, want 4", s.TotalControls)
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
	if s.AddressedPartial != 1 {
		t.Errorf("AddressedPartial = %d, want 1", s.AddressedPartial)
	}

	// Coverage = (1 + 0.5*1) / (4-1) * 100 = 1.5/3 * 100 = 50.0
	if s.CoveragePercent < 49.9 || s.CoveragePercent > 50.1 {
		t.Errorf("CoveragePercent = %.1f, want ~50.0", s.CoveragePercent)
	}
}
