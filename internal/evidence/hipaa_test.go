package evidence

import (
	"testing"
)

func TestHIPAAFramework_HasCorrectMetadata(t *testing.T) {
	fw := HIPAAFramework()
	if fw.ID != "hipaa" {
		t.Errorf("ID = %q, want %q", fw.ID, "hipaa")
	}
	if fw.Name != "HIPAA Security Rule" {
		t.Errorf("Name = %q, want %q", fw.Name, "HIPAA Security Rule")
	}
}

func TestHIPAAFramework_Has5Controls(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()
	if len(controls) != 5 {
		t.Fatalf("expected 5 controls, got %d", len(controls))
	}
}

func TestHIPAAFramework_ControlIDs(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()

	expectedIDs := []string{
		"164.312(a)(1)", "164.312(b)", "164.312(c)(1)",
		"164.312(d)", "164.312(e)(1)",
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

func TestHIPAAFramework_NoDuplicateIDs(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()
	seen := make(map[string]bool)
	for _, c := range controls {
		if seen[c.ID] {
			t.Errorf("duplicate control ID: %q", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestHIPAAFramework_164312d_AlwaysNA(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()

	var found bool
	for _, c := range controls {
		if c.ID == "164.312(d)" {
			found = true
			if len(c.Layers) != 0 {
				t.Errorf("164.312(d) should have no layers, got %d", len(c.Layers))
			}
			if c.NotApplicableReason == "" {
				t.Error("164.312(d) should have a NotApplicableReason")
			}
		}
	}
	if !found {
		t.Error("164.312(d) not found in HIPAA controls")
	}
}

func TestHIPAAFramework_164312e1_AlwaysNA(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()

	var found bool
	for _, c := range controls {
		if c.ID == "164.312(e)(1)" {
			found = true
			if len(c.Layers) != 0 {
				t.Errorf("164.312(e)(1) should have no layers, got %d", len(c.Layers))
			}
			if c.NotApplicableReason == "" {
				t.Error("164.312(e)(1) should have a NotApplicableReason")
			}
		}
	}
	if !found {
		t.Error("164.312(e)(1) not found in HIPAA controls")
	}
}

func TestHIPAAFramework_ValidLayerNames(t *testing.T) {
	fw := HIPAAFramework()
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
		}
	}
}

func TestHIPAAFramework_AllControlsHaveRequiredFields(t *testing.T) {
	fw := HIPAAFramework()
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

func TestHIPAAFramework_164312a1_HasPrimaryLayers(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()

	for _, c := range controls {
		if c.ID == "164.312(a)(1)" {
			if len(c.Layers) < 1 {
				t.Error("164.312(a)(1) should have at least one layer")
			}
			hasPrimary := false
			for _, l := range c.Layers {
				if l.Relevance == "primary" {
					hasPrimary = true
				}
			}
			if !hasPrimary {
				t.Error("164.312(a)(1) should have at least one primary layer")
			}
			return
		}
	}
	t.Error("164.312(a)(1) not found")
}

func TestHIPAAFramework_164312c1_HasLockFileEnforcement(t *testing.T) {
	fw := HIPAAFramework()
	controls := fw.Controls()

	for _, c := range controls {
		if c.ID == "164.312(c)(1)" {
			if len(c.Layers) != 1 {
				t.Fatalf("164.312(c)(1) expected 1 layer, got %d", len(c.Layers))
			}
			if c.Layers[0].LayerName != "lock-file-enforcement" {
				t.Errorf("expected lock-file-enforcement, got %q", c.Layers[0].LayerName)
			}
			if c.Layers[0].Relevance != "primary" {
				t.Errorf("expected primary relevance, got %q", c.Layers[0].Relevance)
			}
			return
		}
	}
	t.Error("164.312(c)(1) not found")
}
