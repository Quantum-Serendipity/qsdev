package cmdscan

import "testing"

// TestParse_QuoteAndEscapeRemoval covers the shell's quote removal in
// argument words (F135): a protected path spelled with split quotes,
// backslash escapes, or an ANSI-C string must reach the rules as the literal
// path the command receives.
func TestParse_QuoteAndEscapeRemoval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    string
		wantExp bool
	}{
		{"empty double quotes", `rm .cl""aude/settings.json`, ".claude/settings.json", false},
		{"empty single quotes", `rm .cl''aude/settings.json`, ".claude/settings.json", false},
		{"quoted tail", `rm .cl'aude'`, ".claude", false},
		{"backslash escape", `rm .cl\aude/settings.json`, ".claude/settings.json", false},
		{"escaped space", `rm my\ file`, "my file", false},
		{"double-quoted escaped quote", `rm "a\"b"`, `a"b`, false},
		{"double-quoted literal backslash", `rm "a\qb"`, `a\qb`, false},
		{"ansi-c hex escape", `rm $'\x2e'claude/settings.json`, ".claude/settings.json", false},
		{"ansi-c octal escape", `rm $'\056claude'`, ".claude", false},
		{"ansi-c unicode escape", `rm $'.claude'`, ".claude", false},
		{"ansi-c simple escapes", `rm $'a\tb\\c'`, "a\tb\\c", false},
		{"ansi-c unmodelled escape is opaque", `rm $'\cA.claude'`, `\cA.claude`, true},
		{"glob characters are kept", `rm .c*e/settings.json`, ".c*e/settings.json", false},
		{"braces are kept", `rm .{claude,x}`, ".{claude,x}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.command, err)
			}
			if len(cmds) != 1 || len(cmds[0].Args) != 1 {
				t.Fatalf("Parse(%q) = %+v, want one command with one argument", tt.command, cmds)
			}
			if got := cmds[0].Args[0]; got != tt.want {
				t.Errorf("Parse(%q) argument = %q, want %q", tt.command, got, tt.want)
			}
			if cmds[0].HasExpansion != tt.wantExp {
				t.Errorf("Parse(%q) HasExpansion = %v, want %v", tt.command, cmds[0].HasExpansion, tt.wantExp)
			}
		})
	}
}

func TestParse_EscapedCommandWord(t *testing.T) {
	t.Parallel()

	cmds, err := Parse(`r\m -f x && r''m -f y`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 2 || cmds[0].Name != "rm" || cmds[1].Name != "rm" {
		t.Errorf("escaped/quote-split command words not decoded: %+v", cmds)
	}
}
