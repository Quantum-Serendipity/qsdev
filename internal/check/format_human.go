package check

import (
	"fmt"
	"io"
)

// categoryOrder defines the display order for categories in human and JUnit output.
var categoryOrder = []CheckCategory{
	CategoryBinaryCompat,
	CategoryConfigIntegrity,
	CategoryRequiredTools,
	CategoryFileState,
	CategorySecurityHarden,
	CategoryDenyConflicts,
}

// orderedCategories returns the categories present in results: those listed in
// categoryOrder first, in that order, followed by any other category in
// first-seen order. Formatters iterate this rather than categoryOrder directly
// so a category missing from categoryOrder can never silently disappear from
// the report while still counting toward the summary and exit code.
func orderedCategories(results []CheckResult) []CheckCategory {
	present := make(map[CheckCategory]bool)
	var firstSeen []CheckCategory
	for _, r := range results {
		if !present[r.Category] {
			present[r.Category] = true
			firstSeen = append(firstSeen, r.Category)
		}
	}

	ordered := make([]CheckCategory, 0, len(firstSeen))
	listed := make(map[CheckCategory]bool, len(categoryOrder))
	for _, cat := range categoryOrder {
		listed[cat] = true
		if present[cat] {
			ordered = append(ordered, cat)
		}
	}
	for _, cat := range firstSeen {
		if !listed[cat] {
			ordered = append(ordered, cat)
		}
	}
	return ordered
}

func formatHuman(report *CheckReport, w io.Writer, useColor bool) error {
	// Group results by category.
	byCategory := make(map[CheckCategory][]CheckResult)
	for _, r := range report.Checks {
		byCategory[r.Category] = append(byCategory[r.Category], r)
	}

	for _, cat := range orderedCategories(report.Checks) {
		results := byCategory[cat]

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
