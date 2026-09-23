package merge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// CanonicalJSON re-serializes a JSON document in the canonical form qsdev
// writes for generated JSON config files (.claude/settings.json, .mcp.json):
// two-space indentation, object keys sorted at every level, array order and
// number text preserved, and a trailing newline. The generators and the
// three-way merge both emit this form. Otherwise the first update after init
// rewrites a struct-ordered file into map (sorted) order, a large diff with no
// semantic change that hides real permission changes in review.
func CanonicalJSON(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("parsing JSON: unexpected data after the top-level value")
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling canonical JSON: %w", err)
	}
	return append(out, '\n'), nil
}
