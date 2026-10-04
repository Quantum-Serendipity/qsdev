package cmdscan

import (
	"slices"
	"strings"
	"unicode"
)

// This file finds where a command line may run a program, judged on the raw
// text rather than a parse (see InvokedSpecs): the text is split into simple
// commands at shell operators and into words at blanks, with quotes removed.
// A word in command position counts as the program; a word anywhere else
// that names it (an argument, a word of a quoted string, an assignment's
// value) counts as a mention, which matches only with its subcommand written
// out.

// simpleCommand is the words of one simple command of the raw text, from a
// command separator to the next.
type simpleCommand struct {
	// paths holds each word with its quotes removed and its backslashes
	// kept, as a Windows path is written.
	paths []string
	// texts holds each word with its quotes and backslashes removed, as the
	// shell's escapes read it (q\sdev is qsdev).
	texts []string
	// invs are the places the command may run the program.
	invs []invocation
}

// invocation is one place a command may run the program: the word at index
// at of its command, and what is known about it.
type invocation struct {
	at int
	// literalProgram is set when the program word names the program itself;
	// otherwise it is a word whose value is unknown (a variable, a command
	// substitution, a word xargs replaces), and at least one path word must
	// be literal for a match, so that every `cp $a $b` is not one, unless
	// anchored is set.
	literalProgram bool
	// anchored is set when the program word is computed and the text assigns
	// the program's name to a variable (`Q=qsdev; $Q $S`), which the word may
	// expand: a computed path word after it then matches too.
	anchored bool
	// mention is set when the word is not in command position: an argument
	// of another program, which may run it (`find -exec`, `devenv shell`) or
	// only name it (`grep qsdev`), a word of a quoted string, which another
	// program may run as code (`trap "..." EXIT`, `git rebase -x "..."`), or
	// an assignment's value (`x="qsdev ..."; $x`). Only a fully literal
	// invocation then matches, so a glob or a variable after it does not
	// stand for a computed subcommand.
	mention bool
	// open is set when words the text does not show may follow the command:
	// xargs appends what it reads from its input.
	open bool
}

// commandBreaks end a simple command: list, pipeline and background
// operators, (sub)shell and brace groups, command substitutions and lines.
const commandBreaks = ";|&()`{}\n\r"

// wordBreaks end a word within a simple command: blanks and redirections.
const wordBreaks = " \t<>"

// splitCommands splits text into simple commands at commandBreaks, and each
// into words at wordBreaks, dropping quotes. A quoted separator splits too,
// so a script quoted for `sh -c` is seen command by command; data that holds
// one may then be taken for a command, which errs towards the human gate.
// Quotes nest as the shell reads them (a " inside '...' does not open a
// string), and every quote character is dropped from the words, so a quote
// cannot hide a program name.
func splitCommands(text string) []*simpleCommand {
	var cmds []*simpleCommand
	cur := &simpleCommand{}
	var b strings.Builder
	// quote is the quote character of the open string, or 0.
	var quote byte
	endWord := func() {
		if b.Len() > 0 {
			cur.paths = append(cur.paths, b.String())
			cur.texts = append(cur.texts, strings.ReplaceAll(b.String(), `\`, ""))
		}
		b.Reset()
	}
	for i := 0; i < len(text); i++ {
		r := text[i]
		if r == '\'' || r == '"' {
			switch quote {
			case 0:
				quote = r
			case r:
				quote = 0
			}
			continue
		}
		switch {
		case strings.IndexByte(commandBreaks, r) >= 0:
			endWord()
			if len(cur.paths) > 0 {
				cmds = append(cmds, cur)
				cur = &simpleCommand{}
			}
		case strings.IndexByte(wordBreaks, r) >= 0:
			endWord()
		default:
			b.WriteByte(r)
		}
	}
	endWord()
	if len(cur.paths) > 0 {
		cmds = append(cmds, cur)
	}
	return cmds
}

// commandPrefixWords are the reserved words that may start a command before
// the word naming its program.
var commandPrefixWords = map[string]bool{
	"!": true, "if": true, "then": true, "elif": true, "else": true,
	"do": true, "while": true, "until": true, "coproc": true,
}

// commandsInvoking returns the simple commands of command that may run the
// program app, each with its invocations, judged in the text as written, in
// the text with every expansion made opaque (opaqueExpansions), which keeps a
// substitution's output from splitting into words of its own, and in the
// text with its quoted escapes printed (printedEscapes).
func commandsInvoking(command, app string) []*simpleCommand {
	var out []*simpleCommand
	for _, text := range []string{command, opaqueExpansions(command), printedEscapes(command)} {
		cmds := splitCommands(text)
		refs := newProgramRefs(app, cmds)
		for _, c := range cmds {
			c.findPrograms(refs)
			c.findMentions(refs)
			if len(c.invs) > 0 {
				out = append(out, c)
			}
		}
	}
	return out
}

// printedEscapes returns command with the escapes \n, \r and \t inside quotes
// replaced by the newline, carriage return and tab printf, echo -e and $'...'
// make of them, so a script printed for a shell (`printf 'qsdev ...\n' | sh`)
// splits into the commands it runs. It is a reading of its own, beside the
// text as written, where a backslash may be a Windows path separator
// ("C:\tools\qsdev.exe").
func printedEscapes(command string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case c == '\'' || c == '"':
			switch quote {
			case 0:
				quote = c
			case c:
				quote = 0
			}
		case c == '\\' && quote != 0 && i+1 < len(command):
			if printed, ok := printedEscape[command[i+1]]; ok {
				b.WriteByte(printed)
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// printedEscape maps the escapes printedEscapes reads to what they print.
var printedEscape = map[byte]byte{'n': '\n', 'r': '\r', 't': '\t'}

// programRefs is how the simple commands of a text may refer to the program
// app.
type programRefs struct {
	app string
	// aliases are the names the text defines as aliases for app.
	aliases []string
	// assigned is set when the text assigns app's name to a variable
	// (`Q=qsdev`, `export Q=/usr/bin/qsdev`).
	assigned bool
}

// newProgramRefs finds how cmds refer to app.
func newProgramRefs(app string, cmds []*simpleCommand) programRefs {
	r := programRefs{app: app}
	for _, c := range cmds {
		r.aliases = append(r.aliases, appAliases(c.texts, app)...)
		r.assigned = r.assigned || slices.ContainsFunc(c.texts, func(w string) bool {
			_, value, _ := strings.Cut(w, "=")
			return isAssignment(w) && strings.EqualFold(ProgramName(value), app)
		})
	}
	return r
}

// names reports whether a word, as written (path) and as the shell reads it
// (text), names the program: by its program name with or without
// backslashes (case-folded: Windows and macOS resolve QSDEV to qsdev), or as
// an alias the text defines for it.
func (r programRefs) names(path, text string) bool {
	return strings.EqualFold(ProgramName(path), r.app) || strings.EqualFold(ProgramName(text), r.app) ||
		slices.Contains(r.aliases, text)
}

// findPrograms records the invocations of the program refs refers to in
// command position in c: the program word of c, or of a command line it runs
// (`sh -c`, eval, `script -c`), followed past reserved words, assignments and
// wrappers (see Program).
func (c *simpleCommand) findPrograms(refs programRefs) {
	for start := 0; start < len(c.texts); {
		for start < len(c.texts) && (commandPrefixWords[c.texts[start]] || isAssignment(c.texts[start])) {
			start++
		}
		run := Program(c.texts[start:])
		var p int
		switch {
		case run.CommandString:
			// The wrapper's command line starts with its option's argument.
			p = start + run.StringRest - 1
			c.setWord(p, run.StringHead)
			start = p
			continue
		case run.Index < 0:
			start = len(c.texts)
			continue
		}
		p = start + run.Index
		viaXargs := slices.ContainsFunc(c.texts[start:p], func(w string) bool { return wrapperName(w) == "xargs" })
		inv := invocation{at: p, open: viaXargs}
		switch name := ProgramName(c.texts[p]); {
		case refs.names(c.paths[p], c.texts[p]):
			inv.literalProgram = true
			c.invs = append(c.invs, inv)
		case IsDynamicWord(name):
			inv.anchored = refs.assigned
			c.invs = append(c.invs, inv)
		case viaXargs:
			// xargs may replace this word with the program (-I, -J).
			inv.open = false
			c.invs = append(c.invs, inv)
		}
		head, rest, ok := CommandLine(c.texts[p:])
		if !ok {
			break
		}
		start = p + rest - 1
		c.setWord(start, head)
	}
}

// findMentions records, as mentions, the words of c that name the program
// refs refers to, or that end in its name after a character other than a
// letter, digit or underscore (`x=qsdev`, `--grep=qsdev`, `!qsdev`): an
// argument a program may run (`find -exec`, `devenv shell`), a word of a
// quoted string a program may run as code (`trap`, `git rebase -x`, `ssh`,
// `echo ... | sh`), or an assignment's value a later word expands (`x="...";
// $x`). Which program takes the text, and whether it runs it, is not
// judged, so text that only names a command (`git commit -m "explain qsdev
// teardown"`) matches too. A word findPrograms found in command position is
// no mention.
func (c *simpleCommand) findMentions(refs programRefs) {
	programs := make(map[int]bool, len(c.invs))
	for _, inv := range c.invs {
		programs[inv.at] = true
	}
	for i, text := range c.texts {
		if !programs[i] && (refs.names(c.paths[i], text) || endsWithName(text, refs.app)) {
			c.invs = append(c.invs, invocation{at: i, literalProgram: true, mention: true})
		}
	}
}

// endsWithName reports whether word ends in name (case-folded) after nothing
// or a character other than a letter, digit or underscore: where the name
// starts a word of the text the shell or another program may run.
func endsWithName(word, name string) bool {
	n := len(word) - len(name)
	if n < 0 || !strings.EqualFold(word[n:], name) {
		return false
	}
	if n == 0 {
		return true
	}
	r := rune(word[n-1])
	return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// setWord makes the word at i read as text: the first word of a command line
// whose option holds it attached (`--command=qsdev`).
func (c *simpleCommand) setWord(i int, text string) {
	if c.texts[i] != text {
		c.paths[i] = text
		c.texts[i] = text
	}
}

// appAliases returns the names words define as aliases for app: each
// name=value word after an alias word whose value names app or is computed.
func appAliases(words []string, app string) []string {
	at := slices.Index(words, "alias")
	if at < 0 {
		return nil
	}
	var names []string
	for _, w := range words[at+1:] {
		name, value, ok := strings.Cut(w, "=")
		if ok && name != "" && (strings.EqualFold(ProgramName(value), app) || IsDynamicWord(value)) {
			names = append(names, name)
		}
	}
	return names
}
