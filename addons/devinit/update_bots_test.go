package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
	handDependabot  = "version: 2\nupdates: []\n"
	overviewFile    = "docs/security-overview.md"
	dependabotFile  = ".github/dependabot.yml"
	renovateFile    = "renovate.json"
	profileDepbot   = "**Update Tool**: dependabot\n"
	profileRenovate = "**Update Tool**: renovate\n"
)

// writeProjectFile writes content to dir/rel, creating parent directories.
func writeProjectFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func projectFileExists(t *testing.T, dir, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

func mustInit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := executeInitCmd(t, dir, args...); err != nil {
		t.Fatalf("init %v: %v\n%s", args, err, out)
	}
}

// startupGitHubConfig returns the committed config of a project switched to
// the startup-github infra profile (Dependabot), as a teammate who switched
// it would commit it.
func startupGitHubConfig(t *testing.T) string {
	t.Helper()
	dir := initLifecycleProject(t)
	mustInit(t, dir, "--yes", "--force", "--infra-profile", "startup-github", "--nix-cache", "none")
	return readProjectFile(t, dir, branding.Get().ConfigFile)
}

// TestProjectManagedBot_RecordedUnmodifiedIsQsdevOwned checks that qsdev's
// own unmodified renovate.json, recorded by the local state or (in a fresh
// clone) only by the committed manifest, is not a bot the project manages.
func TestProjectManagedBot_RecordedUnmodifiedIsQsdevOwned(t *testing.T) {
	for name, dir := range map[string]string{
		"local state": initLifecycleProject(t),
		"fresh clone": freshCloneProject(t),
	} {
		t.Run(name, func(t *testing.T) {
			if got := projectUpdateBots(dir, []string{renovateFile}); got != nil {
				t.Errorf("projectUpdateBots = %v, want none for qsdev's own renovate.json", got)
			}
			// qsdev manages a bot, so a second config is not the project's.
			writeProjectFile(t, dir, dependabotFile, handDependabot)
			if got := projectUpdateBots(dir, []string{dependabotFile, renovateFile}); got != nil {
				t.Errorf("projectUpdateBots = %v, want none while qsdev manages renovate.json", got)
			}
		})
	}
}

// TestProjectManagedBot_RecordedModifiedIsProjectOwned is the W1N-37
// regression test: a recorded renovate.json the project edited is the
// project's own bot on the first join after a switch to a Dependabot
// profile, so the first and second joins agree and neither adds a second
// bot.
func TestProjectManagedBot_RecordedModifiedIsProjectOwned(t *testing.T) {
	cfg := startupGitHubConfig(t)
	dir := freshCloneProject(t)
	edited := strings.Replace(readProjectFile(t, dir, renovateFile), "{", "{\n  \"labels\": [\"deps\"],", 1)
	writeProjectFile(t, dir, renovateFile, edited)
	writeProjectFile(t, dir, branding.Get().ConfigFile, cfg)

	if got := projectUpdateBots(dir, []string{renovateFile}); !slices.Equal(got, []string{"renovate"}) {
		t.Errorf("projectUpdateBots = %v, want [renovate] for an edited recorded config", got)
	}

	var manifests [2]string
	for run := range 2 {
		mustInit(t, dir, "--mode", "join", "--yes")
		if projectFileExists(t, dir, dependabotFile) {
			t.Errorf("join %d generated %s next to the project's edited renovate.json", run+1, dependabotFile)
		}
		if got := readProjectFile(t, dir, renovateFile); got != edited {
			t.Errorf("join %d rewrote the project's renovate.json:\n%s", run+1, got)
		}
		if doc := readProjectFile(t, dir, overviewFile); !strings.Contains(doc, "**Update Tool**: renovate, configured by the project\n") {
			t.Errorf("join %d: overview does not name the project's renovate:\n%s", run+1, doc)
		}
		manifests[run] = readProjectFile(t, dir, state.ManifestFile())
	}
	if manifests[0] != manifests[1] {
		t.Errorf("the second join changed the generated files:\nfirst:\n%s\nsecond:\n%s", manifests[0], manifests[1])
	}
}

// TestJoin_HandWrittenDependabotNoRenovate checks D9 end to end: a project
// with a hand-written Dependabot config and no infra profile gets no
// renovate.json from create, join or update, and its security overview names
// Dependabot as the project's without an invented PR delay.
func TestJoin_HandWrittenDependabotNoRenovate(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T) string
	}{
		{"create", func(t *testing.T) string {
			dir := createGoFixture(t)
			writeProjectFile(t, dir, dependabotFile, handDependabot)
			mustInit(t, dir, "--yes", "--lang", "go", "--tier", "full")
			return dir
		}},
		{"join", func(t *testing.T) string {
			dir := cloneCommitted(t, initLifecycleProject(t), branding.Get().ConfigFile, "go.mod")
			writeProjectFile(t, dir, dependabotFile, handDependabot)
			mustInit(t, dir, "--mode", "join", "--yes")
			return dir
		}},
		{"update", func(t *testing.T) string {
			dir := createGoFixture(t)
			writeProjectFile(t, dir, dependabotFile, handDependabot)
			mustInit(t, dir, "--yes", "--lang", "go", "--tier", "full")
			mustInit(t, dir, "--update")
			return dir
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.run(t)
			if projectFileExists(t, dir, renovateFile) {
				t.Errorf("%s generated next to the project's Dependabot config", renovateFile)
			}
			if got := readProjectFile(t, dir, dependabotFile); got != handDependabot {
				t.Errorf("the project's dependabot.yml was rewritten:\n%s", got)
			}
			doc := readProjectFile(t, dir, overviewFile)
			if !strings.Contains(doc, "**Update Tool**: dependabot, configured by the project\n") || strings.Contains(doc, "day(s)") {
				t.Errorf("security overview does not describe the project's own bot:\n%s", doc)
			}
		})
	}
}

// TestCreate_HandWrittenRenovateDependabotProfile is the reverse case: a
// hand-written Renovate config plus the Dependabot profile gives no
// dependabot.yml.
func TestCreate_HandWrittenRenovateDependabotProfile(t *testing.T) {
	dir := createGoFixture(t)
	const own = "{\"extends\": [\"config:recommended\"]}\n"
	writeProjectFile(t, dir, ".renovaterc.json", own)
	mustInit(t, dir, "--yes", "--lang", "go", "--infra-profile", "startup-github", "--nix-cache", "none")

	if projectFileExists(t, dir, dependabotFile) {
		t.Errorf("%s generated next to the project's Renovate config", dependabotFile)
	}
	if got := readProjectFile(t, dir, ".renovaterc.json"); got != own {
		t.Errorf("the project's .renovaterc.json was rewritten:\n%s", got)
	}
	if doc := readProjectFile(t, dir, overviewFile); !strings.Contains(doc, "**Update Tool**: renovate, configured by the project\n") {
		t.Errorf("overview does not name the project's renovate:\n%s", doc)
	}
}

// TestProfileSwitch_GeneratesNewProfileBotConfig checks that switching from
// the default infra profile (Renovate) to startup-github still generates its
// Dependabot config, as on base, through create --force and through a join
// of the committed switch, and that a second run agrees with the first.
// qsdev's replaced renovate.json is left in place, as on base.
func TestProfileSwitch_GeneratesNewProfileBotConfig(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		args  []string
	}{
		{"init --force", initLifecycleProject,
			[]string{"--yes", "--force", "--infra-profile", "startup-github", "--nix-cache", "none"}},
		{"join", func(t *testing.T) string {
			cfg := startupGitHubConfig(t)
			dir := freshCloneProject(t)
			writeProjectFile(t, dir, branding.Get().ConfigFile, cfg)
			return dir
		}, []string{"--mode", "join", "--yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			if !projectFileExists(t, dir, renovateFile) {
				t.Fatalf("the default profile generated no %s", renovateFile)
			}
			for run := 1; run <= 2; run++ {
				mustInit(t, dir, tt.args...)
				if !projectFileExists(t, dir, dependabotFile) {
					t.Errorf("run %d: startup-github generated no %s", run, dependabotFile)
				}
				if doc := readProjectFile(t, dir, overviewFile); !strings.Contains(doc, profileDepbot) {
					t.Errorf("run %d: overview does not name the profile's dependabot:\n%s", run, doc)
				}
			}
		})
	}
}

// TestNoBotConfig_UnchangedFromBase checks that a consulting-default project
// keeps getting the profile's renovate.json, and an overview naming it, on
// create, join and update.
func TestNoBotConfig_UnchangedFromBase(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T) string
	}{
		{"create", initLifecycleProject},
		{"join", func(t *testing.T) string {
			dir := cloneCommitted(t, initLifecycleProject(t), branding.Get().ConfigFile, "go.mod")
			mustInit(t, dir, "--mode", "join", "--yes")
			return dir
		}},
		{"update", func(t *testing.T) string {
			dir := initLifecycleProject(t)
			mustInit(t, dir, "--update")
			return dir
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.run(t)
			if !projectFileExists(t, dir, renovateFile) {
				t.Errorf("no %s for a project without a bot config", renovateFile)
			}
			if doc := readProjectFile(t, dir, overviewFile); !strings.Contains(doc, profileRenovate) {
				t.Errorf("overview does not name the profile's renovate:\n%s", doc)
			}
		})
	}
}
