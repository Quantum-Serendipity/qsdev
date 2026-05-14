package check

import (
	"fmt"
	"io"
)

// FormatReport writes the report in the specified format to w.
func FormatReport(report *CheckReport, format OutputFormat, w io.Writer, useColor bool) error {
	switch format {
	case FormatHuman:
		return formatHuman(report, w, useColor)
	case FormatJSON:
		return formatJSON(report, w)
	case FormatSARIF:
		return formatSARIF(report, w)
	case FormatJUnit:
		return formatJUnit(report, w)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}
