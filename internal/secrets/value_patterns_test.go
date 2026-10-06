package secrets_test

import (
	"regexp"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
)

// TestValuePatterns_Canon checks the canon's shape and its sample table: every
// entry is named uniquely, compiles in RE2, has samples, and matches each of
// them; the sample table names nothing outside the canon.
func TestValuePatterns_Canon(t *testing.T) {
	t.Parallel()
	if len(secrets.ValuePatterns) == 0 {
		t.Fatal("ValuePatterns is empty")
	}
	samples := secretstest.ValuePatternSamples()
	seen := make(map[string]bool, len(secrets.ValuePatterns))
	for _, vp := range secrets.ValuePatterns {
		if vp.Name == "" {
			t.Errorf("entry with regex %q has an empty Name", vp.Regex)
			continue
		}
		if seen[vp.Name] {
			t.Errorf("duplicate Name %q", vp.Name)
		}
		seen[vp.Name] = true
		t.Run(vp.Name, func(t *testing.T) {
			t.Parallel()
			re, err := regexp.Compile(vp.Regex)
			if err != nil {
				t.Fatalf("regex %q does not compile: %v", vp.Regex, err)
			}
			got := samples[vp.Name]
			if len(got) == 0 {
				t.Fatalf("no sample in secretstest.ValuePatternSamples() for %q", vp.Name)
			}
			for _, s := range got {
				if !re.MatchString(s) {
					t.Errorf("regex %q does not match its sample %q", vp.Regex, s)
				}
			}
		})
	}
	for name := range samples {
		if !seen[name] {
			t.Errorf("samples name %q, which is not a canon entry", name)
		}
	}
}

// TestValuePatterns_Negatives holds near-miss values each entry must reject
// (migrated from the claudecode token-shape tables).
func TestValuePatterns_Negatives(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		input   string
	}{
		{"short AKIA", "aws", "AKIA1234"},
		{"lower-case AKIA", "aws", "AKIAiosfodnn7realkey"},
		{"GitHub wrong prefix", "github", "ghx_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"},
		{"short GitHub token", "github", "ghp_short"},
		{"GitLab no dash", "gitlab", "glpat_no_dash_here"},
		{"certificate", "private-key", "-----BEGIN CERTIFICATE-----"},
		{"public key", "private-key", "-----BEGIN PUBLIC KEY-----"},
		{"short JWT", "jwt", "eyJ.eyJ.abc"},
		{"HTTP URL", "db-url", "https://example.com/api/endpoint"},
		{"PostgreSQL no credentials", "db-url", "postgres://localhost:5432/testdb"},
		{"Redis no credentials", "db-url", "redis://localhost:6379"},
		{"MongoDB no credentials", "db-url", "mongodb://localhost:27017/mydb"},
		{"Slack wrong prefix", "slack", "xoxx-not-a-token"},
		{"Stripe publishable key", "stripe", "pk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZabcde"},
		{"short Stripe key", "stripe", "sk_live_short"},
		{"Stripe publishable is not restricted", "stripe-restricted", "pk_test_ABCDEFGHIJKLMNOPQRSTUVWXYZabcde"},
		{"Anthropic prefix only", "anthropic", "sk-ant-short"},
		{"unrelated sk- word", "openai", "sk-learn-is-a-library"},
		{"Google short", "google-api", "AIzaShort"},
		{"npm word", "npm", "npm_install"},
		{"short npm token", "npm", "npm_" + "abc123"},
		{"Vault wrong prefix", "vault", "hvx.ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"short age key", "age", "AGE-SECRET-KEY-1ABC"},
		{"short Docker PAT", "docker", "dckr_pat_short"},
		{"short Hugging Face token", "huggingface", "hf_short"},
		{"PyPI word", "pypi", "pypi-server"},
	}
	byName := make(map[string]*regexp.Regexp, len(secrets.ValuePatterns))
	for _, vp := range secrets.ValuePatterns {
		byName[vp.Name] = regexp.MustCompile(vp.Regex)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re, ok := byName[tt.pattern]
			if !ok {
				t.Fatalf("no canon entry named %q", tt.pattern)
			}
			if re.MatchString(tt.input) {
				t.Errorf("%s regex %q should not match %q", tt.pattern, re, tt.input)
			}
		})
	}
}

// TestCompiledValuePatterns_Memoized checks CompiledValuePatterns compiles once
// and returns the canon in order.
func TestCompiledValuePatterns_Memoized(t *testing.T) {
	t.Parallel()
	first := secrets.CompiledValuePatterns()
	second := secrets.CompiledValuePatterns()
	if len(first) != len(secrets.ValuePatterns) {
		t.Fatalf("CompiledValuePatterns() has %d entries, want %d", len(first), len(secrets.ValuePatterns))
	}
	if len(first) > 0 && &first[0] != &second[0] {
		t.Error("CompiledValuePatterns() returned a fresh slice on the second call; want the memoized one")
	}
	for i, vp := range secrets.ValuePatterns {
		if first[i] != second[i] {
			t.Errorf("entry %d (%s) recompiled on the second call", i, vp.Name)
		}
		if got := first[i].String(); got != vp.Regex {
			t.Errorf("entry %d = %q, want %q (canon order)", i, got, vp.Regex)
		}
	}
}

// TestPrivateKeyHeaderPattern_IsCanonEntry pins the exported header constant to
// the canon's private-key entry, which logging extends into a block redaction.
func TestPrivateKeyHeaderPattern_IsCanonEntry(t *testing.T) {
	t.Parallel()
	for _, vp := range secrets.ValuePatterns {
		if vp.Name == "private-key" {
			if vp.Regex != secrets.PrivateKeyHeaderPattern {
				t.Errorf("private-key regex = %q, want PrivateKeyHeaderPattern %q", vp.Regex, secrets.PrivateKeyHeaderPattern)
			}
			return
		}
	}
	t.Error("canon has no private-key entry")
}
