package claudecode_test

import (
	"regexp"
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

// TestScanHeuristicPatterns_PositiveNegative covers the scan-only KEY="value"
// assignment heuristics, keyed by heuristic name. Token shapes are covered by
// the canon tests in internal/secrets.
func TestScanHeuristicPatterns_PositiveNegative(t *testing.T) {
	t.Parallel()
	byName := make(map[string]*regexp.Regexp, len(claudecode.ExportScanHeuristicPatterns))
	for _, h := range claudecode.ExportScanHeuristicPatterns {
		if h.Name == "" || byName[h.Name] != nil {
			t.Fatalf("heuristic name %q is empty or duplicated", h.Name)
		}
		byName[h.Name] = regexp.MustCompile(h.Regex)
	}
	tests := []struct {
		name      string
		heuristic string
		input     string
		want      bool
	}{
		{"AWS secret key assignment", "aws-assignment", "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYzzzzzz", true},
		{"AWS session token", "aws-assignment", "aws_session_token: ABCDEFGHIJKLMNOPQRSTzzzz", true},
		{"AWS secret key UPPERCASE", "aws-assignment", "AWS_SECRET_ACCESS_KEY = wJalrXUtnFEMI/K7MDENG/bPxRfiCYzzzzzz", true},
		{"AWS session token UPPERCASE", "aws-assignment", "AWS_SESSION_TOKEN: ABCDEFGHIJKLMNOPQRSTzzzz", true},
		{"AWS region is not a secret", "aws-assignment", "aws_region = us-east-1", false},
		{"AWS secret key short value", "aws-assignment", "aws_secret_access_key = short", false},
		{"API key double-quoted", "api-key-assignment", `API_KEY = "abcdefghijklmnopqrstuvwx"`, true},
		{"API key single-quoted", "api-key-assignment", `api_key: 'abcdefghijklmnopqrstuvwx'`, true},
		{"API key no value", "api-key-assignment", "API_KEY = ", false},
		{"API key short value", "api-key-assignment", `API_KEY = "short"`, false},
		{"Password assignment", "secret-assignment", `password = "supersecretpassword123"`, true},
		{"Secret assignment", "secret-assignment", `secret: "my_very_secret_value_here"`, true},
		{"Token assignment", "secret-assignment", `token = "abcdefghijklmnopqrstuvwxyz"`, true},
		{"PASSWORD uppercase", "secret-assignment", `PASSWORD = "supersecretpassword123"`, true},
		{"SECRET uppercase", "secret-assignment", `SECRET: "my_very_secret_value_here"`, true},
		{"TOKEN uppercase", "secret-assignment", `TOKEN = "abcdefghijklmnopqrstuvwxyz"`, true},
		{"JSON password", "secret-assignment", `{"password": "hunter2hunter2"}`, true},
		{"Short password", "secret-assignment", `password = "short"`, false},
		{"Password no quotes", "secret-assignment", "password = noquotes", false},
		{"Env var reference", "secret-assignment", "password = ${DB_PASSWORD}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := byName[tt.heuristic]
			if re == nil {
				t.Fatalf("no scan heuristic named %q", tt.heuristic)
			}
			if got := re.MatchString(tt.input); got != tt.want {
				t.Errorf("heuristic %q on %q = %v, want %v", tt.heuristic, tt.input, got, tt.want)
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
