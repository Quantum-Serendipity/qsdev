package cmdscan

import (
	"slices"
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
	// Piped reports that the command receives pipeline input: it follows a
	// `|` (`gci .npmrc | Remove-Item`), so a path it acts on may come from
	// the command before it rather than from its own words.
	Piped bool
	// GroupArg reports that an argument is a parenthesised group
	// (`Set-Content ('.npmrc') x`, `Remove-Item -Path (Resolve-Path f)`,
	// `$f.Delete()`), whose value the tokenizer does not attach to the
	// command: the group's commands are commands of their own. The `(` of a
	// statement keyword's condition (`if (...)`) or of an operator (`-and
	// (...)`) is not an argument.
	GroupArg bool
	// Nesting is how many `(` groups enclose the command (counted within the
	// expandable string a subexpression's commands come from). The words
	// after a group argument's `)` are the next command at the same Nesting
	// as the one holding the group (`Start-Process -FilePath (gcm x).Source
	// -ArgumentList y`).
	Nesting int
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

// psGroupKeywords are the statement keywords whose `(` opens a condition or
// a loop header, not an argument (see PSCommand.GroupArg).
var psGroupKeywords = map[string]bool{
	"if": true, "elseif": true, "while": true, "until": true, "for": true,
	"foreach": true, "switch": true, "param": true,
}

// psAliasVerbs are the cmdlets and aliases that define an alias.
var psAliasVerbs = map[string]bool{"set-alias": true, "sal": true, "new-alias": true, "nal": true}

// IsPSAliasVerb reports whether name (a PSCommand.Name) defines an alias.
func IsPSAliasVerb(name string) bool {
	return psAliasVerbs[name]
}

// MayRedefine reports whether c may give a command name another meaning: it
// defines an alias (IsPSAliasVerb), a function or filter (the function and
// filter keywords), or names the function: drive (`$function:gh = {...}`,
// `Set-Item function:gh`). An alias or function takes precedence over a
// cmdlet or program of the same name.
func (c PSCommand) MayRedefine() bool {
	if IsPSAliasVerb(c.Name) || c.Name == "function" || c.Name == "filter" {
		return true
	}
	return slices.ContainsFunc(append([]string{c.Name}, c.Args...), func(w string) bool {
		return strings.Contains(strings.ToLower(w), "function:")
	})
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
// subexpression or script block are commands of their own. The commands of
// a `$(...)` subexpression inside an expandable string ("...", @"..."@) are
// too, ahead of the command whose argument holds it, as they run first. An
// assignment (`$x = git ...`, `$x=git ...`) is a command of its own, and the
// word after its operator starts the next one. A command after a `|` is
// marked Piped, and one whose argument is a `(` group is marked GroupArg, as
// either may act on a path its own words do not show. It honours
// '...' (a doubled quote is a literal one), "..." (with backtick escapes),
// @'...'@ and @"..."@ here-strings, the typographic quotes PowerShell
// accepts as the same, backtick escapes and line continuations, and
// comments. The `&` and `.` call operators are dropped from the command word.
//
// Words end at every rune PowerShell reads as whitespace (IsPowerShellSpace),
// not only at an ASCII blank.
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

// maxPSSubexpressionDepth bounds how deep subexpressions nested in
// expandable strings are tokenized; a deeper one is an opaque command (see
// subexpression).
const maxPSSubexpressionDepth = 32

type psParser struct {
	text string
	i    int
	cmds []PSCommand
	cur  PSCommand
	// depth is how many expandable-string subexpressions p is nested in;
	// parens counts the `(` p has read and not yet closed. A nested p stops
	// (closed) at the `)` that closes its subexpression.
	depth, parens int
	closed        bool
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
	case c == '|' && strings.HasPrefix(p.text[p.i:], "||"):
		p.endCommand()
		p.i += 2
	case c == '\n' || c == ';' || c == '|' || c == '(' || c == ')' || c == '{' || c == '}':
		if c == '(' {
			p.endWord()
			p.cur.GroupArg = p.named && !psGroupKeywords[p.cur.Name] && !strings.HasPrefix(p.cur.Name, "-")
		}
		piped := c == '|' || (c == '\n' && p.cur.Piped && !p.named)
		p.endCommand()
		p.cur.Piped = piped
		p.i++
		switch {
		case c == '(':
			p.parens++
		case c == ')' && p.parens > 0:
			p.parens--
		case c == ')' && p.depth > 0:
			p.closed = true
		}
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
		case IsPowerShellSpace(r): // \v, \f, NEL, NBSP, Unicode separators
			p.endWord()
			p.i += size
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
			end := p.subexpression(p.text, p.i)
			p.buf = append(p.buf, p.text[p.i:end]...)
			p.i = end
			continue
		}
		p.buf = append(p.buf, p.text[p.i:p.i+size]...)
		p.i += size
	}
}

// subexpression tokenizes the `$(` subexpression of an expandable string at
// text[at:], appending its commands to p.cmds, and returns the index just
// past the `)` that closes it, or len(text) when none does. Nested deeper
// than maxPSSubexpressionDepth, it appends a command no read check accepts
// (Subexpression is set) instead, and returns the index past `$(`.
func (p *psParser) subexpression(text string, at int) int {
	if p.depth >= maxPSSubexpressionDepth {
		p.cmds = append(p.cmds, PSCommand{Name: "$(", Subexpression: true})
		return at + 2
	}
	sub := psParser{text: text, i: at + 2, depth: p.depth + 1}
	for sub.i < len(sub.text) && !sub.closed {
		sub.step()
	}
	sub.endCommand()
	p.cmds = append(p.cmds, sub.cmds...)
	return sub.i
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
	if j := strings.Index(body, "$("); !single && j >= 0 {
		p.cur.Subexpression = true
		for j >= 0 {
			end := p.subexpression(body, j)
			if j = strings.Index(body[end:], "$("); j >= 0 {
				j += end
			}
		}
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
		if lhs, rhs, ok := strings.Cut(word, "="); ok && strings.HasPrefix(word, "$") {
			p.cur.Name, p.named = PSCommandName(lhs+"="), true
			p.assigned(rhs)
			return
		}
		p.cur.Name, p.named = PSCommandName(word), true
	case !quoted && len(p.cur.Args) == 0 && strings.HasPrefix(p.cur.Name, "$") && isPSAssignOp(word):
		p.cur.Args = append(p.cur.Args, word)
		p.assigned("")
	default:
		p.cur.Args = append(p.cur.Args, word)
	}
}

// isPSAssignOp reports whether word is one of PowerShell's assignment
// operators.
func isPSAssignOp(word string) bool {
	switch word {
	case "=", "+=", "-=", "*=", "/=", "%=", "??=":
		return true
	}
	return false
}

// assigned closes the open command, an assignment to a variable, and starts
// the next command with rhs, the word after the operator, when it is not
// empty: the value of `$x = git commit` is the output of a command.
func (p *psParser) assigned(rhs string) {
	p.endCommand()
	if rhs != "" {
		p.cur.Name, p.named = PSCommandName(rhs), true
	}
}

// endCommand closes the open command, keeping it when it has a command word,
// a redirect or a subexpression.
func (p *psParser) endCommand() {
	p.endWord()
	if p.named || len(p.cur.WriteRedirects) > 0 || p.cur.Subexpression {
		p.cur.Nesting = p.parens
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

// IsPowerShellSpace reports whether PowerShell's tokenizer reads r as
// whitespace between words (CharExtensions.IsWhitespace): space, tab,
// vertical tab, form feed, NEL, NBSP and the Unicode space, line and
// paragraph separators. A newline ends a statement instead, so it is not one.
// package-guard's _PS_SPACES lists the same runes (TestPSSpacesMatchCmdscan).
func IsPowerShellSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\v', '\f', 0x85, 0xa0: // NEL, NBSP
		return true
	}
	return r >= utf8.RuneSelf && unicode.In(r, unicode.Zs, unicode.Zl, unicode.Zp)
}

// PowerShellSpaced returns s with every rune PowerShell reads as whitespace
// (IsPowerShellSpace) mapped to an ASCII space, so a POSIX reading of a
// PowerShell line (ParseWithVars) splits its words where PowerShell does:
// `claude<NBSP>--add-dir /` passes --add-dir as an argument of its own.
// A rune inside a quoted string stays in that string's word.
func PowerShellSpaced(s string) string {
	return strings.Map(func(r rune) rune {
		if IsPowerShellSpace(r) {
			return ' '
		}
		return r
	}, s)
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
// with `\` mapped to `/` and every rune PowerShell reads as whitespace
// (IsPowerShellSpace) mapped to an ASCII space, so a pattern matching the
// text sees `$env:X<NBSP>=` as PowerShell does; loose is norm without quotes,
// backticks and `+` (with the spaces around it), so a name split across a
// concatenation (`'.cl' + 'aude'`) or an escape (".cl`aude") reads whole
// again.
func PowerShellText(s string) (norm, loose string) {
	norm = strings.Map(func(r rune) rune {
		switch {
		case r == '\\':
			return '/'
		case IsPowerShellSpace(r):
			return ' '
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
// variable already is. A [Diagnostics.Process]::Start line needs nothing
// more: the program it names followed by its argument list is a mention the
// scan matches once the subcommand is written out. A Start-Process line is
// also rendered as the command line it starts (see startProcessLine), as its
// parameters may come in any order.
func PowerShellAsPOSIX(cmds []PSCommand) string {
	var b strings.Builder
	next := psNextAtNesting(cmds)
	// continued[i] is set when cmds[i] holds the words after a group
	// argument of an earlier command, not a command word of its own.
	continued := make([]bool, len(cmds))
	for i, c := range cmds {
		if c.GroupArg && next[i] >= 0 {
			continued[next[i]] = true
		}
	}
	for i, c := range cmds {
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
		if continued[i] || !psStartProcessVerbs[c.Name] {
			continue
		}
		if line, ok := startProcessLine(psWordsThroughGroups(cmds, next, i)); ok {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// psStartProcessVerbs are Start-Process and its aliases.
var psStartProcessVerbs = map[string]bool{"start-process": true, "saps": true, "start": true}

// psStartProcessSwitches are Start-Process's switch parameters, which take
// no value.
var psStartProcessSwitches = []string{"wait", "nonewwindow", "passthru", "usenewenvironment", "loaduserprofile"}

// psWordsThroughGroups returns the words of cmds[i] with each of its group
// arguments in place: the tokenizer reports a `(` group's commands as
// commands of their own, and the words after its `)` as the next command at
// the same Nesting. A group is the dynamic word "$_" (also when a `@` or `$`
// is written before it: `@(...)`, `$(...)`), followed by the words after it;
// a member access on the group (`(gcm x).Source`) is part of its value.
// next is psNextAtNesting(cmds).
func psWordsThroughGroups(cmds []PSCommand, next []int, i int) []string {
	words := slices.Clone(cmds[i].Args)
	for cmds[i].GroupArg {
		if n := len(words); n > 0 && (words[n-1] == "@" || words[n-1] == "$") {
			words = words[:n-1]
		}
		words = append(words, "$_")
		if i = next[i]; i < 0 {
			break
		}
		if c := cmds[i]; !strings.HasPrefix(c.Name, ".") {
			words = append(words, c.Name)
		}
		words = append(words, cmds[i].Args...)
	}
	return words
}

// psNextAtNesting returns, for each command of cmds, the index of the next
// command at its Nesting with only deeper ones between (the words after its
// group's `)`), or -1, in one pass.
func psNextAtNesting(cmds []PSCommand) []int {
	next := make([]int, len(cmds))
	var open []int // commands awaiting their next, by increasing Nesting
	for j, c := range cmds {
		next[j] = -1
		for len(open) > 0 && cmds[open[len(open)-1]].Nesting >= c.Nesting {
			if top := open[len(open)-1]; cmds[top].Nesting == c.Nesting {
				next[top] = j
			}
			open = open[:len(open)-1]
		}
		open = append(open, j)
	}
	return next
}

// startProcessLine returns the command line a Start-Process command with the
// words args (see psWordsThroughGroups) starts: its FilePath followed by the elements of its
// ArgumentList, each comma a word break, whatever order the parameters come
// in. FilePath is -FilePath or any prefix of it (-f), -PSPath or -Path, or the
// first positional argument; ArgumentList is -ArgumentList or any prefix of
// it (-a), -Args, or the second positional one. A value may be attached
// (`-FilePath:qsdev`), and an array written with blanks after its commas
// (`'a', 'b'`) is one value. A value the line computes (a group, an array
// literal or a splat: `@('a','b')`, `@args`, `(gcm x).Source`) is the dynamic
// word "$_". A switch (psStartProcessSwitches, or a prefix of one) takes no
// value; any other parameter takes the next word.
func startProcessLine(args []string) (string, bool) {
	var file, list string
	var positional []string
	// arrayRest appends to value the words that continue it as an array
	// after a trailing comma; a computed value is "$_".
	arrayRest := func(value string, i *int) string {
		for strings.HasSuffix(value, ",") && *i+1 < len(args) {
			*i++
			value += args[*i]
		}
		if strings.HasPrefix(value, "@") {
			return "$_"
		}
		return value
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, isParam := PSParamName(a)
		if !isParam {
			positional = append(positional, arrayRest(a, &i))
			continue
		}
		if slices.ContainsFunc(psStartProcessSwitches, func(s string) bool { return strings.HasPrefix(s, name) }) {
			continue
		}
		_, value, attached := strings.Cut(a, ":")
		if !attached && i+1 < len(args) {
			i++
			value = args[i]
		}
		value = arrayRest(value, &i)
		switch {
		case name == "pspath" || name == "path" || strings.HasPrefix("filepath", name):
			file = value
		case name == "args" || strings.HasPrefix("argumentlist", name):
			list = value
		}
	}
	if file == "" && len(positional) > 0 {
		file, positional = positional[0], positional[1:]
	}
	if list == "" && len(positional) > 0 {
		list = positional[0]
	}
	if file == "" {
		return "", false
	}
	words := append([]string{file}, strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == ' ' })...)
	return strings.Join(words, " "), true
}
