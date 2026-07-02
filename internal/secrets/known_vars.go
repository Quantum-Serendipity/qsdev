package secrets

import "strings"

// KnownCredentialVars is the canonical list of environment variable names
// that carry credentials or secrets. Used by the log redaction handler and
// the devenv addon's environment stripping.
var KnownCredentialVars = []string{
	// AWS
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",
	"AWS_DEFAULT_REGION",
	// GitHub
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GITHUB_PAT",
	// GitLab
	"GITLAB_TOKEN",
	"GL_TOKEN",
	// GCP
	"GOOGLE_APPLICATION_CREDENTIALS",
	"GCLOUD_PROJECT",
	"CLOUDSDK_CORE_PROJECT",
	// Azure
	"AZURE_CLIENT_ID",
	"AZURE_CLIENT_SECRET",
	"AZURE_TENANT_ID",
	"AZURE_SUBSCRIPTION_ID",
	// Package registries
	"NPM_TOKEN",
	"PYPI_TOKEN",
	// Docker
	"DOCKER_PASSWORD",
	"DOCKER_AUTH_CONFIG",
	// Nix
	"CACHIX_AUTH_TOKEN",
	// Databases
	"DATABASE_URL",
	"DATABASE_PASSWORD",
	"PGPASSWORD",
	"MYSQL_PWD",
	"REDIS_PASSWORD",
	// Secrets management
	"VAULT_TOKEN",
	// Third-party services
	"SENTRY_DSN",
	"STRIPE_SECRET_KEY",
	"SENDGRID_API_KEY",
	// Communication
	"SLACK_TOKEN",
	"SLACK_WEBHOOK_URL",
	// Generic
	"API_KEY",
	"API_SECRET",
	"SECRET_KEY",
	"PRIVATE_KEY",
	"ENCRYPTION_KEY",
}

// SensitiveKeyPatterns are substrings that, when found in a variable or log
// attribute key name (case-insensitive, token-boundary matched), indicate the
// value should be redacted. The bare "key" token is only ever matched at a
// token boundary (see IsSensitiveName), so it catches ACCESS_KEY / *_KEY /
// KEY_* without matching KEYBOARD, KEYWORD, or MONKEY.
var SensitiveKeyPatterns = []string{
	"password",
	"secret",
	"token",
	"credential",
	"auth",
	"api_key",
	"apikey",
	"private_key",
	"bearer",
	"session",
	"passwd",
	"pwd",
	"access_key",
	"key",
}

// knownCredentialSet is the case-insensitive lookup form of KnownCredentialVars,
// built once for IsSensitiveName. KnownCredentialVars entries are all uppercase,
// so the map is keyed on the uppercase name.
var knownCredentialSet = func() map[string]bool {
	m := make(map[string]bool, len(KnownCredentialVars))
	for _, v := range KnownCredentialVars {
		m[strings.ToUpper(v)] = true
	}
	return m
}()

// IsSensitiveName reports whether a variable or attribute key name denotes a
// credential-bearing value. Matching is case-insensitive: the name is sensitive
// if it exactly matches a KnownCredentialVars entry OR contains a
// SensitiveKeyPatterns token at a token boundary (see MatchesSensitiveKeyPattern).
//
// This is the single shared predicate for the redaction seams: the slog
// RedactingHandler (attribute keys and NAME=value pairs in messages), the
// external-log / bug-report line scrubber, and the RedactStructured MCP path.
func IsSensitiveName(name string) bool {
	if name == "" {
		return false
	}
	if knownCredentialSet[strings.ToUpper(name)] {
		return true
	}
	return MatchesSensitiveKeyPattern(name)
}

// MatchesSensitiveKeyPattern reports whether name contains a SensitiveKeyPatterns
// token at a token boundary (case-insensitive, with hyphens normalized to
// underscores so "api-key" matches "api_key"). Unlike IsSensitiveName it does not
// consult the exact KnownCredentialVars canon, so a diagnostic-safe connection
// var (e.g. DATABASE_URL) whose credentials live in its value — not its name —
// is not matched here; the env-probe uses this to withhold opaque secrets while
// still value-scrubbing (and thus preserving the host of) such connection vars.
func MatchesSensitiveKeyPattern(name string) bool {
	lower := strings.ToLower(name)
	normalized := strings.ReplaceAll(lower, "-", "_")
	// When the name contains no hyphen, normalized is identical to lower, so
	// scanning it again per pattern would be pure duplicate work. This predicate
	// is hot (per slog attribute key and per NAME=value token), so skip it.
	normalizedDiffers := normalized != lower
	for _, pat := range SensitiveKeyPatterns {
		if matchesTokenBoundary(lower, pat) {
			return true
		}
		if normalizedDiffers && matchesTokenBoundary(normalized, pat) {
			return true
		}
	}
	return false
}

// matchesTokenBoundary reports whether pattern appears in s delimited by a token
// boundary on both sides — the start/end of the string or a non-alphanumeric
// separator (e.g. "_" or "-"). It scans every occurrence, so a boundary match
// later in the string is still found. The boundary rule keeps a short generic
// token like "key" from matching inside "keyboard", "keyword", or "monkey".
func matchesTokenBoundary(s, pattern string) bool {
	from := 0
	for {
		rel := strings.Index(s[from:], pattern)
		if rel < 0 {
			return false
		}
		idx := from + rel
		leftOK := idx == 0 || !isAlphaNum(s[idx-1])
		end := idx + len(pattern)
		rightOK := end == len(s) || !isAlphaNum(s[end])
		if leftOK && rightOK {
			return true
		}
		from = idx + 1
	}
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
