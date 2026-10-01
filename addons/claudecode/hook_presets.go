package claudecode

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ErrUnselectableHookPreset reports a hook preset name that is not in the
// catalog or that no Claude Code hook implements.
var ErrUnselectableHookPreset = errors.New("unknown or unimplemented hook preset")

// SelectableHookPresets returns, in catalog order, the catalog hook presets a
// user may select: those that register at least one hook in settings.json.
// It is derived from the hook registry, so a preset the catalog names but no
// generator implements (e.g. one managed by devenv git hooks) is excluded.
func SelectableHookPresets() []string {
	defs := defaultHookRegistry().Definitions()
	base := types.WizardAnswers{ClaudeCode: true}
	var out []string
	for _, name := range validation.HookPresets() {
		withPreset := base
		if err := withPreset.Hooks.EnableHook(name); err != nil {
			continue
		}
		if enablesNewHook(defs, base, withPreset) {
			out = append(out, name)
		}
	}
	return out
}

// enablesNewHook reports whether any definition is enabled for after but not
// for before.
func enablesNewHook(defs []HookDefinition, before, after types.WizardAnswers) bool {
	for _, d := range defs {
		if d.EnabledFunc != nil && !d.EnabledFunc(before) && d.EnabledFunc(after) {
			return true
		}
	}
	return false
}

// ValidateHookPreset returns an error wrapping ErrUnselectableHookPreset,
// listing the valid presets, when name is not selectable. For a preset the
// catalog names but no Claude Code hook implements, it also says that
// commit-time checks are devenv git hooks set by the compliance level.
func ValidateHookPreset(name string) error {
	valid := SelectableHookPresets()
	if slices.Contains(valid, name) {
		return nil
	}
	err := fmt.Errorf("%w %q; valid presets: %s", ErrUnselectableHookPreset, name, strings.Join(valid, ", "))
	if validation.IsValidHookPreset(name) {
		return fmt.Errorf("%w (it registers no Claude Code hook; commit-time checks such as pre-commit are devenv git hooks set by the compliance level's required_pre_commit_hooks)", err)
	}
	return err
}

// ValidateHookChoices rejects every catalog hook preset h turns on that is
// not selectable (see ValidateHookPreset), so an input that sets hook flags
// directly, such as an answers file, cannot record a preset nothing
// implements. Hook flags outside the catalog preset vocabulary (e.g.
// self-protection) are internal and not checked.
func ValidateHookChoices(h types.HookChoices) error {
	var errs []error
	for _, name := range h.EnabledNames() {
		if !validation.IsValidHookPreset(name) {
			continue
		}
		if err := ValidateHookPreset(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
