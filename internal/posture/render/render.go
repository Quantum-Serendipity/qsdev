package render

import (
	"fmt"
	"io"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// Report writes a PostureReport to w in the specified format.
func Report(report *posture.PostureReport, format Format, w io.Writer, opts Options) error {
	switch format {
	case Text:
		return RenderText(report, w, opts)
	case JSON:
		data, err := RenderJSON(report)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case SARIF:
		data, err := RenderSARIF(report)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case Badge:
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
