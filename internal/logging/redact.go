package logging

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

const redacted = "[REDACTED]"

// Redactor scrubs secret values from log attributes.
type Redactor struct {
	valuePatterns []*regexp.Regexp
	keyDeny       map[string]bool
	envNames      map[string]bool
	urlCredRe     *regexp.Regexp
}

// NewRedactor creates a Redactor with default secret patterns.
func NewRedactor() *Redactor {
	r := &Redactor{
		keyDeny:  make(map[string]bool, len(secrets.SensitiveKeyPatterns)),
		envNames: make(map[string]bool, len(secrets.KnownCredentialVars)),
	}

	for _, p := range secrets.SensitiveKeyPatterns {
		r.keyDeny[strings.ToLower(p)] = true
	}
	for _, v := range secrets.KnownCredentialVars {
		r.envNames[v] = true
	}

	r.valuePatterns = compileValuePatterns()
	r.urlCredRe = regexp.MustCompile(`://[^:@\s]+:[^:@\s]+@`)

	return r
}

func compileValuePatterns() []*regexp.Regexp {
	patterns := []string{
		`AKIA[A-Z0-9]{16}`,
		`gh[pousr]_[A-Za-z0-9_]{36,}`,
		`github_pat_[A-Za-z0-9_]{22,}`,
		`glpat-[A-Za-z0-9\-_]{20,}`,
		`sk_(live|test)_[A-Za-z0-9]{24,}`,
		`npm_[A-Za-z0-9]{36,}`,
		`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+`,
		`-----BEGIN\s+(?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return compiled
}

// RedactAttr scrubs secret values from a single slog.Attr.
func (r *Redactor) RedactAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		scrubbed := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			scrubbed[i] = r.RedactAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(scrubbed...)}
	}

	if r.isKeyDenied(a.Key) {
		return slog.String(a.Key, redacted)
	}

	if a.Value.Kind() == slog.KindString {
		scrubbed := r.RedactString(a.Value.String())
		if scrubbed != a.Value.String() {
			return slog.String(a.Key, scrubbed)
		}
	}

	return a
}

// RedactString scrubs secret patterns from a string value.
func (r *Redactor) RedactString(s string) string {
	for _, p := range r.valuePatterns {
		s = p.ReplaceAllString(s, redacted)
	}
	if r.urlCredRe.MatchString(s) {
		s = r.redactURLCredentials(s)
	}
	return s
}

func (r *Redactor) redactURLCredentials(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.User == nil {
		return r.urlCredRe.ReplaceAllString(s, "://"+redacted+":"+redacted+"@")
	}
	u.User = url.UserPassword(redacted, redacted)
	return u.String()
}

// isKeyDenied checks whether an attribute key indicates a secret value.
// Uses word-boundary matching: "token" matches but "tokenizer" does not.
func (r *Redactor) isKeyDenied(key string) bool {
	lower := strings.ToLower(key)

	if r.envNames[key] || r.envNames[strings.ToUpper(key)] {
		return true
	}

	// Normalize hyphens to underscores so "api-key" matches "api_key".
	normalized := strings.ReplaceAll(lower, "-", "_")

	for pattern := range r.keyDeny {
		if matchesWordBoundary(lower, pattern) || matchesWordBoundary(normalized, pattern) {
			return true
		}
	}
	return false
}

// matchesWordBoundary checks if pattern appears in s at a word boundary.
// A word boundary is: start/end of string, underscore, hyphen, or transition
// between non-letter and letter.
func matchesWordBoundary(s, pattern string) bool {
	idx := strings.Index(s, pattern)
	if idx < 0 {
		return false
	}

	if idx > 0 {
		prev := s[idx-1]
		if isWordChar(prev) && prev != '_' && prev != '-' {
			return false
		}
	}

	end := idx + len(pattern)
	if end < len(s) {
		next := s[end]
		if isWordChar(next) && next != '_' && next != '-' {
			return false
		}
	}

	return true
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
