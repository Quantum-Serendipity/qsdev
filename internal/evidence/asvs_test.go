package evidence

import (
	"strings"
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

func TestASVSFramework_Has5Controls(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()
	if len(controls) != 5 {
		t.Fatalf("expected 5 controls, got %d", len(controls))
	}
}

func TestASVSFramework_ControlIDs(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	expectedIDs := []string{
		"10.2.1", "10.3.2",
		"14.2.1", "14.2.2", "1.2.1",
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

// TestASVSFramework_IDsMatchRequirementText pins each control ID to its
// ASVS 4.0.3 requirement text, guarding against the misattributions fixed in
// F335 ("phone home" is 10.2.1, the low-privilege account is 1.2.1, and
// 10.3.3 is subdomain takeover, which qsdev does not map).
func TestASVSFramework_IDsMatchRequirementText(t *testing.T) {
	want := map[string]string{
		"10.2.1": "unauthorized phone home",
		"10.3.2": "integrity protections",
		"14.2.1": "all components are up to date",
		"14.2.2": "unnecessary features",
		"1.2.1":  "low-privilege operating system accounts",
	}
	for _, c := range ASVSFramework().Controls() {
		phrase, ok := want[c.ID]
		if !ok {
			t.Errorf("unexpected control ID %q", c.ID)
			continue
		}
		if !strings.Contains(c.Desc, phrase) {
			t.Errorf("control %s Desc = %q, want it to contain %q", c.ID, c.Desc, phrase)
		}
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

func TestASVSFramework_1021_HasAgeGatingAndInstallScriptBlocking(t *testing.T) {
	fw := ASVSFramework()
	controls := fw.Controls()

	for _, c := range controls {
		if c.ID == "10.2.1" {
			layerNames := make(map[string]bool)
			for _, l := range c.Layers {
				layerNames[l.LayerName] = true
			}
			if !layerNames["age-gating"] {
				t.Error("10.2.1 should reference age-gating")
			}
			if !layerNames["install-script-blocking"] {
				t.Error("10.2.1 should reference install-script-blocking")
			}
			return
		}
	}
	t.Error("10.2.1 not found")
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
