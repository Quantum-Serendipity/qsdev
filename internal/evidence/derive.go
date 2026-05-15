package evidence

import (
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// DeriveControlMapping determines the status of a control mapping based on
// the control definition and the current state of defense layers.
//
// Status derivation rules:
//   - If NotApplicableReason is set and no layers are mapped: not-applicable.
//   - All primary layers enabled: addressed.
//   - All primary layers disabled: not-addressed.
//   - Mix of primary layer statuses: partial.
//   - No primary layers but supporting layers present: partial if any enabled,
//     not-addressed if all disabled.
//   - No layers mapped and no NotApplicableReason: not-addressed.
func DeriveControlMapping(def ControlDefinition, layers []posture.DefenseLayer) ControlMapping {
	cm := ControlMapping{
		ControlID:   def.ID,
		ControlName: def.Name,
		ControlDesc: def.Desc,
		Category:    def.Category,
		GdevLayers:  make([]LayerEvidence, 0, len(def.Layers)),
		Artifacts:   []EvidenceArtifact{},
	}

	// If no layers are mapped, determine status from NotApplicableReason.
	if len(def.Layers) == 0 {
		if def.NotApplicableReason != "" {
			cm.Status = StatusNotApplicable
			cm.Notes = def.NotApplicableReason
		} else {
			cm.Status = StatusNotAddressed
			cm.Notes = "No qsdev defense layers are directly mapped to this control. Consider artifact-based evidence from external monitoring or logging systems."
		}
		return cm
	}

	// Build a layer lookup from the posture report's defense layers.
	layerLookup := make(map[string]posture.DefenseLayer, len(layers))
	for _, l := range layers {
		layerLookup[l.Name] = l
	}

	// Evaluate each mapped layer.
	var primaryCount, primaryEnabled, primaryPartial, primaryDisabled int

	for _, mapping := range def.Layers {
		postureLayer, found := layerLookup[mapping.LayerName]

		le := LayerEvidence{
			LayerName:   mapping.LayerName,
			Relevance:   mapping.Relevance,
			Description: mapping.Description,
		}

		if !found {
			le.Status = string(posture.LayerDisabled)
		} else {
			le.Status = string(postureLayer.Status)
		}

		cm.GdevLayers = append(cm.GdevLayers, le)

		if mapping.Relevance == "primary" {
			primaryCount++
			status := posture.LayerDisabled
			if found {
				status = postureLayer.Status
			}
			switch status {
			case posture.LayerEnabled:
				primaryEnabled++
			case posture.LayerPartial:
				primaryPartial++
			case posture.LayerDisabled:
				primaryDisabled++
			case posture.LayerNotApplicable:
				// N/A primary layers don't count toward addressed/not-addressed.
				primaryCount--
			}
		}
	}

	// Derive status from primary layers.
	if primaryCount == 0 {
		// No primary layers — check supporting layers.
		hasAnyEnabled := false
		for _, le := range cm.GdevLayers {
			if le.Status == string(posture.LayerEnabled) || le.Status == string(posture.LayerPartial) {
				hasAnyEnabled = true
				break
			}
		}
		if hasAnyEnabled {
			cm.Status = StatusPartial
		} else {
			cm.Status = StatusNotAddressed
		}
	} else if primaryEnabled == primaryCount {
		cm.Status = StatusAddressed
	} else if primaryDisabled == primaryCount {
		cm.Status = StatusNotAddressed
	} else {
		cm.Status = StatusPartial
	}

	return cm
}
