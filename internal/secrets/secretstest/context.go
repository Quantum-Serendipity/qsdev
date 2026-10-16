package secretstest

import (
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// privateKeyHeaderRe matches a canon private-key header sample.
var privateKeyHeaderRe = regexp.MustCompile(secrets.PrivateKeyHeaderPattern)

// InContext embeds a canon sample between a fixed prefix and suffix, so a
// redaction test can assert that the sample is gone while the surrounding text
// survives. A private-key header sample is closed with a key body and its END
// line, because a header with no END is redacted through the end of the input
// by design.
func InContext(sample string) (input, prefix, suffix string) {
	prefix, suffix = "error: got ", " here"
	if privateKeyHeaderRe.MatchString(sample) {
		sample += "\nMIIEsecretkeybody\n" + strings.Replace(sample, "BEGIN", "END", 1)
	}
	return prefix + sample + suffix, prefix, suffix
}
