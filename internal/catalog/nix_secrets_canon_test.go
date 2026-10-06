package catalog

import (
	"regexp"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
)

// TestNixSecretsCheckPatternsSubsetOfCanon keeps the nix-secrets-check hook's
// credential_patterns inside the internal/secrets value-pattern canon: each
// entry, compiled as a Go regexp, must match a sample of some canon entry, so
// the hook never looks for a token shape the canon does not know.
func TestNixSecretsCheckPatternsSubsetOfCanon(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	var patterns []string
	for _, h := range cat.CustomHooks() {
		if h.ID == "nix-secrets-check" {
			patterns = h.CredentialPatterns
		}
	}
	if len(patterns) == 0 {
		t.Fatal("nix-secrets-check has no credential_patterns")
	}

	samples := secretstest.ValuePatternSamples()
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			t.Errorf("credential_patterns entry %q does not compile: %v", p, err)
			continue
		}
		if name, ok := matchedCanonEntry(re, samples); ok {
			t.Logf("%q is covered by canon entry %q", p, name)
			continue
		}
		t.Errorf("credential_patterns entry %q matches no sample of any canon entry in internal/secrets", p)
	}
}

// matchedCanonEntry returns the name of the first canon entry with a sample re
// matches.
func matchedCanonEntry(re *regexp.Regexp, samples map[string][]string) (string, bool) {
	for _, vp := range secrets.ValuePatterns {
		for _, s := range samples[vp.Name] {
			if re.MatchString(s) {
				return vp.Name, true
			}
		}
	}
	return "", false
}
