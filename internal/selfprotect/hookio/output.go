package hookio

import (
	"fmt"
	"io"
)

// WriteDeny writes a denial message to the writer (typically stderr).
// Format: "qsdev-selfprotect: <ruleID> — <reason>"
func WriteDeny(w io.Writer, ruleID string, reason string) {
	fmt.Fprintf(w, "qsdev-selfprotect: %s — %s\n", ruleID, reason)
}

// WriteEvasionDeny writes an evasion denial message to the writer.
// Format: "qsdev-selfprotect: EVASION-<category> — <reason>"
func WriteEvasionDeny(w io.Writer, category string, reason string) {
	fmt.Fprintf(w, "qsdev-selfprotect: EVASION-%s — %s\n", category, reason)
}

// WriteError writes an internal error message to the writer.
// Format: "qsdev-selfprotect: internal error: <message>"
func WriteError(w io.Writer, message string) {
	fmt.Fprintf(w, "qsdev-selfprotect: internal error: %s\n", message)
}
