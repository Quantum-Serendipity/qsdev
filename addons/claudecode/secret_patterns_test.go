package claudecode_test

import (
	"regexp"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestDefaultSecretPatterns_AllCompile(t *testing.T) {
	t.Parallel()
	for i, pattern := range claudecode.ExportDefaultSecretPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			t.Errorf("pattern %d (%q) failed to compile: %v", i, pattern, err)
		}
	}
}

func TestDefaultSecretPatterns_Count(t *testing.T) {
	t.Parallel()
	if len(claudecode.ExportDefaultSecretPatterns) != 12 {
		t.Errorf("expected 12 default patterns, got %d", len(claudecode.ExportDefaultSecretPatterns))
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
