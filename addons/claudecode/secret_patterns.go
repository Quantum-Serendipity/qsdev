package claudecode

// DefaultSecretPatterns contains the default credential detection regex
// patterns used by the scan-secrets hook. Like PlaceholderIndicators, it mirrors
// templates/hooks/scan-secrets.py and serves as the Go-side test oracle
// (kept in sync by TestSecretPatterns_MatchPythonHook).
var DefaultSecretPatterns = []string{
	`(AKIA|ASIA)[0-9A-Z]{16}`,
	`(?i)aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}`,
	`gh[pousr]_[A-Za-z0-9_]{36,}`,
	`glpat-[A-Za-z0-9_-]{20,}`,
	`["']?[Aa](pi|PI)[_-]?[Kk](ey|EY)["']?\s*[=:]\s*["'][A-Za-z0-9_-]{20,}["']`,
	`-----BEGIN ((RSA|EC|DSA|OPENSSH|ENCRYPTED|PGP) )?PRIVATE KEY( BLOCK)?-----`,
	`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`,
	`(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"':]+:[^\s"'@]+@[^\s"']{5,}`,
	`xox[bprase]-[A-Za-z0-9-]{10,}`,
	`sk_(live|test)_[A-Za-z0-9]{20,}`,
	`SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`,
	`(?i)(password|passwd|secret|token|credential)["']?\s*[=:]\s*["'][^\s"']{8,}["']`,
	`github_pat_[A-Za-z0-9_]{22,}`,
	`sk-ant-[A-Za-z0-9_-]{20,}`,
	`sk-(proj|svcacct|admin)-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20}`,
	`AIza[0-9A-Za-z_-]{35}`,
	`npm_[A-Za-z0-9]{36}`,
	`pypi-[A-Za-z0-9_-]{50,}`,
	`https://hooks\.slack\.com/services/T[A-Za-z0-9]+/B[A-Za-z0-9]+/[A-Za-z0-9]+`,
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
