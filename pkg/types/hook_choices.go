package types

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// ErrUnknownHook reports a hook name that has no HookChoices field.
var ErrUnknownHook = errors.New("unknown hook")

// hookChoiceFields maps every hook name to its HookChoices flag. It is the
// single translation from hook names (CLI flags, project profiles,
// `claude add-hook`) to HookChoices; which names a user may select is decided
// by the catalog's hook presets, not by this table.
var hookChoiceFields = map[string]func(*HookChoices) *bool{
	"auto-format":            func(h *HookChoices) *bool { return &h.AutoFormat },
	"safety-block":           func(h *HookChoices) *bool { return &h.SafetyBlock },
	"pre-commit":             func(h *HookChoices) *bool { return &h.PreCommit },
	"audit-log":              func(h *HookChoices) *bool { return &h.AuditLog },
	"credential-scan":        func(h *HookChoices) *bool { return &h.CredentialScan },
	"destructive-prevention": func(h *HookChoices) *bool { return &h.DestructivePrevention },
	"soc2-audit":             func(h *HookChoices) *bool { return &h.SOC2Audit },
	"file-boundary":          func(h *HookChoices) *bool { return &h.FileBoundary },
	"tool-gates":             func(h *HookChoices) *bool { return &h.ToolGates },
	"sandbox":                func(h *HookChoices) *bool { return &h.SandboxEnabled },
	"security-enforcement":   func(h *HookChoices) *bool { return &h.SecurityEnforcement },
	"self-protection":        func(h *HookChoices) *bool { return &h.SelfProtection },
}

// EnableHook turns on the HookChoices flag for the named hook. It returns an
// error wrapping ErrUnknownHook when the name has no corresponding flag.
func (h *HookChoices) EnableHook(name string) error {
	field, ok := hookChoiceFields[name]
	if !ok {
		return fmt.Errorf("%w %q", ErrUnknownHook, name)
	}
	*field(h) = true
	return nil
}

// HookChoiceNames returns every name EnableHook accepts, sorted.
func HookChoiceNames() []string {
	return slices.Sorted(maps.Keys(hookChoiceFields))
}
