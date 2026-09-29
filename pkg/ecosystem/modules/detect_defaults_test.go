package modules

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// stubDefaults satisfies types.DefaultsProvider with zero values; the tests
// below only exercise language selection.
type stubDefaults struct{}

func (stubDefaults) DefaultPostmortem() bool          { return false }
func (stubDefaults) DefaultVersionSentinel() bool     { return false }
func (stubDefaults) DefaultVersionSentinelHours() int { return 0 }
func (stubDefaults) DefaultSembleEnabled() bool       { return false }
func (stubDefaults) DefaultSembleMode() string        { return "" }
func (stubDefaults) DefaultMCPServers() []string      { return nil }
func (stubDefaults) DefaultTier() string              { return "" }
func (stubDefaults) TierCompliance(string) string     { return "" }
func (stubDefaults) TierEnabledTools(string) []string { return nil }

// writeTree creates files (relative slash paths) under a new temp dir.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fillFromDetection runs the --yes path: detection, then FillDefaults.
func fillFromDetection(t *testing.T, dir string) []types.LanguageChoice {
	t.Helper()
	summary := ecosystem.DefaultRegistry().DetectWithEnvironment(dir)
	var answers types.WizardAnswers
	answers.FillDefaults(summary.Project, stubDefaults{})
	return answers.Languages
}

func languageByName(langs []types.LanguageChoice, name string) (types.LanguageChoice, bool) {
	for _, l := range langs {
		if l.Name == name {
			return l, true
		}
	}
	return types.LanguageChoice{}, false
}

// TestFillDefaults_CarriesSuggestedConfig verifies each module's detected
// SuggestedConfig reaches the LanguageChoice generators consume, instead of
// being reduced to a bare ecosystem name.
func TestFillDefaults_CarriesSuggestedConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		files      map[string]string
		lang       string
		wantPM     string
		wantExtras []string
	}{
		{
			name:       "gradle kotlin",
			files:      map[string]string{"build.gradle.kts": "plugins { kotlin(\"jvm\") }\n", "src/main/kotlin/App.kt": "fun main() {}\n"},
			lang:       ecosystem.NameJava,
			wantExtras: []string{"build_tool=gradle", "kotlin=true"},
		},
		{
			name:   "uv",
			files:  map[string]string{"pyproject.toml": "[project]\nname = \"x\"\n", "uv.lock": "version = 1\n"},
			lang:   ecosystem.NamePython,
			wantPM: "uv",
		},
		{
			name:       "opentofu with aws provider",
			files:      map[string]string{"infra/main.tofu": "provider \"aws\" {}\n"},
			lang:       ecosystem.NameTerraform,
			wantExtras: []string{"variant=opentofu", "cloud_providers=aws"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			langs := fillFromDetection(t, writeTree(t, tt.files))
			lc, ok := languageByName(langs, tt.lang)
			if !ok {
				t.Fatalf("language %q not selected; got %+v", tt.lang, langs)
			}
			if tt.wantPM != "" && lc.PackageManager != tt.wantPM {
				t.Errorf("PackageManager = %q, want %q", lc.PackageManager, tt.wantPM)
			}
			for _, e := range tt.wantExtras {
				if !slices.Contains(lc.Extras, e) {
					t.Errorf("Extras = %v, missing %q", lc.Extras, e)
				}
			}
		})
	}
}

// TestFillDefaults_ProbableEcosystemsNotAutoEnabled verifies generic,
// probable-only markers do not auto-enable a tier 2+ ecosystem (toolchain,
// hooks and build/test tasks) for an unrelated project.
func TestFillDefaults_ProbableEcosystemsNotAutoEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		files   map[string]string
		wantOn  []string
		wantOff []string
	}{
		{
			name:    "go service with Makefile and helper scripts",
			files:   map[string]string{"go.mod": "module x\n\ngo 1.24\n", "Makefile": "build:\n\tgo build ./...\n", "scripts/gen.py": "print(1)\n"},
			wantOn:  []string{ecosystem.NameGo},
			wantOff: []string{"cpp", "shell"},
		},
		{
			name:    "powershell install script only",
			files:   map[string]string{"go.mod": "module x\n", "install.ps1": "Write-Host hi\n"},
			wantOn:  []string{ecosystem.NameGo},
			wantOff: []string{"powershell"},
		},
		{
			name:    "generic project/ and roles/ directories",
			files:   map[string]string{"go.mod": "module x\n", "project/README.md": "docs\n", "roles/README.md": "docs\n"},
			wantOn:  []string{ecosystem.NameGo},
			wantOff: []string{"scala", "ansible"},
		},
		{
			name:    "kubernetes app.yaml",
			files:   map[string]string{"go.mod": "module x\n", "app.yaml": "apiVersion: apps/v1\nkind: Deployment\n"},
			wantOn:  []string{ecosystem.NameGo},
			wantOff: []string{"gcp"},
		},
		{
			name:   "certain tier 2 marker still enabled",
			files:  map[string]string{"Gemfile": "source 'https://rubygems.org'\n"},
			wantOn: []string{"ruby"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			langs := fillFromDetection(t, writeTree(t, tt.files))
			for _, name := range tt.wantOn {
				if _, ok := languageByName(langs, name); !ok {
					t.Errorf("%s not enabled; got %+v", name, langs)
				}
			}
			for _, name := range tt.wantOff {
				if _, ok := languageByName(langs, name); ok {
					t.Errorf("%s auto-enabled from a generic marker; got %+v", name, langs)
				}
			}
		})
	}
}
