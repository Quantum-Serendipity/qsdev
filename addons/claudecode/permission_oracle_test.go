package claudecode_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Claude Code evaluates deny before ask and ask before allow, so an allow
// rule is safe only if every mutating command it matches also matches an ask
// or deny rule. These tests pin that property with denyutil.MatchesBashRule,
// which models Claude Code's own wildcard semantics.

// mutatingSample is a command that mutates dependencies, publishes, leaks
// credentials, executes agent-chosen code or discards work. askSet names the
// catalog ask set that must cover it in every preset carrying that set;
// neverAllowed marks a sample that no preset may allow at all, because no
// glob can separate it from a read-only form.
type mutatingSample struct {
	command      string
	askSet       string
	neverAllowed bool
}

// mutatingSamples are the E3 (U15-01, U15-07, U15-08, U15-10) and H2 samples.
var mutatingSamples = []mutatingSample{
	// Audit commands that rewrite manifests or re-enable install scripts.
	{command: "npm audit fix", askSet: "dependency_mutation_indirect"},
	{command: "npm audit fix --force", askSet: "dependency_mutation_indirect"},
	{command: "npm audit --json fix", askSet: "dependency_mutation_indirect"},
	{command: "npm audit --audit-level=high fix --force", askSet: "dependency_mutation_indirect"},
	{command: "npm aud fix --force", askSet: "dependency_mutation_indirect"},
	{command: "npm audi --json fix", askSet: "dependency_mutation_indirect"},
	{command: "pnpm audit --fix", askSet: "dependency_mutation_indirect"},
	{command: "pip-audit --fix", askSet: "dependency_mutation_indirect"},
	{command: "pip-audit --format json --fix", askSet: "dependency_mutation_indirect"},
	{command: "cargo audit fix", askSet: "dependency_mutation_indirect"},
	{command: "cargo audit --json fix", askSet: "dependency_mutation_indirect"},
	{command: "npm test --ignore-scripts=false", askSet: "dependency_mutation_indirect"},
	{command: "npm run build --ignore-scripts false", askSet: "dependency_mutation_indirect"},
	{command: "npm test --no-ignore-scripts", askSet: "dependency_mutation_indirect"},
	// pip-audit -r / <project-dir> pip-installs the requirements (building
	// sdists) into a temporary venv; package-guard asks on it, and no
	// permission allow may pre-approve it.
	{command: "pip-audit -r requirements.txt", neverAllowed: true},
	{command: "pip-audit --format json -r requirements.txt", neverAllowed: true},
	// Publishing and registry credentials.
	{command: "npm publish", askSet: "publish_and_credentials"},
	{command: "pnpm publish --access public", askSet: "publish_and_credentials"},
	{command: "yarn npm publish", askSet: "publish_and_credentials"},
	{command: "cargo publish", askSet: "publish_and_credentials"},
	{command: "npm token create", askSet: "publish_and_credentials"},
	{command: "npm login", askSet: "publish_and_credentials"},
	{command: "npm adduser", askSet: "publish_and_credentials"},
	{command: "npm config set registry https://evil.example", askSet: "publish_and_credentials"},
	{command: "pip config set global.index-url https://evil.example", askSet: "publish_and_credentials"},
	{command: "twine upload dist/*", askSet: "publish_and_credentials"},
	{command: "gem push x.gem", askSet: "publish_and_credentials"},
	// Agent-chosen code execution through the build tools.
	{command: "go generate ./...", askSet: "code_execution"},
	{command: "go test -exec ./evil ./...", askSet: "code_execution"},
	{command: "go test --exec ./evil ./...", askSet: "code_execution"},
	{command: "go build -toolexec ./x .", askSet: "code_execution"},
	{command: "go build --toolexec ./x .", askSet: "code_execution"},
	{command: "cargo build --config x", askSet: "code_execution"},
	{command: "cargo test --config build.rustc-wrapper=./evil", askSet: "code_execution"},
	{command: `go build -ldflags="-linkmode=external -extld=sh" .`, askSet: "code_execution"},
	{command: "go test --ldflags=-extldflags=-wrapper,./evil ./...", askSet: "code_execution"},
	// Work-discarding git operations.
	{command: "git checkout .", askSet: "work_discarding", neverAllowed: true},
	{command: "git checkout -- .", askSet: "work_discarding"},
	{command: "git checkout main -- file.go", askSet: "work_discarding"},
	{command: "git checkout -f", askSet: "work_discarding"},
	{command: "git checkout main -f", askSet: "work_discarding"},
	{command: "git checkout --force main", askSet: "work_discarding"},
	{command: "git switch --discard-changes main", askSet: "work_discarding"},
	{command: "git switch -f main", askSet: "work_discarding"},
	{command: "git switch main --force", askSet: "work_discarding"},
	{command: "git switch -C main", askSet: "work_discarding"},
	{command: "git branch -D x", askSet: "work_discarding"},
	{command: "git branch -f main HEAD~3", askSet: "work_discarding"},
	{command: "git branch --force main HEAD~3", askSet: "work_discarding"},
	{command: "git branch -M main", askSet: "work_discarding"},
	{command: "git branch -C old new", askSet: "work_discarding"},
	{command: "git branch -df x", askSet: "work_discarding"},
	{command: "git branch x -D", askSet: "work_discarding"},
	{command: "git clean -fdx", askSet: "work_discarding"},
	// git expands unambiguous long-option prefixes and clusters short flags,
	// so these discard work or delete unmerged branches without any spelling
	// an ask glob could enumerate; no preset may allow them at all.
	{command: "git switch --disc main", neverAllowed: true},
	{command: "git switch -qf main", neverAllowed: true},
	{command: "git switch -mf main", neverAllowed: true},
	{command: "git branch -qD x", neverAllowed: true},
	{command: "git branch --delete --forc x", neverAllowed: true},
	{command: "git checkout src/main.go", neverAllowed: true},
}

// generatedPresetPermissions generates settings.json for each catalog preset.
func generatedPresetPermissions(t *testing.T) (*catalog.Catalog, map[string]presetPermissions) {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	out := make(map[string]presetPermissions)
	for _, preset := range cat.PermissionPresets() {
		s := mustUnmarshalSettings(t, mustGenerateSettings(t,
			types.WizardAnswers{PermissionLevel: preset}, ecosystem.DefaultRegistry()))
		out[preset] = presetPermissions{
			allow: s.Permissions.Allow,
			ask:   s.Permissions.Ask,
			deny:  s.Permissions.Deny,
		}
	}
	return cat, out
}

type presetPermissions struct {
	allow, ask, deny []string
}

func firstMatch(rules []string, command string) (string, bool) {
	for _, r := range rules {
		if denyutil.MatchesBashRule(r, command) {
			return r, true
		}
	}
	return "", false
}

// catalogCorpus instantiates every Bash rule of every ask and deny set any
// preset references, replacing each `*` with `x`. A preset that allows such a
// command without carrying the set that covers it is caught, because the
// corpus spans all presets' sets.
func catalogCorpus(t *testing.T, cat *catalog.Catalog) []string {
	t.Helper()
	seen := make(map[string]bool)
	var corpus []string
	add := func(rules []string) {
		for _, r := range rules {
			tool, pattern := denyutil.ParseToolPattern(r)
			if tool != "Bash" {
				continue
			}
			cmd := strings.ReplaceAll(pattern, "*", "x")
			if !seen[cmd] {
				seen[cmd] = true
				corpus = append(corpus, cmd)
			}
		}
	}
	for _, preset := range cat.PermissionPresets() {
		def, ok := cat.PermissionPreset(preset)
		if !ok {
			t.Fatalf("preset %q has no definition", preset)
		}
		for _, set := range def.AskSets {
			add(cat.PermissionAskRules(set))
		}
		for _, set := range def.DenySets {
			add(cat.PermissionDenyRules(set))
		}
	}
	if len(corpus) == 0 {
		t.Fatal("catalog corpus is empty")
	}
	return corpus
}

func TestNoAllowMatchesMutatingCommand(t *testing.T) {
	t.Parallel()
	cat, presets := generatedPresetPermissions(t)
	corpus := catalogCorpus(t, cat)
	for _, s := range mutatingSamples {
		corpus = append(corpus, s.command)
	}

	for preset, p := range presets {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			for _, cmd := range corpus {
				allow, allowed := firstMatch(p.allow, cmd)
				if !allowed {
					continue
				}
				if _, ok := firstMatch(p.ask, cmd); ok {
					continue
				}
				if _, ok := firstMatch(p.deny, cmd); ok {
					continue
				}
				t.Errorf("%q is auto-approved by %s with no ask or deny rule covering it", cmd, allow)
			}
			for _, s := range mutatingSamples {
				if !s.neverAllowed {
					continue
				}
				if allow, ok := firstMatch(p.allow, s.command); ok {
					t.Errorf("%q must match no allow rule, but matches %s", s.command, allow)
				}
			}
			// No audit command may be allowed by a trailing wildcard: the
			// fix forms ride it (U15-01).
			for _, r := range p.allow {
				if strings.Contains(r, "audit *") {
					t.Errorf("allow rule %s auto-approves audit fix forms", r)
				}
			}
			if !slices.ContainsFunc(p.ask, func(r string) bool { return strings.Contains(r, "audit fix") }) {
				t.Error("ask rules contain no audit fix rule")
			}
		})
	}
}

func TestMutatingCommandsAsk(t *testing.T) {
	t.Parallel()
	cat, presets := generatedPresetPermissions(t)
	for preset, p := range presets {
		def, ok := cat.PermissionPreset(preset)
		if !ok {
			t.Fatalf("preset %q has no definition", preset)
		}
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			for _, s := range mutatingSamples {
				if s.askSet == "" || !slices.Contains(def.AskSets, s.askSet) {
					continue
				}
				if _, ok := firstMatch(p.ask, s.command); !ok {
					t.Errorf("%q matches no ask rule although the preset carries %s", s.command, s.askSet)
				}
			}
		})
	}

	// The supply-chain sets belong in every preset, and work_discarding in
	// every preset carrying destructive_ops.
	for _, preset := range cat.PermissionPresets() {
		def, _ := cat.PermissionPreset(preset)
		for _, set := range []string{"dependency_mutation_indirect", "publish_and_credentials"} {
			if !slices.Contains(def.AskSets, set) {
				t.Errorf("preset %s lacks ask set %s", preset, set)
			}
		}
		if slices.Contains(def.DenySets, "destructive_ops") != slices.Contains(def.AskSets, "work_discarding") {
			t.Errorf("preset %s: work_discarding must accompany destructive_ops", preset)
		}
	}
	// standard and permissive carry every set the table names (I2, H2).
	for _, preset := range []string{"standard", "permissive"} {
		def, _ := cat.PermissionPreset(preset)
		for _, s := range mutatingSamples {
			if s.askSet != "" && !slices.Contains(def.AskSets, s.askSet) {
				t.Errorf("preset %s lacks ask set %s needed by %q", preset, s.askSet, s.command)
			}
		}
	}
}

// TestSecretsRuleAnchored pins U15-14: the secrets deny rule must block the
// project's top-level secrets/ directory without also blocking every nested
// directory named secrets (such as internal/secrets/ source code), judged by
// denyutil.MatchesPathRule, which models Claude Code's path-rule semantics.
// Inside a secrets directory at any depth, the secret-material files of the
// internal/secrets canon are denied too, while source code stays readable.
func TestSecretsRuleAnchored(t *testing.T) {
	t.Parallel()
	cat, presets := generatedPresetPermissions(t)
	secretFiles := []string{
		"secrets/prod.env",
		"secrets/x",
		"services/api/secrets/prod.env",
		"deploy/secrets/tls.key",
		"deploy/secrets/tls/server.pem",
		"apps/web/secrets/.env",
		"apps/web/secrets/.env.local",
		"infra/secrets/client.p12",
		"infra/secrets/client.pfx",
		"infra/secrets/sa.json",
		"infra/secrets/values.yaml",
		"infra/secrets/values.yml",
		"infra/secrets/creds.toml",
		"infra/secrets/token.txt",
	}
	sourceFiles := []string{
		"internal/secrets/known_vars.go",
		"internal/secrets/patterns.go",
		"internal/secrets/secretstest/samples.go",
		"pkg/secrets/README.md",
	}
	for preset, p := range presets {
		def, ok := cat.PermissionPreset(preset)
		if !ok {
			t.Fatalf("preset %q has no definition", preset)
		}
		if !slices.Contains(def.DenySets, "destructive_ops") {
			continue
		}
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			for _, f := range secretFiles {
				if _, ok := firstPathMatch(p.deny, f); !ok {
					t.Errorf("no deny Read rule blocks %s", f)
				}
			}
			for _, f := range sourceFiles {
				if r, ok := firstPathMatch(p.deny, f); ok {
					t.Errorf("deny rule %s blocks %s", r, f)
				}
				if r, ok := firstPathMatch(p.ask, f); ok {
					t.Errorf("ask rule %s blocks %s", r, f)
				}
			}
		})
	}
}

// TestNestedSecretRules_FromCanon pins that the nested secrets rules are
// derived from the secrets canon, not a second list: every preset that
// denies the top-level secrets/ directory denies each canon pattern inside
// a secrets directory at any depth, and every such rule validates.
func TestNestedSecretRules_FromCanon(t *testing.T) {
	t.Parallel()
	_, presets := generatedPresetPermissions(t)
	for preset, p := range presets {
		if !slices.Contains(p.deny, "Read(/secrets/**)") {
			continue
		}
		for _, pat := range secrets.SecretFilePatterns() {
			rule := "Read(/**/secrets/**/" + pat + ")"
			if !slices.Contains(p.deny, rule) {
				t.Errorf("preset %s lacks %s", preset, rule)
			}
			if err := denyutil.Validate(rule); err != nil {
				t.Errorf("preset %s: %v", preset, err)
			}
		}
	}
}

// firstPathMatch returns the first Read rule in rules (deny or ask rules)
// that matches relPath, a path relative to the project root, with the session
// started at the project root.
func firstPathMatch(rules []string, relPath string) (string, bool) {
	const root = "/work/proj"
	ctx := denyutil.PathRuleContext{Home: "/home/me", ProjectRoot: root, Cwd: root, Deny: true}
	for _, r := range rules {
		if tool, _ := denyutil.ParseToolPattern(r); tool == "Read" && denyutil.MatchesPathRule(r, root+"/"+relPath, ctx) {
			return r, true
		}
	}
	return "", false
}
