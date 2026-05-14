package evidence

import (
	"encoding/json"
	"fmt"
	"io"
)

// RenderJSON writes the EvidenceReport as formatted JSON to the given writer.
func RenderJSON(report *EvidenceReport, w io.Writer) error {
	if report == nil {
		return fmt.Errorf("evidence report must not be nil")
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(report)
}
