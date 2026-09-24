package devinit

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// findModuleField returns lang's module field key, failing the test when the
// form has no such field.
func findModuleField(t *testing.T, fs *formState, lang, key string) *moduleField {
	t.Helper()
	for _, lf := range fs.moduleFields {
		if lf.lang != lang {
			continue
		}
		for _, f := range lf.fields {
			if f.spec.Key == key {
				return f
			}
		}
	}
	t.Fatalf("form has no %s/%s module field", lang, key)
	return nil
}

// customizeFormState returns the wizard's form state for detected, switched
// to the customize path with langs selected.
func customizeFormState(t *testing.T, detected types.DetectedProject, langs ...string) *formState {
	t.Helper()
	partial, flags := parseInitFlags(t)
	partial.Detected = detected
	fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)
	fs.quickChoice = "customize"
	fs.selectedLanguages = langs
	return fs
}

func languageChoice(t *testing.T, langs []types.LanguageChoice, name string) types.LanguageChoice {
	t.Helper()
	i := slices.IndexFunc(langs, func(lc types.LanguageChoice) bool { return lc.Name == name })
	if i < 0 {
		t.Fatalf("answers have no %s language: %+v", name, langs)
	}
	return langs[i]
}

func TestNewModuleFields_Seeding(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{
		HasGoMod:  true,
		GoVersion: "1.24",
		Suggested: map[string]types.LanguageChoice{
			"java": {Name: "java", PackageManager: "gradle", Extras: []string{"build_tool=gradle", "kotlin"}},
		},
	}
	fs := customizeFormState(t, detected)

	tests := []struct {
		lang, key, want string
	}{
		{"go", types.SettingVersion, "1.24"},                                    // from the detected choice
		{"java", types.SettingPackageManager, "gradle"},                         // from the module's suggestion
		{"java", "kotlin", "true"},                                              // bare flag extra
		{"java", types.SettingVersion, strconv.Itoa(ecosystem.DefaultJDKMajor)}, // field Default
		{"rust", "channel", "stable"},                                           // field Default
		{"dart", "flutter", "false"},                                            // unset confirm
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.key, func(t *testing.T) {
			t.Parallel()
			if got := findModuleField(t, fs, tt.lang, tt.key).answer(); got != tt.want {
				t.Errorf("seeded answer = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapFormToAnswers_RecordsModuleFieldAnswers(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	fs := customizeFormState(t, detected, "go", "clojure", "cpp", "dart", "rust")
	findModuleField(t, fs, "clojure", "build_tool").text = "leiningen"
	findModuleField(t, fs, "cpp", types.SettingPackageManager).text = "conan"
	findModuleField(t, fs, "rust", "channel").text = "nightly"

	answers := mapFormToAnswers(fs, "/tmp/project", "project", detected)

	tests := []struct {
		lang, key, want string
	}{
		{"clojure", "build_tool", "leiningen"},
		{"cpp", types.SettingPackageManager, "conan"},
		{"cpp", "build_system", "cmake"}, // the default the form showed is recorded
		{"rust", "channel", "nightly"},
		{"go", types.SettingVersion, "1.24"},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.key, func(t *testing.T) {
			t.Parallel()
			lc := languageChoice(t, answers.Languages, tt.lang)
			if got, _ := lc.Setting(tt.key); got != tt.want {
				t.Errorf("%s setting %q = %q, want %q (choice %+v)", tt.lang, tt.key, got, tt.want, lc)
			}
		})
	}

	t.Run("unchanged answers add nothing", func(t *testing.T) {
		t.Parallel()
		if lc := languageChoice(t, answers.Languages, "dart"); len(lc.Extras) != 0 {
			t.Errorf("dart extras = %v, want none for an unanswered confirm", lc.Extras)
		}
		if lc := languageChoice(t, answers.Languages, "go"); len(lc.Extras) != 0 {
			t.Errorf("go extras = %v, want none", lc.Extras)
		}
	})

	t.Run("cpp package manager reaches the module", func(t *testing.T) {
		t.Parallel()
		mod, ok := ecosystem.DefaultRegistry().ByName("cpp")
		if !ok {
			t.Fatal("cpp module not registered")
		}
		cfg := ecosystem.ToModuleConfig(languageChoice(t, answers.Languages, "cpp"))
		if rules := mod.(interface {
			DenyRules(ecosystem.ModuleConfig) []string
		}).DenyRules(cfg); slices.Contains(rules, "Bash(vcpkg install *)") {
			t.Errorf("DenyRules = %v, want only the conan rule for a conan project", rules)
		}
	})
}

// TestMapFormToAnswers_JavaBuildToolFlag verifies the wizard's Java build
// tool field shows the --java-build-tool value rather than detection's, and
// that changing it takes effect: the module reads the build tool from
// PackageManager before Extras["build_tool"], so an answer recorded only as
// the extra would be shadowed by the flag.
func TestMapFormToAnswers_JavaBuildToolFlag(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{
		HasPomXML:  true,
		Ecosystems: map[string]bool{"java": true},
		Suggested: map[string]types.LanguageChoice{
			"java": {Name: "java", PackageManager: "maven", Extras: []string{"build_tool=maven"}},
		},
	}
	partial, flags := parseInitFlags(t, "--java-build-tool", "gradle")
	partial.Detected = detected
	fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)
	fs.quickChoice = "customize"
	fs.selectedLanguages = []string{"java"}

	field := findModuleField(t, fs, "java", types.SettingPackageManager)
	if got := field.answer(); got != "gradle" {
		t.Fatalf("seeded build tool = %q, want the --java-build-tool value %q", got, "gradle")
	}
	field.text = "both"

	answers := mapFormToAnswers(fs, "/tmp/project", "project", detected)
	cfg := ecosystem.ToModuleConfig(languageChoice(t, answers.Languages, "java"))
	if got := cfg.PM(cfg.Extra("build_tool", "")); got != "both" {
		t.Errorf("effective java build tool = %q, want the wizard answer %q", got, "both")
	}
}

func TestModuleFieldSteps_AccessibleFlow(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{}
	fs := customizeFormState(t, detected, "clojure")

	var out bytes.Buffer
	if err := runAccessibleSteps(moduleFieldSteps(fs), huh.ThemeBase16(), &out, &lineReader{lines: []string{"2"}}); err != nil {
		t.Fatalf("runAccessibleSteps: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Clojure settings") || !strings.Contains(out.String(), "Build tool") {
		t.Errorf("Clojure build tool not prompted:\n%s", out.String())
	}
	for _, unwanted := range []string{"Go version", "Rust channel"} {
		if strings.Contains(out.String(), unwanted) {
			t.Errorf("prompted %q for an unselected language:\n%s", unwanted, out.String())
		}
	}

	answers := mapFormToAnswers(fs, "/tmp/project", "project", detected)
	if got, _ := languageChoice(t, answers.Languages, "clojure").Setting("build_tool"); got != "leiningen" {
		t.Errorf("clojure build_tool = %q, want leiningen", got)
	}
}

func TestModuleFieldSteps_HiddenOnQuickPath(t *testing.T) {
	t.Parallel()
	fs := customizeFormState(t, types.DetectedProject{}, "clojure")
	fs.quickChoice = "yes"
	for _, step := range moduleFieldSteps(fs) {
		if !step.hidden() {
			t.Error("module field step shown on the quick path")
		}
	}
}

func TestHuhOptions_KeepsCurrentValue(t *testing.T) {
	t.Parallel()
	options := []ecosystem.WizardOption{{Label: "Eight", Value: "8"}, {Label: "Nine", Value: "9"}}
	tests := []struct {
		name    string
		current string
		want    []string
	}{
		{"listed value", "9", []string{"8", "9"}},
		{"empty value", "", []string{"8", "9"}},
		{"unlisted value first", "8.0.100", []string{"8.0.100", "8", "9"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, o := range huhOptions(options, tt.current) {
				got = append(got, o.Value)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("option values = %v, want %v", got, tt.want)
			}
		})
	}
}
