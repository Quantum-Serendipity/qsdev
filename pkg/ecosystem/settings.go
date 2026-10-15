package ecosystem

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CommittableFields returns the wizard fields of m that opt in to being
// committed settings (WizardField.Committable): the only keys .qsdev.yaml may
// set as m's language extras. The version and package_manager fields have
// dedicated keys and are never extras. A module that is not a
// WizardFieldProvider has none, and detected extras (directory, manifests,
// toolchain, ...) are never fields, so a committed extra cannot set them.
func CommittableFields(m EcosystemModule) []WizardField {
	p, ok := m.(WizardFieldProvider)
	if !ok {
		return nil
	}
	return slices.DeleteFunc(slices.Clone(p.WizardFields()), func(f WizardField) bool {
		return !f.Committable || f.Key == types.SettingVersion || f.Key == types.SettingPackageManager
	})
}

// CommittableSettings maps each language to its module's CommittableFields:
// the allowlist of keys a committed languages[].extras entry may set, and the
// values each accepts. For these keys .qsdev.yaml is the single source of
// truth.
type CommittableSettings map[string][]WizardField

// CommittableSettings returns the committable settings of every registered
// module that has any.
func (r *Registry) CommittableSettings() CommittableSettings {
	c := CommittableSettings{}
	for _, m := range r.All() {
		if fields := CommittableFields(m); len(fields) > 0 {
			c[m.Name()] = fields
		}
	}
	return c
}

// CheckExtras returns one error for each entry of extras (the
// languages[].extras of lang in .qsdev.yaml) that names a key that is not a
// committable setting of lang or gives one a value it does not accept.
func (c CommittableSettings) CheckExtras(lang string, extras []string) []error {
	var errs []error
	for _, e := range extras {
		for key, value := range ExtrasMap([]string{e}) {
			if err := c.checkExtra(lang, key, value); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Committed returns the entries of extras that set a committable setting of
// lang to a value it accepts, as a ModuleConfig extras map; every other entry
// is dropped (CheckExtras reports it). A later entry for a key wins, as in
// ExtrasMap.
func (c CommittableSettings) Committed(lang string, extras []string) map[string]string {
	m := ExtrasMap(extras)
	maps.DeleteFunc(m, func(k, v string) bool { return c.checkExtra(lang, k, v) != nil })
	if len(m) == 0 {
		return nil
	}
	return m
}

// Recorded returns, as sorted "key=value" entries, the answers in extras (a
// LanguageChoice's) for the committable settings of lang that hold an
// accepted value other than the field's Default: what .qsdev.yaml records
// for them. Every other key, detected ones included, is left out.
func (c CommittableSettings) Recorded(lang string, extras []string) []string {
	answers := c.Committed(lang, extras)
	var out []string
	for _, f := range c[lang] {
		if v, ok := answers[f.Key]; ok && v != f.Default {
			out = append(out, f.Key+"="+v)
		}
	}
	slices.Sort(out)
	return out
}

func (c CommittableSettings) checkExtra(lang, key, value string) error {
	fields := c[lang]
	i := slices.IndexFunc(fields, func(f WizardField) bool { return f.Key == key })
	if i < 0 {
		if len(fields) == 0 {
			return fmt.Errorf("%q is not a committable setting: %s has none", key, lang)
		}
		return fmt.Errorf("%q is not a committable setting for %s; committable settings: %s",
			key, lang, strings.Join(fieldKeys(fields), ", "))
	}
	if err := fields[i].CheckValue(value); err != nil {
		return fmt.Errorf("setting %q for %s: %w", key, lang, err)
	}
	return nil
}

// CheckValue reports whether value is one f accepts, as the wizard would
// record it: one of the options of a select, a comma-separated subset of
// them for a multi-select, "true" or "false" for a confirm, and for free
// input a single bare word (IsValidToken).
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
		if !IsValidToken(value) {
			return fmt.Errorf("invalid value %q; must be a single word of letters, digits, '.', '_' or '-'", value)
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
