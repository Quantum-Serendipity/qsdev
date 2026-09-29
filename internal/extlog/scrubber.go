package extlog

import (
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// Scrubber applies privacy scrubbing to external log content.
// It uses the same base patterns as the slog RedactingHandler plus
// additional patterns for external tool log content.
type Scrubber struct {
	redactor    *logging.Redactor
	extraPats   []*regexp.Regexp
	homeDir     string
	projectRoot string
	publicHosts map[string]bool
}

// NewScrubber creates a Scrubber with default patterns.
func NewScrubber(homeDir, projectRoot string) *Scrubber {
	return &Scrubber{
		redactor:    logging.NewRedactor(),
		extraPats:   compileExtraPatterns(),
		homeDir:     homeDir,
		projectRoot: projectRoot,
		publicHosts: map[string]bool{
			"registry.npmjs.org": true,
			"pypi.org":           true,
			"crates.io":          true,
			"github.com":         true,
			"gitlab.com":         true,
			"maven.org":          true,
			"repo1.maven.org":    true,
			"plugins.gradle.org": true,
			"nuget.org":          true,
			"rubygems.org":       true,
			"packagist.org":      true,
			"nixos.org":          true,
			"cache.nixos.org":    true,
		},
	}
}

func compileExtraPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)_authToken=\S+`,
		`(?i)//[^:]+/:_authToken=\S+`,
		`(?i)access-tokens\s*=\s*\S+`,
		`(?i)(--index-url|--extra-index-url)\s+\S+`,
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return compiled
}

// redactedHost replaces the host of any URL that is not a known public host.
const redactedHost = "[REDACTED_HOST]"

// hostExpr matches a DNS-style or IPv4 host.
const hostExpr = `[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?)*`

// hostPatterns locate hosts in log lines. In each, submatch 1 is everything
// before the host and submatch 2 is the host.
var hostPatterns = []*regexp.Regexp{
	// URL: scheme, optional userinfo, host.
	regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://(?:[^\s/?#@]*@)?)(` + hostExpr + `)`),
	// scp-like git remote (git@host:owner/repo.git), which has no scheme.
	regexp.MustCompile(`(?i)((?:^|[\s"'(=,])[a-z0-9_.\-]+@)(` + hostExpr + `):`),
}

// Scrub applies privacy scrubbing to a single line of external log content.
func (s *Scrubber) Scrub(line string) string {
	line = s.redactor.RedactString(line)

	for _, p := range s.extraPats {
		line = p.ReplaceAllString(line, "[REDACTED]")
	}

	line = s.scrubHosts(line)

	// The project root is the more specific path (it usually lives under the
	// home directory), so it must be replaced first; otherwise the home prefix
	// is rewritten to "~" and the project path — often a client or product
	// name — survives verbatim.
	if s.projectRoot != "" {
		line = replacePathPrefix(line, s.projectRoot, ".")
	}
	if s.homeDir != "" {
		line = replacePathPrefix(line, s.homeDir, "~")
	}

	// Second-pass redaction catches credential patterns that may emerge after
	// path substitution (e.g. a home-dir prefix was masking the pattern boundary).
	line = s.redactor.RedactString(line)

	return line
}

// scrubHosts replaces every URL or scp-style git remote host that is not a
// known public host (or a subdomain of one) with [REDACTED_HOST], so private registry and
// cache hostnames never reach a shared bug report.
func (s *Scrubber) scrubHosts(line string) string {
	for _, p := range hostPatterns {
		line = p.ReplaceAllStringFunc(line, func(m string) string {
			sub := p.FindStringSubmatch(m)
			if s.isPublicHost(sub[2]) {
				return m
			}
			return sub[1] + redactedHost + m[len(sub[1])+len(sub[2]):]
		})
	}
	return line
}

// isPublicHost reports whether host is a known public host or a subdomain of
// one.
func (s *Scrubber) isPublicHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for {
		if s.publicHosts[host] {
			return true
		}
		dot := strings.IndexByte(host, '.')
		if dot < 0 {
			return false
		}
		host = host[dot+1:]
	}
}

// replacePathPrefix replaces every occurrence of the path prefix in line with
// repl, but only where the occurrence is a whole path: the byte before it must
// not be part of a path segment and the byte after it must end the segment.
// This keeps "/home/alice2" from being rewritten when prefix is "/home/alice".
func replacePathPrefix(line, prefix, repl string) string {
	prefix = strings.TrimRight(prefix, `/\`)
	if prefix == "" {
		return line
	}
	var b strings.Builder
	rest := line
	for {
		i := strings.Index(rest, prefix)
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		end := i + len(prefix)
		if startsPathBoundary(rest, i) && endsPathBoundary(rest, end) {
			b.WriteString(rest[:i])
			b.WriteString(repl)
		} else {
			b.WriteString(rest[:end])
		}
		rest = rest[end:]
	}
}

// startsPathBoundary reports whether a match beginning at s[i] is not the
// continuation of a longer path segment.
func startsPathBoundary(s string, i int) bool {
	return i == 0 || !isPathNameByte(s[i-1])
}

// endsPathBoundary reports whether a match ending at s[end] ends a path
// segment. A trailing '.' counts as a boundary only when it is not followed by
// more name characters (sentence punctuation rather than "alice.bak").
func endsPathBoundary(s string, end int) bool {
	if end >= len(s) {
		return true
	}
	c := s[end]
	if c == '.' {
		return end+1 >= len(s) || !isPathNameByte(s[end+1])
	}
	return !isPathNameByte(c)
}

func isPathNameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '_' || c == '-' || c == '.'
}
