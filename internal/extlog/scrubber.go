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

// Scrub applies privacy scrubbing to a single line of external log content.
func (s *Scrubber) Scrub(line string) string {
	line = s.redactor.RedactString(line)

	for _, p := range s.extraPats {
		line = p.ReplaceAllString(line, "[REDACTED]")
	}

	if s.homeDir != "" {
		line = strings.ReplaceAll(line, s.homeDir, "~")
	}
	if s.projectRoot != "" {
		line = strings.ReplaceAll(line, s.projectRoot, ".")
	}

	// Second-pass redaction catches credential patterns that may emerge after
	// path substitution (e.g. a home-dir prefix was masking the pattern boundary).
	line = s.redactor.RedactString(line)

	return line
}
