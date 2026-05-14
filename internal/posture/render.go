package posture

import (
	"fmt"
	"io"
)

// OutputFormat enumerates supported output formats for PostureReport rendering.
type OutputFormat string

const (
	FormatText  OutputFormat = "text"
	FormatJSON  OutputFormat = "json"
	FormatSARIF OutputFormat = "sarif"
	FormatBadge OutputFormat = "badge"
)

// RenderReport writes a PostureReport to w in the specified format.
func RenderReport(report *PostureReport, format OutputFormat, w io.Writer, opts RenderOptions) error {
	switch format {
	case FormatText:
		return RenderText(report, w, opts)
	case FormatJSON:
		data, err := RenderJSON(report)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case FormatSARIF:
		data, err := RenderSARIF(report)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case FormatBadge:
		variant := opts.BadgeType
		if variant == "" {
			variant = "score"
		}
		data, err := RenderBadge(report, variant)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}
