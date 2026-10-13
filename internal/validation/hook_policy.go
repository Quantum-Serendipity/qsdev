package validation

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// PolicyEntryError is one invalid entry of a list in a committed .qsdev.yaml
// policy block (the `hooks` block or claude_code.permissions).
type PolicyEntryError struct {
	// Field is the list the entry belongs to, e.g. "hooks.tool_gates.denied".
	Field string
	// Index is the entry's position in that list.
	Index int
	Value string
	Err   error
}

func (e PolicyEntryError) Error() string {
	return fmt.Sprintf("%s entry %q: %v", e.Field, e.Value, e.Err)
}

func (e PolicyEntryError) Unwrap() error { return e.Err }

// checkEntries appends to errs a PolicyEntryError for each entry of the list
// field that check rejects.
func checkEntries(errs []PolicyEntryError, field string, values []string, check func(string) error) []PolicyEntryError {
	for i, v := range values {
		if err := check(v); err != nil {
			errs = append(errs, PolicyEntryError{Field: field, Index: i, Value: v, Err: err})
		}
	}
	return errs
}

// CheckHookPolicy returns every invalid entry of the hooks block h, in field
// order: file-boundary extra read paths (see CheckBoundaryReadPath) and
// tool-gates entries (see CheckToolNamePattern). It is the one validation
// shared by .qsdev.yaml parsing, answers validation and settings generation,
// so they agree on what reaches the hooks.
func CheckHookPolicy(h types.HooksConfig) []PolicyEntryError {
	var errs []PolicyEntryError
	errs = checkEntries(errs, "hooks.file_boundary.extra_read_paths", h.FileBoundary.ExtraReadPaths, CheckBoundaryReadPath)
	errs = checkEntries(errs, "hooks.tool_gates.allowed", h.ToolGates.Allowed, CheckToolNamePattern)
	errs = checkEntries(errs, "hooks.tool_gates.denied", h.ToolGates.Denied, CheckToolNamePattern)
	return errs
}
