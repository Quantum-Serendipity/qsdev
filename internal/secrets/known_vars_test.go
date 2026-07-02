package secrets

import "testing"

func TestIsSensitiveName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		// Exact KnownCredentialVars entries (case-insensitive).
		{"exact aws secret", "AWS_SECRET_ACCESS_KEY", true},
		{"exact lowercased", "github_token", true},
		{"exact database url", "DATABASE_URL", true},
		// Leak set from the finding: keyword-less names caught by new tokens.
		{"database password", "DATABASE_PASSWORD", true},
		{"pg password", "PGPASSWORD", true},
		{"bitwarden session", "BW_SESSION", true},
		{"new relic license key", "NEW_RELIC_LICENSE_KEY", true},
		{"rails master key", "RAILS_MASTER_KEY", true},
		{"meili master key", "MEILI_MASTER_KEY", true},
		{"bare access key", "ACCESS_KEY", true},
		{"bare key", "KEY", true},
		{"passwd", "MY_PASSWD", true},
		{"pwd suffix", "MYSQL_PWD", true},
		{"hyphenated api key", "api-key", true},
		// Negatives that must NOT be treated as sensitive.
		{"path", "PATH", false},
		{"keyboard", "KEYBOARD", false},
		{"keyword", "KEYWORD", false},
		{"monkey", "MONKEY", false},
		{"home", "HOME", false},
		{"editor", "EDITOR", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsSensitiveName(tt.key); got != tt.want {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// TestMatchesSensitiveKeyPattern confirms the pattern-only predicate (used by the
// env probe) matches on keyword tokens but not on the exact connection-var canon,
// so a DATABASE_URL value can still be emitted (value-scrubbed) rather than withheld.
func TestMatchesSensitiveKeyPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"session token", "BW_SESSION", true},
		{"passwd", "MY_PASSWD", true},
		{"access key", "ACCESS_KEY", true},
		{"suffix key", "NEW_RELIC_LICENSE_KEY", true},
		// Exact-canon connection var carries no keyword token -> not matched here.
		{"database url not a token", "DATABASE_URL", false},
		{"keyboard", "KEYBOARD", false},
		{"path", "PATH", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesSensitiveKeyPattern(tt.key); got != tt.want {
				t.Errorf("MatchesSensitiveKeyPattern(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
