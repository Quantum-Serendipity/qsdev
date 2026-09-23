package detect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDetect_FillDefaults_CarriesSuggestedConfig runs the non-interactive
// (--yes) path end to end: detect -> FillDefaults -> ToModuleConfig, and
// asserts each module's detected sub-configuration reaches the module.
func TestDetect_FillDefaults_CarriesSuggestedConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		files  map[string]string
		module string
		key    string
		want   string
	}{
		{"maven", map[string]string{"pom.xml": "<project/>"}, "java", "build_tool", "maven"},
		{"flutter", map[string]string{"pubspec.yaml": "name: app\nflutter:\n  uses-material-design: true\n"}, "dart", "flutter", "true"},
		{"meson", map[string]string{"meson.build": "project('x', 'c')\n"}, "cpp", "build_system", "meson"},
		{"mill", map[string]string{"build.sc": "import mill._\n"}, "scala", "build_tool", "mill"},
		{"stack", map[string]string{"stack.yaml": "resolver: lts-22.0\n", "app.cabal": "name: app\n"}, "haskell", "build_tool", "stack"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, filepath.Join(dir, name), content)
			}

			var a types.WizardAnswers
			a.FillDefaults(Detect(dir), catalog.MustDefault())

			idx := slices.IndexFunc(a.Languages, func(l types.LanguageChoice) bool { return l.Name == tt.module })
			if idx < 0 {
				t.Fatalf("module %q not selected: %+v", tt.module, a.Languages)
			}
			cfg := ecosystem.ToModuleConfig(a.Languages[idx])
			if got := cfg.Extra(tt.key, ""); got != tt.want {
				t.Errorf("%s Extra(%q) = %q, want %q (extras %v)", tt.module, tt.key, got, tt.want, cfg.Extras)
			}
		})
	}
}

// TestDetect_FillDefaults_MavenDenyRules verifies the security consequence of
// carrying build_tool: a Maven project's --yes defaults produce the mvn deny
// rules.
func TestDetect_FillDefaults_MavenDenyRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pom.xml"), "<project/>")

	var a types.WizardAnswers
	a.FillDefaults(Detect(dir), catalog.MustDefault())

	m, ok := ecosystem.DefaultRegistry().ByName("java")
	if !ok {
		t.Fatal("java module not registered")
	}
	mod, ok := m.(ecosystem.DenyRuleProvider)
	if !ok {
		t.Fatal("java module does not provide deny rules")
	}
	idx := slices.IndexFunc(a.Languages, func(l types.LanguageChoice) bool { return l.Name == "java" })
	if idx < 0 {
		t.Fatalf("java not selected: %+v", a.Languages)
	}
	rules := mod.DenyRules(ecosystem.ToModuleConfig(a.Languages[idx]))
	if !slices.Contains(rules, "Bash(mvn install *)") {
		t.Errorf("java deny rules = %v, want Bash(mvn install *)", rules)
	}
}

func TestDetect_RedactsRemoteURLCredentials(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"), "[remote \"origin\"]\n\turl = https://x-access-token:ghs_secret@github.com/org/repo.git\n")

	dp := Detect(dir)
	if want := "https://github.com/org/repo.git"; dp.RemoteURL != want {
		t.Errorf("RemoteURL = %q, want %q", dp.RemoteURL, want)
	}
}
