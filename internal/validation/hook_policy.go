package validation

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// HookPolicyError is one invalid entry of a .qsdev.yaml `hooks` block.
type HookPolicyError struct {
	// Field is the list the entry belongs to, e.g. "hooks.tool_gates.denied".
	Field string
	// Index is the entry's position in that list.
	Index int
	Value string
	Err   error
}

func (e HookPolicyError) Error() string {
	return fmt.Sprintf("%s entry %q: %v", e.Field, e.Value, e.Err)
}

func (e HookPolicyError) Unwrap() error { return e.Err }

// CheckHookPolicy returns every invalid entry of the hooks block h, in field
// order: file-boundary extra read paths (see CheckBoundaryReadPath) and
// tool-gates entries (see CheckToolNamePattern). It is the one validation
// shared by .qsdev.yaml parsing, answers validation and settings generation,
// so they agree on what reaches the hooks.
func CheckHookPolicy(h types.HooksConfig) []HookPolicyError {
	var errs []HookPolicyError
	check := func(field string, values []string, fn func(string) error) {
		for i, v := range values {
			if err := fn(v); err != nil {
				errs = append(errs, HookPolicyError{Field: field, Index: i, Value: v, Err: err})
			}
		}
	}
	check("hooks.file_boundary.extra_read_paths", h.FileBoundary.ExtraReadPaths, CheckBoundaryReadPath)
	check("hooks.tool_gates.allowed", h.ToolGates.Allowed, CheckToolNamePattern)
	check("hooks.tool_gates.denied", h.ToolGates.Denied, CheckToolNamePattern)
	return errs
}
