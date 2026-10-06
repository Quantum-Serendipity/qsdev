package secrets

import (
	"slices"
	"testing"
)

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

// TestSensitiveCamelCaseNames covers credential keys in camelCase/PascalCase
// (the norm for JSON and structured payloads), where the credential word is
// joined to its neighbour by a case change rather than a separator and so has
// no token boundary until the words are split.
func TestSensitiveCamelCaseNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"camel access token", "accessToken", true},
		{"camel client secret", "clientSecret", true},
		{"camel db password", "dbPassword", true},
		{"camel refresh token", "refreshToken", true},
		{"camel secret key", "secretKey", true},
		{"camel github token", "githubToken", true},
		{"pascal access token", "AccessToken", true},
		{"acronym api key", "APIKey", true},
		{"digit before word", "s3SecretKey", true},
		{"camel pwd", "mysqlPwd", true},
		{"all caps concatenated client secret", "CLIENTSECRET", true},
		{"all caps concatenated access token", "ACCESSTOKEN", true},
		// Benign camelCase names must stay unredacted.
		{"camel keyboard layout", "keyboardLayout", false},
		{"camel tokenizer", "tokenizerConfig", false},
		{"camel user name", "userName", false},
		{"pascal working dir", "WorkingDir", false},
		{"camel monkey patch", "monkeyPatch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsSensitiveName(tt.key); got != tt.want {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tt.key, got, tt.want)
			}
			if got := MatchesSensitiveKeyPattern(tt.key); got != tt.want {
				t.Errorf("MatchesSensitiveKeyPattern(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSplitCaseWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"accessToken", "access_Token"},
		{"APIKey", "API_Key"},
		{"OAuthToken", "O_Auth_Token"},
		{"s3SecretKey", "s3_Secret_Key"},
		{"GITHUB_TOKEN", "GITHUB_TOKEN"},
		{"github_token", "github_token"},
		{"api-key", "api-key"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := splitCaseWords(tt.in); got != tt.want {
				t.Errorf("splitCaseWords(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestSensitiveSubstringFallback covers the credential-root substring fallback:
// token-boundary matching alone leaks values whose credential keyword is
// concatenated to another word with no separator — "auth" inside AUTHORIZATION /
// PROXY_AUTHORIZATION (no right-hand boundary) and "private" inside bare PRIVATE
// / PRIVATE_FOO. Both by-name predicates must now flag them, while obvious
// non-secrets (including the ubiquitous PWD) must stay emittable.
func TestSensitiveSubstringFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		// Previously leaked: concatenated credential words the boundary missed.
		{"authorization", "AUTHORIZATION", true},
		{"proxy authorization", "PROXY_AUTHORIZATION", true},
		{"bare private", "PRIVATE", true},
		{"private prefix", "PRIVATE_FOO", true},
		{"oauth token", "OAUTH_TOKEN", true},
		// Obvious non-secrets must not be over-withheld.
		{"path", "PATH", false},
		{"home", "HOME", false},
		{"user", "USER", false},
		{"pwd", "PWD", false},
		{"lang", "LANG", false},
		{"term", "TERM", false},
		{"shell", "SHELL", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsSensitiveName(tt.key); got != tt.want {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tt.key, got, tt.want)
			}
			if got := MatchesSensitiveKeyPattern(tt.key); got != tt.want {
				t.Errorf("MatchesSensitiveKeyPattern(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// TestIsSensitiveName_Plurals covers plural credential names (one trailing "s"
// accepted as a token's right boundary), the connection-string / cookie /
// passphrase tokens, and "pass" as an embedded-only token: DB_PASS is sensitive
// but a bare "pass" (qsdev's own check/posture pass-count field) is not.
func TestIsSensitiveName_Plurals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want bool
	}{
		{"credentials", true},
		{"secrets", true},
		{"tokens", true},
		{"passwords", true},
		{"api_keys", true},
		{"keys", true},
		{"sessions", true},
		{"passphrase", true},
		{"cookie", true},
		{"set-cookie", true},
		{"Set-Cookie", true},
		{"dsn", true},
		{"connection_string", true},
		{"conn_string", true},
		{"DB_PASS", true},
		{"MYSQL_PASS", true},
		{"smtp-pass", true},
		{"pass", false},
		{"Pass", false},
		{"passthrough", false},
		{"compass", false},
		{"bypass", false},
		{"tokenizer_version", false},
		{"status", false},
		{"keyboard", false},
		{"PWD", false},
		{"PATH", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			if got := IsSensitiveName(tt.key); got != tt.want {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// TestKnownCredentialVars_ExcludeSelectors guards that the credential canon
// carries no non-secret selector variables. A region, profile name, project
// ID, tenant ID or subscription ID picks an account context but grants no
// access; listing one here strips it from the devenv shell (catalog
// unset_vars must be a superset of this list), which discards the value a
// cloud module or the user deliberately set.
func TestKnownCredentialVars_ExcludeSelectors(t *testing.T) {
	t.Parallel()

	selectors := []string{
		"AWS_DEFAULT_REGION",
		"AWS_REGION",
		"AWS_PROFILE",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
		"AZURE_TENANT_ID",
		"AZURE_SUBSCRIPTION_ID",
	}
	for _, s := range selectors {
		if slices.Contains(KnownCredentialVars, s) {
			t.Errorf("KnownCredentialVars contains selector %q, which is not a credential", s)
		}
	}
}
