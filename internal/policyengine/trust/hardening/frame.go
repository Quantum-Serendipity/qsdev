package hardening

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

type TrustLevel string

const (
	TrustUntrusted TrustLevel = "untrusted"
	TrustModerate  TrustLevel = "moderate"
	TrustTrusted   TrustLevel = "trusted"
)

// frameTagPrefix is the element-name prefix of every frame Frame emits.
const frameTagPrefix = "qsdev:data-"

// frameTagRe matches the opening of any qsdev start or end tag in untrusted
// content, tolerating case changes and whitespace an attacker might use to
// slip a forged tag past an exact-match check ("< /QSDEV:data"). Private Use
// Area runes count as whitespace: datamarking, which runs before framing,
// replaces the whitespace of such a tag with a PUA marker rune.
var frameTagRe = regexp.MustCompile(`(?i)<([\s\p{Co}]*/?[\s\p{Co}]*qsdev:)`)

// Frame wraps MCP tool output in a provenance element for the model. The
// element name carries a per-call random nonce the content cannot predict, and
// every qsdev tag opener inside the content is escaped, so untrusted output can
// neither close the frame early nor open a forged (for example "trusted") one.
func Frame(input, serverName string, tier int, source string) string {
	tag := frameTagPrefix + rand.Text()
	return fmt.Sprintf(
		"<%s server=%q tier=\"tier-%d\" source=%q trust=%q>\n%s\n</%s>",
		tag, serverName, tier, source, tierToTrustLevel(tier), neutralizeFrameTags(input), tag,
	)
}

// neutralizeFrameTags escapes the "<" of every qsdev start or end tag in s so
// the content cannot be parsed as frame markup.
func neutralizeFrameTags(s string) string {
	return frameTagRe.ReplaceAllString(s, "&lt;$1")
}

func tierToTrustLevel(tier int) TrustLevel {
	switch tier {
	case 1:
		return TrustTrusted
	case 2:
		return TrustModerate
	default:
		return TrustUntrusted
	}
}
