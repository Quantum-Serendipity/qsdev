package claudecode

// DefaultSecretPatterns contains the 12 default credential detection regex
// patterns used by the scan-secrets hook. Like PlaceholderIndicators, it mirrors
// templates/hooks/scan-secrets.py and serves as the Go-side test oracle.
var DefaultSecretPatterns = []string{
	`AKIA[0-9A-Z]{16}`,
	`(?i)aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}`,
	`gh[ps]_[A-Za-z0-9_]{36,}`,
	`glpat-[A-Za-z0-9_-]{20,}`,
	`["']?[Aa](pi|PI)[_-]?[Kk](ey|EY)["']?\s*[=:]\s*["'][A-Za-z0-9_-]{20,}["']`,
	`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
	`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`,
	`(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"':]+:[^\s"'@]+@[^\s"']{5,}`,
	`xox[bprase]-[A-Za-z0-9-]{10,}`,
	`sk_(live|test)_[A-Za-z0-9]{20,}`,
	`SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`,
	`(?i)(password|passwd|secret|token|credential)\s*[=:]\s*["'][^\s"']{8,}["']`,
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
