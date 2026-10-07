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
	Name            string            `yaml:"name"`
	If              string            `yaml:"if"`
	ContinueOnError string            `yaml:"continue-on-error"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
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

// stringList is a workflow key that takes a single string or a list of
// strings (needs, runs-on). Both forms decode to a slice.
type stringList []string

func (l *stringList) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		*l = stringList{n.Value}
		return nil
	}
	var s []string
	if err := n.Decode(&s); err != nil {
		return fmt.Errorf("decoding string or list: %w", err)
	}
	*l = s
	return nil
}

type workflowJob struct {
	Name            string              `yaml:"name"`
	If              string              `yaml:"if"`
	ContinueOnError string              `yaml:"continue-on-error"`
	Strategy        workflowStrategy    `yaml:"strategy"`
	RunsOn          stringList          `yaml:"runs-on"`
	Needs           stringList          `yaml:"needs"`
	Outputs         map[string]string   `yaml:"outputs"`
	Permissions     workflowPermissions `yaml:"permissions"`
	Env             map[string]string   `yaml:"env"`
	Steps           []workflowStep      `yaml:"steps"`
}

// workflowStrategy is a job's strategy block; the matrix stays a node
// because its axes and include entries are arbitrary keys.
type workflowStrategy struct {
	Matrix yaml.Node `yaml:"matrix"`
}

type workflowFile struct {
	On          map[string]yaml.Node   `yaml:"on"`
	Permissions workflowPermissions    `yaml:"permissions"`
	Env         map[string]string      `yaml:"env"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
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

// TestDogfoodJob guards the CI job that runs scripts/dogfood.sh (U26-WS1):
// qsdev check and a join regeneration of a `git archive HEAD` copy of this
// repository must pass with no drift. The job uses the same SHA-pinned
// harden-runner, checkout and setup-go steps as its siblings, with the
// catalog's pins where the catalog has one, and does not persist the token.
// It is not yet a required gate: no other job needs it, and it does not hide
// a failure behind continue-on-error.
func TestDogfoodJob(t *testing.T) {
	t.Parallel()

	var wf workflowFile
	readRepoYAML(t, filepath.Join(".github", "workflows", "ci.yml"), &wf)
	job, ok := wf.Jobs["dogfood"]
	if !ok {
		t.Fatal("ci.yml has no dogfood job")
	}
	if !slices.Equal(job.RunsOn, stringList{"ubuntu-latest"}) {
		t.Errorf("dogfood runs-on = %v, want [ubuntu-latest]", job.RunsOn)
	}
	if job.ContinueOnError != "" {
		t.Errorf("dogfood continue-on-error = %q; a drift must show as a failed job", job.ContinueOnError)
	}
	for name, other := range wf.Jobs {
		if slices.Contains(other.Needs, "dogfood") {
			t.Errorf("job %s needs dogfood; the dogfood gate is not required yet", name)
		}
	}

	pins := map[string]ActionRef{}
	for _, ref := range AllActionRefs() {
		pins[ref.Owner+"/"+ref.Repo] = ref
	}
	var uses []string
	runsDogfood := false
	for i, s := range job.Steps {
		step := fmt.Sprintf("dogfood step %d (%s)", i+1, s.Name)
		if strings.TrimSpace(s.Run) == "make dogfood" {
			runsDogfood = true
		}
		if s.Uses == "" {
			continue
		}
		if !shaPinnedUsesRe.MatchString(s.Uses) {
			t.Errorf("%s: uses %q is not pinned to a commit SHA", step, s.Uses)
			continue
		}
		path, sha, _ := strings.Cut(s.Uses, "@")
		uses = append(uses, path)
		if ref, ok := pins[path]; ok && ref.SHA != sha {
			t.Errorf("%s: %s@%s, want the catalog pin %s", step, path, sha, ref.SHA)
		}
		if path == "actions/checkout" && s.With["persist-credentials"] != "false" {
			t.Errorf("%s: checkout persist-credentials = %q, want false", step, s.With["persist-credentials"])
		}
		if path == "actions/setup-go" && s.With["go-version-file"] != "go.mod" {
			t.Errorf("%s: setup-go go-version-file = %q, want go.mod", step, s.With["go-version-file"])
		}
	}
	wantUses := []string{"step-security/harden-runner", "actions/checkout", "actions/setup-go"}
	if !slices.Equal(uses, wantUses) {
		t.Errorf("dogfood actions = %v, want %v", uses, wantUses)
	}
	if !runsDogfood {
		t.Error("no dogfood step runs `make dogfood`")
	}
}

// readDogfoodScript returns scripts/dogfood.sh.
func readDogfoodScript(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "dogfood.sh"))
	if err != nil {
		t.Fatalf("reading scripts/dogfood.sh: %v", err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// scriptIndex returns the offset of the first line of script, ignoring
// comments, that contains want, or -1.
func scriptIndex(script, want string) int {
	off := 0
	for _, line := range strings.SplitAfter(script, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			if i := strings.Index(line, want); i >= 0 {
				return off + i
			}
		}
		off += len(line)
	}
	return -1
}

// dogfoodResetRe matches dogfood.sh's loop over the three-way-merged JSON
// files it resets to exactly what qsdev generates.
var dogfoodResetRe = regexp.MustCompile(`(?m)^for merged in ([^;]+); do$`)

// TestDogfoodScript_ResetsMergedJSON pins that scripts/dogfood.sh resets the
// three-way-merged JSON files after the join and before its drift check.
// join keeps entries it no longer generates in those files as user content
// and qsdev check accepts them, so without the reset a stale hook or allow
// rule (U26-01) would pass the gate.
func TestDogfoodScript_ResetsMergedJSON(t *testing.T) {
	t.Parallel()

	script := readDogfoodScript(t)
	m := dogfoodResetRe.FindStringSubmatchIndex(script)
	if m == nil {
		t.Fatal("dogfood.sh has no `for merged in ...; do` reset loop")
	}
	files := strings.Fields(script[m[2]:m[3]])
	for _, want := range []string{".claude/settings.json", ".mcp.json"} {
		if !slices.Contains(files, want) {
			t.Errorf("dogfood.sh does not reset %s (resets %v)", want, files)
		}
	}
	if !strings.Contains(script[m[0]:], `qsdev repair --reset --file "$merged"`) {
		t.Error("dogfood.sh's reset loop does not run `qsdev repair --reset --file \"$merged\"`")
	}
	join := scriptIndex(script, "qsdev init --mode join --quiet")
	drift := scriptIndex(script, "git status --porcelain")
	if join < 0 || drift < 0 || join >= m[0] || m[0] >= drift {
		t.Errorf("dogfood.sh must reset the merged files after the join (at %d) and before the drift check (at %d); reset at %d", join, drift, m[0])
	}

	// The reset files are human-edited (three-way-merge), so the committed
	// manifest of machine-owned files, when there is one, must not list them.
	manifest, err := os.ReadFile(filepath.Join(repoRoot, ".qsdev-generated.sha256"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(f))); err != nil {
			t.Errorf("dogfood.sh resets %s, which the repository does not have: %v", f, err)
		}
		if strings.Contains(string(manifest), "  "+f+"\n") {
			t.Errorf("dogfood.sh resets %s, but it is machine-owned (in the manifest), not three-way-merged", f)
		}
	}
}

// TestDogfoodScript_CanonicalOrigin pins how scripts/dogfood.sh picks the
// origin remote of its copy. Generated files derive from it
// (.cosign/policy.yaml names the image repository and signing identity), so
// a copy without the canonical origin would report drift, or pass with a
// non-enforcing policy. The order is DOGFOOD_ORIGIN, the upstream remote (a
// fork's checkout), then origin, and with none the script fails before it
// builds anything.
func TestDogfoodScript_CanonicalOrigin(t *testing.T) {
	t.Parallel()

	script := readDogfoodScript(t)
	steps := []string{
		`origin=${DOGFOOD_ORIGIN:-}`,
		`remote get-url upstream`,
		`remote get-url origin`,
		`if [ -z "$origin" ]; then`,
		`exit 1`,
		`mktemp -d`,
		`git remote add origin "$origin"`,
		`qsdev check`,
	}
	from, prevStep := 0, "the start"
	for _, step := range steps {
		i := scriptIndex(script[from:], step)
		if i < 0 {
			t.Errorf("dogfood.sh has no %q after %q", step, prevStep)
			continue
		}
		from, prevStep = from+i+len(step), step
	}
}

// dogfoodRmRe matches a command line that removes files.
var dogfoodRmRe = regexp.MustCompile(`(^|[\s;&|(])rm\s`)

// TestDogfoodScript_CleanupScopedToMktemp pins that scripts/dogfood.sh
// removes nothing but its own mktemp directory, from its EXIT trap, and keeps
// that directory when DOGFOOD_KEEP=1 (for inspecting a failure, and for agents
// that may not delete files).
func TestDogfoodScript_CleanupScopedToMktemp(t *testing.T) {
	t.Parallel()

	script := readDogfoodScript(t)
	if scriptIndex(script, `tmp=$(mktemp -d)`) < 0 {
		t.Error("dogfood.sh does not create its directory with `tmp=$(mktemp -d)`")
	}
	var removals []string
	for _, line := range strings.Split(script, "\n") {
		code := strings.TrimSpace(line)
		if !strings.HasPrefix(code, "#") && dogfoodRmRe.MatchString(code) {
			removals = append(removals, code)
		}
	}
	if want := []string{`rm -rf -- "$tmp"`}; !slices.Equal(removals, want) {
		t.Errorf("dogfood.sh removal commands = %q, want exactly %q", removals, want)
	}

	start := scriptIndex(script, "cleanup() {")
	if start < 0 {
		t.Fatal("dogfood.sh has no cleanup() function")
	}
	end := strings.Index(script[start:], "\n}\n")
	if end < 0 {
		t.Fatal("dogfood.sh's cleanup() function is not closed")
	}
	body := script[start : start+end]
	keep, rm := strings.Index(body, `"${DOGFOOD_KEEP:-}" = 1`), strings.Index(body, `rm -rf -- "$tmp"`)
	if keep < 0 || rm < 0 || keep > rm {
		t.Errorf("dogfood.sh's cleanup() must test DOGFOOD_KEEP=1 before removing $tmp:\n%s", body)
	}
	trap, mk := scriptIndex(script, "trap cleanup EXIT"), scriptIndex(script, `tmp=$(mktemp -d)`)
	if trap < 0 || trap < mk {
		t.Error("dogfood.sh must set `trap cleanup EXIT` after creating $tmp")
	}
}

// makeTargetRe matches a Makefile rule line and captures its target.
var makeTargetRe = regexp.MustCompile(`^([A-Za-z0-9_.-]+):`)

// readMakeRecipes returns each Makefile target's prerequisites and recipe
// lines (without the leading tab), in order.
func readMakeRecipes(t *testing.T) map[string][]string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	recipes := map[string][]string{}
	target := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		switch m := makeTargetRe.FindStringSubmatch(line); {
		case m != nil:
			target = m[1]
			recipes[target] = append(recipes[target], strings.Fields(line[len(m[0]):])...)
		case strings.HasPrefix(line, "\t") && target != "":
			recipes[target] = append(recipes[target], strings.TrimPrefix(line, "\t"))
		case !strings.HasPrefix(line, "\t"):
			target = ""
		}
	}
	return recipes
}

// TestMakefileTargets pins the maintainer entry points U26-WS1 adds, next to
// the existing on-demand bench target:
// `make dogfood` runs the CI gate's script, `make dev-install` installs a
// plain source build from the vendored tree (into devenv's $GOPATH/bin, ahead
// of the system qsdev the .mcp.json servers would otherwise run), and
// `make clean` also removes the root-level binary `go build ./cmd/qsdev`
// leaves behind.
func TestMakefileTargets(t *testing.T) {
	t.Parallel()

	recipes := readMakeRecipes(t)
	contains := func(target string, want ...string) {
		t.Helper()
		body, ok := recipes[target]
		if !ok {
			t.Errorf("Makefile has no %s target", target)
			return
		}
		text := strings.Join(body, "\n")
		for _, w := range want {
			if !strings.Contains(text, w) {
				t.Errorf("make %s recipe %q does not contain %q", target, text, w)
			}
		}
	}
	contains("dogfood", "scripts/dogfood.sh")
	contains("dev-install", "GOWORK=off", "go install", "-mod=vendor", "./cmd/qsdev")
	contains("bench", "go test", "-bench")

	var removed []string
	for _, line := range recipes["clean"] {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "rm" {
			for _, arg := range f[1:] {
				removed = append(removed, strings.TrimSuffix(strings.TrimPrefix(arg, "./"), "/"))
			}
		}
	}
	for _, want := range []string{"qsdev", "bin", "dist"} {
		if !slices.Contains(removed, want) {
			t.Errorf("make clean removes %v, want it to include %s", removed, want)
		}
	}
	for _, target := range []string{"bench", "dogfood", "dev-install"} {
		if !slices.Contains(recipes[".PHONY"], target) {
			t.Errorf(".PHONY = %v, want it to include %s", recipes[".PHONY"], target)
		}
	}
}
