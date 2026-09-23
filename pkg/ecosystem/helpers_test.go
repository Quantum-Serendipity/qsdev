package ecosystem

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestToModuleConfigWithProxy_JavaBuildTool verifies the proxy key follows
// the Java build tool wherever it is recorded: explicitly as PackageManager
// (--java-build-tool) or via detection in Extras["build_tool"].
func TestToModuleConfigWithProxy_JavaBuildTool(t *testing.T) {
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
			got := ToModuleConfigWithProxy(tt.lang, infra).RegistryProxy
			if got != tt.want {
				t.Errorf("RegistryProxy = %q, want %q", got, tt.want)
			}
		})
	}
}
