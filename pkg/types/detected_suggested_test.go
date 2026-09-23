package types

import (
	"slices"
	"testing"
)

func TestWithSuggested(t *testing.T) {
	t.Parallel()
	detected := DetectedProject{
		Suggested: map[string]LanguageChoice{
			"java":   {Name: "java", Extras: []string{"build_tool=gradle", "kotlin=true"}},
			"python": {Name: "python", Version: "3.12", PackageManager: "uv"},
		},
	}
	tests := []struct {
		name string
		in   LanguageChoice
		want LanguageChoice
	}{
		{
			name: "fills missing fields",
			in:   LanguageChoice{Name: "python"},
			want: LanguageChoice{Name: "python", Version: "3.12", PackageManager: "uv", Extras: []string{}},
		},
		{
			name: "explicit values win",
			in:   LanguageChoice{Name: "python", Version: "3.11", PackageManager: "poetry"},
			want: LanguageChoice{Name: "python", Version: "3.11", PackageManager: "poetry", Extras: []string{}},
		},
		{
			name: "extras merged by key",
			in:   LanguageChoice{Name: "java", Extras: []string{"build_tool=maven"}},
			want: LanguageChoice{Name: "java", Extras: []string{"build_tool=maven", "kotlin=true"}},
		},
		{
			name: "no suggestion leaves choice untouched",
			in:   LanguageChoice{Name: "go", Version: "1.24"},
			want: LanguageChoice{Name: "go", Version: "1.24"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := slices.Clone(tt.in.Extras)
			got := detected.WithSuggested(tt.in)
			if got.Version != tt.want.Version || got.PackageManager != tt.want.PackageManager {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if !slices.Equal(got.Extras, tt.want.Extras) && len(got.Extras)+len(tt.want.Extras) > 0 {
				t.Errorf("Extras = %v, want %v", got.Extras, tt.want.Extras)
			}
			if !slices.Equal(tt.in.Extras, orig) {
				t.Errorf("input Extras mutated: %v", tt.in.Extras)
			}
		})
	}
}

// TestFillDefaults_SkipsProbableTier2 verifies probable-only tier 2+
// ecosystems are not auto-enabled and the order is deterministic.
func TestFillDefaults_SkipsProbableTier2(t *testing.T) {
	t.Parallel()
	detected := DetectedProject{
		Ecosystems:         map[string]bool{"shell": true, "ruby": true, "cpp": true, "php": true},
		ProbableEcosystems: map[string]bool{"shell": true, "cpp": true},
	}
	var a WizardAnswers
	a.FillDefaults(detected, nil)
	var names []string
	for _, l := range a.Languages {
		names = append(names, l.Name)
	}
	if want := []string{"php", "ruby"}; !slices.Equal(names, want) {
		t.Errorf("Languages = %v, want %v", names, want)
	}
}
