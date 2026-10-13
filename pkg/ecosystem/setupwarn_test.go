package ecosystem_test

import (
	"fmt"
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

// mockProxyKeyed is a module with a package manager that routes through the
// registry proxy (it implements ProxyKeyProvider).
type mockProxyKeyed struct {
	ecosystem.MockModule
	key string
}

func (m *mockProxyKeyed) ProxyKey(_ ecosystem.ModuleConfig) string { return m.key }

// TestSetupWarningsUnsupportedProxy checks that a registry proxy the project
// sets is reported as unused for every selected module that installs packages
// but does not route them through it, and for no other module.
func TestSetupWarningsUnsupportedProxy(t *testing.T) {
	t.Parallel()

	pms := []ecosystem.PackageManagerInfo{{Name: "cabal"}, {Name: "stack"}}
	r := ecosystem.NewRegistry()
	for _, m := range []ecosystem.EcosystemModule{
		&ecosystem.MockModule{NameVal: "unrouted", DisplayNameVal: "Unrouted", PackageManagersVal: pms},
		&mockProxyKeyed{ecosystem.MockModule{NameVal: "routed", DisplayNameVal: "Routed", PackageManagersVal: pms}, "maven"},
		&mockProxyKeyed{ecosystem.MockModule{NameVal: "keyless", DisplayNameVal: "Keyless", PackageManagersVal: pms}, ""},
		&ecosystem.MockModule{NameVal: "nopm", DisplayNameVal: "NoPM"},
	} {
		if err := r.Register(m); err != nil {
			t.Fatalf("Register(%q): %v", m.Name(), err)
		}
	}
	const unsupported = "registry_proxy is set but %s has no proxy support; cabal, stack will fetch directly from the public registries"

	tests := []struct {
		name  string
		lang  string
		proxy string
		want  []string
	}{
		{"module without a provider warns", "unrouted", "https://proxy.corp.internal", []string{"Unrouted: " + fmt.Sprintf(unsupported, "Unrouted")}},
		{"provider configuration without a key warns", "keyless", "https://proxy.corp.internal", []string{"Keyless: " + fmt.Sprintf(unsupported, "Keyless")}},
		{"provider is silent", "routed", "https://proxy.corp.internal", nil},
		{"module without package managers is silent", "nopm", "https://proxy.corp.internal", nil},
		{"proxy unset is silent", "unrouted", "", nil},
		{"proxy none is silent", "unrouted", types.InfraDisabled, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				Languages:      []types.LanguageChoice{{Name: tt.lang}},
				Infrastructure: types.InfraConfig{RegistryProxy: tt.proxy},
			}
			if got := r.SetupWarnings("/proj", answers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupWarnings() = %q, want %q", got, tt.want)
			}
		})
	}
}
