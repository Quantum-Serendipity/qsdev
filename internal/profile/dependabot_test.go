package profile

import (
	"slices"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type dependabotFile struct {
	Version int `yaml:"version"`
	Updates []struct {
		PackageEcosystem string `yaml:"package-ecosystem"`
		Cooldown         *struct {
			DefaultDays int `yaml:"default-days"`
		} `yaml:"cooldown"`
	} `yaml:"updates"`
}

func parseDependabot(t *testing.T, f types.GeneratedFile) dependabotFile {
	t.Helper()
	var cfg dependabotFile
	if err := yaml.Unmarshal(f.Content, &cfg); err != nil {
		t.Fatalf("dependabot.yml is not valid YAML: %v\n%s", err, f.Content)
	}
	return cfg
}

func dependabotEcosystems(cfg dependabotFile) []string {
	var ecos []string
	for _, u := range cfg.Updates {
		ecos = append(ecos, u.PackageEcosystem)
	}
	return ecos
}

// Regression: Dependabot entries came from the registry proxy's ecosystem
// list (startup-github: npm, maven) plus always-on docker/terraform, so a Go
// project got no gomod updates and failing npm/maven jobs.
func TestDependabot_UsesProjectEcosystems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    []string
		notWant []string
	}{
		{
			name: "go service",
			answers: types.WizardAnswers{
				Languages: []types.LanguageChoice{{Name: "go"}},
				Detected:  types.DetectedProject{HasGoMod: true},
			},
			want:    []string{"gomod", "github-actions"},
			notWant: []string{"npm", "maven", "docker", "terraform"},
		},
		{
			name: "python with dockerfile",
			answers: types.WizardAnswers{
				Languages: []types.LanguageChoice{{Name: "python"}},
				Detected:  types.DetectedProject{HasPyProject: true, HasDockerfile: true},
			},
			want:    []string{"pip", "docker", "github-actions"},
			notWant: []string{"npm", "maven", "terraform", "gomod"},
		},
		{
			name: "gradle java",
			answers: types.WizardAnswers{
				Languages: []types.LanguageChoice{{Name: "java", PackageManager: "gradle"}},
			},
			want:    []string{"gradle", "github-actions"},
			notWant: []string{"maven"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := mustConfigFiles(t, StartupGitHub, ProjectInputsFromAnswers(tt.answers))
			f, ok := findFile(files, ".github/dependabot.yml")
			if !ok {
				t.Fatal("startup-github did not produce .github/dependabot.yml")
			}
			got := dependabotEcosystems(parseDependabot(t, f))
			for _, w := range tt.want {
				if !slices.Contains(got, w) {
					t.Errorf("ecosystems = %v, want %q", got, w)
				}
			}
			for _, nw := range tt.notWant {
				if slices.Contains(got, nw) {
					t.Errorf("ecosystems = %v, must not contain %q", got, nw)
				}
			}
		})
	}
}

func TestDependabot_AgeGatingEmitsCooldown(t *testing.T) {
	t.Parallel()
	in := ProjectInputs{Ecosystems: []string{"go"}}

	gated := *StartupGitHub
	gated.Updates.AgeGatingDays = 5
	f, err := gated.generateDependabotYML(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range parseDependabot(t, f).Updates {
		if u.Cooldown == nil || u.Cooldown.DefaultDays != 5 {
			t.Errorf("%s: cooldown = %+v, want default-days 5", u.PackageEcosystem, u.Cooldown)
		}
	}

	ungated := *StartupGitHub
	ungated.Updates.AgeGatingDays = 0
	f, err = ungated.generateDependabotYML(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range parseDependabot(t, f).Updates {
		if u.Cooldown != nil {
			t.Errorf("%s: cooldown = %+v, want none when age gating is off", u.PackageEcosystem, u.Cooldown)
		}
	}
}

func TestProjectInputsFromAnswers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    []string
	}{
		{"empty", types.WizardAnswers{}, nil},
		{
			"languages and manifests deduplicated and sorted",
			types.WizardAnswers{
				Languages: []types.LanguageChoice{{Name: "javascript"}, {Name: "go"}, {Name: "terraform"}},
				Detected:  types.DetectedProject{HasGoMod: true, HasPackageJSON: true, HasCargoToml: true},
			},
			[]string{"cargo", "go", "npm", "terraform"},
		},
		{
			"language without a package ecosystem",
			types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "shell"}}},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ProjectInputsFromAnswers(tt.answers).Ecosystems
			if !slices.Equal(got, tt.want) {
				t.Errorf("Ecosystems = %v, want %v", got, tt.want)
			}
		})
	}
}
