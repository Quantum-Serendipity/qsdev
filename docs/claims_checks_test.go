package docs_test

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Quantum-Serendipity/qsdev/internal/archtest"
)

// fixtureTests is the Go test source, by package directory, every fixture
// claims file may bind to.
var fixtureTests = map[string]string{
	"x": `package x

import "testing"

func TestGuard(t *testing.T) { t.Log("ok"); ping(t) }

func ping(t *testing.T) { pong(t) }

func pong(t *testing.T) { ping(t) }

func TestFlaky(t *testing.T) {
	t.Run("sub", func(t *testing.T) { t.Skipf("flaky on %s", "ci") })
}

func TestHelped(t *testing.T) { newEnv(t) }

func newEnv(t *testing.T) { requireUnix(t) }

func requireUnix(t *testing.T) { t.SkipNow() }

func TestDup(t *testing.T) {}
`,
	"y": `package y

import "testing"

func TestDup(t *testing.T) {}
`,
}

const fixtureReadme = "# Tool\r\n\r\nThe  guard blocks\r\ncurl | sh.\r\n\r\n```\r\nit blocks nothing here\r\n```\r\n"

// fixtureClaims renders a claims file over fixtureReadme with the given
// claims and retracted entries, in YAML.
func fixtureClaims(claims, retracted string) []byte {
	return []byte("version: 1\nscan: [README.md, docs/*.md]\nkeywords: [blocks, prevents]\n" +
		"claims:\n" + claims + "retracted:\n" + retracted)
}

const goodClaim = "  - {id: guard, text: 'The guard blocks curl | sh.', files: [README.md], tests: [TestGuard], tiers: [full]}\n"

func fixtureIndex(t *testing.T) testIndex {
	t.Helper()
	repo := &archtest.Repo{}
	for pkg, src := range fixtureTests {
		name := pkg + "/" + pkg + "_test.go"
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		repo.Files = append(repo.Files, &archtest.File{Pkg: pkg, Path: name, IsTest: true, AST: f})
	}
	return indexTests(repo)
}

func TestSecurityClaimsChecks(t *testing.T) {
	t.Parallel()
	tests := fixtureIndex(t)
	cases := []struct {
		name        string
		readme      string
		claims      []byte
		wantFailure string // "" means no failure
		wantWarning string // "" means no warning
	}{
		{
			name:   "valid claim over CRLF wrapped text",
			readme: fixtureReadme,
			claims: fixtureClaims(goodClaim, "  []\n"),
		},
		{
			name:        "missing test",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "TestGuard", "TestGone", 1), "  []\n"),
			wantFailure: `claim "guard": test TestGone does not exist`,
		},
		{
			name:        "skipped test",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "TestGuard", "TestFlaky", 1), "  []\n"),
			wantFailure: `claim "guard": test TestFlaky calls t.Skip`,
		},
		{
			name:        "retracted phrase present, fenced code included",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim, "  - {phrase: \"blocks\\x20nothing\", reason: overclaim, finding: U00-01}\n"),
			wantFailure: `README.md:7: retracted phrase "blocks nothing" matched "blocks nothing" (finding U00-01: overclaim)`,
		},
		{
			name:        "test skips through a helper chain",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "TestGuard", "TestHelped", 1), "  []\n"),
			wantFailure: `claim "guard": test TestHelped calls t.Skip via requireUnix`,
		},
		{
			name:        "test name ambiguous across packages",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "TestGuard", "TestDup", 1), "  []\n"),
			wantFailure: `claim "guard": test TestDup is ambiguous (packages `,
		},
		{
			name:   "package-qualified test name",
			readme: fixtureReadme,
			claims: fixtureClaims(strings.Replace(goodClaim, "TestGuard", "y.TestDup", 1), "  []\n"),
		},
		{
			name:        "retracted pattern through markdown emphasis",
			readme:      fixtureReadme + "Rules **cannot** be skipped.\n",
			claims:      fixtureClaims(goodClaim, "  - {pattern: \"\\\\bcan(not|'t) be skipped\", reason: overclaim, finding: U00-03}\n"),
			wantFailure: `README.md:9: retracted pattern "\\bcan(not|'t) be skipped" matched "cannot be skipped" (finding U00-03: overclaim)`,
		},
		{
			name:        "retracted pattern with a typographic apostrophe",
			readme:      fixtureReadme + "Rules can\u2019t be skipped.\n",
			claims:      fixtureClaims(goodClaim, "  - {pattern: \"\\\\bcan(not|'t) be skipped\", reason: overclaim, finding: U00-03}\n"),
			wantFailure: `README.md:9: retracted pattern "\\bcan(not|'t) be skipped" matched "can't be skipped"`,
		},
		{
			name:        "retracted pattern invalid",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim, "  - {pattern: '(', reason: overclaim, finding: U00-04}\n"),
			wantFailure: `retracted #1: pattern "(":`,
		},
		{
			name:        "retracted entry with both phrase and pattern",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim, "  - {phrase: x, pattern: 'z{2}', reason: overclaim, finding: U00-05}\n"),
			wantFailure: "retracted #1 needs one of phrase or pattern",
		},
		{
			name:        "retracted phrase wrapped across lines, any case",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim, "  - {phrase: \"TOOL\\x20THE GUARD\", reason: overclaim, finding: U00-02}\n"),
			wantFailure: `README.md:1: retracted phrase "TOOL THE GUARD" matched "tool the guard" (finding U00-02: overclaim)`,
		},
		{
			name:        "retracted phrase spelled out in the claims file itself",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim, "  - {phrase: 'guard stops', reason: overclaim, finding: U00-06}\n"),
			wantFailure: `security-claims.yaml:7: retracted phrase "guard stops" matched "guard stops" (finding U00-06: overclaim)`,
		},
		{
			name:        "retracted pattern split by a markdown link target",
			readme:      fixtureReadme + "Built with [SLSA](https://slsa.dev/) Level 9.\n",
			claims:      fixtureClaims(goodClaim, "  - {pattern: 'slsa level[ ]9', reason: overclaim, finding: U00-07}\n"),
			wantFailure: `README.md:9: retracted pattern "slsa level[ ]9" matched "slsa level 9"`,
		},
		{
			name:        "claim text drifted",
			readme:      strings.Replace(fixtureReadme, "guard blocks", "guard stops", 1),
			claims:      fixtureClaims(goodClaim, "  []\n"),
			wantFailure: `claim "guard": text not found in README.md`,
		},
		{
			name:        "duplicate id",
			readme:      fixtureReadme,
			claims:      fixtureClaims(goodClaim+goodClaim, "  []\n"),
			wantFailure: `duplicate claim id "guard"`,
		},
		{
			name:        "unknown field",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "tiers:", "tier:", 1), "  []\n"),
			wantFailure: "field tier not found",
		},
		{
			name:        "uncovered keyword sentence warns only",
			readme:      fixtureReadme + "Hooks prevents drift.\n",
			claims:      fixtureClaims(goodClaim, "  []\n"),
			wantWarning: "README.md:9: uncovered security sentence: Hooks prevents drift.",
		},
		{
			name:        "claim without tests warns only",
			readme:      fixtureReadme,
			claims:      fixtureClaims(strings.Replace(goodClaim, "[TestGuard]", "[]", 1), "  []\n"),
			wantWarning: `claim "guard": no test bound yet`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := fstest.MapFS{
				"README.md":      {Data: []byte(tc.readme)},
				"docs/extra.md":  {Data: []byte("Nothing to see.\n")},
				"docs/claims.go": {Data: []byte("package docs\n")},
			}
			res, err := checkClaims(root, tc.claims, tests)
			if err != nil {
				if tc.wantFailure == "" || !strings.Contains(err.Error(), tc.wantFailure) {
					t.Fatalf("checkClaims: %v", err)
				}
				return
			}
			assertOne(t, "failure", res.failures, tc.wantFailure)
			assertOne(t, "warning", res.warnings, tc.wantWarning)
		})
	}
}

// assertOne requires got to be exactly one entry containing want, or empty
// when want is "".
func assertOne(t *testing.T, kind string, got []string, want string) {
	t.Helper()
	switch {
	case want == "" && len(got) != 0:
		t.Errorf("unexpected %ss: %q", kind, got)
	case want != "" && (len(got) != 1 || !strings.Contains(got[0], want)):
		t.Errorf("%ss = %q, want one containing %q", kind, got, want)
	}
}

// TestSecurityClaimsRetractedVariants feeds variants of every withdrawn
// overclaim to the retractions in the real claims file; each must be caught.
// The sentences are split with "+" so this file matches no overclaim grep.
func TestSecurityClaimsRetractedVariants(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(claimsFile)
	if err != nil {
		t.Fatalf("reading %s: %v", claimsFile, err)
	}
	doc, err := parseClaims(data)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		line    string
		finding string
	}{
		{"Ships with 14" + " layers of defense.", "U27-01"},
		{"Ships with 14" + " defense layers.", "U27-01"},
		{"Ships with fourteen" + " layers of defense.", "U27-01"},
		{"Everything is configured out of the" + " box.", "U27-01"},
		{"Everything is configured out-of-the" + "-box.", "U27-01"},
		{"The policy engine" + " evaluates tool calls.", "U27-01"},
		{"Policies live in .qsdev/" + "policy/*.yaml.", "U27-01"},
		{"MCP file tools are path-checked by the confused" + "-deputy hook.", "U27-01"},
		{"Self-protection rules cannot" + " be bypassed.", "U18-14"},
		{"Self-protection rules can not" + " be bypassed.", "U18-14"},
		{"Deny rules can\u2019t" + " be circumvented.", "U18-14"},
		{"The hook runs with fail-closed" + " semantics.", "U18-14"},
		{"Hooks run in" + " a sandbox (bubblewrap).", "U05-01"},
		{"All hooks are executed inside" + " the sandbox.", "U05-01"},
		{"Package-guard hooks are" + " sandboxed.", "U05-01"},
		{"Releases are built with [SLSA Level" + " 3](https://slsa.dev/) provenance.", "U26-06"},
		{"Built with [SLSA](https://slsa.dev/) Level" + " 3 provenance.", "U26-06"},
		{"Provenance meets SLSA" + " L3.", "U26-06"},
		{"Version-Sentinel guards" + " dependency changes.", "U22-03"},
		{"Enhanced gates packages for 168" + " hours.", "U27-03"},
		{"Enhanced gates packages for 1 week (168" + "h).", "U27-03"},
		{"Strict gates packages for 336" + " hours.", "U27-03"},
		{"The OS sandbox prevents" + " bypassing the rules.", "XS-N2"},
		{"Harden-runner constrains" + " network egress.", "U26-V01"},
		{"It provides policy enforcement" + " for GitHub Actions.", "U26-V01"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			t.Parallel()
			var res claimsResult
			checkRetracted([]string{"README.md"}, map[string][]string{"README.md": {tc.line}}, doc, &res)
			for _, f := range res.failures {
				if strings.Contains(f, "(finding "+tc.finding+":") {
					return
				}
			}
			t.Errorf("no %s retraction caught %q; failures: %q", tc.finding, tc.line, res.failures)
		})
	}
}
