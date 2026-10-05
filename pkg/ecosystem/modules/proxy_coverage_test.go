package modules

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// The registry proxy key comes from the registered modules (ProxyKeyProvider),
// so these tests live here, where every module is imported, rather than in
// pkg/ecosystem.

func TestProxyKeyForLanguage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		lang   string
		pm     string
		extras []string
		want   string
	}{
		{"javascript", "npm", nil, "npm"},
		{"javascript", "pnpm", nil, "npm"},
		{"python", "pip", nil, "pypi"},
		{"python", "", nil, "pypi"},
		{"python", "uv", nil, ""},
		{"python", "poetry", nil, ""},
		{"go", "", nil, "go"},
		{"java", "maven", nil, "maven"},
		{"java", "gradle", nil, "gradle"},
		{"java", "both", nil, "maven"},
		{"java", "", []string{"build_tool=gradle"}, "gradle"},
		{"rust", "", nil, "cargo"},
		{"dotnet", "", nil, "nuget"},
		{"php", "", nil, "composer"},
		{"scala", "", nil, "maven"},
		{"scala", "", []string{"build_tool=sbt"}, "maven"},
		{"scala", "", []string{"build_tool=mill"}, ""},
		{"clojure", "", nil, ""},
		{"haskell", "", nil, ""},
		{"unknown", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.pm+"/"+strings.Join(tt.extras, ","), func(t *testing.T) {
			t.Parallel()
			lang := types.LanguageChoice{Name: tt.lang, PackageManager: tt.pm, Extras: tt.extras}
			if got := ecosystem.ProxyKeyForLanguage(lang); got != tt.want {
				t.Errorf("ProxyKeyForLanguage(%+v) = %q, want %q", lang, got, tt.want)
			}
		})
	}
}

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
		{"package manager gradle", types.LanguageChoice{Name: ecosystem.NameJava, PackageManager: "gradle"}, "https://proxy.example.com/gradle/"},
		{"detected gradle extra", types.LanguageChoice{Name: ecosystem.NameJava, Extras: []string{"build_tool=gradle"}}, "https://proxy.example.com/gradle/"},
		{"package manager wins over extra", types.LanguageChoice{Name: ecosystem.NameJava, PackageManager: "maven", Extras: []string{"build_tool=gradle"}}, "https://proxy.example.com/maven/"},
		{"detected maven extra", types.LanguageChoice{Name: ecosystem.NameJava, Extras: []string{"build_tool=maven"}}, "https://proxy.example.com/maven/"},
		{"unset defaults to maven", types.LanguageChoice{Name: ecosystem.NameJava}, "https://proxy.example.com/maven/"},
		{"scala sbt uses maven", types.LanguageChoice{Name: "scala"}, "https://proxy.example.com/maven/"},
		{"scala mill is not routed", types.LanguageChoice{Name: "scala", Extras: []string{"build_tool=mill"}}, ""},
		{"clojure is not routed", types.LanguageChoice{Name: "clojure"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.ToModuleConfigWithInfra(tt.lang, infra).RegistryProxy
			if got != tt.want {
				t.Errorf("RegistryProxy = %q, want %q", got, tt.want)
			}
		})
	}
}

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
			got := ecosystem.ToModuleConfigWithInfra(types.LanguageChoice{Name: ecosystem.NameJavaScript}, tt.infra).RegistryProxy
			if got != tt.want {
				t.Errorf("RegistryProxy = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSetupWarnings_UnsupportedProxyCatalog checks the "no proxy support"
// warning against the real modules: modules that install packages without
// routing them through registry_proxy warn, routed modules and modules with
// no package manager do not.
func TestSetupWarnings_UnsupportedProxyCatalog(t *testing.T) {
	t.Parallel()
	const marker = "has no proxy support"
	tests := []struct {
		lang types.LanguageChoice
		want bool
	}{
		{types.LanguageChoice{Name: "haskell"}, true},
		{types.LanguageChoice{Name: "clojure"}, true},
		{types.LanguageChoice{Name: "scala", Extras: []string{"build_tool=mill"}}, true},
		{types.LanguageChoice{Name: "python", PackageManager: "uv"}, true},
		{types.LanguageChoice{Name: "python", PackageManager: "pip"}, false},
		{types.LanguageChoice{Name: "scala"}, false},
		{types.LanguageChoice{Name: "go"}, false},
		{types.LanguageChoice{Name: "java", PackageManager: "maven"}, false},
		{types.LanguageChoice{Name: "java", PackageManager: "gradle"}, false},
		{types.LanguageChoice{Name: "shell"}, false},
	}
	root := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.lang.Name+"/"+tt.lang.PackageManager+"/"+strings.Join(tt.lang.Extras, ","), func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				Languages:      []types.LanguageChoice{tt.lang},
				Infrastructure: types.InfraConfig{RegistryProxy: "https://proxy.corp.internal"},
			}
			got := slices.ContainsFunc(ecosystem.DefaultRegistry().SetupWarnings(root, answers), func(w string) bool {
				return strings.Contains(w, marker)
			})
			if got != tt.want {
				t.Errorf("SetupWarnings contains %q = %v, want %v", marker, got, tt.want)
			}
		})
	}
}

// Markers around the registry proxy coverage matrix in
// docs/configuration-reference.md.
const (
	coverageStart = "<!-- registry-proxy-coverage:start -->"
	coverageEnd   = "<!-- registry-proxy-coverage:end -->"
)

// TestRegistryProxyCoverageMatrixDocumented keeps the registry proxy coverage
// matrix in docs/configuration-reference.md in step with the catalog: one row
// per module with package managers, sorted by display name, naming its
// package managers and whether it implements ecosystem.ProxyKeyProvider.
func TestRegistryProxyCoverageMatrixDocumented(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "configuration-reference.md"))
	if err != nil {
		t.Fatalf("reading configuration reference: %v", err)
	}
	doc := strings.ReplaceAll(string(data), "\r\n", "\n")
	_, rest, ok := strings.Cut(doc, coverageStart)
	if !ok {
		t.Fatalf("configuration reference lacks %s", coverageStart)
	}
	table, _, ok := strings.Cut(rest, coverageEnd)
	if !ok {
		t.Fatalf("configuration reference lacks %s", coverageEnd)
	}

	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(table), "\n") {
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 3 {
			t.Fatalf("matrix row %q has fewer than 3 columns", line)
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		got = append(got, strings.Join(cells[:3], " | "))
	}
	if len(got) < 2 {
		t.Fatalf("matrix has no rows:\n%s", table)
	}
	got = got[2:] // header and separator

	var want []string
	for _, m := range ecosystem.DefaultRegistry().All() {
		pms := m.PackageManagers()
		if len(pms) == 0 {
			continue
		}
		names := make([]string, len(pms))
		for i, pm := range pms {
			names[i] = pm.Name
		}
		routed := "No"
		if _, ok := m.(ecosystem.ProxyKeyProvider); ok {
			routed = "Yes"
		}
		want = append(want, strings.Join([]string{m.DisplayName(), strings.Join(names, ", "), routed}, " | "))
	}
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Errorf("registry proxy coverage matrix is out of date; want rows (Module | Package managers | Routed):\n%s\n\ngot:\n%s",
			strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}
