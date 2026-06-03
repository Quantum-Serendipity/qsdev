package evidence

import (
	"testing"
)

func TestSOC2Framework_HasCorrectMetadata(t *testing.T) {
	fw := SOC2Framework()
	if fw.ID != "soc2" {
		t.Errorf("ID = %q, want %q", fw.ID, "soc2")
	}
	if fw.Name != "SOC 2 Type II" {
		t.Errorf("Name = %q, want %q", fw.Name, "SOC 2 Type II")
	}
}

func TestSOC2Framework_Has8Controls(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()
	if len(controls) != 8 {
		t.Fatalf("expected 8 controls, got %d", len(controls))
	}
}

func TestSOC2Framework_ControlIDs(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()

	expectedIDs := []string{
		"CC6.1", "CC6.6", "CC6.8", "CC7.1", "CC7.2",
		"CC8.1", "CC8.2", "CC8.3",
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

func TestSOC2Framework_NoDuplicateIDs(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()
	seen := make(map[string]bool)
	for _, c := range controls {
		if seen[c.ID] {
			t.Errorf("duplicate control ID: %q", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestSOC2Framework_ValidLayerNames(t *testing.T) {
	fw := SOC2Framework()
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

func TestSOC2Framework_CC61Layers(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()

	var cc61 *ControlDefinition
	for i := range controls {
		if controls[i].ID == "CC6.1" {
			cc61 = &controls[i]
			break
		}
	}
	if cc61 == nil {
		t.Fatal("CC6.1 not found")
		return
	}

	if len(cc61.Layers) != 2 {
		t.Fatalf("CC6.1 expected 2 layers, got %d", len(cc61.Layers))
	}
	if cc61.Layers[0].LayerName != "pretooluse-hooks" || cc61.Layers[0].Relevance != "primary" {
		t.Errorf("CC6.1 first layer: got %s/%s, want pretooluse-hooks/primary",
			cc61.Layers[0].LayerName, cc61.Layers[0].Relevance)
	}
	if cc61.Layers[1].LayerName != "nix-hardening" || cc61.Layers[1].Relevance != "supporting" {
		t.Errorf("CC6.1 second layer: got %s/%s, want nix-hardening/supporting",
			cc61.Layers[1].LayerName, cc61.Layers[1].Relevance)
	}
}

func TestSOC2Framework_CC68HasThreePrimaryLayers(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()

	var cc68 *ControlDefinition
	for i := range controls {
		if controls[i].ID == "CC6.8" {
			cc68 = &controls[i]
			break
		}
	}
	if cc68 == nil {
		t.Fatal("CC6.8 not found")
		return
	}

	primaryCount := 0
	for _, l := range cc68.Layers {
		if l.Relevance == "primary" {
			primaryCount++
		}
	}
	if primaryCount != 3 {
		t.Errorf("CC6.8 expected 3 primary layers, got %d", primaryCount)
	}
}

func TestSOC2Framework_CC72HasNoLayers(t *testing.T) {
	fw := SOC2Framework()
	controls := fw.Controls()

	var cc72 *ControlDefinition
	for i := range controls {
		if controls[i].ID == "CC7.2" {
			cc72 = &controls[i]
			break
		}
	}
	if cc72 == nil {
		t.Fatal("CC7.2 not found")
		return
	}

	if len(cc72.Layers) != 0 {
		t.Errorf("CC7.2 expected 0 layers, got %d", len(cc72.Layers))
	}
}

func TestSOC2Framework_AllControlsHaveRequiredFields(t *testing.T) {
	fw := SOC2Framework()
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
