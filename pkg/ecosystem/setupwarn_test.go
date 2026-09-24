package ecosystem_test

import (
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// mockSetupWarner implements both EcosystemModule and SetupWarner. It warns
// with the project root and package manager it was given, so tests can see
// both were passed through.
type mockSetupWarner struct {
	ecosystem.MockModule
}

func (m *mockSetupWarner) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.PackageManager == "" {
		return nil
	}
	return []string{projectRoot + " lacks " + config.PackageManager + " files"}
}

func TestRegistry_SetupWarnings(t *testing.T) {
	t.Parallel()

	r := ecosystem.NewRegistry()
	for _, m := range []ecosystem.EcosystemModule{
		&mockSetupWarner{ecosystem.MockModule{NameVal: "warner", DisplayNameVal: "Warner"}},
		&mockSetupWarner{ecosystem.MockModule{NameVal: "other", DisplayNameVal: "Other"}},
		&ecosystem.MockModule{NameVal: "plain", DisplayNameVal: "Plain"},
	} {
		if err := r.Register(m); err != nil {
			t.Fatalf("Register(%q): %v", m.Name(), err)
		}
	}

	tests := []struct {
		name  string
		langs []types.LanguageChoice
		want  []string
	}{
		{name: "no languages", langs: nil, want: nil},
		{
			name:  "module without SetupWarner",
			langs: []types.LanguageChoice{{Name: "plain", PackageManager: "x"}},
			want:  nil,
		},
		{
			name:  "unregistered language",
			langs: []types.LanguageChoice{{Name: "missing", PackageManager: "x"}},
			want:  nil,
		},
		{
			name:  "warner with nothing to report",
			langs: []types.LanguageChoice{{Name: "warner"}},
			want:  nil,
		},
		{
			name: "prefixed with display name in language order",
			langs: []types.LanguageChoice{
				{Name: "other", PackageManager: "b"},
				{Name: "plain", PackageManager: "x"},
				{Name: "warner", PackageManager: "a"},
			},
			want: []string{"Other: /proj lacks b files", "Warner: /proj lacks a files"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := r.SetupWarnings("/proj", tt.langs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupWarnings() = %q, want %q", got, tt.want)
			}
		})
	}
}
