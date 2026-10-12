package validation

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// PolicyEntryError is one invalid entry of a list, or one invalid scalar, in
// a committed .qsdev.yaml policy block (the `hooks` block or
// claude_code.permissions).
type PolicyEntryError struct {
	// Field is the list the entry belongs to, e.g. "hooks.tool_gates.denied",
	// or the scalar's key path.
	Field string
	// Index is the entry's position in that list, or -1 when Field is a
	// scalar.
	Index int
	Value string
	Err   error
}

func (e PolicyEntryError) Error() string {
	if e.Index < 0 {
		return fmt.Sprintf("%s %q: %v", e.Field, e.Value, e.Err)
	}
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
// order: file-boundary extra read paths (see CheckBoundaryReadPath),
// tool-gates entries (see CheckToolNamePattern) and the package-guard policy
// (see checkPackageGuard). It is the one validation shared by .qsdev.yaml
// parsing, answers validation and settings generation, so they agree on what
// reaches the hooks.
func CheckHookPolicy(h types.HooksConfig) []PolicyEntryError {
	var errs []PolicyEntryError
	errs = checkEntries(errs, "hooks.file_boundary.extra_read_paths", h.FileBoundary.ExtraReadPaths, CheckBoundaryReadPath)
	errs = checkEntries(errs, "hooks.tool_gates.allowed", h.ToolGates.Allowed, CheckToolNamePattern)
	errs = checkEntries(errs, "hooks.tool_gates.denied", h.ToolGates.Denied, CheckToolNamePattern)
	return checkPackageGuard(errs, h.PackageGuard)
}

// checkPackageGuard appends to errs the invalid values of hooks.package_guard:
// list entries (see CheckPackageGuardEntry), a min_age_days below one day and
// a new_dep_gate outside types.NewDepGates(). Scalar errors carry Index -1.
func checkPackageGuard(errs []PolicyEntryError, g types.PackageGuardConfig) []PolicyEntryError {
	errs = checkEntries(errs, "hooks.package_guard.allowlist", g.Allowlist, CheckPackageGuardEntry)
	errs = checkEntries(errs, "hooks.package_guard.denylist", g.Denylist, CheckPackageGuardEntry)
	if g.MinAgeDays != nil && *g.MinAgeDays < 1 {
		errs = append(errs, PolicyEntryError{
			Field: "hooks.package_guard.min_age_days", Index: -1,
			Value: strconv.Itoa(*g.MinAgeDays), Err: ErrPackageGuardMinAgeDays,
		})
	}
	if g.NewDepGate != "" && !slices.Contains(types.NewDepGates(), g.NewDepGate) {
		errs = append(errs, PolicyEntryError{
			Field: "hooks.package_guard.new_dep_gate", Index: -1,
			Value: g.NewDepGate, Err: fmt.Errorf("unknown value; valid values: %s", strings.Join(types.NewDepGates(), ", ")),
		})
	}
	return errs
}
