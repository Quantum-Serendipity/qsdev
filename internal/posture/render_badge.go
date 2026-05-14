package posture

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// BadgeJSON represents a shields.io endpoint badge JSON structure.
type BadgeJSON struct {
	SchemaVersion int    `json:"schemaVersion"` // Always 1
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// RenderBadge generates a shields.io-compatible badge JSON for the given variant.
// Supported variants: "score" (default), "conformance", "defense".
func RenderBadge(report *PostureReport, variant string) ([]byte, error) {
	var badge BadgeJSON

	switch variant {
	case "", "score":
		score := int(math.Round(report.Score.Total))
		badge = BadgeJSON{
			SchemaVersion: 1,
			Label:         "security posture",
			Message:       fmt.Sprintf("%d/100 %s", score, report.Score.Grade),
			Color:         scoreColor(report.Score.Total),
		}
	case "conformance":
		msg := "baseline PASS"
		if !report.Conformance.Baseline.Pass {
			msg = "baseline FAIL"
		} else if report.Conformance.Enhanced.Pass {
			msg = "enhanced PASS"
		}
		badge = BadgeJSON{
			SchemaVersion: 1,
			Label:         "conformance",
			Message:       msg,
			Color:         conformanceColor(report),
		}
	case "defense":
		badge = BadgeJSON{
			SchemaVersion: 1,
			Label:         "defense coverage",
			Message:       fmt.Sprintf("%d/%d layers", report.Defense.Enabled, report.Defense.Total),
			Color:         scoreColor(report.Defense.Score),
		}
	default:
		return nil, fmt.Errorf("unknown badge variant: %q (supported: score, conformance, defense)", variant)
	}

	data, err := json.MarshalIndent(badge, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	return data, nil
}

// RenderAllBadges writes all three badge variants to the given output directory.
// Files are named: badge-score.json, badge-conformance.json, badge-defense.json.
func RenderAllBadges(report *PostureReport, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating badge output directory: %w", err)
	}

	variants := []string{"score", "conformance", "defense"}
	for _, v := range variants {
		data, err := RenderBadge(report, v)
		if err != nil {
			return fmt.Errorf("rendering %s badge: %w", v, err)
		}
		path := filepath.Join(outputDir, fmt.Sprintf("badge-%s.json", v))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("writing %s badge: %w", v, err)
		}
	}
	return nil
}

// scoreColor returns a shields.io color string based on a 0-100 score.
func scoreColor(score float64) string {
	rounded := int(math.Round(score))
	switch {
	case rounded >= 90:
		return "brightgreen"
	case rounded >= 80:
		return "green"
	case rounded >= 70:
		return "yellow"
	case rounded >= 60:
		return "orange"
	default:
		return "red"
	}
}

// conformanceColor returns a badge color based on conformance results.
func conformanceColor(report *PostureReport) string {
	if report.Conformance.Enhanced.Pass {
		return "brightgreen"
	}
	if report.Conformance.Baseline.Pass {
		return "green"
	}
	return "red"
}
