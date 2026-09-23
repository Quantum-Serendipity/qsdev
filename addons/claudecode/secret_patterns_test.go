package claudecode_test

import (
	"regexp"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestDefaultSecretPatterns_AllCompile(t *testing.T) {
	t.Parallel()
	all := append(append([]string{}, claudecode.ExportDefaultSecretPatterns...), claudecode.ExportConfigSecretPatterns...)
	for i, pattern := range all {
		if _, err := regexp.Compile(pattern); err != nil {
			t.Errorf("pattern %d (%q) failed to compile: %v", i, pattern, err)
		}
	}
}

func TestDefaultSecretPatterns_Count(t *testing.T) {
	t.Parallel()
	if len(claudecode.ExportDefaultSecretPatterns) != 19 {
		t.Errorf("expected 19 default patterns, got %d", len(claudecode.ExportDefaultSecretPatterns))
	}
}

func TestSecretPatterns_PositiveMatches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern int
		input   string
	}{
		{"AWS access key", 0, "AKIAIOSFODNN7REALKEY"},
		{"AWS secret key assignment", 1, "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYzzzzzz"},
		{"AWS session token", 1, "aws_session_token: ABCDEFGHIJKLMNOPQRSTzzzz"},
		{"GitHub PAT", 2, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"},
		{"GitHub secret", 2, "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"},
		{"GitLab PAT", 3, "glpat-xxxxxxxxxxxxxxxxxxxx"},
		{"API key double-quoted", 4, `API_KEY = "abcdefghijklmnopqrstuvwx"`},
		{"API key single-quoted", 4, `api_key: 'abcdefghijklmnopqrstuvwx'`},
		{"RSA private key", 5, "-----BEGIN RSA PRIVATE KEY-----"},
		{"EC private key", 5, "-----BEGIN EC PRIVATE KEY-----"},
		{"Generic private key", 5, "-----BEGIN PRIVATE KEY-----"},
		{"OPENSSH private key", 5, "-----BEGIN OPENSSH PRIVATE KEY-----"},
		{"JWT token", 6, "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"},
		{"MongoDB connection string", 7, "mongodb://admin:password@db.example.com:27017/mydb"},
		{"PostgreSQL connection string", 7, "postgresql://user:pass@localhost:5432/database"},
		{"Redis connection string", 7, "redis://default:secretpass@redis.example.com:6379"},
		{"MySQL connection string", 7, "mysql://root:rootpass@127.0.0.1:3306/testdb"},
		{"Slack bot token", 8, "xoxb-AAAAAAAAAA-AAAAAAAAAAAAA-AAAAAAAAAAAAAAAAAAAAAAAA"},
		{"Stripe live key", 9, "sk_live_AAAAAAAAAAAAAAAAAAAA"},
		{"Stripe test key", 9, "sk_test_AAAAAAAAAAAAAAAAAAAA"},
		{"SendGrid key", 10, "SG.abcdefghijklmnopqrstuv.ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr"},
		{"Password assignment", 11, `password = "supersecretpassword123"`},
		{"Secret assignment", 11, `secret: "my_very_secret_value_here"`},
		{"Token assignment", 11, `token = "abcdefghijklmnopqrstuvwxyz"`},
		{"AWS secret key UPPERCASE", 1, "AWS_SECRET_ACCESS_KEY = wJalrXUtnFEMI/K7MDENG/bPxRfiCYzzzzzz"},
		{"AWS session token UPPERCASE", 1, "AWS_SESSION_TOKEN: ABCDEFGHIJKLMNOPQRSTzzzz"},
		{"PASSWORD uppercase", 11, `PASSWORD = "supersecretpassword123"`},
		{"SECRET uppercase", 11, `SECRET: "my_very_secret_value_here"`},
		{"TOKEN uppercase", 11, `TOKEN = "abcdefghijklmnopqrstuvwxyz"`},
		{"Slack enterprise token", 8, "xoxe-AAAAAAAAAA-AAAAAAAAAAAAA"},
		// W044: formats the scan used to miss.
		{"AWS temporary key", 0, "ASIA" + strings.Repeat("Q", 16)},
		{"GitHub OAuth token", 2, "gho_" + strings.Repeat("a", 36)},
		{"GitHub refresh token", 2, "ghr_" + strings.Repeat("a", 36)},
		{"Encrypted private key", 5, "-----BEGIN ENCRYPTED PRIVATE KEY-----"},
		{"PGP private key block", 5, "-----BEGIN PGP PRIVATE KEY BLOCK-----"},
		{"JSON password", 11, `{"password": "hunter2hunter2"}`},
		{"GitHub fine-grained PAT", 12, "github_pat_" + strings.Repeat("A1b2", 6)},
		{"Anthropic key", 13, "sk-ant-api03-" + strings.Repeat("Ab9_", 6)},
		{"OpenAI project key", 14, "sk-proj-" + strings.Repeat("Ab9-", 6)},
		{"OpenAI legacy key", 14, "sk-" + strings.Repeat("a", 20) + "T3BlbkFJ" + strings.Repeat("b", 20)},
		{"Google API key", 15, "AIza" + strings.Repeat("x", 35)},
		{"npm token", 16, "npm_" + strings.Repeat("a1", 18)},
		{"PyPI token", 17, "pypi-" + strings.Repeat("AgE", 20)},
		{"Slack webhook", 18, "https://hooks.slack.com/services/T0001/B0002/" + strings.Repeat("x", 24)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := regexp.MustCompile(claudecode.ExportDefaultSecretPatterns[tt.pattern])
			if !re.MatchString(tt.input) {
				t.Errorf("pattern %d should match %q", tt.pattern, tt.input)
			}
		})
	}
}

func TestSecretPatterns_NegativeMatches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern int
		input   string
	}{
		{"Short AKIA prefix", 0, "AKIA1234"},
		{"Non-uppercase AKIA", 0, "AKIAiosfodnn7realkey"},
		{"GitHub wrong prefix", 2, "ghx_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"},
		{"Short GitHub token", 2, "ghp_short"},
		{"GitLab wrong prefix", 3, "glpat_no_dash_here"},
		{"API key no value", 4, "API_KEY = "},
		{"API key short value", 4, `API_KEY = "short"`},
		{"Not a private key", 5, "-----BEGIN CERTIFICATE-----"},
		{"Short JWT", 6, "eyJ.eyJ.abc"},
		{"HTTP URL not DB", 7, "https://example.com/api/endpoint"},
		{"PostgreSQL no credentials", 7, "postgres://localhost:5432/testdb"},
		{"Redis no credentials", 7, "redis://localhost:6379"},
		{"MongoDB no credentials", 7, "mongodb://localhost:27017/mydb"},
		{"Slack wrong prefix", 8, "xoxx-not-a-token"},
		{"Stripe wrong prefix", 9, "pk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZabcde"},
		{"Short Stripe key", 9, "sk_live_short"},
		{"Short password", 11, `password = "short"`},
		{"Password no quotes", 11, "password = noquotes"},
		{"Env var reference", 11, "password = ${DB_PASSWORD}"},
		{"Anthropic prefix only", 13, "sk-ant-short"},
		{"OpenAI unrelated sk- word", 14, "sk-learn-is-a-library"},
		{"Google short", 15, "AIzaShort"},
		{"npm short", 16, "npm_install"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := regexp.MustCompile(claudecode.ExportDefaultSecretPatterns[tt.pattern])
			if re.MatchString(tt.input) {
				t.Errorf("pattern %d should NOT match %q", tt.pattern, tt.input)
			}
		})
	}
}

func TestPlaceholderIndicators_Defined(t *testing.T) {
	t.Parallel()
	if len(claudecode.ExportPlaceholderIndicators) == 0 {
		t.Error("expected non-empty PlaceholderIndicators")
	}
}

// TestConfigSecretPatterns matches the unquoted assignments the hook checks in
// dotenv and config files (W044) and leaves variable references alone.
func TestConfigSecretPatterns(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(claudecode.ExportConfigSecretPatterns[0])
	tests := []struct {
		input string
		want  bool
	}{
		{"DB_PASSWORD=hunter2hunter2", true},
		{"export API_KEY=abcdef0123456789", true},
		{"db:\n  password: hunter2hunter2", true},
		{"client_secret = s3cr3tv4lu3", true},
		{"DB_PASSWORD=${DB_PASSWORD}", false},
		{"password: <password>", false},
		{"password: short", false},
		{"DB_HOST=db.example.com", false},
		// Keys that name or point at a secret, not hold one.
		{"      secretName: tls-cert-production", false},
		{"      tokenUrl: https://auth.example.com/oauth/token", false},
		{"POSTGRES_PASSWORD_FILE=/run/secrets/db_password", false},
		// The value must be on the key's own line.
		{"password:\n  valueFrom: vault-secret-ref", false},
	}
	for _, tt := range tests {
		if got := re.MatchString(tt.input); got != tt.want {
			t.Errorf("config pattern on %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}
