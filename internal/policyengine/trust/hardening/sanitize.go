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

func containsBase64Padding(s string) bool {
	// Look for base64-encoded blocks (strings of 20+ base64 chars ending in padding)
	for i := 0; i < len(s)-20; i++ {
		if isBase64Block(s[i:]) {
			return true
		}
	}
	return false
}

func isBase64Block(s string) bool {
	count := 0
	for _, c := range s {
		if isBase64Char(byte(c)) {
			count++
		} else if c == '=' && count >= 20 {
			return true
		} else {
			return false
		}
	}
	return false
}

func isBase64Char(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/'
}
