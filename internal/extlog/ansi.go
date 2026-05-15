package extlog

import "regexp"

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape codes from s.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
