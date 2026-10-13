// Package secretstest provides sample credential values for the
// internal/secrets value-pattern canon, so the canon, log redaction, the
// scan-secrets hook mirror and the catalog tests share one sample table. It is
// a regular package rather than a _test file so tests in other packages can
// share it; only _test.go files may import it (internal/archtest rule
// secretstest-test-only).
//
// Samples are assembled with strings.Repeat and concatenation: no literal
// token appears in source, which keeps them out of the binary's string table
// and away from the ripsecrets and gitleaks pre-commit hooks. No sample may
// contain a scan-secrets placeholder indicator (XXXX, TODO, EXAMPLE, ...), or
// the hook would silently treat it as a placeholder.
package secretstest

import "strings"

// ValuePatternSamples returns sample values keyed by canon entry Name
// (secrets.ValuePatterns). Each entry lists one sample per alternative of the
// pattern, so every prefix variant is exercised. It returns a fresh map on
// every call; callers may modify it.
func ValuePatternSamples() map[string][]string {
	rep := strings.Repeat
	return map[string][]string{
		"aws": {
			"AKIA" + rep("Q7", 8),
			"ASIA" + rep("Z3", 8),
		},
		"github": {
			"ghp_" + rep("Ab1", 12),
			"gho_" + rep("Ab1", 12),
			"ghu_" + rep("Ab1", 12),
			"ghs_" + rep("Ab1", 12),
			"ghr_" + rep("Ab1", 12),
		},
		"github-pat": {"github_" + "pat_" + rep("A1b2_", 5)},
		"gitlab":     {"glpat-" + rep("Ab1-", 5)},
		"stripe": {
			"sk_" + "live_" + rep("Ab1", 8),
			"sk_" + "test_" + rep("Ab1", 8),
		},
		"stripe-restricted": {
			"rk_" + "live_" + rep("Ab1", 8),
			"rk_" + "test_" + rep("Ab1", 8),
		},
		"npm": {"npm_" + rep("Ab1", 12)},
		"jwt": {
			"eyJ" + rep("hb9", 4) + ".eyJ" + rep("zd8", 4) + "." + rep("Sf7_", 4),
			// A short signature (a truncated line) and an unsigned alg:none token.
			"eyJ" + rep("hb9", 4) + ".eyJ" + rep("zd8", 4) + ".Sf7",
			"eyJ" + rep("hb9", 4) + ".eyJ" + rep("zd8", 4) + ".",
		},
		"private-key": {
			pemHeader("", ""),
			pemHeader("RSA ", ""),
			pemHeader("EC ", ""),
			pemHeader("DSA ", ""),
			pemHeader("OPENSSH ", ""),
			pemHeader("ENCRYPTED ", ""),
			pemHeader("PGP ", " BLOCK"),
		},
		"azure-storage": {"AccountKey=" + rep("Ab1+", 11) + "=="},
		"google-api":    {"AIza" + rep("Ab1_-", 7)},
		"db-url": {
			"mongodb://admin:" + password() + "@db.internal.lan:27017/app",
			"mongodb+srv://admin:" + password() + "@cluster0.internal.lan/app",
			"postgres://app:" + password() + "@pg.internal.lan:5432/app",
			"postgresql://app:" + password() + "@pg.internal.lan:5432/app",
			"mysql://root:" + password() + "@127.0.0.1:3306/app",
			"redis://default:" + password() + "@cache.internal.lan:6379",
		},
		"vault": {
			"hvs." + rep("Ab1_", 6),
			"hvb." + rep("Ab1_", 6),
			"hvr." + rep("Ab1_", 6),
		},
		"slack": {
			"xoxb-" + slackBody(),
			"xoxp-" + slackBody(),
			"xoxr-" + slackBody(),
			"xoxa-" + slackBody(),
			"xoxs-" + slackBody(),
			"xoxe-" + slackBody(),
		},
		"sendgrid": {"SG." + rep("Ab1_-", 4) + "Ab" + "." + rep("Cd2", 14) + "E"},
		"anthropic": {
			"sk-ant-" + "api03-" + rep("Ab9_", 6),
			"sk-ant-" + "admin01-" + rep("Ab9_", 6),
		},
		"openai": {
			"sk-proj-" + rep("Ab9-", 6),
			"sk-svcacct-" + rep("Ab9-", 6),
			"sk-admin-" + rep("Ab9-", 6),
			"sk-" + rep("Ab1Cd", 4) + "T3Blbk" + "FJ" + rep("Ef2Gh", 4),
		},
		"pypi": {"pypi-" + "AgE" + rep("Ab1_", 13)},
		"slack-webhook": {
			"https://hooks.slack.com/services/T0" + "1AB2CD3" + "/B0" + "4EF5GH6" + "/" + rep("Ab1", 8),
		},
		"docker":      {"dckr_" + "pat_" + rep("Ab1_", 6)},
		"huggingface": {"hf_" + rep("Ab1", 11)},
		"age":         {"AGE-SECRET-" + "KEY-1" + rep("Q8Z3", 14) + "QZ"},
	}
}

// pemHeader assembles a private-key header line for a key kind ("RSA ", or ""
// for PKCS#8) and trailer ("" or " BLOCK" for PGP).
func pemHeader(kind, trailer string) string {
	return "-----BEGIN " + kind + "PRIVATE " + "KEY" + trailer + "-----"
}

// password is a credential-looking userinfo password for connection strings.
func password() string { return "pw" + strings.Repeat("9z", 4) }

// slackBody is the token part after a Slack xox?- prefix.
func slackBody() string { return "1234567890-" + strings.Repeat("Ab1", 8) }
