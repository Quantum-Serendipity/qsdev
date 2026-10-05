package ecosystem

import (
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestToModuleConfigWithInfra_BuildCache verifies infrastructure.build_cache
// reaches modules as the build_cache extra (it was persisted but never
// applied), without overriding a language's own setting.
func TestToModuleConfigWithInfra_BuildCache(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		lang  types.LanguageChoice
		infra types.InfraConfig
		want  string
	}{
		{"infra sets extra", types.LanguageChoice{Name: NameRust}, types.InfraConfig{BuildCache: "sccache"}, "sccache"},
		{"language extra wins", types.LanguageChoice{Name: NameRust, Extras: []string{"build_cache=none"}}, types.InfraConfig{BuildCache: "sccache"}, "none"},
		{"unset stays unset", types.LanguageChoice{Name: NameRust}, types.InfraConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ToModuleConfigWithInfra(tt.lang, tt.infra).Extra(ExtraBuildCache, ""); got != tt.want {
				t.Errorf("build_cache = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestToGenerationConfig_MinReleaseAge checks the compliance release-age
// window the caller resolves reaches the module unchanged.
func TestToGenerationConfig_MinReleaseAge(t *testing.T) {
	t.Parallel()
	for _, age := range []time.Duration{0, 72 * time.Hour, 168 * time.Hour, 336 * time.Hour} {
		got := ToGenerationConfig(types.LanguageChoice{Name: NameJavaScript}, types.WizardAnswers{}, age)
		if got.MinReleaseAge != age {
			t.Errorf("ToGenerationConfig(..., %v).MinReleaseAge = %v", age, got.MinReleaseAge)
		}
	}
}
