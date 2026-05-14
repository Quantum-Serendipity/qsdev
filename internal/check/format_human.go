package check

import (
	"fmt"
	"io"
)

// categoryOrder defines the display order for categories in human output.
var categoryOrder = []CheckCategory{
	CategoryBinaryCompat,
	CategoryConfigIntegrity,
	CategoryRequiredTools,
	CategoryFileState,
	CategorySecurityHarden,
}

func formatHuman(report *CheckReport, w io.Writer, useColor bool) error {
	// Group results by category.
	byCategory := make(map[CheckCategory][]CheckResult)
	for _, r := range report.Checks {
		byCategory[r.Category] = append(byCategory[r.Category], r)
	}

	for _, cat := range categoryOrder {
		results, ok := byCategory[cat]
		if !ok || len(results) == 0 {
			continue
		}

		// Category header.
		header := categoryDisplayName(cat)
		if useColor {
			fmt.Fprintf(w, "\n%s%s%s\n", colorBold, header, colorReset)
		} else {
			fmt.Fprintf(w, "\n%s\n", header)
		}

		for _, r := range results {
			sym := statusSymbol(r.Status, useColor)
			fmt.Fprintf(w, "  %s %s: %s\n", sym, r.Name, r.Message)

			if r.Status == StatusFail && r.Remediation != "" {
				if useColor {
					fmt.Fprintf(w, "      %s→ %s%s\n", colorDim, r.Remediation, colorReset)
				} else {
					fmt.Fprintf(w, "      -> %s\n", r.Remediation)
				}
			}
		}
	}

	// Summary.
	fmt.Fprintln(w)
	s := report.Summary
	summary := fmt.Sprintf("Summary: %d checks, %d passed, %d failed, %d warnings",
		s.Total, s.Pass, s.Fail, s.Warn)
	if s.Skip > 0 {
		summary += fmt.Sprintf(", %d skipped", s.Skip)
	}

	if useColor {
		if s.Fail > 0 {
			fmt.Fprintf(w, "%s%s%s\n", colorRed, summary, colorReset)
		} else if s.Warn > 0 {
			fmt.Fprintf(w, "%s%s%s\n", colorYellow, summary, colorReset)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", colorGreen, summary, colorReset)
		}
	} else {
		fmt.Fprintln(w, summary)
	}

	return nil
}
