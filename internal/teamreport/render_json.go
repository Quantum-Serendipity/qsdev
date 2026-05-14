package teamreport

import (
	"encoding/json"
	"fmt"
)

// RenderJSON serializes a TeamReport to indented JSON with a trailing newline.
// Nil slices are normalized to empty slices for consistent output.
func RenderJSON(report *TeamReport) ([]byte, error) {
	// Create a copy to avoid mutating the original.
	r := *report

	// Normalize nil slices to empty slices.
	if r.Projects == nil {
		r.Projects = []ProjectSummary{}
	}
	if r.Alerts == nil {
		r.Alerts = []PostureAlert{}
	}
	if r.Trends == nil {
		r.Trends = []ProjectTrend{}
	}

	data, err := json.MarshalIndent(&r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling team report: %w", err)
	}

	data = append(data, '\n')
	return data, nil
}
