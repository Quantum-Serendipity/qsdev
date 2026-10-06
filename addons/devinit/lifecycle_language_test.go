package devinit

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// `enable|disable <language>` (U11-05): a registered language module is
// added or removed through the full update pipeline, so its Claude Code deny
// rules land in settings.json in the same step as its devenv files.

// gcloudTokenCommand is a credential-printing gcloud command the gcp module's
// deny rules must block.
const gcloudTokenCommand = "gcloud auth print-access-token"

// settingsDenyRules returns the permissions.deny list of the project's
// Claude Code settings.
func settingsDenyRules(t *testing.T, dir string) []string {
	t.Helper()
	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	return settings.Permissions.Deny
}

// blocksBash reports whether any rule in rules matches command as Claude Code
// would.
func blocksBash(rules []string, command string) bool {
	return slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, command) })
}

// committedLanguages returns the language names in the project's .qsdev.yaml.
func committedLanguages(t *testing.T, dir string) []string {
	t.Helper()
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
	if err != nil {
		t.Fatalf("parsing %s: %v", branding.Get().ConfigFile, err)
	}
	var names []string
	for _, l := range cfg.Languages {
		names = append(names, l.Name)
	}
	return names
}

// answersLanguages returns the language names in the saved answers.
func answersLanguages(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	for _, l := range loadProjectAnswers(t, dir).Languages {
		names = append(names, l.Name)
	}
	return names
}

func TestEnableLanguage_RegeneratesClaudeDenyRules(t *testing.T) {
	dir := initLifecycleProject(t)
	if blocksBash(settingsDenyRules(t, dir), gcloudTokenCommand) {
		t.Fatal("precondition: settings.json already blocks gcloud before gcp is enabled")
	}

	out, err := enableTool(t, dir, "gcp")
	if err != nil {
		t.Fatalf("enable gcp: %v\n%s", err, out)
	}

	if !blocksBash(settingsDenyRules(t, dir), gcloudTokenCommand) {
		t.Errorf("settings.json has no deny rule matching %q after enable gcp", gcloudTokenCommand)
	}
	if !slices.Contains(settingsDenyRules(t, dir), "Read(~/.config/gcloud/credentials.db)") {
		t.Errorf("settings.json lacks the gcloud credential Read deny after enable gcp")
	}
	if got := committedLanguages(t, dir); !slices.Contains(got, "gcp") || !slices.Contains(got, "go") {
		t.Errorf(".qsdev.yaml languages = %v, want go and gcp", got)
	}
	if got := answersLanguages(t, dir); !slices.Contains(got, "gcp") {
		t.Errorf("answers languages = %v, want gcp", got)
	}
	if !strings.Contains(readProjectFile(t, dir, "devenv.nix"), "google-cloud-sdk") {
		t.Error("devenv.nix lacks google-cloud-sdk after enable gcp")
	}
}

// projectSnapshot returns every file under dir with its content.
func projectSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		snap[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotting %s: %v", dir, err)
	}
	return snap
}

func TestEnableLanguage_DryRunWritesNothing(t *testing.T) {
	dir := initLifecycleProject(t)
	before := projectSnapshot(t, dir)

	out, err := enableTool(t, dir, "gcp", "--dry-run")
	if err != nil {
		t.Fatalf("enable gcp --dry-run: %v\n%s", err, out)
	}

	after := projectSnapshot(t, dir)
	for path, content := range after {
		if before[path] != content {
			t.Errorf("--dry-run changed %s", path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			t.Errorf("--dry-run removed %s", path)
		}
	}
	if !strings.Contains(out, ".claude/settings.json") {
		t.Errorf("--dry-run output does not preview settings.json:\n%s", out)
	}
}

func TestEnableLanguage_AlreadyPresentIsNoop(t *testing.T) {
	dir := initLifecycleProject(t)
	before := projectSnapshot(t, dir)

	out, err := enableTool(t, dir, "go")
	if err != nil {
		t.Fatalf("enable go (already a language): %v\n%s", err, out)
	}
	if !strings.Contains(out, "already enabled") {
		t.Errorf("enable of a present language should report a no-op, got:\n%s", out)
	}
	after := projectSnapshot(t, dir)
	for path, content := range after {
		if before[path] != content {
			t.Errorf("no-op enable changed %s", path)
		}
	}

	out, err = disableTool(t, dir, "gcp")
	if err != nil {
		t.Fatalf("disable gcp (absent language): %v\n%s", err, out)
	}
	if !strings.Contains(out, "already disabled") {
		t.Errorf("disable of an absent language should report a no-op, got:\n%s", out)
	}
}

func TestDisableLanguage_RemovesModuleRules(t *testing.T) {
	dir := initLifecycleProject(t)
	mustEnable(t, dir, "gcp")

	out, err := disableTool(t, dir, "gcp")
	if err != nil {
		t.Fatalf("disable gcp: %v\n%s", err, out)
	}

	if blocksBash(settingsDenyRules(t, dir), gcloudTokenCommand) {
		t.Errorf("settings.json still blocks %q after disable gcp", gcloudTokenCommand)
	}
	if got := committedLanguages(t, dir); slices.Contains(got, "gcp") || !slices.Contains(got, "go") {
		t.Errorf(".qsdev.yaml languages = %v, want go without gcp", got)
	}
	if got := answersLanguages(t, dir); slices.Contains(got, "gcp") {
		t.Errorf("answers languages = %v, want no gcp", got)
	}
}

func TestEnable_UnknownNameMentionsToolsAndLanguages(t *testing.T) {
	dir := initLifecycleProject(t)
	for _, run := range []func(*testing.T, string, ...string) (string, error){enableTool, disableTool} {
		_, err := run(t, dir, "not-a-thing")
		if err == nil {
			t.Fatal("unknown name succeeded")
		}
		msg := err.Error()
		if !strings.Contains(msg, "list") || !strings.Contains(msg, "language") {
			t.Errorf("error %q should point to both '%s list' and the supported languages", msg, branding.Get().AppName)
		}
	}
}

// addUserEdits appends a user deny rule to settings.json and a comment to
// devenv.nix, the edits a language change must never discard.
func addUserEdits(t *testing.T, dir, rule, nixEdit string) {
	t.Helper()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	var settings map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	perms, _ := settings["permissions"].(map[string]any)
	deny, _ := perms["deny"].([]any)
	perms["deny"] = append(deny, rule)
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatalf("encoding settings.json: %v", err)
	}
	if err := os.WriteFile(settingsPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("writing settings.json: %v", err)
	}
	nix := readProjectFile(t, dir, "devenv.nix") + nixEdit + "\n"
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte(nix), 0o644); err != nil {
		t.Fatalf("writing devenv.nix: %v", err)
	}
}

// TestLanguageChange_PreservesUserEdits guards against enable/disable --force
// on a language name becoming update's overwrite-modified mode: --force is
// refused without touching the project, and the plain change merges the
// user's settings.json and devenv.nix edits instead of discarding them.
func TestLanguageChange_PreservesUserEdits(t *testing.T) {
	const (
		rule    = "Bash(my-custom-rule *)"
		nixEdit = "# my devenv edit"
	)
	dir := initLifecycleProject(t)
	addUserEdits(t, dir, rule, nixEdit)

	steps := []struct {
		name string
		run  func(*testing.T, string, ...string) (string, error)
	}{
		{name: "enable", run: enableTool},
		{name: "disable", run: disableTool},
	}
	for _, step := range steps {
		before := projectSnapshot(t, dir)
		out, err := step.run(t, dir, "gcp", "--force")
		if err == nil || !strings.Contains(err.Error(), overwriteModifiedFlag) {
			t.Fatalf("%s gcp --force: err = %v, want a refusal naming %s\n%s", step.name, err, overwriteModifiedFlag, out)
		}
		after := projectSnapshot(t, dir)
		if !maps.Equal(before, after) {
			t.Errorf("refused %s gcp --force changed the project", step.name)
		}

		if out, err := step.run(t, dir, "gcp"); err != nil {
			t.Fatalf("%s gcp: %v\n%s", step.name, err, out)
		}
		if !slices.Contains(settingsDenyRules(t, dir), rule) {
			t.Errorf("%s gcp dropped the user's settings.json deny rule %q", step.name, rule)
		}
		if !strings.Contains(readProjectFile(t, dir, "devenv.nix"), nixEdit) {
			t.Errorf("%s gcp dropped the user's devenv.nix edit", step.name)
		}
	}
}
