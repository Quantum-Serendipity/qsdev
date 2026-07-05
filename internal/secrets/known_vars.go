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
// KEY_* without matching KEYBOARD, KEYWORD, or MONKEY. The "pwd" token is
// handled separately in MatchesSensitiveKeyPattern (as a non-whole-name token)
// so it catches MYSQL_PWD / *_PWD without withholding the ubiquitous, non-secret
// PWD (present working directory) variable.
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
	"access_key",
	"key",
}

// credentialRootSubstrings are credential-word roots matched case-insensitively
// ANYWHERE in a name (plain substring), complementing the token-boundary matcher
// in MatchesSensitiveKeyPattern. The token-boundary matcher misses a credential
// word concatenated to another word with no separator — e.g. "auth" in
// AUTHORIZATION / PROXY_AUTHORIZATION has no right-hand boundary, and bare
// PRIVATE / PRIVATE_FOO carry no separator — so those values would otherwise leak.
//
// Only high-precision roots belong here, because MatchesSensitiveKeyPattern feeds
// the log redactor, which must NOT over-redact benign words that merely start
// with a credential prefix (tokenizer, passwordless, secretariat). "auth" and
// "private" are whole credential words with no common benign superword, so a
// plain substring match is safe. Broader, lower-precision roots (secret,
// password, token, …) would over-match and therefore live in
// EnvCredentialRootSubstrings, used only by the env-info probe where
// over-withholding a value is fail-safe.
var credentialRootSubstrings = []string{
	"auth",
	"private",
}

// EnvCredentialRootSubstrings are additional credential-word roots matched as a
// plain case-insensitive substring by the env-info probe ONLY (see
// ContainsEnvCredentialRoot). They catch separator-less concatenations such as
// SECRETKEY, PASSWORDHASH, ACCESSKEY, and MYAPIKEY that the token-boundary
// matcher misses. These roots are intentionally lower-precision than
// credentialRootSubstrings: as bare substrings they also match benign words
// (secretariat, passwordless, tokenizer), so they must never feed
// MatchesSensitiveKeyPattern / the log redactor. The env probe tolerates this
// because there over-withholding a value is fail-safe while leaking is not.
var EnvCredentialRootSubstrings = []string{
	"secret",
	"password",
	"passwd",
	"credential",
	"token",
	"apikey",
	"accesskey",
	"privatekey",
	"signingkey",
}

// ContainsEnvCredentialRoot reports whether name contains (case-insensitively) an
// EnvCredentialRootSubstrings root anywhere. It is the env-info probe's fail-safe
// fallback for opaque credential names the precise predicates miss; it is
// deliberately not used by the log-redaction path (see EnvCredentialRootSubstrings).
func ContainsEnvCredentialRoot(name string) bool {
	lower := strings.ToLower(name)
	for _, root := range EnvCredentialRootSubstrings {
		if strings.Contains(lower, root) {
			return true
		}
	}
	return false
}

// CloudCredentialPrefixes are environment-variable name prefixes for cloud
// provider namespaces whose values must be withheld even when the specific name
// carries no credential keyword (e.g. GCP_PROJECT, GOOGLE_CLOUD_PROJECT). This is
// the shared canon; the env-info probe consumes it (via HasCloudCredentialPrefix)
// as defense-in-depth alongside the value-level redactor. Prefixes are matched
// case-insensitively against the start of the name.
var CloudCredentialPrefixes = []string{"AWS_", "AZURE_", "GCP_", "GOOGLE_", "GH_", "GITHUB_"}

// HasCloudCredentialPrefix reports whether name begins (case-insensitively) with
// one of the CloudCredentialPrefixes cloud-provider namespaces.
func HasCloudCredentialPrefix(name string) bool {
	up := strings.ToUpper(name)
	for _, p := range CloudCredentialPrefixes {
		if strings.HasPrefix(up, p) {
			return true
		}
	}
	return false
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
// if it exactly matches a KnownCredentialVars entry OR matches the by-name
// predicate MatchesSensitiveKeyPattern (a SensitiveKeyPatterns token at a token
// boundary, or a credentialRootSubstrings root anywhere in the name).
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

// MatchesSensitiveKeyPattern reports whether name looks credential-bearing by
// name alone. It matches when name contains a SensitiveKeyPatterns token at a
// token boundary (case-insensitive, with hyphens normalized to underscores so
// "api-key" matches "api_key"), OR when it contains a credentialRootSubstrings
// root anywhere (the substring fallback that catches concatenated credential
// words the token boundary misses — e.g. AUTHORIZATION, PROXY_AUTHORIZATION,
// bare PRIVATE, PRIVATE_FOO). Unlike IsSensitiveName it does not consult the
// exact KnownCredentialVars canon, so a diagnostic-safe connection var (e.g.
// DATABASE_URL) whose credentials live in its value — not its name — is not
// matched here; the env-probe uses this to withhold opaque secrets while still
// value-scrubbing (and thus preserving the host of) such connection vars.
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
	// Substring fallback for credential words concatenated to another word with
	// no separator, which the token-boundary loop above cannot see (e.g. "auth"
	// in AUTHORIZATION has no right-hand boundary). See credentialRootSubstrings
	// for the rationale and the fail-safe (over-withhold, never leak) tradeoff.
	for _, root := range credentialRootSubstrings {
		if strings.Contains(lower, root) {
			return true
		}
	}
	// "pwd" is treated as sensitive only when it is a separated token inside a
	// longer name (MYSQL_PWD, *_PWD), never as the whole name — otherwise the
	// ubiquitous, non-secret PWD (present working directory) var would be withheld.
	if lower != "pwd" && (matchesTokenBoundary(lower, "pwd") ||
		(normalizedDiffers && matchesTokenBoundary(normalized, "pwd"))) {
		return true
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
