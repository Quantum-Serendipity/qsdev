package profile

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestConfigFiles_ProjectManagedOtherBotSkipsProfileBot checks D9 (one
// dependency bot): a project that configures the other bot itself gets no
// config for the profile's bot, in both directions and for the implicit
// default profile, and still gets its security overview.
func TestConfigFiles_ProjectManagedOtherBotSkipsProfileBot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		p       *InfraProfile
		project UpdateToolType
	}{
		{"default renovate, project dependabot", ConsultingDefault.ConfigOnly(), UpdateToolDependabot},
		{"consulting renovate, project dependabot", ConsultingDefault, UpdateToolDependabot},
		{"enterprise renovate, project dependabot", Enterprise, UpdateToolDependabot},
		{"startup dependabot, project renovate", StartupGitHub, UpdateToolRenovate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := ProjectInputs{Ecosystems: []string{"go", "npm"}, ProjectUpdateTools: []UpdateToolType{tt.project}}
			files := mustConfigFiles(t, tt.p, in)
			for _, f := range files {
				if _, isBot := UpdateToolForConfig(f.Path); isBot {
					t.Errorf("generated %s next to the project's own %s config", f.Path, tt.project)
				}
			}
			if _, ok := findFile(files, "docs/security-overview.md"); !ok {
				t.Error("skipping the bot config dropped the security overview")
			}
		})
	}
}

// TestConfigFiles_SameTypeBotStillGenerated checks that a project whose own
// config is for the profile's bot still gets that config (and that a project
// without one is unchanged), so updates keep managing it.
func TestConfigFiles_SameTypeBotStillGenerated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		p       *InfraProfile
		project []UpdateToolType
		path    string
	}{
		{"renovate project, renovate profile", ConsultingDefault.ConfigOnly(), []UpdateToolType{UpdateToolRenovate}, "renovate.json"},
		{"dependabot project, dependabot profile", StartupGitHub, []UpdateToolType{UpdateToolDependabot}, ".github/dependabot.yml"},
		{"no project bot, renovate profile", ConsultingDefault.ConfigOnly(), nil, "renovate.json"},
		{"no project bot, dependabot profile", StartupGitHub, nil, ".github/dependabot.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := ProjectInputs{Ecosystems: []string{"npm"}, ProjectUpdateTools: tt.project}
			if _, ok := findFile(mustConfigFiles(t, tt.p, in), tt.path); !ok {
				t.Errorf("%s not generated", tt.path)
			}
		})
	}
}

// TestSecurityOverview_ProjectManagedBotNoInventedDelay checks that the
// security overview names the project's own bot and claims no update-PR
// delay qsdev did not configure.
func TestSecurityOverview_ProjectManagedBotNoInventedDelay(t *testing.T) {
	t.Parallel()
	in := ProjectInputs{ProjectUpdateTools: []UpdateToolType{UpdateToolDependabot}, MinReleaseAge: 72 * time.Hour}
	doc, ok := findFile(mustConfigFiles(t, ConsultingDefault.ConfigOnly(), in), "docs/security-overview.md")
	if !ok {
		t.Fatal("no docs/security-overview.md")
	}
	s := string(doc.Content)
	if !strings.Contains(s, "**Update Tool**: dependabot, configured by the project") {
		t.Errorf("overview does not name the project-managed bot:\n%s", s)
	}
	if strings.Contains(strings.ToLower(s), "renovate") {
		t.Errorf("overview names the profile's skipped bot:\n%s", s)
	}
	for _, invented := range []string{"day(s)", "proposes releases immediately"} {
		if strings.Contains(s, invented) {
			t.Errorf("overview states a PR delay (%q) qsdev did not configure:\n%s", invented, s)
		}
	}
}

// TestProjectInputsFromAnswers_ProjectUpdateTools checks that the bots the
// project manages itself reach the profile.
func TestProjectInputsFromAnswers_ProjectUpdateTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		bots []string
		want []UpdateToolType
	}{
		{"none", nil, nil},
		{"dependabot", []string{"dependabot"}, []UpdateToolType{UpdateToolDependabot}},
		{"renovate", []string{"renovate"}, []UpdateToolType{UpdateToolRenovate}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := types.WizardAnswers{Detected: types.DetectedProject{ProjectUpdateBots: tt.bots}}
			if got := ProjectInputsFromAnswers(a).ProjectUpdateTools; !slices.Equal(got, tt.want) {
				t.Errorf("ProjectUpdateTools = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUpdateBotConfigPaths checks the single path list: every generated bot
// config is in it and maps back to its tool, and paths of no bot do not.
func TestUpdateBotConfigPaths(t *testing.T) {
	t.Parallel()
	paths := UpdateBotConfigPaths()
	if !slices.IsSorted(paths) {
		t.Errorf("UpdateBotConfigPaths() = %v, want sorted", paths)
	}
	for path, want := range map[string]UpdateToolType{
		"renovate.json":          UpdateToolRenovate,
		".github/dependabot.yml": UpdateToolDependabot,
	} {
		if !slices.Contains(paths, path) {
			t.Errorf("UpdateBotConfigPaths() lacks generated %s", path)
		}
		if got, ok := UpdateToolForConfig(path); !ok || got != want {
			t.Errorf("UpdateToolForConfig(%q) = %q, %v; want %q", path, got, ok, want)
		}
	}
	if _, ok := UpdateToolForConfig("dependabot.yml"); ok {
		t.Error("UpdateToolForConfig(dependabot.yml) matched a path no bot reads")
	}
}
