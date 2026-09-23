package ecosystem

import (
	"fmt"
	"strings"
)

// NixLangConfig describes a devenv.nix language block and associated settings.
// Use BuildLanguageFragment to render it into a Nix fragment string.
type NixLangConfig struct {
	// EnablePath is the devenv language attribute path, e.g. "languages.go".
	EnablePath string

	// Properties are key-value pairs rendered inside the language block,
	// after the "enable = true;" line. Values are emitted verbatim (no quoting).
	// Use NixString to produce a properly quoted Nix string value.
	Properties []NixProperty

	// EnvVars are rendered as  env.KEY = VALUE;  lines after the language
	// block. Values are emitted verbatim (caller must quote if needed).
	EnvVars []NixEnvVar

	// ExtraBlocks are rendered verbatim after the language block and env
	// vars, separated by blank lines. Each entry should be a complete
	// Nix expression (indented with two leading spaces to match devenv
	// conventions).
	ExtraBlocks []string
}

// NixProperty is a single key = value line inside a language block.
type NixProperty struct {
	Key   string
	Value string // emitted verbatim — use NixString() for quoted strings
}

// NixEnvVar is an env.KEY = VALUE line.
type NixEnvVar struct {
	Key     string
	Value   string // emitted verbatim — use NixString() for quoted strings
	Comment string // optional inline comment rendered on the line above
}

// NixString returns a properly quoted and escaped Nix string literal,
// e.g. NixString("hello") returns `"hello"`.
func NixString(s string) string {
	return `"` + NixEscapeString(s) + `"`
}

// NixPkgsAttrOr returns a Nix expression selecting `pkgs.<attr>`, or
// `fallback` (a Nix expression) with an evaluation warning when nixpkgs does
// not provide attr. A plain `pkgs.<attr> or <fallback>` is not enough: nixpkgs
// keeps removed versions (bazel_6, zig_0_12, ...) as throw aliases, which `or`
// does not catch, so the whole shell would fail to evaluate. warning is the
// body of a Nix string literal, emitted verbatim so it may interpolate
// (`${pkgs.zig.version}`); callers must only put validated text in it.
func NixPkgsAttrOr(attr, fallback, warning string) string {
	return fmt.Sprintf(
		`(let r = builtins.tryEval (pkgs.%s or null); in if r.success && r.value != null then r.value else lib.warn "%s" %s)`,
		attr, warning, fallback)
}

// BuildLanguageFragment renders a NixLangConfig into a devenv.nix fragment.
// The output uses two-space indentation and matches the hand-written style
// used across ecosystem modules.
func BuildLanguageFragment(cfg NixLangConfig) string {
	var b strings.Builder

	// Language block.
	if len(cfg.Properties) == 0 {
		// Single-line form: languages.X.enable = true;
		fmt.Fprintf(&b, "  %s.enable = true;\n", cfg.EnablePath)
	} else {
		// Block form.
		fmt.Fprintf(&b, "  %s = {\n", cfg.EnablePath)
		b.WriteString("    enable = true;\n")
		for _, p := range cfg.Properties {
			fmt.Fprintf(&b, "    %s = %s;\n", p.Key, p.Value)
		}
		b.WriteString("  };\n")
	}

	// Environment variables.
	if len(cfg.EnvVars) > 0 {
		b.WriteString("\n")
		for _, ev := range cfg.EnvVars {
			if ev.Comment != "" {
				fmt.Fprintf(&b, "  # %s\n", ev.Comment)
			}
			fmt.Fprintf(&b, "  env.%s = %s;\n", ev.Key, ev.Value)
		}
	}

	// Extra blocks.
	for _, block := range cfg.ExtraBlocks {
		b.WriteString("\n")
		b.WriteString(block)
		// Ensure trailing newline.
		if !strings.HasSuffix(block, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}
