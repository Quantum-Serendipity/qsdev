package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxSecretPatternLen caps one hooks.credential_scan.extra_patterns entry,
// in bytes.
const maxSecretPatternLen = 1024

const (
	// caseFoldPrefix is the one inline flag group allowed, and only at the
	// start of an entry.
	caseFoldPrefix = "(?i)"
	// secretGroup opens the named group whose match scan-secrets.py judges
	// as the credential value; an entry may hold it at most once.
	secretGroup = "?P<secret>"
	// asciiPunct are the characters a backslash may escape to a literal in
	// both RE2 and Python re.
	asciiPunct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	// valueEscapes are the escape letters both engines read alike that stand
	// for characters: Perl classes and \t \n \r.
	valueEscapes = "dDwWsStnr"
)

// ErrSecretPattern is wrapped by every CheckSecretPattern failure.
var ErrSecretPattern = errors.New("invalid secret pattern")

// CheckSecretPattern validates one hooks.credential_scan.extra_patterns
// entry. qsdev hands the entries to scan-secrets.py, which compiles them with
// Python's re, one per line of CREDENTIAL_SCAN_EXTRA_PATTERNS. The check is
// an allowlist over the syntax RE2 and Python share, so an accepted entry
// means the same in both engines; anything else is rejected. See
// docs/configuration-reference.md for the accepted subset.
func CheckSecretPattern(p string) error {
	switch {
	case p == "":
		return fmt.Errorf("%w: empty", ErrSecretPattern)
	case len(p) > maxSecretPatternLen:
		return fmt.Errorf("%w: longer than %d bytes", ErrSecretPattern, maxSecretPatternLen)
	case !utf8.ValidString(p):
		return fmt.Errorf("%w: not valid UTF-8", ErrSecretPattern)
	case strings.Trim(p, " \t\r") != p:
		return fmt.Errorf("%w: leading or trailing space, tab or carriage return", ErrSecretPattern)
	}
	l := patternLexer{src: []rune(p)}
	if strings.HasPrefix(p, caseFoldPrefix) {
		l.pos = len(caseFoldPrefix)
	}
	if err := l.lex(); err != nil {
		return fmt.Errorf("%w: %w", ErrSecretPattern, err)
	}
	if _, err := regexp.Compile(p); err != nil {
		return fmt.Errorf("%w: %w", ErrSecretPattern, err)
	}
	return nil
}

// patternLexer walks a secret pattern and rejects any syntax outside the
// allowlist CheckSecretPattern documents.
type patternLexer struct {
	src       []rune
	pos       int
	depth     int  // open groups
	named     bool // the secret group has been seen
	canRepeat bool // the previous token is an atom a quantifier may follow
}

func (l *patternLexer) lex() error {
	for l.pos < len(l.src) {
		at, r := l.pos, l.src[l.pos]
		l.pos++
		var err error
		switch r {
		case '\\':
			err = l.escape(false)
		case '.':
			l.canRepeat = true
		case '^', '$', '|':
			l.canRepeat = false
		case '*', '+', '?', '{':
			err = l.quantifier(r, at)
		case '(':
			err = l.group(at)
		case ')':
			if l.depth == 0 {
				return fmt.Errorf("unbalanced ')' at offset %d", at)
			}
			l.depth--
			l.canRepeat = true
		case '[':
			err = l.class(at)
		case ']', '}':
			err = fmt.Errorf("unescaped %q at offset %d", r, at)
		default:
			err = checkPatternLiteral(r, at)
			l.canRepeat = true
		}
		if err != nil {
			return err
		}
	}
	if l.depth != 0 {
		return errors.New("unclosed group")
	}
	return nil
}

// escape consumes the character after a backslash: a value escape, an
// escaped ASCII punctuation character, or outside a class \b or \B.
func (l *patternLexer) escape(inClass bool) error {
	at := l.pos - 1
	if l.pos == len(l.src) {
		return fmt.Errorf("trailing backslash at offset %d", at)
	}
	r := l.src[l.pos]
	l.pos++
	switch {
	case strings.ContainsRune(valueEscapes, r) || strings.ContainsRune(asciiPunct, r):
		l.canRepeat = true
	case (r == 'b' || r == 'B') && !inClass:
		l.canRepeat = false
	default:
		return fmt.Errorf("unsupported escape %q at offset %d", `\`+string(r), at)
	}
	return nil
}

// quantifier consumes the rest of a quantifier whose first rune op was just
// read, and its optional lazy '?'.
func (l *patternLexer) quantifier(op rune, at int) error {
	if !l.canRepeat {
		return fmt.Errorf("%q at offset %d has nothing to repeat", op, at)
	}
	if op == '{' && !l.counted() {
		return fmt.Errorf(`'{' at offset %d does not start {n}, {n,} or {n,m}; escape it as \{`, at)
	}
	if l.peek('?') {
		l.pos++
	}
	l.canRepeat = false
	return nil
}

// counted consumes "n}", "n,}" or "n,m}" after a '{'.
func (l *patternLexer) counted() bool {
	if !l.digits() {
		return false
	}
	if l.peek(',') {
		l.pos++
		l.digits()
	}
	if !l.peek('}') {
		return false
	}
	l.pos++
	return true
}

func (l *patternLexer) digits() bool {
	start := l.pos
	for l.pos < len(l.src) && '0' <= l.src[l.pos] && l.src[l.pos] <= '9' {
		l.pos++
	}
	return l.pos > start
}

func (l *patternLexer) peek(r rune) bool {
	return l.pos < len(l.src) && l.src[l.pos] == r
}

// group consumes the opening of a group: "(", "(?:" or the one "(?P<secret>".
func (l *patternLexer) group(at int) error {
	l.depth++
	l.canRepeat = false
	if !l.peek('?') {
		return nil
	}
	rest := string(l.src[l.pos:])
	switch {
	case strings.HasPrefix(rest, "?:"):
		l.pos += len("?:")
	case strings.HasPrefix(rest, secretGroup) && !l.named:
		l.named = true
		l.pos += len(secretGroup)
	default:
		return fmt.Errorf("unsupported group at offset %d: only (, (?: and one (?P<secret> are allowed, and (?i) only at the start", at)
	}
	return nil
}

// class consumes a character class after its '[': literals, value escapes
// and ranges between two plain literals; '-' unescaped only first or last.
func (l *patternLexer) class(at int) error {
	if l.peek('^') {
		l.pos++
	}
	first := l.pos
	for l.pos < len(l.src) {
		i, r := l.pos, l.src[l.pos]
		l.pos++
		switch r {
		case ']':
			if i == first {
				return fmt.Errorf("empty class at offset %d; escape a literal ']' as \\]", at)
			}
			l.canRepeat = true
			return nil
		case '\\':
			if err := l.escape(true); err != nil {
				return err
			}
		case '-':
			if i != first && !l.peek(']') {
				return fmt.Errorf("unescaped '-' at offset %d; put it first or last in the class, or escape it", i)
			}
		case '[':
			return fmt.Errorf("unescaped '[' in class at offset %d", i)
		default:
			if err := l.rangeFrom(r, i); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("unclosed '[' at offset %d", at)
}

// rangeFrom checks the class literal lo at offset at and, when a range
// "lo-hi" follows, consumes it; hi must be a plain literal.
func (l *patternLexer) rangeFrom(lo rune, at int) error {
	if err := checkPatternLiteral(lo, at); err != nil {
		return err
	}
	if !l.peek('-') || l.pos+1 >= len(l.src) || l.src[l.pos+1] == ']' {
		return nil
	}
	hi := l.src[l.pos+1]
	if strings.ContainsRune(`\[]-`, hi) {
		return fmt.Errorf("range at offset %d must end in a plain character", at)
	}
	l.pos += 2
	return checkPatternLiteral(hi, l.pos-1)
}

// checkPatternLiteral rejects control, format and line or paragraph
// separator characters, which no pattern needs as literals.
func checkPatternLiteral(r rune, at int) error {
	if !unicode.IsGraphic(r) {
		return fmt.Errorf("control or invisible character %U at offset %d", r, at)
	}
	return nil
}
