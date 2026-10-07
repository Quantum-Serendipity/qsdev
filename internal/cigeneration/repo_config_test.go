package cigeneration

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// repoRoot is the repository root, relative to the package directory tests
// run in.
var repoRoot = filepath.Join("..", "..")

// readRepoYAML decodes a YAML file under the repository root into out.
func readRepoYAML(t *testing.T, rel string, out any) {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		t.Fatalf("parsing %s: %v", rel, err)
	}
}

// baselineAgeGateDays is the catalog's baseline age gate in whole days, the
// floor every Dependabot cooldown must meet. Reading it from the catalog keeps
// the repository's own bot no looser than what qsdev enforces on users.
func baselineAgeGateDays(t *testing.T) int {
	t.Helper()

	cat, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("loading embedded catalog: %v", err)
	}
	level, ok := cat.ComplianceLevel("baseline")
	if !ok {
		t.Fatal("embedded catalog has no baseline compliance level")
	}
	if level.AgeGatingThresholdHours <= 0 {
		t.Fatalf("baseline age_gating_threshold_hours = %d, want > 0", level.AgeGatingThresholdHours)
	}
	// Round up: a 3.5-day gate needs a 4-day cooldown, never 3.
	return (level.AgeGatingThresholdHours + 23) / 24
}

type dependabotConfig struct {
	Updates []dependabotUpdate `yaml:"updates"`
}

type dependabotUpdate struct {
	Ecosystem string `yaml:"package-ecosystem"`
	Directory string `yaml:"directory"`
	Vendor    bool   `yaml:"vendor"`
	Cooldown  *struct {
		DefaultDays     int `yaml:"default-days"`
		SemverMajorDays int `yaml:"semver-major-days"`
		SemverMinorDays int `yaml:"semver-minor-days"`
		SemverPatchDays int `yaml:"semver-patch-days"`
	} `yaml:"cooldown"`
}

// semverCooldownEcosystems are the Dependabot ecosystems whose cooldown
// accepts semver-*-days. GitHub Actions, Docker and Nix flakes ignore them,
// so setting them there would claim a gate the bot does not apply.
var semverCooldownEcosystems = []string{"gomod"}

// TestDependabotConfigAgeGated is the U26-02 regression test (Decision D9):
// Dependabot is the repository's one dependency bot, and every update it
// proposes must have aged at least as long as qsdev's own baseline gate.
func TestDependabotConfigAgeGated(t *testing.T) {
	t.Parallel()

	var cfg dependabotConfig
	readRepoYAML(t, filepath.Join(".github", "dependabot.yml"), &cfg)
	floor := baselineAgeGateDays(t)

	seen := map[string]bool{}
	for _, u := range cfg.Updates {
		seen[u.Ecosystem] = true
		name := u.Ecosystem + " " + u.Directory
		if u.Cooldown == nil {
			t.Errorf("%s: no cooldown; every entry must wait at least %d days", name, floor)
			continue
		}
		if u.Cooldown.DefaultDays < floor {
			t.Errorf("%s: cooldown.default-days = %d, want >= %d (catalog baseline age gate)",
				name, u.Cooldown.DefaultDays, floor)
		}
		semver := u.Cooldown.SemverMajorDays + u.Cooldown.SemverMinorDays + u.Cooldown.SemverPatchDays
		if semver != 0 && !slices.Contains(semverCooldownEcosystems, u.Ecosystem) {
			t.Errorf("%s: sets semver-*-days, which Dependabot does not support for this ecosystem", name)
		}
		if u.Ecosystem == "gomod" {
			if !u.Vendor {
				t.Errorf("%s: vendor: true is required; the module is vendored", name)
			}
			if u.Cooldown.SemverMajorDays < 14 {
				t.Errorf("%s: cooldown.semver-major-days = %d, want >= 14", name, u.Cooldown.SemverMajorDays)
			}
		}
	}
	for _, eco := range []string{"gomod", "github-actions", "docker", "nix"} {
		if !seen[eco] {
			t.Errorf("dependabot.yml has no %s entry", eco)
		}
	}
}

type fixupWorkflow struct {
	On          map[string]yaml.Node `yaml:"on"`
	Permissions map[string]string    `yaml:"permissions"`
	Jobs        map[string]struct {
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []struct {
			Name string            `yaml:"name"`
			If   string            `yaml:"if"`
			Uses string            `yaml:"uses"`
			Run  string            `yaml:"run"`
			With map[string]string `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// dependabotActorGateRe is the only job condition the fixup accepts: an
// equality test on the Dependabot actor, so the fixup's own push and any
// other actor's push to dependabot/** never run with the PAT.
var dependabotActorGateRe = regexp.MustCompile(`^\s*github\.actor\s*==\s*'dependabot\[bot\]'\s*$`)

// shaPinnedUsesRe matches an action reference pinned to a full commit SHA.
var shaPinnedUsesRe = regexp.MustCompile(`^[\w.-]+/[\w./-]+@[0-9a-f]{40}$`)

// TestDependabotFixupWorkflowScope pins the trust boundary of the fixup
// workflow. It pushes to Dependabot branches with a PAT, so it must run only
// on those branches, never on a pull_request_target event, with a read-only
// GITHUB_TOKEN, and only for the step that matches the branch's ecosystem.
func TestDependabotFixupWorkflowScope(t *testing.T) {
	t.Parallel()

	var wf fixupWorkflow
	readRepoYAML(t, filepath.Join(".github", "workflows", "dependabot-fixup.yml"), &wf)

	if len(wf.On) != 1 {
		t.Errorf("on: has %d triggers, want only push", len(wf.On))
	}
	if _, ok := wf.On["pull_request_target"]; ok {
		t.Error("on: includes pull_request_target")
	}
	push, ok := wf.On["push"]
	if !ok {
		t.Fatal("on: has no push trigger")
	}
	var pushCfg struct {
		Branches []string `yaml:"branches"`
	}
	if err := push.Decode(&pushCfg); err != nil {
		t.Fatalf("decoding on.push: %v", err)
	}
	if !slices.Equal(pushCfg.Branches, []string{"dependabot/**"}) {
		t.Errorf("on.push.branches = %v, want [dependabot/**]", pushCfg.Branches)
	}
	if len(wf.Permissions) != 1 || wf.Permissions["contents"] != "read" {
		t.Errorf("top-level permissions = %v, want {contents: read}", wf.Permissions)
	}

	wantSteps := map[string]string{
		"dependabot/go_modules/":     "go mod vendor",
		"dependabot/github_actions/": "go generate ./internal/cigeneration/",
		"dependabot/nix/":            "devenv.lock",
	}
	found := map[string]bool{}
	for name, job := range wf.Jobs {
		if !dependabotActorGateRe.MatchString(job.If) {
			t.Errorf("job %s: if = %q, want exactly github.actor == 'dependabot[bot]'", name, job.If)
		}
		for perm, level := range job.Permissions {
			if level != "read" && level != "none" {
				t.Errorf("job %s: permissions %s: %s; pushes use the PAT, the token stays read-only", name, perm, level)
			}
		}
		for _, s := range job.Steps {
			if s.Uses != "" && !shaPinnedUsesRe.MatchString(s.Uses) {
				t.Errorf("job %s: uses %q is not pinned to a commit SHA", name, s.Uses)
			}
			for prefix, cmd := range wantSteps {
				if strings.Contains(s.If, "'"+prefix+"'") && strings.Contains(s.Run, cmd) {
					found[prefix] = true
				}
			}
		}
	}
	for prefix, cmd := range wantSteps {
		if !found[prefix] {
			t.Errorf("no step gated on branch prefix %s runs %q", prefix, cmd)
		}
	}
}

// TestNoRenovateConfig keeps Dependabot the only dependency bot (Decision
// D9). Renovate was never installed, so its config and the workflow comments
// that credited it with bumping pins were false assurance.
func TestNoRenovateConfig(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"renovate.json", "renovate.json5", ".renovaterc", ".renovaterc.json"} {
		_, err := os.Stat(filepath.Join(repoRoot, name))
		if err == nil {
			t.Errorf("%s exists; Dependabot is the repository's only dependency bot", name)
		} else if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("stat %s: %v", name, err)
		}
	}

	dir := filepath.Join(repoRoot, ".github")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(string(b)), "renovate") {
			t.Errorf("%s mentions Renovate, which does not run on this repository", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}

// toolVersionsFile is the single source of the CI tool versions that no
// Dependabot ecosystem covers (U26-02).
var toolVersionsFile = filepath.Join(".github", "tool-versions.env")

var (
	// toolVersionLineRe is one KEY=value assignment in tool-versions.env.
	toolVersionLineRe = regexp.MustCompile(`^([A-Z][A-Z0-9_]*)=(.*)$`)
	// toolSourceLineRe is the "# source:" annotation bump-tool-pins.sh reads;
	// it captures the source kind and its target.
	toolSourceLineRe = regexp.MustCompile(`^# source: (github|goproxy|nix-release|nix-sha256) ([\w.-]+(?:/[\w.-]+)*)$`)
	// nixVersionRe is an exact Nix release; Nix tags carry no leading v.
	nixVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	// sha256HexRe is a SHA-256 digest in lower-case hex.
	sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// envRefRe is a ${{ env.NAME }} expression.
	envRefRe = regexp.MustCompile(`\$\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	// goInstallRe captures the version of a `go install pkg@version` command.
	goInstallRe = regexp.MustCompile(`go install\s+[^\s@]+@(\$\{\{[^}]*\}\}|\S+)`)
	// toolEnvRefRe is a whole-value ${{ env.NAME }} reference.
	toolEnvRefRe = regexp.MustCompile(`^\$\{\{\s*env\.([A-Z][A-Z0-9_]*)\s*\}\}$`)
	// workflowScriptRe is a repository CI script a step runs; such a script
	// reads its pins from tool-versions.env itself.
	workflowScriptRe = regexp.MustCompile(`\.github/scripts/[\w.-]+`)
	// scriptPinKeyRe is a name in a CI script shaped like a tool pin key.
	scriptPinKeyRe = regexp.MustCompile(`\b(?:[A-Z][A-Z0-9_]*_VERSION|NIX_SHA256_[A-Z0-9_]+)\b`)
)

// repoScriptsDir holds the scripts workflows run; they may read pins from
// tool-versions.env but must not carry versions of their own.
var repoScriptsDir = filepath.Join(repoRoot, ".github", "scripts")

// readToolVersions parses tool-versions.env. Workflows append its assignment
// lines to $GITHUB_ENV, which rejects anything but NAME=value, so every
// non-comment line must be one, defined once, with a "# source:" line
// directly above it for the updater and a value of the shape that source
// yields: an exact vX.Y.Z release, an exact X.Y.Z Nix release, or the SHA-256
// of that Nix release's tarball for one system.
func readToolVersions(t *testing.T) map[string]string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, toolVersionsFile))
	if err != nil {
		t.Fatalf("reading %s: %v", toolVersionsFile, err)
	}
	versions := map[string]string{}
	prev := ""
	nixRelease := false
	for i, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		switch m := toolVersionLineRe.FindStringSubmatch(line); {
		case line == "" || strings.HasPrefix(line, "#"):
		case m == nil:
			t.Errorf("%s:%d: %q is neither a comment nor KEY=value", toolVersionsFile, i+1, line)
		default:
			where := fmt.Sprintf("%s:%d", toolVersionsFile, i+1)
			if _, dup := versions[m[1]]; dup {
				t.Errorf("%s: %s is defined twice", where, m[1])
			}
			src := toolSourceLineRe.FindStringSubmatch(prev)
			if src == nil {
				t.Errorf("%s: %s has no \"# source: github|goproxy|nix-release|nix-sha256 <target>\" line above it", where, m[1])
			} else {
				nixRelease = checkToolPin(t, where, m[1], m[2], src[1], src[2], nixRelease)
			}
			versions[m[1]] = m[2]
		}
		prev = line
	}
	if len(versions) == 0 {
		t.Fatalf("%s defines no tool versions", toolVersionsFile)
	}
	return versions
}

// checkToolPin checks one pin's value against its source kind. nixSeen says
// whether a nix-release pin came earlier; it returns the updated value.
func checkToolPin(t *testing.T, where, key, value, kind, target string, nixSeen bool) bool {
	t.Helper()

	switch kind {
	case "nix-release":
		if !nixVersionRe.MatchString(value) {
			t.Errorf("%s: %s=%q is not an exact Nix release (want X.Y.Z)", where, key, value)
		}
		return true
	case "nix-sha256":
		if !nixSeen {
			t.Errorf("%s: %s comes before the nix-release pin its digest belongs to", where, key)
		}
		if !sha256HexRe.MatchString(value) {
			t.Errorf("%s: %s=%q is not a lower-case hex SHA-256", where, key, value)
		}
		if want := "NIX_SHA256_" + strings.ToUpper(strings.ReplaceAll(target, "-", "_")); key != want {
			t.Errorf("%s: %s pins the %s tarball; name it %s", where, key, target, want)
		}
	default:
		if !exactVersionRe.MatchString(value) {
			t.Errorf("%s: %s=%q is not an exact release (want vX.Y.Z)", where, key, value)
		}
	}
	return nixSeen
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
	Env  map[string]string `yaml:"env"`
}

// workflowPermissions is a permissions block. The shorthand scalar form
// (read-all, write-all) decodes as {"*": value}.
type workflowPermissions map[string]string

func (p *workflowPermissions) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		*p = workflowPermissions{"*": n.Value}
		return nil
	}
	var m map[string]string
	if err := n.Decode(&m); err != nil {
		return fmt.Errorf("decoding permissions: %w", err)
	}
	*p = m
	return nil
}

type workflowFile struct {
	On          map[string]yaml.Node `yaml:"on"`
	Permissions workflowPermissions  `yaml:"permissions"`
	Env         map[string]string    `yaml:"env"`
	Jobs        map[string]struct {
		Permissions workflowPermissions `yaml:"permissions"`
		Env         map[string]string   `yaml:"env"`
		Steps       []workflowStep      `yaml:"steps"`
	} `yaml:"jobs"`
}

// readRepoWorkflows decodes every workflow in .github/workflows by file name.
func readRepoWorkflows(t *testing.T) map[string]workflowFile {
	t.Helper()

	entries, err := os.ReadDir(repoWorkflowsDir)
	if err != nil {
		t.Fatalf("reading %s: %v", repoWorkflowsDir, err)
	}
	out := map[string]workflowFile{}
	for _, e := range entries {
		if e.IsDir() || (filepath.Ext(e.Name()) != ".yml" && filepath.Ext(e.Name()) != ".yaml") {
			continue
		}
		var wf workflowFile
		readRepoYAML(t, filepath.Join(".github", "workflows", e.Name()), &wf)
		out[e.Name()] = wf
	}
	return out
}

// loadsToolVersions reports whether a step appends tool-versions.env to
// $GITHUB_ENV, which makes its keys visible as env.* to later steps.
func loadsToolVersions(s workflowStep) bool {
	return strings.Contains(s.Run, filepath.ToSlash(toolVersionsFile)) && strings.Contains(s.Run, "GITHUB_ENV")
}

// stepText is every expression-bearing field of a step.
func stepText(s workflowStep) string {
	var b strings.Builder
	b.WriteString(s.Run)
	for _, m := range []map[string]string{s.With, s.Env} {
		for _, v := range m {
			b.WriteString("\n" + v)
		}
	}
	return b.String()
}

// markScriptPins marks the tool-versions keys each CI script in scripts
// names as used, and fails on a pin-shaped name a script reads that
// tool-versions.env does not define.
func markScriptPins(t *testing.T, scripts map[string]bool, versions map[string]string, used map[string]bool) {
	t.Helper()

	for rel := range scripts {
		b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("a workflow runs %s, which cannot be read: %v", rel, err)
			continue
		}
		for _, name := range scriptPinKeyRe.FindAllString(string(b), -1) {
			if _, ok := versions[name]; ok {
				used[name] = true
			} else {
				t.Errorf("%s reads %s, which %s does not define", rel, name, toolVersionsFile)
			}
		}
	}
}

// TestToolVersionsSingleSource is the U26-02 regression test for the tool
// pins no Dependabot ecosystem covers. Their versions live only in
// tool-versions.env, which tool-pins.yml keeps current with a cooldown: no
// `go install` or action `version:` input may carry a literal version, every
// ${{ env.X }} must resolve (a tool-versions key only after the job loaded
// the file), and every key must be used somewhere, by a workflow or by a
// .github/scripts script a workflow runs.
func TestToolVersionsSingleSource(t *testing.T) {
	t.Parallel()

	versions := readToolVersions(t)
	used := map[string]bool{}
	scripts := map[string]bool{}
	for file, wf := range readRepoWorkflows(t) {
		for jobName, job := range wf.Jobs {
			where := file + " job " + jobName
			loaded := false
			for i, s := range job.Steps {
				step := fmt.Sprintf("%s step %d (%s)", where, i+1, s.Name)
				for _, script := range workflowScriptRe.FindAllString(s.Run, -1) {
					scripts[script] = true
				}
				for _, m := range goInstallRe.FindAllStringSubmatch(s.Run, -1) {
					ref := toolEnvRefRe.FindStringSubmatch(m[1])
					if ref == nil || versions[ref[1]] == "" {
						t.Errorf("%s: go install version %q, want ${{ env.KEY }} with KEY from %s", step, m[1], toolVersionsFile)
					}
				}
				if v, ok := s.With["version"]; ok && s.Uses != "" {
					ref := toolEnvRefRe.FindStringSubmatch(v)
					if ref == nil || versions[ref[1]] == "" {
						t.Errorf("%s: %s version %q, want ${{ env.KEY }} with KEY from %s", step, s.Uses, v, toolVersionsFile)
					}
				}
				for _, m := range envRefRe.FindAllStringSubmatch(stepText(s), -1) {
					name := m[1]
					_, inStep := s.Env[name]
					_, inJob := job.Env[name]
					_, inWorkflow := wf.Env[name]
					_, isTool := versions[name]
					switch {
					case isTool && !loaded:
						t.Errorf("%s: uses env.%s before any step loads %s into $GITHUB_ENV", step, name, toolVersionsFile)
					case isTool:
						used[name] = true
					case !inStep && !inJob && !inWorkflow:
						t.Errorf("%s: env.%s is defined nowhere", step, name)
					}
				}
				loaded = loaded || loadsToolVersions(s)
			}
		}
	}
	markScriptPins(t, scripts, versions, used)
	for key := range versions {
		if !used[key] {
			t.Errorf("%s: %s is used by no workflow; remove it", toolVersionsFile, key)
		}
	}
	assertNoLiteralToolVersions(t, versions)
}

// literalVersionRe matches version (vX.Y.Z) anywhere in text, with or
// without its leading v, unless it is part of a longer version number.
func literalVersionRe(version string) *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^0-9.])v?` + regexp.QuoteMeta(strings.TrimPrefix(version, "v")) + `(?:$|[^0-9.])`)
}

// assertNoLiteralToolVersions scans the raw text of every workflow and CI
// script for the pinned tool versions and digests, so a literal pin outside
// the structural checks (`go run pkg@v1.2.3`, a release download URL, an env
// value, a hard-coded checksum) still fails.
func assertNoLiteralToolVersions(t *testing.T, versions map[string]string) {
	t.Helper()

	var files []string
	for _, dir := range []string{repoWorkflowsDir, repoScriptsDir} {
		entries, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) && dir == repoScriptsDir {
			continue
		}
		if err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		name := filepath.ToSlash(strings.TrimPrefix(path, repoRoot+string(filepath.Separator)))
		for key, version := range versions {
			re := literalVersionRe(version)
			for i, line := range strings.Split(string(b), "\n") {
				if re.MatchString(line) {
					t.Errorf("%s:%d contains %s's pin %s literally; read %s from %s",
						name, i+1, key, version, key, toolVersionsFile)
				}
			}
		}
	}
}

// bumpMinAgeRe captures the cooldown bump-tool-pins.sh applies.
var bumpMinAgeRe = regexp.MustCompile(`(?m)^readonly MIN_AGE_DAYS=(\d+)$`)

// TestToolPinsWorkflowScope pins the scope of the tool pin updater. It holds
// a PAT that can push branches and open PRs, so it runs only on its schedule
// or by hand, never on an event a contributor controls, with a read-only
// GITHUB_TOKEN. Its cooldown must meet the catalog's baseline age gate, the
// same floor as Dependabot's.
func TestToolPinsWorkflowScope(t *testing.T) {
	t.Parallel()

	var wf workflowFile
	readRepoYAML(t, filepath.Join(".github", "workflows", "tool-pins.yml"), &wf)

	var triggers []string
	for name := range wf.On {
		triggers = append(triggers, name)
	}
	slices.Sort(triggers)
	if !slices.Equal(triggers, []string{"schedule", "workflow_dispatch"}) {
		t.Errorf("on: = %v, want only [schedule workflow_dispatch]", triggers)
	}
	if len(wf.Permissions) != 1 || wf.Permissions["contents"] != "read" {
		t.Errorf("top-level permissions = %v, want {contents: read}", wf.Permissions)
	}
	runsScript := false
	for name, job := range wf.Jobs {
		for perm, level := range job.Permissions {
			if level != "read" && level != "none" {
				t.Errorf("job %s: permissions %s: %s; the PR uses the PAT, the token stays read-only", name, perm, level)
			}
		}
		for _, s := range job.Steps {
			if s.Uses != "" && !shaPinnedUsesRe.MatchString(s.Uses) {
				t.Errorf("job %s: uses %q is not pinned to a commit SHA", name, s.Uses)
			}
			if strings.Contains(s.Run, "scripts/bump-tool-pins.sh") {
				runsScript = true
			}
		}
	}
	if !runsScript {
		t.Error("no step runs scripts/bump-tool-pins.sh")
	}

	b, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "bump-tool-pins.sh"))
	if err != nil {
		t.Fatalf("reading bump-tool-pins.sh: %v", err)
	}
	m := bumpMinAgeRe.FindSubmatch(b)
	if m == nil {
		t.Fatal("bump-tool-pins.sh has no `readonly MIN_AGE_DAYS=<n>` line")
	}
	if days, _ := strconv.Atoi(string(m[1])); days < baselineAgeGateDays(t) {
		t.Errorf("bump-tool-pins.sh MIN_AGE_DAYS = %d, want >= %d (catalog baseline age gate)", days, baselineAgeGateDays(t))
	}
}

// TestLiteralVersionRe covers the raw-text scan behind
// TestToolVersionsSingleSource.
func TestLiteralVersionRe(t *testing.T) {
	t.Parallel()

	re := literalVersionRe("v1.3.0")
	tests := []struct {
		line string
		want bool
	}{
		{"run: go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...", true},
		{"curl -L https://example.com/releases/download/v1.3.0/tool_1.3.0_Linux.tar.gz", true},
		{"  TOOL: 1.3.0", true},
		{"v1.3.0", true},
		{"tool-1.3.0-linux-amd64", true},
		{"version: ${{ env.TOOL_VERSION }}", false},
		{"v11.3.0", false},
		{"v1.3.01", false},
		{"v1.3.0.1", false},
		{"2.1.3.0", false},
	}
	for _, tc := range tests {
		if got := re.MatchString(tc.line); got != tc.want {
			t.Errorf("literalVersionRe(v1.3.0).MatchString(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}
