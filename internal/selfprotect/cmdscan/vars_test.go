package cmdscan

import "testing"

// TestParseWithVars covers the rendering of known variables: a plain $NAME or
// ${NAME} expands to its value, in or out of double quotes and in redirect
// targets, as does a plain echo of literal words, while any other expansion
// (an operator, an unknown name, another command substitution) still
// contributes no text. Every such word is an expansion.
func TestParseWithVars(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"HOME": "/h", "ORG": "/o/d.yaml"}

	tests := []struct {
		name     string
		command  string
		wantArg  string
		wantRedi string
		wantExp  bool
	}{
		{"bare", "sed -i x $HOME/.config/a", "/h/.config/a", "", true},
		{"braced", "sed -i x ${HOME}/.config/a", "/h/.config/a", "", true},
		{"double quoted", `sed -i x "$HOME/.config/a"`, "/h/.config/a", "", true},
		{"redirect", `printf x > "$ORG"`, "x", "/o/d.yaml", true},
		{"single quoted stays literal", `sed -i x '$HOME'`, "$HOME", "", false},
		{"unknown name", "sed -i x $OTHER/a", "/a", "", true},
		{"default operator", "sed -i x ${HOME:-/z}/a", "/a", "", true},
		{"length", "sed -i x ${#HOME}", "", "", true},
		{"longer name", "sed -i x $HOMEDIR/a", "/a", "", true},
		{"command substitution", "sed -i x $(cat /h)/a", "/a", "", true},
		// A plain echo of literal words prints them (see echoOutput).
		{"echo substitution", "sed -i x $(echo /h)/a", "/h/a", "", true},
		{"quoted echo substitution", `rm -rf "$(echo ~/.co)nfig"`, "~/.config", "", true},
		{"echo substitution of a variable", "sed -i x $(echo $HOME)/a", "/a", "", true},
		{"echo substitution with an option", "sed -i x $(echo -n /h)/a", "/a", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmds, err := ParseWithVars(tt.command, vars)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("ParseWithVars(%q) = %v, %v; want a command", tt.command, cmds, err)
			}
			c := cmds[0]
			if got := c.Args[len(c.Args)-1]; got != tt.wantArg {
				t.Errorf("last arg = %q, want %q", got, tt.wantArg)
			}
			if tt.wantRedi != "" && (len(c.WriteRedirects) != 1 || c.WriteRedirects[0] != tt.wantRedi) {
				t.Errorf("write redirects = %q, want [%q]", c.WriteRedirects, tt.wantRedi)
			}
			if c.HasExpansion != tt.wantExp {
				t.Errorf("HasExpansion = %v, want %v", c.HasExpansion, tt.wantExp)
			}
		})
	}
}
