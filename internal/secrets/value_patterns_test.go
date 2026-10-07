package secrets_test

import (
	"regexp"
	"strings"
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

// TestValuePatterns_FixedLengthMatch pins the entries whose token has a fixed
// length: the match is the token alone, never the word written after it, so
// a placeholder word (`npm_<36>TODO`) cannot drag a real token into the
// scan-secrets placeholder filter.
func TestValuePatterns_FixedLengthMatch(t *testing.T) {
	t.Parallel()
	npm := "npm_" + strings.Repeat("Ab1", 12)
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{"npm", npm + "XXXX", npm},
		{"npm", npm + "TODO", npm},
		{"npm", npm + "Example", npm},
		{"npm", npm, npm},
	}
	byName := make(map[string]*regexp.Regexp, len(secrets.ValuePatterns))
	for _, vp := range secrets.ValuePatterns {
		byName[vp.Name] = regexp.MustCompile(vp.Regex)
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.input, func(t *testing.T) {
			t.Parallel()
			if got := byName[tt.pattern].FindString(tt.input); got != tt.want {
				t.Errorf("%s matched %q in %q, want %q", tt.pattern, got, tt.input, tt.want)
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

// canonEntry returns the compiled canon entry named name.
func canonEntry(t *testing.T, name string) *regexp.Regexp {
	t.Helper()
	for _, vp := range secrets.ValuePatterns {
		if vp.Name == name {
			return regexp.MustCompile(vp.Regex)
		}
	}
	t.Fatalf("canon has no %s entry", name)
	return nil
}

// TestValuePatterns_SlackWebhookInContext pins that the slack-webhook entry
// still finds a webhook wherever it sits in text, and that its leading
// boundary is zero-width: the match is exactly the URL, so the log redactor
// and the scan-secrets hook replace or report nothing around it.
func TestValuePatterns_SlackWebhookInContext(t *testing.T) {
	t.Parallel()
	re := canonEntry(t, "slack-webhook")
	webhook := "https://hooks.slack.com/services/T0" + "1AB2CD3" + "/B0" + "4EF5GH6" + "/" + strings.Repeat("Ab1", 8)
	tests := []struct {
		name, text, want string
	}{
		{"whole input", webhook, webhook},
		{"start of line", webhook + " is the alert channel", webhook},
		{"start of a later line", "config:\n" + webhook + "\n", webhook},
		{"mid-line", "posting the alert to " + webhook + " now", webhook},
		{"inside double quotes", `url = "` + webhook + `"`, webhook},
		{"inside single quotes", "URL = '" + webhook + "'", webhook},
		{"after =", "SLACK_WEBHOOK_URL=" + webhook, webhook},
		{"JSON value", `{"webhook":"` + webhook + `"}`, webhook},
		{"in parentheses", "(" + webhook + ")", webhook},
		// \b needs a non-word byte or the input start before https, so a
		// URL glued onto a preceding word is not a webhook URL here.
		{"glued to a preceding word", "x" + webhook, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := re.FindString(tt.text); got != tt.want {
				t.Errorf("FindString(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

// codeQLUnanchoredURLRe and codeQLAnchorRe restate CodeQL's
// go/regex/missing-regexp-anchor heuristic (isInterestingUnanchoredRegexpString
// in github/codeql go/ql/src/Security/CWE-020/MissingRegexpAnchor.ql): a
// pattern made only of host-like characters that ends in a common TLD, plus an
// optional path, is reported unless it contains ^, $, \A or \z anywhere. The
// query's (?![a-z0-9]) after the TLD is implied by this whole-string match.
var (
	codeQLUnanchoredURLRe = regexp.MustCompile(`(?i)^[():|?a-z0-9\-\\./]+[.](?:com|org|edu|gov|uk|net|io)(?:[/#?():]\S*)?$`)
	codeQLAnchorRe        = regexp.MustCompile(`\$|\^|\\A|\\z`)
)

// codeQLReportsUnanchoredURL reports whether CodeQL's heuristic flags re.
func codeQLReportsUnanchoredURL(re string) bool {
	return codeQLUnanchoredURLRe.MatchString(re) && !codeQLAnchorRe.MatchString(re)
}

// TestValuePatterns_NoUnanchoredURLAlert keeps the canon clear of CodeQL's
// missing-anchor alert. The canon's patterns detect credentials anywhere in
// text, so a URL-shaped one cannot be anchored to the whole input; it carries
// a zero-width (?:^|\b) boundary instead, which the query accepts.
func TestValuePatterns_NoUnanchoredURLAlert(t *testing.T) {
	t.Parallel()
	// The pre-fix slack-webhook entry, which CodeQL reported: proves the
	// restated heuristic is not vacuous.
	if !codeQLReportsUnanchoredURL(`https://hooks\.slack\.com/services/T[A-Za-z0-9]+/B[A-Za-z0-9]+/[A-Za-z0-9]+`) {
		t.Fatal("the restated CodeQL heuristic does not flag the pattern CodeQL reported")
	}
	// A bare \b is no anchor to the query.
	if !codeQLReportsUnanchoredURL(`\bhttps://hooks\.slack\.com/services/T[A-Za-z0-9]+`) {
		t.Error("the restated CodeQL heuristic accepts a bare \\b, which the query does not")
	}
	for _, vp := range secrets.ValuePatterns {
		if codeQLReportsUnanchoredURL(vp.Regex) {
			t.Errorf("%s regex %q matches a URL host with no anchor; CodeQL reports go/regex/missing-regexp-anchor", vp.Name, vp.Regex)
		}
	}
}
