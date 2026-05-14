package evidence

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RenderMarkdown writes the EvidenceReport as a Markdown audit document
// to the given writer.
func RenderMarkdown(report *EvidenceReport, w io.Writer) error {
	if report == nil {
		return fmt.Errorf("evidence report must not be nil")
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# Compliance Evidence Report: %s\n\n", report.Framework)
	fmt.Fprintf(&b, "**Project:** %s\n", report.ProjectName)
	fmt.Fprintf(&b, "**Framework:** %s (v%s)\n", report.Framework, report.FrameworkVer)
	fmt.Fprintf(&b, "**Generated:** %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "**gdev Version:** %s\n", report.GdevVersion)
	fmt.Fprintf(&b, "**Schema Version:** %s\n\n", report.SchemaVersion)

	b.WriteString("## Disclaimer\n\n")
	fmt.Fprintf(&b, "> %s\n\n", report.Disclaimer)

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(&b, "| Total Controls | %d |\n", report.Summary.TotalControls)
	fmt.Fprintf(&b, "| Fully Addressed | %d |\n", report.Summary.AddressedFully)
	fmt.Fprintf(&b, "| Partially Addressed | %d |\n", report.Summary.AddressedPartial)
	fmt.Fprintf(&b, "| Not Addressed | %d |\n", report.Summary.NotAddressed)
	fmt.Fprintf(&b, "| Not Applicable | %d |\n", report.Summary.NotApplicable)
	fmt.Fprintf(&b, "| Coverage | %.1f%% |\n\n", report.Summary.CoveragePercent)

	b.WriteString("## Control Mappings\n\n")
	b.WriteString("| Control ID | Control Name | Category | Status |\n")
	b.WriteString("|------------|-------------|----------|--------|\n")
	for _, cm := range report.Controls {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			cm.ControlID, cm.ControlName, cm.Category, statusEmoji(cm.Status))
	}
	b.WriteString("\n")

	b.WriteString("## Control Details\n\n")
	for _, cm := range report.Controls {
		fmt.Fprintf(&b, "### %s: %s\n\n", cm.ControlID, cm.ControlName)
		fmt.Fprintf(&b, "**Category:** %s\n", cm.Category)
		fmt.Fprintf(&b, "**Status:** %s\n\n", statusEmoji(cm.Status))
		fmt.Fprintf(&b, "> %s\n\n", cm.ControlDesc)

		if len(cm.GdevLayers) > 0 {
			b.WriteString("**Defense Layers:**\n\n")
			b.WriteString("| Layer | Relevance | Status | Description |\n")
			b.WriteString("|-------|-----------|--------|-------------|\n")
			for _, le := range cm.GdevLayers {
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
					le.LayerName, le.Relevance, le.Status, le.Description)
			}
			b.WriteString("\n")
		}

		if len(cm.Artifacts) > 0 {
			b.WriteString("**Artifacts:**\n\n")
			for _, a := range cm.Artifacts {
				fmt.Fprintf(&b, "- **%s** (`%s`): %s\n", a.Type, a.Path, a.Description)
			}
			b.WriteString("\n")
		}

		if cm.Notes != "" {
			fmt.Fprintf(&b, "**Notes:** %s\n\n", cm.Notes)
		}

		b.WriteString("---\n\n")
	}

	if report.Posture != nil {
		b.WriteString("## Full Posture Report\n\n")
		b.WriteString("<details>\n")
		b.WriteString("<summary>Click to expand raw PostureReport JSON</summary>\n\n")
		b.WriteString("```json\n")
		postureJSON, err := json.MarshalIndent(report.Posture, "", "  ")
		if err != nil {
			fmt.Fprintf(&b, "Error serializing posture report: %v\n", err)
		} else {
			b.WriteString(string(postureJSON))
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
		b.WriteString("</details>\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func statusEmoji(s ControlStatus) string {
	switch s {
	case StatusAddressed:
		return "Addressed"
	case StatusPartial:
		return "Partial"
	case StatusNotAddressed:
		return "Not Addressed"
	case StatusNotApplicable:
		return "N/A"
	default:
		return string(s)
	}
}
