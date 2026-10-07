package secrets

import (
	"regexp"
	"sync"
)

// ValuePattern is one credential token shape in the canon: a stable Name
// (used to key test samples) and the regular expression that matches a
// credential value of that shape.
type ValuePattern struct {
	Name  string
	Regex string
}

// PrivateKeyHeaderPattern matches the header line of a PEM or PGP private key
// ("-----BEGIN RSA PRIVATE KEY-----", "-----BEGIN PGP PRIVATE KEY BLOCK-----").
// The canon owns only the header shape; extending a match to the END line is a
// redaction mechanic that belongs to the consumer.
const PrivateKeyHeaderPattern = `-----BEGIN [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?-----`

// ValuePatterns is the single canon of credential token shapes, shared by the
// log redactor, the scan-secrets hook and the nix secrets check. It holds only
// token shapes: KEY="value" assignment heuristics stay with the scanner.
//
// Every Regex must work unchanged in both Go RE2 and Python re, because the
// same strings are mirrored into the scan-secrets.py hook: no lookarounds, no
// \z, no mid-pattern inline flags. Every entry needs at least one sample in
// secretstest.ValuePatternSamples.
var ValuePatterns = []ValuePattern{
	// AWS access key IDs: long-term (AKIA) and temporary STS (ASIA).
	{Name: "aws", Regex: `(AKIA|ASIA)[0-9A-Z]{16}`},
	// GitHub classic, OAuth, user-to-server, server and refresh tokens.
	{Name: "github", Regex: `gh[pousr]_[A-Za-z0-9_]{36,}`},
	// GitHub fine-grained personal access tokens.
	{Name: "github-pat", Regex: `github_pat_[A-Za-z0-9_]{22,}`},
	// GitLab personal access tokens.
	{Name: "gitlab", Regex: `glpat-[A-Za-z0-9_-]{20,}`},
	// Stripe secret keys.
	{Name: "stripe", Regex: `sk_(live|test)_[A-Za-z0-9]{20,}`},
	// Stripe restricted keys.
	{Name: "stripe-restricted", Regex: `rk_(live|test)_[A-Za-z0-9]{20,}`},
	// npm access tokens: exactly 36 characters (30 random, 6 checksum). An
	// open-ended count would take a placeholder word written after a real
	// token (`npm_<36>TODO`) into the match, and scan-secrets would skip it.
	{Name: "npm", Regex: `npm_[A-Za-z0-9]{36}`},
	// JSON Web Tokens (three base64url segments). The signature may be short or
	// empty: a truncated log line or an unsigned alg:none token still carries
	// the header and claims, which are the sensitive part.
	{Name: "jwt", Regex: `eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]*`},
	// PEM private keys, including encrypted keys and PGP secret key blocks.
	{Name: "private-key", Regex: PrivateKeyHeaderPattern},
	// Azure storage account keys in connection strings.
	{Name: "azure-storage", Regex: `AccountKey=[A-Za-z0-9+/=]{44,}`},
	// Google API keys.
	{Name: "google-api", Regex: `AIza[0-9A-Za-z_-]{35}`},
	// Database connection strings with userinfo credentials.
	{Name: "db-url", Regex: `(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"':]+:[^\s"'@]+@[^\s"']{5,}`},
	// HashiCorp Vault service, batch and recovery tokens.
	{Name: "vault", Regex: `hv[sbr]\.[A-Za-z0-9_-]{24,}`},
	// Slack API tokens.
	{Name: "slack", Regex: `xox[bprase]-[A-Za-z0-9-]{10,}`},
	// SendGrid API keys.
	{Name: "sendgrid", Regex: `SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`},
	// Anthropic API and admin keys.
	{Name: "anthropic", Regex: `sk-ant-[A-Za-z0-9_-]{20,}`},
	// OpenAI project, service-account, admin and legacy keys.
	{Name: "openai", Regex: `sk-(proj|svcacct|admin)-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20}`},
	// PyPI API tokens.
	{Name: "pypi", Regex: `pypi-[A-Za-z0-9_-]{50,}`},
	// Slack incoming-webhook URLs, found anywhere in text. The leading
	// (?:^|\b) is zero-width, so the match is still exactly the URL: \b
	// starts it at a token boundary (line start, whitespace, a quote, '=').
	// The ^ alternative is redundant for matching but is what CodeQL's
	// go/regex/missing-regexp-anchor takes as an anchor; a bare \b is not,
	// and this detector must not be anchored to the whole input
	// (TestValuePatterns_NoUnanchoredURLAlert).
	{Name: "slack-webhook", Regex: `(?:^|\b)https://hooks\.slack\.com/services/T[A-Za-z0-9]+/B[A-Za-z0-9]+/[A-Za-z0-9]+`},
	// Docker Hub personal access tokens.
	{Name: "docker", Regex: `dckr_pat_[A-Za-z0-9_-]{20,}`},
	// Hugging Face access tokens.
	{Name: "huggingface", Regex: `hf_[A-Za-z0-9]{30,}`},
	// age secret keys.
	{Name: "age", Regex: `AGE-SECRET-KEY-1[0-9A-Z]{58}`},
}

var compiledValuePatterns = sync.OnceValue(func() []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, len(ValuePatterns))
	for i, vp := range ValuePatterns {
		compiled[i] = regexp.MustCompile(vp.Regex)
	}
	return compiled
})

// CompiledValuePatterns returns ValuePatterns compiled, in canon order. The
// slice is compiled once and shared: callers must not modify it.
func CompiledValuePatterns() []*regexp.Regexp {
	return compiledValuePatterns()
}
