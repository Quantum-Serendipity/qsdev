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
)

// annotation is a fixture expectation: the line after a "ruleid: <id>"
// comment must be reported by rule <id>, the line after "ok: <id>" must not.
type annotation struct {
	file   string
	line   int
	ruleID string
	want   bool
}

// scannerCommand returns the argv prefix for an available OpenGrep-compatible
// scanner (opengrep, or semgrep, which shares the rule format), or nil.
func scannerCommand() []string {
	if _, err := exec.LookPath("opengrep"); err == nil {
		return []string{"opengrep", "scan"}
	}
	if _, err := exec.LookPath("semgrep"); err == nil {
		return []string{"semgrep", "scan", "--metrics=off", "--disable-version-check"}
	}
	return nil
}

// TestCoreRules_FixtureAnnotations runs the injection rules against the
// annotated fixtures under core/testdata, so a sink pattern that misses a
// canonical vulnerable call shape (bind arguments, *Context variants, Node
// callbacks) fails the build rather than silently passing the scan gate.
func TestCoreRules_FixtureAnnotations(t *testing.T) {
	scanner := scannerCommand()
	if scanner == nil {
		t.Skip("neither opengrep nor semgrep is available")
	}

	fixtures, _ := filepath.Glob(filepath.Join("core", "testdata", "*", "*"))
	var annotated []string
	var expectations []annotation
	for _, f := range fixtures {
		anns := readAnnotations(t, f)
		if len(anns) > 0 {
			annotated = append(annotated, f)
			expectations = append(expectations, anns...)
		}
	}
	if len(expectations) == 0 {
		t.Fatal("no annotated fixtures found")
	}

	args := append(scanner[1:], "--json", "--quiet",
		"--config", filepath.Join("core", "injection", "go-sql-injection.yaml"),
		"--config", filepath.Join("core", "injection", "command-injection.yaml"))
	args = append(args, annotated...)
	out, err := exec.Command(scanner[0], args...).Output()
	if err != nil && len(out) == 0 {
		t.Fatalf("%s failed: %v", scanner[0], err)
	}

	var report struct {
		Results []struct {
			CheckID string `json:"check_id"`
			Path    string `json:"path"`
			Start   struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("decoding scanner output: %v\n%s", err, out)
	}

	found := map[string]bool{}
	for _, r := range report.Results {
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
		for prefix, want := range map[string]bool{"// ruleid:": true, "// ok:": false} {
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
