package hardening

import (
	"crypto/rand"
	"regexp"
)

// datamarkPrefixRe matches the opening of any qsdev datamark marker in
// untrusted content, tolerating case changes and whitespace ("[ qsdev:END").
var datamarkPrefixRe = regexp.MustCompile(`(?i)\[(\s*qsdev:)`)

// Datamark brackets MCP tool output between begin and end markers that carry
// a per-call random nonce, and escapes every qsdev marker opener inside the
// content, so the output can neither end the marked region early nor forge a
// marker the reader would mistake for a real one.
func Datamark(input string) string {
	nonce := rand.Text()
	return "[QSDEV:BEGIN " + nonce + "]" + neutralizeDatamarks(input) + "[QSDEV:END " + nonce + "]"
}

// neutralizeDatamarks escapes the "[" of every qsdev marker opener in s.
func neutralizeDatamarks(s string) string {
	return datamarkPrefixRe.ReplaceAllString(s, "&#91;$1")
}
