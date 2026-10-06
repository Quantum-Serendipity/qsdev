package claudecode

import "github.com/Quantum-Serendipity/qsdev/internal/secrets"

// ScanHeuristicPatterns are the scan-only KEY="value" assignment heuristics
// the scan-secrets hook checks in every file. They are not credential token
// shapes, so they stay out of the internal/secrets canon (and out of log
// redaction).
var ScanHeuristicPatterns = []secrets.ValuePattern{
	// AWS secret access key and session token assignments.
	{Name: "aws-assignment", Regex: `(?i)aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}`},
	// Quoted API key assignments.
	{Name: "api-key-assignment", Regex: `["']?[Aa](pi|PI)[_-]?[Kk](ey|EY)["']?\s*[=:]\s*["'][A-Za-z0-9_-]{20,}["']`},
	// Quoted password/secret/token assignments; the key may be quoted (JSON).
	{Name: "secret-assignment", Regex: `(?i)(password|passwd|secret|token|credential)["']?\s*[=:]\s*["'][^\s"']{8,}["']`},
}

// DefaultSecretPatterns are the patterns the scan-secrets hook checks in
// every file: the internal/secrets credential canon followed by
// ScanHeuristicPatterns. Like PlaceholderIndicators, it mirrors
// DEFAULT_PATTERNS in templates/hooks/scan-secrets.py and serves as the
// Go-side test oracle (kept in sync by TestSecretPatterns_MatchPythonHook).
var DefaultSecretPatterns = defaultSecretPatterns()

func defaultSecretPatterns() []string {
	patterns := make([]string, 0, len(secrets.ValuePatterns)+len(ScanHeuristicPatterns))
	for _, vp := range secrets.ValuePatterns {
		patterns = append(patterns, vp.Regex)
	}
	for _, h := range ScanHeuristicPatterns {
		patterns = append(patterns, h.Regex)
	}
	return patterns
}

// ConfigSecretPatterns mirror CONFIG_PATTERNS in scan-secrets.py: unquoted
// KEY=value / key: value assignments, which the hook checks only in dotenv and
// config files because in source code the same shape is an ordinary
// variable assignment.
var ConfigSecretPatterns = []string{
	`(?im)^[ \t]*(export[ \t]+)?[A-Za-z0-9_.-]*(password|passwd|secret|token|credential|api[_-]?key|access[_-]?key)["']?[ \t]*[=:][ \t]*[^\s"'#$<%{][^\s"'#]{7,}`,
}

// PlaceholderIndicators are substrings that indicate a matched value is a
// placeholder rather than a real secret. They mirror PLACEHOLDER_INDICATORS in
// templates/hooks/scan-secrets.py, which is what enforces them; this copy is
// the Go-side test oracle, kept in sync by TestPlaceholderIndicators_MatchPythonHook.
// The hook compares them against the upper-cased match, so every entry must
// be upper case to take effect.
var PlaceholderIndicators = []string{
	"EXAMPLE",
	"PLACEHOLDER",
	"YOUR_",
	"REPLACE",
	"CHANGEME",
	"INSERT_",
	"TODO",
	"XXXX",
	"SAMPLE",
	"DUMMY",
	"TEST_KEY",
}
