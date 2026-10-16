package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// secretPatternAccepted are entries CheckSecretPattern must accept; the
// Python cross-check compiles each of them with python3 re as well.
var secretPatternAccepted = []string{
	`corp_[a-z0-9]{20,40}`,
	`(?i)corp_(?P<secret>[a-z0-9]{20})`,
	`[-a-z_.]{8,}`,
	`tok\.[A-Z]{4}`,
	`x y`,
	`a+?b*?c??d{2}?e{2,}?f{2,3}?`,
	`(?:ab|cd)+(ef)?^$\b\B\d\D\w\W\s\S\t\n\r.`,
	`[^\]\[\\a-z\d\s-]`,
	`\{\}\(\)\*\+\?\|\^\$\.\-\/\_\#`,
}

func TestCheckSecretPattern(t *testing.T) {
	t.Parallel()
	// Invisible characters are built from their code points so the source
	// holds none.
	lineSeparator, zeroWidthSpace := string(rune(0x2028)), string(rune(0x200b))
	rejected := []struct{ name, pattern string }{
		{"empty", ""},
		{"leading space", " x"},
		{"trailing tab", "x\t"},
		{"trailing carriage return", "x\r"},
		{"embedded newline", "x\n y"},
		{"unclosed class", "corp_["},
		{"lookahead", "(?=x)"},
		{"negative lookahead", "(?!x)"},
		{"lookbehind", "(?<=x)y"},
		{"bare named group", "(?<n>x)"},
		{"other group name", "(?P<n>x)"},
		{"repeated secret group", "(?P<secret>a)(?P<secret>b)"},
		{"backreference", `a\1`},
		{"unicode class", `\pL+`},
		{"end of text", `\z`},
		{"quoted literal", `\Qa\E`},
		{"posix class", "[[:alpha:]]+"},
		{"range from escape", `[\d-x]`},
		{"word boundary in class", `[\b]`},
		{"open lower bound", "a{,3}"},
		{"unescaped brace", "a{"},
		{"mid-pattern flag", "a(?i)b"},
		{"line separator", "x" + lineSeparator + "y"},
		{"zero-width space", "x" + zeroWidthSpace + "y"},
		{"too long", strings.Repeat("a", maxSecretPatternLen+1)},
		{"nothing to repeat", "*a"},
		{"repeated anchor", `\b*`},
		{"nested quantifier", "a**"},
		{"possessive quantifier", "a*+"},
		{"unbalanced close", "a)"},
		{"unclosed group", "(a"},
		{"empty class", "[]a]"},
		{"middle dash", "[a-z-_]"},
		{"unescaped close bracket", "a]"},
		{"trailing backslash", `a\`},
		{"reversed range", "[z-a]"},
		{"repeat count too large", "a{1001}"},
		{"invalid UTF-8", "a\xffb"},
	}
	for _, tt := range rejected {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			t.Parallel()
			if err := CheckSecretPattern(tt.pattern); !errors.Is(err, ErrSecretPattern) {
				t.Errorf("CheckSecretPattern(%q) = %v, want an error wrapping ErrSecretPattern", tt.pattern, err)
			}
		})
	}
	for _, p := range secretPatternAccepted {
		t.Run("accept/"+p, func(t *testing.T) {
			t.Parallel()
			if err := CheckSecretPattern(p); err != nil {
				t.Errorf("CheckSecretPattern(%q) = %v, want nil", p, err)
			}
		})
	}
	if err := CheckSecretPattern(strings.Repeat("a", maxSecretPatternLen)); err != nil {
		t.Errorf("CheckSecretPattern(%d bytes) = %v, want nil", maxSecretPatternLen, err)
	}
}

// TestCheckSecretPattern_AcceptedCompileInPython checks that every entry the
// Go check accepts also compiles with Python's re, the engine scan-secrets.py
// runs it with.
func TestCheckSecretPattern_AcceptedCompileInPython(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	input, err := json.Marshal(secretPatternAccepted)
	if err != nil {
		t.Fatal(err)
	}
	const script = "import json, re, sys\nfor p in json.load(sys.stdin):\n    re.compile(p)\n"
	cmd := exec.CommandContext(t.Context(), python, "-c", script)
	cmd.Stdin = bytes.NewReader(input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("python3 re.compile failed: %v\n%s", err, out)
	}
}
