package posture

import (
	"encoding/json"
)

// RenderJSON serializes a PostureReport to indented JSON with a trailing newline.
// It forces SchemaVersion to the current SchemaVersion constant and replaces
// nil slices with empty slices to ensure consistent JSON output.
func RenderJSON(report *PostureReport) ([]byte, error) {
	// Create a shallow copy so we don't mutate the caller's report.
	r := *report
	r.SchemaVersion = SchemaVersion

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
func normalizeSlices(r *PostureReport) {
	if r.Tools == nil {
		r.Tools = []ToolStatus{}
	}
	if r.Ecosystems == nil {
		r.Ecosystems = []EcosystemStatus{}
	}
	if r.Defense.Layers == nil {
		r.Defense.Layers = []DefenseLayer{}
	}
	if r.Config.Files == nil {
		r.Config.Files = []ConfigFileInfo{}
	}
	if r.Dependencies.Ecosystems == nil {
		r.Dependencies.Ecosystems = []EcosystemStatus{}
	}
	if r.Drift.Categories == nil {
		r.Drift.Categories = []DriftCategory{}
	}
	if r.Drift.BySeverity == nil {
		r.Drift.BySeverity = make(map[DriftSeverity]int)
	}
	if r.Conformance.Baseline.Checks == nil {
		r.Conformance.Baseline.Checks = []ConformanceCheck{}
	}
	if r.Conformance.Enhanced.Checks == nil {
		r.Conformance.Enhanced.Checks = []ConformanceCheck{}
	}

	// Normalize nested slices in categories.
	for i := range r.Drift.Categories {
		if r.Drift.Categories[i].Findings == nil {
			r.Drift.Categories[i].Findings = []DriftFinding{}
		}
	}

	// Normalize nested slices in ecosystems.
	// (EcosystemStatus doesn't have sub-slices, but DependencyHealth.Ecosystems might.)
}
