package logging

import (
	"log/slog"
	"testing"
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
