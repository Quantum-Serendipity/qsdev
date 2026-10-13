package testutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// GovernedSkip is a test that skipped although its output names a tool that
// a required switch governs: a skip RequireTool or Unavailable would have
// turned into a failure, so it bypassed them.
type GovernedSkip struct {
	Package string
	Test    string
	Tool    string
	Switch  Switch
}

func (g GovernedSkip) String() string {
	return fmt.Sprintf("%s %s skipped naming %s, which %s=1 requires", g.Package, g.Test, g.Tool, g.Switch)
}

// RequiredSwitches returns the switches that are on in the environment.
func RequiredSwitches() []Switch {
	var on []Switch
	for _, s := range Switches() {
		if s.Required() {
			on = append(on, s)
		}
	}
	return on
}

// testEvent is the part of a `go test -json` event FindGovernedSkips reads.
type testEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

// Characters that may delimit a tool name in a skip message. '/' bounds a
// path's last element ("/usr/bin/bwrap") or an alternative ("nix/bwrap"),
// '(' a man-page section ("nix-instantiate(1)").
const (
	mentionLeft  = " \t\"'`([:,=/"
	mentionRight = " \t\"'`)]:,;!?/("
)

// mentions reports whether text names name as a word, ignoring case:
// "Nix not installed", `exec: "nix"` and "/usr/bin/bwrap not found" do, while
// "unix", "devenv.nix", "nix-parse", a "/nix/store" path element and the
// "=== RUN TestX/nix" framing lines of go test do not.
func mentions(text, name string) bool {
	name = strings.ToLower(name)
	for line := range strings.Lines(strings.ToLower(text)) {
		line = strings.TrimRight(line, "\r\n")
		if trimmed := strings.TrimLeft(line, " \t"); strings.HasPrefix(trimmed, "=== ") || strings.HasPrefix(trimmed, "--- ") {
			continue // go test framing names the test, not the reason
		}
		for i := 0; ; {
			j := strings.Index(line[i:], name)
			if j < 0 {
				break
			}
			start, end := i+j, i+j+len(name)
			if mentionBounded(line, start, end) {
				return true
			}
			i = start + 1
		}
	}
	return false
}

// mentionBounded reports whether line[start:end] stands as a word: delimited
// on both sides, a sentence-final '.' counting on the right, and not an inner
// path element such as the nix of /nix/store.
func mentionBounded(line string, start, end int) bool {
	var left, right byte
	if start > 0 {
		left = line[start-1]
		if !strings.ContainsRune(mentionLeft, rune(left)) {
			return false
		}
	}
	if end < len(line) {
		right = line[end]
		finalDot := right == '.' && (end+1 == len(line) || line[end+1] == ' ' || line[end+1] == '\t')
		if !finalDot && !strings.ContainsRune(mentionRight, rune(right)) {
			return false
		}
	}
	return left != '/' || right != '/'
}

// governedTool is a tool a required switch governs, with the names a skip
// message may give it.
type governedTool struct {
	tool  string
	sw    Switch
	names []string
}

func (g governedTool) mentionedIn(text string) bool {
	return slices.ContainsFunc(g.names, func(n string) bool { return mentions(text, n) })
}

// governedTools returns each tool of the required switches once, attributed
// to the first switch that governs it.
func governedTools(required []Switch) []governedTool {
	var tools []governedTool
	for _, s := range required {
		for _, tool := range s.Tools() {
			if !slices.ContainsFunc(tools, func(g governedTool) bool { return g.tool == tool }) {
				names := append([]string{tool}, toolAliases[tool]...)
				tools = append(tools, governedTool{tool: tool, sw: s, names: names})
			}
		}
	}
	return tools
}

// FindGovernedSkips reads a `go test -json` stream and returns every skipped
// test whose output names a tool of one of the required switches. It is an
// error for the stream to hold no test at all, so a misdirected report cannot
// pass vacuously.
func FindGovernedSkips(r io.Reader, required []Switch) ([]GovernedSkip, error) {
	tools := governedTools(required)
	type testKey struct{ pkg, test string }
	output := map[testKey]*strings.Builder{}
	var skips []GovernedSkip
	sawTest := false
	dec := json.NewDecoder(r)
	for {
		var ev testEvent
		if err := dec.Decode(&ev); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decoding test event: %w", err)
		}
		if ev.Test == "" {
			continue // package-level event
		}
		sawTest = true
		key := testKey{ev.Package, ev.Test}
		switch ev.Action {
		case "output":
			b := output[key]
			if b == nil {
				b = &strings.Builder{}
				output[key] = b
			}
			b.WriteString(ev.Output)
		case "skip":
			text := ""
			if b := output[key]; b != nil {
				text = b.String()
			}
			for _, g := range tools {
				if g.mentionedIn(text) {
					skips = append(skips, GovernedSkip{Package: ev.Package, Test: ev.Test, Tool: g.tool, Switch: g.sw})
				}
			}
			delete(output, key)
		case "pass", "fail":
			delete(output, key)
		}
	}
	if !sawTest {
		return nil, errors.New("the stream holds no test events")
	}
	return skips, nil
}
