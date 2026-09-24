package types_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func languageByName(t *testing.T, langs []types.LanguageChoice, name string) types.LanguageChoice {
	t.Helper()
	for _, l := range langs {
		if l.Name == name {
			return l
		}
	}
	t.Fatalf("language %q not found in %+v", name, langs)
	return types.LanguageChoice{}
}

func TestDetectedProject_LanguageChoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		detected   types.DetectedProject
		lang       string
		wantExtras []string
		absent     []string
	}{
		{
			name:       "maven marker yields build_tool=maven",
			detected:   types.DetectedProject{HasPomXML: true},
			lang:       "java",
			wantExtras: []string{"build_tool=maven"},
		},
		{
			name:       "gradle marker yields build_tool=gradle",
			detected:   types.DetectedProject{HasBuildGradle: true},
			lang:       "java",
			wantExtras: []string{"build_tool=gradle"},
		},
		{
			name:       "both markers yield build_tool=both",
			detected:   types.DetectedProject{HasPomXML: true, HasBuildGradle: true},
			lang:       "java",
			wantExtras: []string{"build_tool=both"},
		},
		{
			name: "suggested java build_tool and kotlin are kept",
			detected: types.DetectedProject{
				HasBuildGradle: true,
				Suggested: map[string]types.LanguageChoice{
					"java": {Version: "21", Extras: []string{"build_tool=gradle", "kotlin=true"}},
				},
			},
			lang:       "java",
			wantExtras: []string{"build_tool=gradle", "kotlin=true"},
		},
		{
			name: "tier-2 suggested extras reach the language choice",
			detected: types.DetectedProject{
				Ecosystems: map[string]bool{"dart": true},
				Suggested:  map[string]types.LanguageChoice{"dart": {Extras: []string{"flutter=true"}}},
			},
			lang:       "dart",
			wantExtras: []string{"flutter=true"},
		},
		{
			name: "cpp build_system survives",
			detected: types.DetectedProject{
				Ecosystems: map[string]bool{"cpp": true},
				Suggested:  map[string]types.LanguageChoice{"cpp": {Extras: []string{"build_system=meson"}}},
			},
			lang:       "cpp",
			wantExtras: []string{"build_system=meson"},
		},
		{
			name: "tier-1 terraform variant survives",
			detected: types.DetectedProject{
				HasTerraform: true,
				Ecosystems:   map[string]bool{"terraform": true},
				Suggested:    map[string]types.LanguageChoice{"terraform": {Extras: []string{"variant=opentofu"}}},
			},
			lang:       "terraform",
			wantExtras: []string{"variant=opentofu"},
		},
		{
			name: "gcp with helm gets k8s extra",
			detected: types.DetectedProject{
				Ecosystems: map[string]bool{"gcp": true, "helm": true},
			},
			lang:       "gcp",
			wantExtras: []string{"k8s=true"},
		},
		{
			name: "gcp with only a Dockerfile is not kubernetes",
			detected: types.DetectedProject{
				HasDockerfile: true,
				Ecosystems:    map[string]bool{"gcp": true, "container": true},
			},
			lang:   "gcp",
			absent: []string{"k8s=true"},
		},
		{
			name: "azure with only a Dockerfile is not kubernetes",
			detected: types.DetectedProject{
				HasDockerfile: true,
				Ecosystems:    map[string]bool{"azure": true, "container": true},
			},
			lang:   "azure",
			absent: []string{"k8s=true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := languageByName(t, tt.detected.LanguageChoices(), tt.lang)
			for _, e := range tt.wantExtras {
				if !slices.Contains(got.Extras, e) {
					t.Errorf("%s extras = %v, missing %q", tt.lang, got.Extras, e)
				}
			}
			for _, e := range tt.absent {
				if slices.Contains(got.Extras, e) {
					t.Errorf("%s extras = %v, must not contain %q", tt.lang, got.Extras, e)
				}
			}
		})
	}
}

func TestDetectedProject_LanguageChoices_SuggestedVersionAndPM(t *testing.T) {
	t.Parallel()
	d := types.DetectedProject{
		HasPyProject: true,
		Ecosystems:   map[string]bool{"python": true, "php": true},
		Suggested: map[string]types.LanguageChoice{
			"python": {Version: "3.12", PackageManager: "uv"},
			"php":    {Version: "8.3"},
		},
	}
	langs := d.LanguageChoices()
	py := languageByName(t, langs, "python")
	if py.Version != "3.12" || py.PackageManager != "uv" {
		t.Errorf("python = %+v, want version 3.12 and package manager uv", py)
	}
	if php := languageByName(t, langs, "php"); php.Version != "8.3" {
		t.Errorf("php version = %q, want 8.3", php.Version)
	}
	if len(langs) != 2 {
		t.Errorf("expected python and php only, got %+v", langs)
	}
}

func TestDetectedProject_LanguageChoices_SkipsAliases(t *testing.T) {
	t.Parallel()
	d := types.DetectedProject{
		HasPackageJSON: true,
		HasDockerfile:  true,
		Ecosystems:     map[string]bool{"javascript": true, "node": true, "container": true, "docker": true},
	}
	var names []string
	for _, l := range d.LanguageChoices() {
		names = append(names, l.Name)
	}
	if want := []string{"javascript", "container"}; !reflect.DeepEqual(names, want) {
		t.Errorf("languages = %v, want %v", names, want)
	}
}

func TestDetectedProject_LanguageChoices_DoesNotAliasSuggestedExtras(t *testing.T) {
	t.Parallel()
	d := types.DetectedProject{
		HasDockerfile:    true,
		ContainerRuntime: "podman",
		Suggested:        map[string]types.LanguageChoice{"container": {Extras: []string{"has_compose=true"}}},
	}
	_ = d.LanguageChoices()
	if got := d.Suggested["container"].Extras; !reflect.DeepEqual(got, []string{"has_compose=true"}) {
		t.Errorf("LanguageChoices mutated Suggested extras: %v", got)
	}
}

// TestFillDefaults_DeterministicLanguageOrder guards against map-iteration
// order leaking into generated output.
func TestFillDefaults_DeterministicLanguageOrder(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{
		HasGoMod: true,
		Ecosystems: map[string]bool{
			"go": true, "php": true, "ruby": true, "helm": true, "shell": true,
			"lua": true, "perl": true, "zig": true,
		},
	}
	want := []string{"go", "helm", "lua", "perl", "php", "ruby", "shell", "zig"}
	for i := range 50 {
		a := types.WizardAnswers{}
		a.FillDefaults(detected, catalog.MustDefault())
		var got []string
		for _, l := range a.Languages {
			got = append(got, l.Name)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: languages = %v, want %v", i, got, want)
		}
	}
}

func TestFillDefaults_JavaBuildToolFromMarkers(t *testing.T) {
	t.Parallel()
	a := types.WizardAnswers{}
	a.FillDefaults(types.DetectedProject{HasPomXML: true}, catalog.MustDefault())
	java := languageByName(t, a.Languages, "java")
	if !slices.Contains(java.Extras, "build_tool=maven") {
		t.Errorf("java extras = %v, want build_tool=maven", java.Extras)
	}
}

func TestWizardAnswers_ApplyClaudeHookDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   types.WizardAnswers
		want types.HookChoices
	}{
		{"claude disabled is untouched", types.WizardAnswers{}, types.HookChoices{}},
		{"no hooks gets self-protection and safety block", types.WizardAnswers{ClaudeCode: true},
			types.HookChoices{SelfProtection: true, SafetyBlock: true}},
		{"explicit primary hook keeps safety block off", types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{AutoFormat: true}},
			types.HookChoices{SelfProtection: true, AutoFormat: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := tt.in
			a.ApplyClaudeHookDefaults()
			if a.Hooks != tt.want {
				t.Errorf("Hooks = %+v, want %+v", a.Hooks, tt.want)
			}
		})
	}
}

func TestRedactURLCredentials(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"https://x-access-token:ghs_secret@github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"https://ghp_token@github.com/org/repo", "https://github.com/org/repo"},
		{"https://user:p%40ss@host.example:8443/org/repo", "https://host.example:8443/org/repo"},
		{"ssh://git@github.com/org/repo.git", "ssh://github.com/org/repo.git"},
		{"https://github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"git@github.com:org/repo.git", "git@github.com:org/repo.git"},
		{"/srv/git/repo.git", "/srv/git/repo.git"},
		{"https://tok@host", "https://host"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := types.RedactURLCredentials(tt.in); got != tt.want {
				t.Errorf("RedactURLCredentials(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
