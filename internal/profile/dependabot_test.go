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

// TestDependabot_CooldownByComplianceLevel pins the Dependabot cooldown to
// the larger of the profile's days and the compliance level's window, so
// startup-github (no profile cooldown) gains the tier's and a longer profile
// cooldown is kept.
func TestDependabot_CooldownByComplianceLevel(t *testing.T) {
	t.Parallel()
	long := *StartupGitHub
	long.Updates.AgeGatingDays = 10
	tests := []struct {
		name  string
		p     *InfraProfile
		level string
		want  int
	}{
		{"startup-github baseline", StartupGitHub, "baseline", 3},
		{"startup-github enhanced", StartupGitHub, "enhanced", 7},
		{"startup-github strict", StartupGitHub, "strict", 14},
		{"unknown level falls back to baseline", StartupGitHub, "no-such-level", 3},
		{"longer profile cooldown kept", &long, "enhanced", 10},
		{"tier above profile cooldown", &long, "strict", 14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := ProjectInputsFromAnswers(types.WizardAnswers{
				ComplianceLevel: tt.level,
				Detected:        types.DetectedProject{HasGoMod: true, HasPackageJSON: true},
			})
			cfg := parseDependabot(t, mustFile(t, tt.p, in, ".github/dependabot.yml"))
			if len(cfg.Updates) == 0 {
				t.Fatal("dependabot.yml has no update entries")
			}
			for _, u := range cfg.Updates {
				if u.Cooldown == nil || u.Cooldown.DefaultDays != tt.want {
					t.Errorf("%s: cooldown = %+v, want default-days %d", u.PackageEcosystem, u.Cooldown, tt.want)
				}
			}
		})
	}
}
