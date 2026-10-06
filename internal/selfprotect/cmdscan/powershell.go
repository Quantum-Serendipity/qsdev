package cmdscan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Dialect is the command language a shell tool's command is written in.
type Dialect int

const (
	// POSIX is a POSIX shell (Bash, and Monitor, which runs in Bash's
	// environment), parsed by Parse.
	POSIX Dialect = iota
	// PowerShell is Windows PowerShell or pwsh, tokenized by ParsePowerShell.
	PowerShell
)

// ToolDialect returns the dialect toolName's command is written in. It is the
// one place the PowerShell tool name maps to its dialect; every other tool,
// shell or not, is POSIX.
func ToolDialect(tool string) Dialect {
	if tool == "PowerShell" {
		return PowerShell
	}
	return POSIX
}

// PSCommand is one command of a PowerShell line: its command word, its
// arguments with quotes and escapes removed, and the targets of its write
// redirects.
type PSCommand struct {
	// Name is the command word lower-cased, without a directory or a
	// trailing .exe, so `C:\Tools\Git.EXE` is "git". It is "" for a
	// redirect with no command.
	Name string
	Args []string
	// WriteRedirects are the files `>`, `>>`, `n>` and `*>` write. `$null`
	// and a stream merge (`2>&1`) write nothing and are left out.
	WriteRedirects []string
	// Subexpression reports that an argument is an expandable string
	// ("..." or @"..."@) holding a `$(...)` subexpression, which runs code
	// whatever the command is.
	Subexpression bool
}

// psReadVerbs are the cmdlets and aliases that only read the paths they are
// given.
var psReadVerbs = map[string]bool{
	"get-content": true, "gc": true, "cat": true, "type": true,
	"get-item": true, "gi": true, "get-childitem": true, "gci": true,
	"ls": true, "dir": true, "select-string": true, "sls": true,
	"test-path": true, "resolve-path": true,
}

// IsRead reports whether c only reads: a read verb, or a program whose
// arguments only read (IsSafeReadCommand: `git diff`, `git log`, `rg`), with
// no write redirect and no subexpression.
func (c PSCommand) IsRead() bool {
	if len(c.WriteRedirects) > 0 || c.Subexpression {
		return false
	}
	return psReadVerbs[c.Name] || IsSafeReadCommand(Command{Name: c.Name, Args: c.Args})
}

// PSParamName returns the name of a PowerShell parameter word, lower-cased
// and without any `:value`, and whether word is one: it starts with `-`,
// `--` or `/`, or with the en dash, em dash or horizontal bar that PowerShell
// and pwsh's own command line accept as a dash.
func PSParamName(word string) (string, bool) {
	var name string
	switch r, size := utf8.DecodeRuneInString(word); {
	case strings.HasPrefix(word, "--"):
		name = word[2:]
	case r == '-' || r == '/' || r == '\u2013' || r == '\u2014' || r == '\u2015':
		name = word[size:]
	default:
		return "", false
	}
	name, _, _ = strings.Cut(name, ":")
	return strings.ToLower(name), name != ""
}

// IsPSParamPrefix reports whether word is a parameter (see PSParamName) that
// abbreviates param, as PowerShell accepts any unambiguous prefix.
func IsPSParamPrefix(word, param string) bool {
	name, ok := PSParamName(word)
	return ok && strings.HasPrefix(param, name)
}

// ParsePowerShell splits a PowerShell line into its commands in one linear
// pass. Commands end at an unquoted newline, `;`, `|`, `&&`, `||`, a
// background `&`, and at `(`, `)`, `{` and `}`, so the commands of a
// subexpression or script block are commands of their own. It honours
// '...' (a doubled quote is a literal one), "..." (with backtick escapes),
// @'...'@ and @"..."@ here-strings, the typographic quotes PowerShell
// accepts as the same, backtick escapes and line continuations, and
// comments. The `&` and `.` call operators are dropped from the command word.
//
// It is a tokenizer, not a parser: it errs toward reporting more commands
// than PowerShell runs, never fewer, so a caller that treats any non-read
// command as a mutation fails closed.
func ParsePowerShell(text string) []PSCommand {
	p := psParser{text: text}
	for p.i < len(p.text) {
		p.step()
	}
	p.endCommand()
	return p.cmds
}

type psParser struct {
	text string
	i    int
	cmds []PSCommand
	cur  PSCommand
	// buf holds the open word; inWord is set once a word has started (an
	// empty quoted word counts), quoted once any part of it was quoted or
	// escaped.
	buf            []byte
	inWord, quoted bool
	// named is set once cur has its command word; redirect, when the next
	// word is a redirect target.
	named, redirect bool
}

// step consumes the next token-level construct at p.i.
func (p *psParser) step() {
	c := p.text[p.i]
	switch {
	case c == ' ' || c == '\t' || c == '\r':
		p.endWord()
		p.i++
	case c == '\n' || c == ';' || c == '|' || c == '(' || c == ')' || c == '{' || c == '}':
		p.endCommand()
		p.i++
	case c == '&':
		p.ampersand()
	case c == '>':
		p.redirection()
	case c == '`':
		p.backtick()
	case c == '#' && !p.inWord:
		p.skipTo("\n", false)
	case c == '<' && !p.inWord && strings.HasPrefix(p.text[p.i:], "<#"):
		p.skipTo("#>", true)
	case c == '@' && !p.inWord && p.hereString():
	default:
		r, size := p.runeAt(p.i)
		switch {
		case isPSSingleQuote(r):
			p.i += size
			p.singleQuoted()
		case isPSDoubleQuote(r):
			p.i += size
			p.doubleQuoted()
		default:
			p.buf = append(p.buf, p.text[p.i:p.i+size]...)
			p.inWord = true
			p.i += size
		}
	}
}

func (p *psParser) runeAt(i int) (rune, int) {
	if p.text[i] < utf8.RuneSelf {
		return rune(p.text[i]), 1
	}
	return utf8.DecodeRuneInString(p.text[i:])
}

// ampersand handles `&&`, the `&` call operator before a command word, and
// any other `&`, which PowerShell 7 reads as the background operator ending a
// pipeline.
func (p *psParser) ampersand() {
	p.endWord()
	switch {
	case strings.HasPrefix(p.text[p.i:], "&&"):
		p.i += 2
		p.endCommand()
	case !p.named && len(p.cur.WriteRedirects) == 0:
		p.i++ // call operator
	default:
		p.i++
		p.endCommand()
	}
}

// redirection handles `>`, `>>` and their stream prefixes (`2>`, `*>`): the
// next word is a write target, unless the redirect merges a stream (`2>&1`).
func (p *psParser) redirection() {
	if p.inWord && !p.quoted && len(p.buf) == 1 && (p.buf[0] == '*' || p.buf[0] >= '1' && p.buf[0] <= '6') {
		p.buf, p.inWord = p.buf[:0], false // stream prefix
	} else {
		p.endWord()
	}
	p.i++
	if p.i < len(p.text) && p.text[p.i] == '>' {
		p.i++
	}
	if p.i < len(p.text) && p.text[p.i] == '&' {
		for p.i++; p.i < len(p.text) && p.text[p.i] >= '0' && p.text[p.i] <= '9'; p.i++ {
		}
		return
	}
	p.redirect = true
}

// backtick handles a backtick outside quotes: a line continuation before a
// newline, else an escape that takes the next character literally.
func (p *psParser) backtick() {
	p.i++
	if strings.HasPrefix(p.text[p.i:], "\r\n") {
		p.i += 2
		p.endWord()
		return
	}
	if p.i >= len(p.text) {
		return
	}
	if p.text[p.i] == '\n' {
		p.i++
		p.endWord()
		return
	}
	_, size := p.runeAt(p.i)
	p.buf = append(p.buf, p.text[p.i:p.i+size]...)
	p.i += size
	p.inWord, p.quoted = true, true
}

// skipTo moves past the next end (past it too when consume is set), or to the
// end of the text when there is none.
func (p *psParser) skipTo(end string, consume bool) {
	j := strings.Index(p.text[p.i:], end)
	switch {
	case j < 0:
		p.i = len(p.text)
	case consume:
		p.i += j + len(end)
	default:
		p.i += j
	}
}

// singleQuoted reads a '...' string after its opening quote; a doubled quote
// is a literal one.
func (p *psParser) singleQuoted() {
	p.inWord, p.quoted = true, true
	for p.i < len(p.text) {
		r, size := p.runeAt(p.i)
		if isPSSingleQuote(r) {
			if p.i+size < len(p.text) {
				if next, nsize := p.runeAt(p.i + size); isPSSingleQuote(next) {
					p.buf = append(p.buf, p.text[p.i:p.i+size]...)
					p.i += size + nsize
					continue
				}
			}
			p.i += size
			return
		}
		p.buf = append(p.buf, p.text[p.i:p.i+size]...)
		p.i += size
	}
}

// doubleQuoted reads a "..." string after its opening quote: a backtick
// escapes the next character, a doubled quote is a literal one, and `$(`
// marks a subexpression.
func (p *psParser) doubleQuoted() {
	p.inWord, p.quoted = true, true
	for p.i < len(p.text) {
		r, size := p.runeAt(p.i)
		switch {
		case r == '`' && p.i+1 < len(p.text):
			_, esize := p.runeAt(p.i + 1)
			p.buf = append(p.buf, p.text[p.i+1:p.i+1+esize]...)
			p.i += 1 + esize
			continue
		case isPSDoubleQuote(r):
			if p.i+size < len(p.text) {
				if next, nsize := p.runeAt(p.i + size); isPSDoubleQuote(next) {
					p.buf = append(p.buf, p.text[p.i:p.i+size]...)
					p.i += size + nsize
					continue
				}
			}
			p.i += size
			return
		case r == '$' && strings.HasPrefix(p.text[p.i+1:], "("):
			p.cur.Subexpression = true
		}
		p.buf = append(p.buf, p.text[p.i:p.i+size]...)
		p.i += size
	}
}

// hereString reads an @'...'@ or @"..."@ here-string at p.i, reporting
// whether there is one: the opening quote must end its line, and the closing
// one starts a line.
func (p *psParser) hereString() bool {
	if p.i+1 >= len(p.text) {
		return false
	}
	q, size := p.runeAt(p.i + 1)
	single := isPSSingleQuote(q)
	if !single && !isPSDoubleQuote(q) {
		return false
	}
	// Only blanks may follow the opening quote on its line. Scanning stops
	// at the first other byte, so a run of openers stays linear.
	eol := p.i + 1 + size
	for eol < len(p.text) && (p.text[eol] == ' ' || p.text[eol] == '\t' || p.text[eol] == '\r') {
		eol++
	}
	if eol == len(p.text) || p.text[eol] != '\n' {
		return false
	}
	bodyStart := eol + 1
	body, end := p.text[bodyStart:], len(p.text)
	for line := bodyStart; line < len(p.text); {
		if r, rsize := p.runeAt(line); single == isPSSingleQuote(r) && (single || isPSDoubleQuote(r)) &&
			strings.HasPrefix(p.text[line+rsize:], "@") {
			body, end = p.text[bodyStart:max(bodyStart, line-1)], line+rsize+1
			break
		}
		j := strings.IndexByte(p.text[line:], '\n')
		if j < 0 {
			break
		}
		line += j + 1
	}
	body = strings.TrimSuffix(body, "\r")
	if !single && strings.Contains(body, "$(") {
		p.cur.Subexpression = true
	}
	p.buf = append(p.buf, body...)
	p.inWord, p.quoted = true, true
	p.i = end
	return true
}

// endWord closes the open word into the command word, an argument or a
// redirect target.
func (p *psParser) endWord() {
	if !p.inWord {
		return
	}
	word, quoted := string(p.buf), p.quoted
	p.buf, p.inWord, p.quoted = p.buf[:0], false, false
	switch {
	case p.redirect:
		p.redirect = false
		if !strings.EqualFold(word, "$null") {
			p.cur.WriteRedirects = append(p.cur.WriteRedirects, word)
		}
	case !p.named:
		if !quoted && word == "." {
			return // dot-source operator
		}
		p.cur.Name, p.named = PSCommandName(word), true
	default:
		p.cur.Args = append(p.cur.Args, word)
	}
}

// endCommand closes the open command, keeping it when it has a command word,
// a redirect or a subexpression.
func (p *psParser) endCommand() {
	p.endWord()
	if p.named || len(p.cur.WriteRedirects) > 0 || p.cur.Subexpression {
		p.cmds = append(p.cmds, p.cur)
	}
	p.cur, p.named, p.redirect = PSCommand{}, false, false
}

// PSCommandName returns a PowerShell command word as PSCommand.Name holds
// it: lower-cased, without a directory or a trailing .exe.
func PSCommandName(word string) string {
	if i := strings.LastIndexAny(word, `/\`); i >= 0 {
		word = word[i+1:]
	}
	word = strings.ToLower(word)
	return strings.TrimSuffix(word, ".exe")
}

// isPSSingleQuote reports whether r opens or closes a single-quoted
// PowerShell string: the ASCII quote or a typographic single quote.
func isPSSingleQuote(r rune) bool {
	return r == '\'' || r == '\u2018' || r == '\u2019' || r == '\u201a' || r == '\u201b'
}

// isPSDoubleQuote reports whether r opens or closes a double-quoted
// PowerShell string: the ASCII quote or a typographic double quote.
func isPSDoubleQuote(r rune) bool {
	return r == '"' || r == '\u201c' || r == '\u201d' || r == '\u201e'
}

// PowerShellText returns s in the forms protected paths are searched for in a
// PowerShell line: norm is lower-cased (PowerShell paths are case-insensitive)
// with `\` mapped to `/`; loose is norm without quotes, backticks and `+`
// (with the spaces around it), so a name split across a concatenation
// (`'.cl' + 'aude'`) or an escape (".cl`aude") reads whole again.
func PowerShellText(s string) (norm, loose string) {
	norm = strings.Map(func(r rune) rune {
		if r == '\\' {
			return '/'
		}
		return unicode.ToLower(r)
	}, s)
	var b strings.Builder
	b.Grow(len(norm))
	for i := 0; i < len(norm); {
		r, size := rune(norm[i]), 1
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeRuneInString(norm[i:])
		}
		switch {
		case r == ' ' || r == '\t':
			j := i
			for j < len(norm) && (norm[j] == ' ' || norm[j] == '\t') {
				j++
			}
			if j == len(norm) || norm[j] != '+' {
				b.WriteString(norm[i:j])
			}
			i = j
			continue
		case r == '+':
			for i+size < len(norm) && (norm[i+size] == ' ' || norm[i+size] == '\t') {
				size++
			}
		case r == '`' || isPSSingleQuote(r) || isPSDoubleQuote(r):
		default:
			b.WriteString(norm[i : i+size])
		}
		i += size
	}
	return norm, b.String()
}

// PowerShellAsPOSIX renders PowerShell commands as a POSIX command line, one
// command per line, in the words PowerShell passes a native program, for the
// program-invocation scan (InvokedSpecs). PowerShell passes each element of
// an array argument (`'a','b'`, `a,b`) as an argument of its own, so a comma
// separates words; a quoted comma is split too, which only adds words, so
// the scan errs towards a match. An array literal or splat (`@(...)`,
// `@args`), whose elements the line computes, is the dynamic word "$_", as a
// variable already is. A Start-Process or [Diagnostics.Process]::Start line
// needs nothing more: the program it names followed by its argument list is
// a mention the scan matches once the subcommand is written out.
func PowerShellAsPOSIX(cmds []PSCommand) string {
	var b strings.Builder
	for _, c := range cmds {
		b.WriteString(strings.ReplaceAll(c.Name, ",", " "))
		for _, a := range c.Args {
			b.WriteByte(' ')
			if strings.HasPrefix(a, "@") {
				b.WriteString("$_")
				continue
			}
			b.WriteString(strings.ReplaceAll(a, ",", " "))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
