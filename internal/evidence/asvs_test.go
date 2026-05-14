package evidence

import (
	"testing"
)

func TestASVSFramework_HasCorrectMetadata(t *testing.T) {
	fw := ASVSFramework()
	if fw.ID != "asvs" {
		t.Errorf("ID = %q, want %q", fw.ID, "asvs")
	}
	if fw.Name != "OWASP ASVS" {
		t.Errorf("Name = %q, want %q", fw.Name, "OWASP ASVS")
	}
}

func TestASVSFramework_Has6Controls(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()
	if len(controls) != 6 {
		t.Fatalf("expected 6 controls, got %d", len(controls))
	}
}

func TestASVSFramework_ControlIDs(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	expectedIDs := []string{
		"10.3.1", "10.3.2", "10.3.3",
		"14.2.1", "14.2.2", "1.14.1",
	}

	if len(controls) != len(expectedIDs) {
		t.Fatalf("expected %d controls, got %d", len(expectedIDs), len(controls))
	}

	for i, expected := range expectedIDs {
		if controls[i].ID != expected {
			t.Errorf("control[%d].ID = %q, want %q", i, controls[i].ID, expected)
		}
	}
}

func TestASVSFramework_NoDuplicateIDs(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()
	seen := make(map[string]bool)
	for _, c := range controls {
		if seen[c.ID] {
			t.Errorf("duplicate control ID: %q", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestASVSFramework_1033_AlwaysNA(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	var found bool
	for _, c := range controls {
		if c.ID == "10.3.3" {
			found = true
			if len(c.Layers) != 0 {
				t.Errorf("10.3.3 should have no layers, got %d", len(c.Layers))
			}
			if c.NotApplicableReason == "" {
				t.Error("10.3.3 should have a NotApplicableReason")
			}
		}
	}
	if !found {
		t.Error("10.3.3 not found in ASVS controls")
	}
}

func TestASVSFramework_ValidLayerNames(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	validLayers := map[string]bool{
		"age-gating":              true,
		"install-script-blocking": true,
		"lock-file-enforcement":   true,
		"vulnerability-scanning":  true,
		"pretooluse-hooks":        true,
		"nix-hardening":           true,
		"sast":                    true,
		"secrets-scanning":        true,
		"container-security":      true,
		"license-compliance":      true,
	}

	for _, c := range controls {
		for _, l := range c.Layers {
			if !validLayers[l.LayerName] {
				t.Errorf("control %s references invalid layer %q", c.ID, l.LayerName)
			}
			if l.Relevance != "primary" && l.Relevance != "supporting" {
				t.Errorf("control %s layer %s has invalid relevance %q", c.ID, l.LayerName, l.Relevance)
			}
			if l.Description == "" {
				t.Errorf("control %s layer %s has empty description", c.ID, l.LayerName)
			}
		}
	}
}

func TestASVSFramework_AllControlsHaveRequiredFields(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()
	for _, c := range controls {
		if c.ID == "" {
			t.Error("control has empty ID")
		}
		if c.Name == "" {
			t.Errorf("control %s has empty Name", c.ID)
		}
		if c.Desc == "" {
			t.Errorf("control %s has empty Desc", c.ID)
		}
		if c.Category == "" {
			t.Errorf("control %s has empty Category", c.ID)
		}
	}
}

func TestASVSFramework_1031_HasAgeGatingAndInstallScriptBlocking(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	for _, c := range controls {
		if c.ID == "10.3.1" {
			layerNames := make(map[string]bool)
			for _, l := range c.Layers {
				layerNames[l.LayerName] = true
			}
			if !layerNames["age-gating"] {
				t.Error("10.3.1 should reference age-gating")
			}
			if !layerNames["install-script-blocking"] {
				t.Error("10.3.1 should reference install-script-blocking")
			}
			return
		}
	}
	t.Error("10.3.1 not found")
}

func TestASVSFramework_1032_HasVulnScanningAndSAST(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	for _, c := range controls {
		if c.ID == "10.3.2" {
			layerNames := make(map[string]bool)
			for _, l := range c.Layers {
				layerNames[l.LayerName] = true
			}
			if !layerNames["vulnerability-scanning"] {
				t.Error("10.3.2 should reference vulnerability-scanning")
			}
			if !layerNames["sast"] {
				t.Error("10.3.2 should reference sast")
			}
			return
		}
	}
	t.Error("10.3.2 not found")
}
