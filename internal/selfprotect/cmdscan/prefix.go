package cmdscan

import (
	"errors"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// maxPrefixAttempts bounds how many shorter prefixes ExecutedPrefix parses
// when the lines just before the syntax error do not parse on their own (an
// unclosed construct spanning them), so a hostile line costs a bounded number
// of parses.
const maxPrefixAttempts = 16

// ExecutedPrefix returns the complete lines of an unparseable command that
// bash still runs: bash parses and executes a script line by line, so every
// complete command on the lines before the one holding the syntax error has
// already run when the error stops it (`cp x y` then `fi` copies). It is the
// longest run of whole lines before the error's line that parses, or "" when
// command parses, the error comes on its first line, or no such prefix is
// found within maxPrefixAttempts.
func ExecutedPrefix(command string) string {
	_, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err == nil {
		return ""
	}
	lines := strings.SplitAfter(command, "\n")
	end := len(lines)
	if line, ok := errorLine(err); ok && line-1 < end {
		end = line - 1
	}
	for attempt := 0; end > 0 && attempt < maxPrefixAttempts; attempt, end = attempt+1, end-1 {
		prefix := strings.Join(lines[:end], "")
		if strings.TrimSpace(prefix) == "" {
			return ""
		}
		if _, err := syntax.NewParser().Parse(strings.NewReader(prefix), ""); err == nil {
			return prefix
		}
	}
	return ""
}

// errorLine returns the 1-based line a parse error points at.
func errorLine(err error) (int, bool) {
	var pe syntax.ParseError
	if errors.As(err, &pe) {
		return int(pe.Pos.Line()), true
	}
	var le syntax.LangError
	if errors.As(err, &le) {
		return int(le.Pos.Line()), true
	}
	return 0, false
}
