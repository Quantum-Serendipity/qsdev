package hardening

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
)

// datamarkSource names the content kind in the datamark framing.
const datamarkSource = "MCP tool output"

// datamarkPrefixRe matches the opening of any qsdev datamark marker in
// untrusted content, tolerating case changes and whitespace ("[ qsdev:END").
var datamarkPrefixRe = regexp.MustCompile(`(?i)\[(\s*qsdev:)`)

// Datamark applies the shared contentsign datamarking transform (hardening
// Layer 1) to MCP tool output: prose whitespace is replaced with a randomized
// Private Use Area marker rune so injected instructions stop reading as
// instructions, fenced and inline code is preserved, and the result is wrapped
// in framing that tells the model what the marker means. It deliberately reuses
// contentsign rather than a local copy so hardening fixes land in one place.
//
// Every qsdev marker opener in the content is escaped and the marked result is
// bracketed between begin and end markers that carry a per-call random nonce,
// so the output can neither end the marked region early nor forge a marker the
// reader would mistake for a real one.
func Datamark(input string) string {
	sum := sha256.Sum256([]byte(input))
	opts := contentsign.DefaultDatamarkOptions()
	opts.Source = datamarkSource
	opts.VerificationStatus = "unverified"
	opts.ContentHashPrefix = hex.EncodeToString(sum[:6])
	// Neutralize before the transform too: it replaces whitespace with the
	// marker rune, which would otherwise hide a spaced "[ qsdev:" opener from
	// the escape.
	marked, _ := contentsign.Datamark(neutralizeDatamarks(input), opts)

	nonce := rand.Text()
	return "[QSDEV:BEGIN " + nonce + "]" + neutralizeDatamarks(marked) + "[QSDEV:END " + nonce + "]"
}

// neutralizeDatamarks escapes the "[" of every qsdev marker opener in s.
func neutralizeDatamarks(s string) string {
	return datamarkPrefixRe.ReplaceAllString(s, "&#91;$1")
}
