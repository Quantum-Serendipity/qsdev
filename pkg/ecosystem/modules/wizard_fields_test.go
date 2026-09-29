package modules

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWizardFields_EveryKeyIsReadByItsModule verifies that the init wizard's
// answer to every module wizard field reaches the module: storing the answer
// under the field's key (types.LanguageChoice.WithSetting, exactly as the
// wizard records it) must change what the module generates. A key the module
// never reads (a "cpp_build_system" field for a module that reads
// Extras["build_system"]) leaves the output unchanged and fails here.
//
// Select options must each produce distinct output, a confirm must differ
// between "true" and "false", and an input must differ between its
// Placeholder example and no answer.
func TestWizardFields_EveryKeyIsReadByItsModule(t *testing.T) {
	t.Parallel()
	for _, mod := range ecosystem.DefaultRegistry().All() {
		provider, ok := mod.(ecosystem.WizardFieldProvider)
		if !ok {
			continue
		}
		for _, field := range provider.WizardFields() {
			t.Run(mod.Name()+"/"+field.Key, func(t *testing.T) {
				t.Parallel()
				values := answerValues(t, field)
				seen := make(map[string]string, len(values))
				for _, v := range values {
					out := moduleOutput(t, mod, v.set, field.Key, v.value)
					if prev, dup := seen[out]; dup {
						t.Errorf("answers %q and %q produce identical module output: the module does not read %q",
							prev, v.label, field.Key)
					}
					seen[out] = v.label
				}
			})
		}
	}
}

// TestWizardFields_WellFormed verifies each field declares what the wizard
// needs to render and seed it.
func TestWizardFields_WellFormed(t *testing.T) {
	t.Parallel()
	for _, mod := range ecosystem.DefaultRegistry().All() {
		provider, ok := mod.(ecosystem.WizardFieldProvider)
		if !ok {
			continue
		}
		keys := make(map[string]bool)
		for _, field := range provider.WizardFields() {
			t.Run(mod.Name()+"/"+field.Key, func(t *testing.T) {
				if field.Key == "" || field.Label == "" {
					t.Fatalf("field %+v needs a Key and a Label", field)
				}
				if keys[field.Key] {
					t.Errorf("duplicate key %q", field.Key)
				}
				keys[field.Key] = true
				switch field.Type {
				case ecosystem.FieldTypeSelect, ecosystem.FieldTypeMultiSelect:
					if len(field.Options) < 2 {
						t.Errorf("%s field has %d options, want at least 2", field.Type, len(field.Options))
					}
					if field.Type == ecosystem.FieldTypeSelect && field.Default != "" &&
						!slices.ContainsFunc(field.Options, func(o ecosystem.WizardOption) bool { return o.Value == field.Default }) {
						t.Errorf("Default %q is not one of the options", field.Default)
					}
				case ecosystem.FieldTypeConfirm:
					if field.Default != "" && field.Default != "true" && field.Default != "false" {
						t.Errorf("confirm Default = %q, want \"true\", \"false\" or empty", field.Default)
					}
				case ecosystem.FieldTypeInput:
					if field.Placeholder == "" {
						t.Error("input field needs a Placeholder example")
					}
				default:
					t.Errorf("unknown field type %v", field.Type)
				}
			})
		}
	}
}

// answerValue is one answer to try for a field; set is false for "no answer".
type answerValue struct {
	label, value string
	set          bool
}

func answerValues(t *testing.T, field ecosystem.WizardField) []answerValue {
	t.Helper()
	switch field.Type {
	case ecosystem.FieldTypeSelect:
		vals := make([]answerValue, len(field.Options))
		for i, o := range field.Options {
			vals[i] = answerValue{label: o.Value, value: o.Value, set: true}
		}
		return vals
	case ecosystem.FieldTypeMultiSelect:
		vals := []answerValue{{label: "(none)"}}
		for _, o := range field.Options {
			vals = append(vals, answerValue{label: o.Value, value: o.Value, set: true})
		}
		return vals
	case ecosystem.FieldTypeConfirm:
		return []answerValue{{label: "true", value: "true", set: true}, {label: "false", value: "false", set: true}}
	case ecosystem.FieldTypeInput:
		return []answerValue{{label: "(empty)"}, {label: field.Placeholder, value: field.Placeholder, set: true}}
	}
	t.Fatalf("unknown field type %v", field.Type)
	return nil
}

// moduleOutput renders everything the module generates for a language whose
// only setting is key=value (or nothing when !set): the result of every
// method that takes a ModuleConfig, so no generator is missed.
func moduleOutput(t *testing.T, mod ecosystem.EcosystemModule, set bool, key, value string) string {
	t.Helper()
	lc := types.LanguageChoice{Name: mod.Name()}
	if set {
		lc = lc.WithSetting(key, value)
	}
	cfg := reflect.ValueOf(ecosystem.ToModuleConfig(lc))
	errType := reflect.TypeFor[error]()

	v := reflect.ValueOf(mod)
	out := make(map[string][]any)
	for i := range v.NumMethod() {
		m := v.Method(i)
		if m.Type().NumIn() != 1 || m.Type().In(0) != cfg.Type() {
			continue
		}
		var results []any
		for _, r := range m.Call([]reflect.Value{cfg}) {
			if r.Type() == errType {
				if err, _ := r.Interface().(error); err != nil {
					results = append(results, "error: "+err.Error())
				}
				continue
			}
			results = append(results, r.Interface())
		}
		out[v.Type().Method(i).Name] = results
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshaling %s output: %v", mod.Name(), err)
	}
	return string(data)
}
