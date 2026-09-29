package contentsign

import (
	"bytes"
	"encoding/json"
	"strings"
)

// datamarkJSON datamarks content token-wise when it is a whole JSON object or
// array: every string token (keys included, since a map keyed by untrusted
// data carries untrusted text in its keys) is decoded, datamarked as prose and
// re-encoded, while the structural whitespace, numbers and literals between
// the strings are copied unchanged. The result is therefore the same JSON
// document with datamarked strings and still parses. It reports false, leaving
// the caller to datamark content as prose, when content is not a valid JSON
// object or array.
func datamarkJSON(content string, marker rune, opts DatamarkOptions) (string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') || !json.Valid([]byte(trimmed)) {
		return "", false
	}

	var b strings.Builder
	b.Grow(len(content))
	for i := 0; i < len(content); {
		if content[i] != '"' {
			b.WriteByte(content[i])
			i++
			continue
		}
		end := jsonStringEnd(content, i)
		marked, ok := datamarkJSONString(content[i:end], marker, opts)
		if !ok {
			return "", false
		}
		b.WriteString(marked)
		i = end
	}
	return b.String(), true
}

// jsonStringEnd returns the index just past the closing quote of the JSON
// string token that opens at s[start]. s must be valid JSON, so the token is
// terminated and every backslash starts a complete escape sequence.
func jsonStringEnd(s string, start int) int {
	for j := start + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++ // skip the escaped byte; \uXXXX's hex digits need no special case
		case '"':
			return j + 1
		}
	}
	return len(s)
}

// datamarkJSONString decodes one JSON string token, datamarks its text as
// prose and encodes it back to a JSON string token. HTML characters are left
// unescaped so the model reads them as written.
func datamarkJSONString(token string, marker rune, opts DatamarkOptions) (string, bool) {
	var text string
	if err := json.Unmarshal([]byte(token), &text); err != nil {
		return "", false
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(datamarkBody(text, marker, opts)); err != nil {
		return "", false
	}
	return strings.TrimSuffix(buf.String(), "\n"), true
}
