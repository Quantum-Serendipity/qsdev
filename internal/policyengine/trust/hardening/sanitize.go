package hardening

import "strings"

type SanitizeMode int

const (
	LightMode SanitizeMode = iota
	StrictMode
)

type SanitizeResult struct {
	Output     string
	Detections int
	Patterns   []string
}

var lightPatterns = []string{
	"<script",
	"javascript:",
	"data:text/html",
	"{{",
	"${",
}

var strictPatterns = []string{
	"<qsdev:",
	"</",
	"ignore previous",
	"system:",
}

func Sanitize(input string, mode SanitizeMode) SanitizeResult {
	lower := strings.ToLower(input)

	var matched []string

	for _, p := range lightPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			matched = append(matched, p)
		}
	}

	if mode == StrictMode {
		for _, p := range strictPatterns {
			if strings.Contains(lower, strings.ToLower(p)) {
				matched = append(matched, p)
			}
		}

		if containsBase64Padding(input) {
			matched = append(matched, "base64-encoded")
		}
	}

	return SanitizeResult{
		Output:     input,
		Detections: len(matched),
		Patterns:   matched,
	}
}

// minBase64Run is the shortest run of base64-alphabet characters that, when
// followed by '=' padding, is reported as an encoded block.
const minBase64Run = 20

// containsBase64Padding reports whether s holds a base64-encoded block: a run
// of at least minBase64Run base64-alphabet bytes immediately followed by '='
// padding. It is a single linear pass over bytes, so large tool outputs cannot
// push the hook past its timeout, and multi-byte UTF-8 runes (whose bytes are
// all >= 0x80) never count toward a run.
func containsBase64Padding(s string) bool {
	run := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case isBase64Char(c):
			run++
		case c == '=' && run >= minBase64Run:
			return true
		default:
			run = 0
		}
	}
	return false
}

func isBase64Char(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/'
}
