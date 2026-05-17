package render

import (
	"encoding/json"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

// RenderJSON serializes a PostureReport to indented JSON with a trailing newline.
// It forces SchemaVersion to the current SchemaVersion constant and replaces
// nil slices with empty slices to ensure consistent JSON output.
func RenderJSON(report *posture.PostureReport) ([]byte, error) {
	// Create a shallow copy so we don't mutate the caller's report.
	r := *report
	r.SchemaVersion = posture.SchemaVersion

	// Ensure nil slices are replaced with empty slices.
	normalizeSlices(&r)

	data, err := json.MarshalIndent(&r, "", "  ")
	if err != nil {
		return nil, err
	}

	// Trailing newline.
	data = append(data, '\n')
	return data, nil
}

// normalizeSlices replaces nil slices with empty slices to ensure JSON
// serialization produces [] instead of null.
func normalizeSlices(r *posture.PostureReport) {
	if r.Tools == nil {
		r.Tools = []posture.ToolStatus{}
	}
	if r.Ecosystems == nil {
		r.Ecosystems = []posture.EcosystemStatus{}
	}
	if r.Defense.Layers == nil {
		r.Defense.Layers = []posture.DefenseLayer{}
	}
	if r.Config.Files == nil {
		r.Config.Files = []posture.ConfigFileInfo{}
	}
	if r.Dependencies.Ecosystems == nil {
		r.Dependencies.Ecosystems = []posture.EcosystemStatus{}
	}
	if r.Drift.Categories == nil {
		r.Drift.Categories = []drift.Category{}
	}
	if r.Drift.BySeverity == nil {
		r.Drift.BySeverity = make(map[drift.Severity]int)
	}
	if r.Conformance.Baseline.Checks == nil {
		r.Conformance.Baseline.Checks = []posture.ConformanceCheck{}
	}
	if r.Conformance.Enhanced.Checks == nil {
		r.Conformance.Enhanced.Checks = []posture.ConformanceCheck{}
	}

	// Normalize nested slices in categories.
	for i := range r.Drift.Categories {
		if r.Drift.Categories[i].Findings == nil {
			r.Drift.Categories[i].Findings = []drift.Finding{}
		}
	}

	// Normalize nested slices in ecosystems.
	// (EcosystemStatus doesn't have sub-slices, but DependencyHealth.Ecosystems might.)
}
