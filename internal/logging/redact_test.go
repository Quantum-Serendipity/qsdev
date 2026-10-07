package logging

import (
	"bytes"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
)

func TestRedactString_AWSKey(t *testing.T) {
	r := NewRedactor()
	input := "found key AKIAIOSFODNN7EXAMPLE in config"
	got := r.RedactString(input)
	if got == input {
		t.Errorf("AWS key was not redacted: %s", got)
	}
	if got != "found key [REDACTED] in config" {
		t.Errorf("unexpected result: %s", got)
	}
}

func TestRedactString_GitHubPAT(t *testing.T) {
	r := NewRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"classic", "token ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"},
		{"fine-grained", "token github_pat_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"},
		{"gho", "token gho_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.RedactString(tt.input)
			if got == tt.input {
				t.Errorf("GitHub token was not redacted: %s", got)
			}
		})
	}
}

func TestRedactString_GitLabPAT(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("token glpat-ABCDEFGHIJKLMNOPqrst")
	if got == "token glpat-ABCDEFGHIJKLMNOPqrst" {
		t.Error("GitLab token was not redacted")
	}
}

func TestRedactString_StripeKey(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("key sk_live_ABCDEFGHIJKLMNOPQRSTUVWXyz")
	if got == "key sk_live_ABCDEFGHIJKLMNOPQRSTUVWXyz" {
		t.Error("Stripe key was not redacted")
	}
}

func TestRedactString_NpmToken(t *testing.T) {
	r := NewRedactor()
	token := "npm_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	input := "//registry.npmjs.org/:_authToken=" + token
	got := r.RedactString(input)
	if got == input {
		t.Error("npm token was not redacted")
	}
}

func TestRedactString_JWT(t *testing.T) {
	r := NewRedactor()
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	got := r.RedactString("bearer " + jwt)
	if got == "bearer "+jwt {
		t.Error("JWT was not redacted")
	}
}

func TestRedactString_PrivateKey(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("-----BEGIN RSA PRIVATE KEY-----")
	if got == "-----BEGIN RSA PRIVATE KEY-----" {
		t.Error("private key header was not redacted")
	}
}

func TestRedactString_URLWithCredentials(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("connecting to https://admin:s3cret@registry.example.com/v2/")
	if got == "connecting to https://admin:s3cret@registry.example.com/v2/" {
		t.Error("URL credentials were not redacted")
	}
	if !contains(got, "registry.example.com") {
		t.Errorf("hostname should be preserved: %s", got)
	}
	if contains(got, "admin") || contains(got, "s3cret") {
		t.Errorf("credentials should be scrubbed: %s", got)
	}
}

func TestRedactString_AzureAccountKey(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	input := "AccountKey=dGhpcyBpcyBhIHRlc3QgYmFzZTY0IGVuY29kZWQga2V5IGZvcg=="
	got := r.RedactString(input)
	if got == input {
		t.Errorf("Azure account key was not redacted: %s", got)
	}
}

func TestRedactString_GCPAPIKey(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	input := "key AIzaSyA1234567890abcdefghijklmnopqrstuvw"
	got := r.RedactString(input)
	if got == input {
		t.Errorf("GCP API key was not redacted: %s", got)
	}
}

func TestRedactString_MongoDBURI(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"standard", "mongodb://admin:s3cret@cluster.example.com:27017/db"},
		{"srv", "mongodb+srv://user:pass123@cluster.mongodb.net/mydb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if got == tt.input {
				t.Errorf("MongoDB URI was not redacted: %s", got)
			}
		})
	}
}

func TestRedactString_VaultToken(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"service", "token hvs.ABCDEFGHIJKLMNOPQRSTUVWXyz"},
		{"batch", "token hvb.ABCDEFGHIJKLMNOPQRSTUVWXyz"},
		{"recovery", "token hvr.ABCDEFGHIJKLMNOPQRSTUVWXyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if got == tt.input {
				t.Errorf("Vault token was not redacted: %s", got)
			}
		})
	}
}

func TestRedactString_SlackToken(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"bot", "token xoxb-123456789-abcdefghij"},
		{"user", "token xoxp-123456789-abcdefghij"},
		{"app", "token xoxa-123456789-abcdefghij"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if got == tt.input {
				t.Errorf("Slack token was not redacted: %s", got)
			}
		})
	}
}

func TestRedactString_SafeValues(t *testing.T) {
	r := NewRedactor()
	safe := []string{
		"this is a normal log message",
		"processing 42 files in /tmp/build",
		"https://registry.npmjs.org/express",
		"version 1.2.3-beta.4",
		"Go version go1.26.1 linux/amd64",
	}
	for _, s := range safe {
		got := r.RedactString(s)
		if got != s {
			t.Errorf("safe value was incorrectly modified: %q -> %q", s, got)
		}
	}
}

func TestRedactString_NameValuePairs(t *testing.T) {
	r := NewRedactor()

	positives := []struct {
		name   string
		input  string
		secret string
	}{
		{"database password", "DATABASE_PASSWORD=hunter2", "hunter2"},
		{"pg password", "PGPASSWORD=s3cr3t", "s3cr3t"},
		{"aws secret access key", "AWS_SECRET_ACCESS_KEY=abcdef", "abcdef"},
		{"bitwarden session", "BW_SESSION=toktoktok", "toktoktok"},
		{"new relic license key", "NEW_RELIC_LICENSE_KEY=licabc", "licabc"},
		{"rails master key", "RAILS_MASTER_KEY=masterval", "masterval"},
		{"colon form", "PGPASSWORD: colonsecret", "colonsecret"},
		{"prefixed export", "export DATABASE_PASSWORD=exportedpw", "exportedpw"},
	}
	for _, tt := range positives {
		t.Run(tt.name, func(t *testing.T) {
			got := r.RedactString(tt.input)
			if strings.Contains(got, tt.secret) {
				t.Errorf("RedactString(%q) = %q, still contains secret %q", tt.input, got, tt.secret)
			}
			if !strings.Contains(got, redacted) {
				t.Errorf("RedactString(%q) = %q, expected %s marker", tt.input, got, redacted)
			}
		})
	}

	// Non-sensitive NAME=value pairs must be left untouched.
	negatives := []string{
		"PATH=/usr/bin",
		"KEYBOARD=us",
		"HOME=/home/x",
	}
	for _, in := range negatives {
		t.Run("safe/"+in, func(t *testing.T) {
			if got := r.RedactString(in); got != in {
				t.Errorf("RedactString(%q) = %q, want unchanged", in, got)
			}
		})
	}
}

func TestRedactString_MultiWordNamedValues(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name        string
		input       string
		want        string   // exact expected output ("" to skip the exact check)
		wantAbsent  []string // substrings that must not survive redaction
		wantPresent []string // substrings that must be preserved
	}{
		{
			name:       "space-separated passphrase redacts entire value",
			input:      "password: correct horse battery staple",
			want:       "password: " + redacted,
			wantAbsent: []string{"correct", "horse", "battery", "staple"},
		},
		{
			name:       "multi-word value after spaced separator",
			input:      "token = abc def",
			want:       "token = " + redacted,
			wantAbsent: []string{"abc", "def"},
		},
		{
			name:        "trailing pair redacted independently",
			input:       "user=alice password=hunter2 stuff",
			wantAbsent:  []string{"hunter2"},
			wantPresent: []string{"user=alice", redacted},
		},
		{
			name:        "two sensitive pairs redacted separately",
			input:       "password=my secret token=abc123",
			wantAbsent:  []string{"my", "secret", "abc123"},
			wantPresent: []string{"password=" + redacted, "token=" + redacted},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.RedactString(tt.input)
			if tt.want != "" && got != tt.want {
				t.Errorf("RedactString(%q) = %q, want %q", tt.input, got, tt.want)
			}
			for _, a := range tt.wantAbsent {
				if strings.Contains(got, a) {
					t.Errorf("RedactString(%q) = %q, still contains secret %q", tt.input, got, a)
				}
			}
			for _, p := range tt.wantPresent {
				if !strings.Contains(got, p) {
					t.Errorf("RedactString(%q) = %q, expected to contain %q", tt.input, got, p)
				}
			}
		})
	}
}

func TestRedactAttr_KeyDenyList(t *testing.T) {
	r := NewRedactor()
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"db_password", true},
		{"access_token", true},
		{"api_key", true},
		{"secret", true},
		{"credential", true},
		{"bearer", true},
		// False positives that should NOT be redacted
		{"tokenizer", false},
		{"tokenizer_count", false},
		{"passwordless", false},
		{"secretariat", false},
		// These SHOULD be redacted (underscore/hyphen boundaries)
		{"auth_token", true},
		{"api-key", true},
		{"private_key", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			attr := slog.String(tt.key, "some-value")
			got := r.RedactAttr(attr)
			wasRedacted := got.Value.String() == redacted
			if wasRedacted != tt.expected {
				t.Errorf("key %q: redacted=%v, want %v", tt.key, wasRedacted, tt.expected)
			}
		})
	}
}

func TestRedactAttr_GroupValues(t *testing.T) {
	r := NewRedactor()
	attr := slog.Group("config",
		slog.String("host", "example.com"),
		slog.String("password", "hunter2"),
		slog.String("port", "5432"),
	)
	got := r.RedactAttr(attr)
	group := got.Value.Group()
	for _, a := range group {
		if a.Key == "password" && a.Value.String() != redacted {
			t.Error("nested password should be redacted")
		}
		if a.Key == "host" && a.Value.String() != "example.com" {
			t.Error("host should be preserved")
		}
		if a.Key == "port" && a.Value.String() != "5432" {
			t.Error("port should be preserved")
		}
	}
}

func TestRedactAttr_EnvVarNames(t *testing.T) {
	r := NewRedactor()
	attr := slog.String("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	got := r.RedactAttr(attr)
	if got.Value.String() != redacted {
		t.Errorf("env var name key should trigger redaction: got %s", got.Value.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// redactCase is one RedactString table row: an exact expected output ("" to
// skip) plus substrings that must not survive and ones that must be preserved.
type redactCase struct {
	name        string
	input       string
	want        string
	wantAbsent  []string
	wantPresent []string
}

// runRedactCases runs each case as a parallel subtest against r.
func runRedactCases(t *testing.T, r *Redactor, cases []redactCase) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if tt.want != "" && got != tt.want {
				t.Errorf("RedactString(%q) = %q, want %q", tt.input, got, tt.want)
			}
			for _, a := range tt.wantAbsent {
				if strings.Contains(got, a) {
					t.Errorf("RedactString(%q) = %q, still contains %q", tt.input, got, a)
				}
			}
			for _, p := range tt.wantPresent {
				if !strings.Contains(got, p) {
					t.Errorf("RedactString(%q) = %q, expected to contain %q", tt.input, got, p)
				}
			}
		})
	}
}

// TestRedactString_AuthSchemePadding covers credentials whose base64 padding
// ('=' / '==') used to look like the start of a new NAME= key, scheme-prefixed
// '=' form credentials, and colon-form (header / YAML) values that must be
// redacted through end of line.
func TestRedactString_AuthSchemePadding(t *testing.T) {
	t.Parallel()
	runRedactCases(t, NewRedactor(), []redactCase{
		{
			name:       "basic header with padding",
			input:      "Authorization: Basic dXNlcjpwYXNzd29yZA==",
			want:       "Authorization: " + redacted,
			wantAbsent: []string{"dXNlcjpwYXNzd29yZA"},
		},
		{
			name:       "bearer header with padding",
			input:      "Authorization: Bearer abcDEF123456==",
			want:       "Authorization: " + redacted,
			wantAbsent: []string{"abcDEF123456"},
		},
		{
			name:       "proxy authorization header",
			input:      "proxy-authorization: Basic dXNlcjpwYXNzd29yZA==",
			wantAbsent: []string{"dXNlcjpwYXNzd29yZA"},
		},
		{
			name:        "curl header argument",
			input:       "curl -H 'Authorization: Basic dXNlcjpwYXNzd29yZA=='",
			wantAbsent:  []string{"dXNlcjpwYXNzd29yZA"},
			wantPresent: []string{"curl -H 'Authorization: " + redacted},
		},
		{
			name:       "colon form value with internal equals",
			input:      "token: prefix abcdefXYZ=123",
			want:       "token: " + redacted,
			wantAbsent: []string{"abcdefXYZ", "=123"},
		},
		{
			name:        "equals form padding then a later pair",
			input:       "AUTH_HEADER=Basic dXNlcjpwYXNzd29yZA== next=1",
			want:        "AUTH_HEADER=" + redacted + " next=1",
			wantAbsent:  []string{"dXNlcjpwYXNzd29yZA"},
			wantPresent: []string{"next=1"},
		},
		{
			name:       "equals form bearer credential with internal equals",
			input:      "AUTH_HEADER=Bearer abc=123 next=1",
			want:       "AUTH_HEADER=" + redacted + " next=1",
			wantAbsent: []string{"abc=123"},
		},
		{
			name:       "equals form lowercase token scheme",
			input:      "authorization=token abc=def other=2",
			want:       "authorization=" + redacted + " other=2",
			wantAbsent: []string{"abc=def"},
		},
		{
			name:        "map toString header dump with single padding",
			input:       "headers={Authorization=Basic dXNlcjpwYXNzd29yZE=, Accept=application/json}",
			wantAbsent:  []string{"dXNlcjpwYXNzd29yZE"},
			wantPresent: []string{"headers={Authorization=" + redacted, "Accept=application/json}"},
		},
		{
			name:       "padding followed by closing brace",
			input:      "Authorization=Basic dXNlcjpwYXNzd29yZE=}",
			wantAbsent: []string{"dXNlcjpwYXNzd29yZE"},
		},
		{
			name:        "spaced separator padding then comma pair",
			input:       "Authorization = Basic dXNlcjpwYXNzd29yZE=, x=1",
			wantAbsent:  []string{"dXNlcjpwYXNzd29yZE"},
			wantPresent: []string{"x=1"},
		},
		{
			name:        "multimap header dump with bracketed bearer",
			input:       "{Authorization=[Bearer abcDEF123456=], Accept=[json]}",
			wantAbsent:  []string{"abcDEF123456"},
			wantPresent: []string{"Accept=[json]}"},
		},
		{
			name:       "digest header with internal pairs",
			input:      `Authorization: Digest username="u", response="r3sp"`,
			want:       "Authorization: " + redacted,
			wantAbsent: []string{"r3sp"},
		},
		{
			name:  "logfmt quoted colon value keeps closing quote and later pairs",
			input: `msg="auth: Basic dXNlcjpwYXNzd29yZA==" user=bob`,
			want:  `msg="auth: ` + redacted + `" user=bob`,
		},
		{
			name:  "single-quoted dict repr header",
			input: "{'Authorization': 'Basic dXNlcjpwYXNzd29yZA==', 'X': 1}",
			want:  "{'Authorization': '" + redacted + "', 'X': 1}",
		},
		{
			name:       "single-quoted dict repr unquoted value fails closed",
			input:      "{'password': 12345, 'x': 1}",
			want:       "{'password': " + redacted,
			wantAbsent: []string{"12345"},
		},
		{
			name:       "single-quoted dict repr unterminated value fails closed",
			input:      "{'token': 'abc123",
			want:       "{'token': '" + redacted,
			wantAbsent: []string{"abc123"},
		},
		{
			name:  "single-quoted shell header keeps its closing quote",
			input: "curl -H 'Authorization: Basic dXNlcjpwYXNzd29yZA==' https://x",
			want:  "curl -H 'Authorization: " + redacted + "' https://x",
		},
		{
			name:  "single-quoted benign member unchanged",
			input: "{'status': 'ok', 'pass': 3}",
			want:  "{'status': 'ok', 'pass': 3}",
		},
		{
			name:  "colon form redacts rest of line (fail closed)",
			input: "password: hunter2 status: ok user=bob\nnext: line",
			want:  "password: " + redacted + "\nnext: line",
		},
		{
			name:  "equals pairs redacted independently",
			input: "password=a token=b",
			want:  "password=" + redacted + " token=" + redacted,
		},
		{
			name:  "non-sensitive pairs unchanged",
			input: "a=b c=d",
			want:  "a=b c=d",
		},
		{
			name:  "comma-separated pair after benign pair",
			input: "user=alice,password=hunter2",
			want:  "user=alice,password=" + redacted,
		},
	})
}

// TestRedactString_URLAndFlags covers empty-user URL userinfo, URLs whose path
// holds an '@' (never userinfo), and the "--flag value" form of a sensitive
// command-line flag inside free text.
func TestRedactString_URLAndFlags(t *testing.T) {
	t.Parallel()
	runRedactCases(t, NewRedactor(), []redactCase{
		{
			name:       "empty-user redis url",
			input:      "redis://:hunter2@cache:6379",
			wantAbsent: []string{"hunter2"},
		},
		{
			name:        "empty-user url embedded in text",
			input:       "connect redis://:hunter2@cache:6379 failed",
			wantAbsent:  []string{"hunter2"},
			wantPresent: []string{"connect ", "cache:6379 failed"},
		},
		{
			name:        "password containing a colon",
			input:       "connect redis://u:hun:ter2@cache:6379 failed",
			wantAbsent:  []string{"hun:ter2", "ter2"},
			wantPresent: []string{"cache:6379 failed"},
		},
		{
			name:        "password containing a slash",
			input:       "postgres://u:pw/x@host/db",
			wantAbsent:  []string{"pw/x"},
			wantPresent: []string{"@host/db"},
		},
		{
			name:        "base64-like password in embedded dsn",
			input:       "dial mysql://root:Zm9v/YmFy+cXV4@db:3306/app failed",
			wantAbsent:  []string{"Zm9v", "YmFy+cXV4"},
			wantPresent: []string{"dial ", "@db:3306/app failed"},
		},
		{
			name:       "password containing a question mark",
			input:      "https://u:a?b@host/",
			wantAbsent: []string{"a?b"},
		},
		{
			name:       "password containing a hash",
			input:      "https://u:a#b@host/",
			wantAbsent: []string{"a#b"},
		},
		{
			name:       "password containing an at sign",
			input:      "connect https://user:p@ss@host/ now",
			wantAbsent: []string{"p@ss"},
		},
		{
			name:  "unencoded at in url path unchanged",
			input: "GET http://localhost:4873/@types%2fnode 200",
			want:  "GET http://localhost:4873/@types%2fnode 200",
		},
		{
			name:  "ipv6 host with at sign in path unchanged",
			input: "GET http://[::1]:4873/@types%2fnode 200",
			want:  "GET http://[::1]:4873/@types%2fnode 200",
		},
		{
			name:  "port and colon path before at sign unchanged",
			input: "see http://h:8080/a:b@c",
			want:  "see http://h:8080/a:b@c",
		},
		{
			name:  "scoped package path behind a port unchanged",
			input: "GET http://localhost:4873/@scope/pkg 200",
			want:  "GET http://localhost:4873/@scope/pkg 200",
		},
		{
			name:  "long flag with separate value",
			input: "run --password hunter2 now",
			want:  "run --password " + redacted + " now",
		},
		{
			name:  "quoted multi-word flag value",
			input: `run --password "hunter 2" now`,
			want:  "run --password " + redacted + " now",
		},
		{
			name:       "unterminated quoted flag value fails closed",
			input:      "run --password 'hunter 2 now",
			want:       "run --password " + redacted,
			wantAbsent: []string{"hunter", "2 now"},
		},
		{
			name:  "stdin credential flag takes no value",
			input: "docker login --password-stdin registry.example.com",
			want:  "docker login --password-stdin registry.example.com",
		},
		{
			name:        "hyphenated flag followed by another flag",
			input:       "--api-key k3y --verbose",
			wantAbsent:  []string{"k3y"},
			wantPresent: []string{"--api-key ", "--verbose"},
		},
		{
			name:  "credential file path flag keeps its path",
			input: "run --token-file /home/u/.tok --password-file=/run/pw",
			want:  "run --token-file /home/u/.tok --password-file=/run/pw",
		},
		{
			name:  "credential dir flag keeps its path",
			input: "content verify --KEYS-DIR /home/u/.qsdev/keys",
			want:  "content verify --KEYS-DIR /home/u/.qsdev/keys",
		},
		{
			// Deliberate fail-closed over-redaction: "keys" is a plural of the
			// sensitive "key" token, so a bare --keys value is withheld even
			// when it is a directory (qsdev content verify --keys <dir>).
			name:  "plural credential flag redacts its value",
			input: "content verify --keys /home/u/.qsdev/keys",
			want:  "content verify --keys " + redacted,
		},
		{
			// Deliberate fail-closed over-redaction: "tokens" is a plural of
			// the sensitive "token" token, so a token-count pair is withheld.
			name:  "token count pair redacted",
			input: "max_tokens=4096 temperature=1",
			want:  "max_tokens=" + redacted + " temperature=1",
		},
		{
			name:  "attached short port flag unchanged",
			input: "docker run -p8080:80",
			want:  "docker run -p8080:80",
		},
		{
			name:  "separate short port flag unchanged",
			input: "docker run -p 8080:80",
			want:  "docker run -p 8080:80",
		},
		{
			name:  "sensitive flag at end of string unchanged",
			input: "--password",
			want:  "--password",
		},
		{
			name:  "sensitive flag at end of text unchanged",
			input: "run --password",
			want:  "run --password",
		},
	})
}

// TestRedactStructured_StdinFlagTakesNoValue proves the argv walk shares the
// "-stdin" and path-flag exemptions: the argument after --password-stdin or
// --token-file is not a secret,
// while a value-bearing credential flag still redacts its argument.
func TestRedactStructured_StdinFlagTakesNoValue(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	got := r.RedactStructured([]string{"login", "--password-stdin", "registry.example.com", "--token-file", "/run/tok", "--api-key", "k3y"})
	want := []string{"login", "--password-stdin", "registry.example.com", "--token-file", "/run/tok", "--api-key", redacted}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RedactStructured = %v, want %v", got, want)
	}
}

// TestAuthCredentialEnd_NoAllocs guards the per-pair scheme scan on the
// NAME=value hot path: it must not allocate on any OS.
func TestAuthCredentialEnd_NoAllocs(t *testing.T) {
	inputs := []string{"Bearer abc=123 next=1", "hunter2 user=bob", "Basic dXNlcjpwYXNzd29yZA=="}
	for _, in := range inputs {
		if n := testing.AllocsPerRun(100, func() { _ = authCredentialEnd(in, 0) }); n != 0 {
			t.Errorf("authCredentialEnd(%q) allocated %v times, want 0", in, n)
		}
	}
	if n := testing.AllocsPerRun(100, func() { _ = isSensitiveFlag("--Password-STDIN") || isSensitiveFlag("--token-FILE") }); n != 0 {
		t.Errorf("isSensitiveFlag allocated %v times, want 0", n)
	}
}

// tokenShapeInput embeds a canon sample between fixed prefix and suffix text.
// A private-key header sample is closed with its END line, because a header
// with no END is redacted through the end of the input by design.
func tokenShapeInput(sample string) (input, prefix, suffix string) {
	prefix, suffix = "error: got ", " here"
	if privateKeyBeginRe.MatchString(sample) {
		sample += "\nMIIEsecretkeybody\n" + strings.Replace(sample, "BEGIN", "END", 1)
	}
	return prefix + sample + suffix, prefix, suffix
}

// TestRedactString_TokenShapes proves the log redactor covers every token
// shape in the secrets canon: each sample is scrubbed from a plain string and
// from a JSON slog record routed through RedactingHandler, while the fixed
// text around it survives.
func TestRedactString_TokenShapes(t *testing.T) {
	t.Parallel()

	samples := secretstest.ValuePatternSamples()
	r := NewRedactor()
	for _, vp := range secrets.ValuePatterns {
		if len(samples[vp.Name]) == 0 {
			t.Errorf("canon entry %q has no secretstest sample", vp.Name)
		}
		for i, sample := range samples[vp.Name] {
			input, prefix, suffix := tokenShapeInput(sample)
			t.Run(fmt.Sprintf("%s/%d", vp.Name, i), func(t *testing.T) {
				t.Parallel()

				got := r.RedactString(input)
				if strings.Contains(got, sample) {
					t.Errorf("RedactString left the %s sample: %q", vp.Name, got)
				}
				if !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
					t.Errorf("RedactString lost the surrounding text: %q", got)
				}

				var buf bytes.Buffer
				logger := slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, nil)))
				logger.Info(input, "detail", input)
				out := buf.String()
				if strings.Contains(out, sample) {
					t.Errorf("RedactingHandler left the %s sample: %s", vp.Name, out)
				}
				if !strings.Contains(out, `"msg":"error: got `) || !strings.Contains(out, ` here","detail":"error: got `) {
					t.Errorf("RedactingHandler lost the surrounding text: %s", out)
				}
			})
		}
	}
}
