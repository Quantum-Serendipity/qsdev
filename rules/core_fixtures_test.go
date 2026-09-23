package rules_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/rules"
)

// annotation is a fixture expectation: the line after a "ruleid: <id>"
// comment must be reported by rule <id>, the line after "ok: <id>" must not.
type annotation struct {
	file   string
	line   int
	ruleID string
	want   bool
}

// scanReport is the subset of the OpenGrep/Semgrep --json output the tests use.
type scanReport struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
		} `json:"start"`
	} `json:"results"`
	Errors []struct {
		Type    any    `json:"type"`
		Message string `json:"message"`
	} `json:"errors"`
}

// scannerCommand returns the argv prefix for an available OpenGrep-compatible
// scanner (opengrep, or semgrep, which shares the rule format), or nil.
func scannerCommand() []string {
	if _, err := exec.LookPath("opengrep"); err == nil {
		return []string{"opengrep", "scan"}
	}
	if _, err := exec.LookPath("semgrep"); err == nil {
		// Scanning is semgrep's default command; some packagings (osemgrep)
		// reject an explicit "scan" subcommand, so none is passed.
		return []string{"semgrep", "--metrics=off", "--disable-version-check"}
	}
	return nil
}

// requireScanner returns the scanner argv, skipping the test when none is
// installed. CI sets QSDEV_REQUIRE_RULE_SCANNER so that a missing scanner
// fails instead of silently skipping the rule checks.
func requireScanner(t *testing.T) []string {
	t.Helper()
	scanner := scannerCommand()
	if scanner == nil {
		if os.Getenv("QSDEV_REQUIRE_RULE_SCANNER") != "" {
			t.Fatal("QSDEV_REQUIRE_RULE_SCANNER is set but neither opengrep nor semgrep is on PATH")
		}
		t.Skip("neither opengrep nor semgrep is available")
	}
	return scanner
}

// ruleConfigArgs returns a --config argument for every deliverable rule file,
// exactly the set rules.CoreRuleFiles ships (no manifest, no testdata).
func ruleConfigArgs(t *testing.T) []string {
	t.Helper()
	files, err := rules.CoreRuleFiles()
	if err != nil {
		t.Fatalf("loading rule library: %v", err)
	}
	var args []string
	for _, f := range files {
		args = append(args, "--config", filepath.Join("core", filepath.FromSlash(f.RelPath)))
	}
	return args
}

// runScanner runs the scanner with --json over targets and fails the test on
// any scanner error, including a rule the engine skipped as unparsable: a
// skipped rule is lost coverage that would otherwise surface only as a
// warning in scan output.
func runScanner(t *testing.T, scanner []string, extra ...string) scanReport {
	t.Helper()
	args := append(append([]string{}, scanner[1:]...), "--json", "--quiet")
	args = append(args, ruleConfigArgs(t)...)
	args = append(args, extra...)
	out, err := exec.Command(scanner[0], args...).Output()
	if err != nil && len(out) == 0 {
		t.Fatalf("%s failed: %v", scanner[0], err)
	}
	var report scanReport
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("decoding scanner output: %v\n%s", err, out)
	}
	for _, e := range report.Errors {
		t.Errorf("%s reported an error: %v: %s", scanner[0], e.Type, e.Message)
	}
	return report
}

// TestCoreRules_Validate fails when any shipped rule does not parse. Engines
// drop such a rule with only a warning, silently losing its coverage.
func TestCoreRules_Validate(t *testing.T) {
	scanner := requireScanner(t)
	args := append(append([]string{}, scanner[1:]...), "--validate")
	args = append(args, ruleConfigArgs(t)...)
	if out, err := exec.Command(scanner[0], args...).CombinedOutput(); err != nil {
		t.Fatalf("%s --validate failed: %v\n%s", scanner[0], err, out)
	}
}

// TestCoreRules_FixtureAnnotations runs every rule against the annotated
// fixtures under core/testdata, so a sink pattern that misses a canonical
// vulnerable call shape (bind arguments, *Context variants, Node callbacks,
// SvelteKit form actions) fails the build rather than silently passing the
// scan gate. The fixtures under core/testdata/safe must produce no findings
// at all: they exist to prove the rules do not fire on safe code.
func TestCoreRules_FixtureAnnotations(t *testing.T) {
	scanner := requireScanner(t)

	fixtures, _ := filepath.Glob(filepath.Join("core", "testdata", "*", "*"))
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found")
	}
	var expectations []annotation
	for _, f := range fixtures {
		expectations = append(expectations, readAnnotations(t, f)...)
	}
	if len(expectations) == 0 {
		t.Fatal("no annotated fixtures found")
	}

	report := runScanner(t, scanner, fixtures...)

	safeDir := filepath.Join("core", "testdata", "safe") + string(filepath.Separator)
	found := map[string]bool{}
	for _, r := range report.Results {
		if strings.HasPrefix(filepath.Clean(r.Path), safeDir) {
			t.Errorf("%s:%d: safe fixture reported by %s", r.Path, r.Start.Line, r.CheckID)
		}
		for _, e := range expectations {
			// Scanners may prefix rule IDs with the config path.
			if strings.HasSuffix(r.CheckID, e.ruleID) && filepath.Clean(r.Path) == filepath.Clean(e.file) && r.Start.Line == e.line {
				found[key(e)] = true
			}
		}
	}
	for _, e := range expectations {
		if got := found[key(e)]; got != e.want {
			verb := "not reported"
			if got {
				verb = "reported (false positive)"
			}
			t.Errorf("%s:%d: rule %s %s", e.file, e.line, e.ruleID, verb)
		}
	}
}

func key(a annotation) string { return fmt.Sprintf("%s:%d:%s", a.file, a.line, a.ruleID) }

// annotationPrefixes maps the fixture comment markers (C-style and
// hash-comment languages) to whether the next line must be reported.
var annotationPrefixes = map[string]bool{
	"// ruleid:": true, "// ok:": false,
	"# ruleid:": true, "# ok:": false,
}

func readAnnotations(t *testing.T, path string) []annotation {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	var anns []annotation
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		text := strings.TrimSpace(scanner.Text())
		for prefix, want := range annotationPrefixes {
			if id, ok := strings.CutPrefix(text, prefix); ok {
				anns = append(anns, annotation{file: path, line: lineNo + 1, ruleID: strings.TrimSpace(id), want: want})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return anns
}
