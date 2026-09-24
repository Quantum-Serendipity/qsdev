package devinit

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// languageFields holds the wizard fields an ecosystem module contributes
// through ecosystem.WizardFieldProvider, bound to the form.
type languageFields struct {
	lang    string // language (module) name
	display string // display name shown in the step header
	fields  []*moduleField
}

// moduleField is one module wizard field and the answer the form binds to it.
type moduleField struct {
	spec  ecosystem.WizardField
	text  string   // select and input answer
	on    bool     // confirm answer
	multi []string // multi-select answer
}

// newModuleFields builds the fields of every wizard language whose module
// contributes any, in the language list's order. Each field is seeded from
// the setting the language's choice already carries (from known, completed
// by detection), else from the field's Default.
func newModuleFields(registry *ecosystem.Registry, known map[string]types.LanguageChoice, detected types.DetectedProject) []languageFields {
	var all []languageFields
	for _, lang := range allLanguages {
		mod, ok := registry.ByName(lang.Value)
		if !ok {
			continue
		}
		provider, ok := mod.(ecosystem.WizardFieldProvider)
		if !ok {
			continue
		}
		specs := provider.WizardFields()
		if len(specs) == 0 {
			continue
		}
		lc, ok := known[lang.Value]
		if !ok {
			lc = types.LanguageChoice{Name: lang.Value}
		}
		lc = detected.WithSuggested(lc)
		lf := languageFields{lang: lang.Value, display: lang.Display}
		for _, spec := range specs {
			f := &moduleField{spec: spec}
			seed, ok := lc.Setting(spec.Key)
			if !ok {
				seed = spec.Default
			}
			f.set(seed)
			lf.fields = append(lf.fields, f)
		}
		all = append(all, lf)
	}
	return all
}

// knownLanguageChoices indexes the language choices by name; later lists
// override earlier ones.
func knownLanguageChoices(lists ...[]types.LanguageChoice) map[string]types.LanguageChoice {
	known := make(map[string]types.LanguageChoice)
	for _, list := range lists {
		for _, lc := range list {
			known[lc.Name] = lc
		}
	}
	return known
}

// set stores value as the field's answer.
func (f *moduleField) set(value string) {
	switch f.spec.Type {
	case ecosystem.FieldTypeConfirm:
		f.on, _ = strconv.ParseBool(value)
	case ecosystem.FieldTypeMultiSelect:
		f.multi = splitList(value)
	default:
		f.text = value
	}
}

// answer returns the field's answer in the form it is stored in.
func (f *moduleField) answer() string {
	switch f.spec.Type {
	case ecosystem.FieldTypeConfirm:
		return strconv.FormatBool(f.on)
	case ecosystem.FieldTypeMultiSelect:
		return strings.Join(f.multi, ",")
	default:
		return strings.TrimSpace(f.text)
	}
}

// normalize returns a stored setting in the form answer reports it, so an
// unanswered confirm ("") compares equal to "false".
func (f *moduleField) normalize(value string) string {
	probe := moduleField{spec: f.spec}
	probe.set(value)
	return probe.answer()
}

// binding returns the pointer the form binds, for the Plan Preview hash.
func (f *moduleField) binding() any {
	switch f.spec.Type {
	case ecosystem.FieldTypeConfirm:
		return &f.on
	case ecosystem.FieldTypeMultiSelect:
		return &f.multi
	default:
		return &f.text
	}
}

// huhField renders the field as a huh form field.
func (f *moduleField) huhField() huh.Field {
	s := f.spec
	switch s.Type {
	case ecosystem.FieldTypeConfirm:
		return huh.NewConfirm().Title(s.Label).Description(s.Description).Value(&f.on)
	case ecosystem.FieldTypeMultiSelect:
		return huh.NewMultiSelect[string]().Title(s.Label).Description(s.Description).
			Options(huhOptions(s.Options, f.multi...)...).Value(&f.multi)
	case ecosystem.FieldTypeSelect:
		return huh.NewSelect[string]().Title(s.Label).Description(s.Description).
			Options(huhOptions(s.Options, f.text)...).Value(&f.text)
	default:
		input := huh.NewInput().Title(s.Label).Description(s.Description).Value(&f.text)
		if s.Placeholder != "" {
			input = input.Placeholder("e.g. " + s.Placeholder)
		}
		if s.Required {
			input = input.Validate(func(v string) error {
				if strings.TrimSpace(v) == "" {
					return errors.New(s.Label + " is required")
				}
				return nil
			})
		}
		return input
	}
}

// huhOptions converts module options to huh options. A current value that is
// none of the options (a version detection or an earlier configuration
// recorded) is offered first, so the form keeps it unless the user picks
// another.
func huhOptions(options []ecosystem.WizardOption, current ...string) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(options)+len(current))
	for _, c := range current {
		if c != "" && !slices.ContainsFunc(options, func(o ecosystem.WizardOption) bool { return o.Value == c }) {
			opts = append(opts, huh.NewOption(c+" (current)", c))
		}
	}
	for _, o := range options {
		opts = append(opts, huh.NewOption(o.Label, o.Value))
	}
	return opts
}

// moduleFieldSteps returns one wizard screen per language with module
// fields, shown on the customize path when that language is selected.
func moduleFieldSteps(fs *formState) []wizardStep {
	steps := make([]wizardStep, 0, len(fs.moduleFields))
	for _, lf := range fs.moduleFields {
		steps = append(steps, wizardStep{
			fields: func() []huh.Field {
				fields := []huh.Field{huh.NewNote().Title(lf.display + " settings")}
				for _, f := range lf.fields {
					fields = append(fields, f.huhField())
				}
				return fields
			},
			hidden: func() bool {
				return fs.quickChoice == "yes" || !slices.Contains(fs.selectedLanguages, lf.lang)
			},
		})
	}
	return steps
}

// applyModuleFields records the answers to lang's module fields in lc. An
// answer is stored only when it differs from what lc (completed by
// detection) already resolves to, so accepting a seeded value leaves the
// choice as it was.
func applyModuleFields(fs *formState, lc types.LanguageChoice, detected types.DetectedProject) types.LanguageChoice {
	i := slices.IndexFunc(fs.moduleFields, func(lf languageFields) bool { return lf.lang == lc.Name })
	if i < 0 {
		return lc
	}
	for _, f := range fs.moduleFields[i].fields {
		current, _ := detected.WithSuggested(lc).Setting(f.spec.Key)
		if answer := f.answer(); answer != f.normalize(current) {
			lc = lc.WithSetting(f.spec.Key, answer)
		}
	}
	return lc
}

// splitList splits a comma-separated list into trimmed, non-empty entries.
func splitList(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
