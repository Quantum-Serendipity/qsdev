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

// pmKeyedModule routes only its "routed" package manager through the proxy
// key "pmkey", like Python routes pip but not uv.
type pmKeyedModule struct{ *MockModule }

func (m pmKeyedModule) ProxyKey(cfg ModuleConfig) string {
	if cfg.PM("routed") == "routed" {
		return "pmkey"
	}
	return ""
}

// TestResolveRegistryProxy covers the proxy URL resolution shared by
// generation (ToModuleConfigWithInfra) and qsdev check: it is keyed by the
// language's module and the package manager in the config, honours per-key
// overrides, and is empty without a proxy, an unrouted package manager or a
// module that is not registered.
func TestResolveRegistryProxy(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	if err := reg.Register(pmKeyedModule{&MockModule{NameVal: "keyed"}}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(&MockModule{NameVal: "unrouted"}); err != nil {
		t.Fatal(err)
	}
	proxy := types.InfraConfig{
		RegistryProxy:      "https://nexus.example.test",
		RegistryProxyPaths: map[string]string{"pmkey": "/repository/pmkey-proxy/"},
	}

	tests := []struct {
		name  string
		lang  string
		cfg   ModuleConfig
		infra types.InfraConfig
		want  string
	}{
		{"default package manager is routed", "keyed", ModuleConfig{}, proxy, "https://nexus.example.test/repository/pmkey-proxy/"},
		{"routed package manager", "keyed", ModuleConfig{PackageManager: "routed"}, proxy, "https://nexus.example.test/repository/pmkey-proxy/"},
		{"other package manager is not routed", "keyed", ModuleConfig{PackageManager: "other"}, proxy, ""},
		{"override for the key", "keyed", ModuleConfig{}, types.InfraConfig{RegistryProxy: "https://nexus.example.test", RegistryProxyOverrides: map[string]string{"pmkey": "https://mirror.example.test/pm/"}}, "https://mirror.example.test/pm/"},
		{"key without a known path", "keyed", ModuleConfig{}, types.InfraConfig{RegistryProxy: "https://nexus.example.test"}, ""},
		{"no proxy configured", "keyed", ModuleConfig{}, types.InfraConfig{}, ""},
		{"proxy disabled", "keyed", ModuleConfig{}, types.InfraConfig{RegistryProxy: types.InfraDisabled}, ""},
		{"module without a proxy key", "unrouted", ModuleConfig{}, proxy, ""},
		{"unregistered language", "missing", ModuleConfig{}, proxy, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := reg.resolveRegistryProxy(tt.lang, tt.cfg, tt.infra); got != tt.want {
				t.Errorf("resolveRegistryProxy(%q, %+v) = %q, want %q", tt.lang, tt.cfg, got, tt.want)
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
