package types_test

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// languageNames returns the selected language names in order.
func languageNames(a *types.WizardAnswers) []string {
	var names []string
	for _, l := range a.Languages {
		names = append(names, l.Name)
	}
	return names
}

func TestWizardAnswers_LanguageEdits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		start       []string
		add         bool
		lang        string
		wantChanged bool
		want        []string
	}{
		{name: "add new", start: []string{"go"}, add: true, lang: "gcp", wantChanged: true, want: []string{"go", "gcp"}},
		{name: "add present", start: []string{"go"}, add: true, lang: "go", want: []string{"go"}},
		{name: "remove present", start: []string{"go", "gcp"}, lang: "gcp", wantChanged: true, want: []string{"go"}},
		{name: "remove duplicates", start: []string{"gcp", "go", "gcp"}, lang: "gcp", wantChanged: true, want: []string{"go"}},
		{name: "remove absent", start: []string{"go"}, lang: "gcp", want: []string{"go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var a types.WizardAnswers
			for _, n := range tt.start {
				a.Languages = append(a.Languages, types.LanguageChoice{Name: n, Version: "keep"})
			}
			var changed bool
			if tt.add {
				changed = a.AddLanguage(tt.lang)
			} else {
				changed = a.RemoveLanguage(tt.lang)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if got := languageNames(&a); !slices.Equal(got, tt.want) {
				t.Errorf("languages = %v, want %v", got, tt.want)
			}
			if got := a.HasLanguage(tt.lang); got != tt.add {
				t.Errorf("HasLanguage(%q) = %v after edit, want %v", tt.lang, got, tt.add)
			}
			if a.Languages[0].Version != "keep" {
				t.Errorf("edit reset an existing entry's settings: %+v", a.Languages[0])
			}
		})
	}
}
