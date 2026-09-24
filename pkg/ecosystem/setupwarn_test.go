package ecosystem_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// mockSetupWarner implements both EcosystemModule and SetupWarner. It warns
// with the project root, package manager and repository allowlist it was
// given, so tests can see all were passed through.
type mockSetupWarner struct {
	ecosystem.MockModule
}

func (m *mockSetupWarner) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.PackageManager == "" {
		return nil
	}
	msg := projectRoot + " lacks " + config.PackageManager + " files"
	if len(config.RepositoryAllowlist) > 0 {
		msg += " allowing " + strings.Join(config.RepositoryAllowlist, ",")
	}
	return []string{msg}
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
		java  types.JavaConfig
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
		{
			// Modules see the configuration generation uses, so a warning
			// about the generated files matches what is written.
			name:  "generation config reaches the module",
			langs: []types.LanguageChoice{{Name: "warner", PackageManager: "a"}},
			java:  types.JavaConfig{RepositoryAllowlist: []string{"confluent", "nexus"}},
			want:  []string{"Warner: /proj lacks a files allowing confluent,nexus"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := r.SetupWarnings("/proj", types.WizardAnswers{Languages: tt.langs, Java: tt.java}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupWarnings() = %q, want %q", got, tt.want)
			}
		})
	}
}
