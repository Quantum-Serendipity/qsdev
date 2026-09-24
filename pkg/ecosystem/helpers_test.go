package ecosystem

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestToModuleConfigWithInfra_JavaBuildTool verifies the proxy key follows
// the Java build tool wherever it is recorded: explicitly as PackageManager
// (--java-build-tool) or via detection in Extras["build_tool"].
func TestToModuleConfigWithInfra_JavaBuildTool(t *testing.T) {
	t.Parallel()

	infra := types.InfraConfig{
		RegistryProxyOverrides: map[string]string{
			"maven":  "https://proxy.example.com/maven/",
			"gradle": "https://proxy.example.com/gradle/",
		},
	}
	tests := []struct {
		name string
		lang types.LanguageChoice
		want string
	}{
		{"package manager gradle", types.LanguageChoice{Name: NameJava, PackageManager: "gradle"}, "https://proxy.example.com/gradle/"},
		{"detected gradle extra", types.LanguageChoice{Name: NameJava, Extras: []string{"build_tool=gradle"}}, "https://proxy.example.com/gradle/"},
		{"package manager wins over extra", types.LanguageChoice{Name: NameJava, PackageManager: "maven", Extras: []string{"build_tool=gradle"}}, "https://proxy.example.com/maven/"},
		{"detected maven extra", types.LanguageChoice{Name: NameJava, Extras: []string{"build_tool=maven"}}, "https://proxy.example.com/maven/"},
		{"unset defaults to maven", types.LanguageChoice{Name: NameJava}, "https://proxy.example.com/maven/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToModuleConfigWithInfra(tt.lang, infra).RegistryProxy
			if got != tt.want {
				t.Errorf("RegistryProxy = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestToModuleConfigWithInfra_BuildCache verifies infrastructure.build_cache
// reaches modules as the build_cache extra (it was persisted but never
// applied), without overriding a language's own setting.
// TestToModuleConfigWithInfra_RegistryProxyNone checks registry_proxy "none"
// (the opt-out from an infra profile's proxy) is never used as a base URL.
func TestToModuleConfigWithInfra_RegistryProxyNone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		infra types.InfraConfig
		want  string
	}{
		{"none disables the proxy", types.InfraConfig{RegistryProxy: types.InfraDisabled}, ""},
		{"none keeps explicit overrides", types.InfraConfig{RegistryProxy: types.InfraDisabled,
			RegistryProxyOverrides: map[string]string{"npm": "https://npm.corp.internal/"}}, "https://npm.corp.internal/"},
		{"base URL", types.InfraConfig{RegistryProxy: "https://nexus.corp.internal"}, "https://nexus.corp.internal/repository/npm-proxy/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToModuleConfigWithInfra(types.LanguageChoice{Name: NameJavaScript}, tt.infra).RegistryProxy
			if got != tt.want {
				t.Errorf("RegistryProxy = %q, want %q", got, tt.want)
			}
		})
	}
}

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
