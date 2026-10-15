package ecosystem

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SettingFields returns the settings m declares that .qsdev.yaml may commit
// as language extras: its wizard fields, other than the version and
// package_manager fields, which have dedicated keys. A module that is not a
// WizardFieldProvider declares none. Detected extras (directory, manifests,
// toolchain, ...) are never declared, so a committed extra cannot set them.
func SettingFields(m EcosystemModule) []WizardField {
	p, ok := m.(WizardFieldProvider)
	if !ok {
		return nil
	}
	return slices.DeleteFunc(slices.Clone(p.WizardFields()), func(f WizardField) bool {
		return f.Key == types.SettingVersion || f.Key == types.SettingPackageManager
	})
}

// DeclaredSettings maps each language to the settings its module declares
// (SettingFields): the allowlist of keys a committed languages[].extras
// entry may set, and the values each accepts.
type DeclaredSettings map[string][]WizardField

// DeclaredSettings returns the declared settings of every registered module
// that declares any.
func (r *Registry) DeclaredSettings() DeclaredSettings {
	d := DeclaredSettings{}
	for _, m := range r.All() {
		if fields := SettingFields(m); len(fields) > 0 {
			d[m.Name()] = fields
		}
	}
	return d
}

// CheckExtras returns one error for each entry of extras (the
// languages[].extras of lang in .qsdev.yaml) that names a setting lang does
// not declare or gives a declared setting a value it does not accept.
func (d DeclaredSettings) CheckExtras(lang string, extras []string) []error {
	var errs []error
	for _, e := range extras {
		for key, value := range ExtrasMap([]string{e}) {
			if err := d.checkExtra(lang, key, value); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Committed returns the entries of extras that set a setting lang declares
// to a value it accepts, as a ModuleConfig extras map; every other entry is
// dropped (CheckExtras reports it). A later entry for a key wins, as in
// ExtrasMap.
func (d DeclaredSettings) Committed(lang string, extras []string) map[string]string {
	m := ExtrasMap(extras)
	maps.DeleteFunc(m, func(k, v string) bool { return d.checkExtra(lang, k, v) != nil })
	if len(m) == 0 {
		return nil
	}
	return m
}

func (d DeclaredSettings) checkExtra(lang, key, value string) error {
	fields := d[lang]
	i := slices.IndexFunc(fields, func(f WizardField) bool { return f.Key == key })
	if i < 0 {
		if len(fields) == 0 {
			return fmt.Errorf("unknown setting %q: %s declares no settings", key, lang)
		}
		return fmt.Errorf("unknown setting %q for %s; declared settings: %s", key, lang, strings.Join(fieldKeys(fields), ", "))
	}
	if err := fields[i].CheckValue(value); err != nil {
		return fmt.Errorf("setting %q for %s: %w", key, lang, err)
	}
	return nil
}

// CheckValue reports whether value is one f accepts, as the wizard would
// record it: one of the options of a select, a comma-separated subset of
// them for a multi-select, "true" or "false" for a confirm, and for free
// input one or more comma-separated bare words (IsValidToken).
func (f WizardField) CheckValue(value string) error {
	switch f.Type {
	case FieldTypeSelect:
		if !slices.Contains(f.optionValues(), value) {
			return fmt.Errorf("invalid value %q; valid values: %s", value, strings.Join(f.optionValues(), ", "))
		}
	case FieldTypeMultiSelect:
		for item := range strings.SplitSeq(value, ",") {
			if !slices.Contains(f.optionValues(), item) {
				return fmt.Errorf("invalid value %q; a comma-separated list of: %s", value, strings.Join(f.optionValues(), ", "))
			}
		}
	case FieldTypeConfirm:
		if value != "true" && value != "false" {
			return fmt.Errorf("invalid value %q; valid values: true, false", value)
		}
	default:
		for item := range strings.SplitSeq(value, ",") {
			if !IsValidToken(item) {
				return fmt.Errorf("invalid value %q; must be comma-separated words of letters, digits, '.', '_' or '-'", value)
			}
		}
	}
	return nil
}

func (f WizardField) optionValues() []string {
	values := make([]string, 0, len(f.Options))
	for _, o := range f.Options {
		values = append(values, o.Value)
	}
	return values
}

func fieldKeys(fields []WizardField) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	return keys
}
