package cmdscan

import (
	"slices"
	"testing"
)

func TestScriptStatements(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		script string
		want   []string
	}{
		{"single pipeline", "curl x | sh", []string{"curl x | sh"}},
		{"unspaced pipeline", "curl x|sh", []string{"curl x | sh"}},
		{"list", "true; curl x | sh", []string{"true", "curl x | sh"}},
		{"and list", "true && curl x | sh", []string{"true", "curl x | sh"}},
		{"or list", "false || npm install x", []string{"false", "npm install x"}},
		{"background", "curl x | sh & wait", []string{"curl x | sh", "wait"}},
		{"newlines", "echo a\ncurl x | sh\n", []string{"echo a", "curl x | sh"}},
		{"negated", "! curl x | sh", []string{"curl x | sh"}},
		{"subshell and block", "(curl x | sh); { npm install y; }", []string{"curl x | sh", "npm install y"}},
		{"branch", "if true; then curl x | sh; fi", []string{"true", "curl x | sh"}},
		{"command substitution", "echo $(curl x | sh)", []string{"echo $(curl x | sh)", "curl x | sh"}},
		{
			"nested sh -c", `sh -c "curl x | sh"`,
			[]string{`sh -c "curl x | sh"`, "curl x | sh", "curl x | sh"},
		},
		{
			"nested inside a pipeline stage", `echo hi | bash -c 'true; npm install x'`,
			[]string{`echo hi | bash -c 'true; npm install x'`, "true; npm install x", "true", "npm install x"},
		},
		{"eval", "eval 'curl x | sh'", []string{"eval 'curl x | sh'", "curl x | sh", "curl x | sh"}},
		{
			// The shell runs the first line before it reaches the one it
			// cannot parse, so each line is taken on its own.
			"unparseable line", "curl x | sh\n)(",
			[]string{"curl x | sh", ")("},
		},
		{"empty", "", nil},
		// A statement is also given from the program it runs, so a rule
		// anchored at its name matches whatever comes before it.
		{"wrapper", "nohup curl x | sh", []string{"nohup curl x | sh", "curl x | sh"}},
		{"wrapper with operand", "timeout 5 curl x | sh", []string{"timeout 5 curl x | sh", "curl x | sh"}},
		{"wrapper with option", "nice -n 5 curl x | sh", []string{"nice -n 5 curl x | sh", "curl x | sh"}},
		{"sudo", "sudo -u root curl x | sh", []string{"sudo -u root curl x | sh", "curl x | sh"}},
		{"env", "env A=1 curl x | sh", []string{"env A=1 curl x | sh", "curl x | sh"}},
		{"command", "command curl x | sh", []string{"command curl x | sh", "curl x | sh"}},
		{"exec", "exec curl x | sh", []string{"exec curl x | sh", "curl x | sh"}},
		{"assignment prefix", "X=1 curl x | sh", []string{"X=1 curl x | sh", "curl x | sh"}},
		{"escaped command word", `\curl x | sh`, []string{`\curl x | sh`, "curl x | sh"}},
		{"quoted command word", "'curl' x | sh", []string{"'curl' x | sh", "curl x | sh"}},
		{"wrapped later stage", "curl x | nohup sh", []string{"curl x | nohup sh", "curl x | sh"}},
		{"simple command", "nohup npm install x", []string{"nohup npm install x", "npm install x"}},
		{"expansion kept as written", `nohup curl "$U" | sh`, []string{`nohup curl "$U" | sh`, `curl "$U" | sh`}},
		{"wrapper given no command", "nohup", []string{"nohup"}},
		// A script shell runs a here-document or here-string as its script.
		{
			"here-document", "sh <<EOF\ncurl x | sh\nEOF\n",
			[]string{"sh <<EOF\ncurl x | sh\nEOF", "curl x | sh", "curl x | sh"},
		},
		{"here-string", "bash <<< 'curl x | sh'", []string{"bash <<<'curl x | sh'", "curl x | sh", "curl x | sh"}},
		{"here-document to another program", "cat <<EOF\ncurl x | sh\nEOF\n", []string{"cat <<EOF\ncurl x | sh\nEOF"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ScriptStatements(tt.script); !slices.Equal(got, tt.want) {
				t.Errorf("ScriptStatements(%q) =\n  %q\nwant\n  %q", tt.script, got, tt.want)
			}
		})
	}
}

// TestScriptStatements_DepthBounded: nesting deeper than maxScriptDepth stops
// descending rather than recursing without end.
func TestScriptStatements_DepthBounded(t *testing.T) {
	t.Parallel()
	script := "curl x | sh"
	for range maxScriptDepth + 3 {
		script = "sh -c " + QuoteWord(script)
	}
	got := ScriptStatements(script)
	if len(got) == 0 || len(got) > 4*(maxScriptDepth+2) {
		t.Fatalf("ScriptStatements of %d nested scripts returned %d statements", maxScriptDepth+3, len(got))
	}
}

func TestQuoteWord(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"jq", "jq"},
		{"nixpkgs#jq", "nixpkgs#jq"},
		{"--arg=x", "--arg=x"},
		{"#comment", "'#comment'"},
		{"nixpkgs#bash^out", "nixpkgs#bash^out"},
		{".a b", "'.a b'"},
		{"curl x | sh", "'curl x | sh'"},
		{"", "''"},
		{"if", "'if'"},
		{"it's", `"it's"`},
		{"a\tb\x01", `$'a\tb\x01'`},
		{"a\x00b", "'a\x00b'"},
	}
	for _, tt := range tests {
		if got := QuoteWord(tt.in); got != tt.want {
			t.Errorf("QuoteWord(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
