package cmdscan

import (
	"slices"
	"strings"
)

// This file finds where a command line may run a program, judged on the raw
// text rather than a parse (see InvokedSpecs): the text is split into simple
// commands at shell operators and into words at blanks, with quotes removed,
// and only a word in command position, or one that may be a program another
// program runs, counts.

// shellWord is one word of the raw text.
type shellWord struct {
	// path is the word with its quotes removed and its backslashes kept,
	// as a Windows path is written.
	path string
	// quoted is set when the word holds a quote: as an argument it is data
	// passed to its program.
	quoted bool
}

// simpleCommand is the words of one simple command of the raw text, from a
// command separator to the next.
type simpleCommand struct {
	words []shellWord
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
	// be literal for a match, so that every `cp $a $b` is not one.
	literalProgram bool
	// mention is set when the word is an argument of another program that
	// may run it (`find -exec`, `devenv shell`) or may only name it (`grep
	// qsdev`): only a fully literal invocation then matches, so a glob or a
	// variable after it does not stand for a computed subcommand.
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
func splitCommands(text string) []*simpleCommand {
	var cmds []*simpleCommand
	cur := &simpleCommand{}
	var b strings.Builder
	quoted := false
	endWord := func() {
		if b.Len() > 0 {
			cur.words = append(cur.words, shellWord{path: b.String(), quoted: quoted})
			cur.texts = append(cur.texts, strings.ReplaceAll(b.String(), `\`, ""))
		}
		b.Reset()
		quoted = false
	}
	for _, r := range text {
		switch {
		case r == '\'' || r == '"':
			quoted = true
		case strings.ContainsRune(commandBreaks, r):
			endWord()
			if len(cur.words) > 0 {
				cmds = append(cmds, cur)
				cur = &simpleCommand{}
			}
		case strings.ContainsRune(wordBreaks, r):
			endWord()
		default:
			b.WriteRune(r)
		}
	}
	endWord()
	if len(cur.words) > 0 {
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
// program app, each with its invocations, judged in the text as written and
// in the text with every expansion made opaque (opaqueExpansions), which
// keeps a substitution's output from splitting into words of its own.
func commandsInvoking(command, app string) []*simpleCommand {
	var out []*simpleCommand
	for _, text := range []string{command, opaqueExpansions(command)} {
		cmds := splitCommands(text)
		names := programNames(app, cmds)
		for _, c := range cmds {
			c.findInvocations(names)
			if len(c.invs) > 0 {
				out = append(out, c)
			}
		}
	}
	return out
}

// programNames reports whether a word names the program app: by its program
// name with or without backslashes (case-folded: Windows and macOS resolve
// QSDEV to qsdev), or as an alias cmds define for it.
func programNames(app string, cmds []*simpleCommand) func(w shellWord, text string) bool {
	var aliases []string
	for _, c := range cmds {
		aliases = append(aliases, appAliases(c.texts, app)...)
	}
	return func(w shellWord, text string) bool {
		return strings.EqualFold(ProgramName(w.path), app) || strings.EqualFold(ProgramName(text), app) ||
			slices.Contains(aliases, text)
	}
}

// findInvocations records the places c may run the program names names: the
// program word of c, or of a command line it runs (`sh -c`, eval, `script
// -c`), followed past reserved words, assignments and wrappers (see
// Program); and every unquoted argument naming it, as a mention.
func (c *simpleCommand) findInvocations(names func(shellWord, string) bool) {
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
		case names(c.words[p], c.texts[p]):
			inv.literalProgram = true
			c.invs = append(c.invs, inv)
		case IsDynamicWord(name):
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
	for i, w := range c.words {
		if !w.quoted && names(w, c.texts[i]) {
			c.invs = append(c.invs, invocation{at: i, literalProgram: true, mention: true})
		}
	}
}

// setWord makes the word at i read as text: the first word of a command line
// whose option holds it attached (`--command=qsdev`).
func (c *simpleCommand) setWord(i int, text string) {
	if c.texts[i] != text {
		c.words[i] = shellWord{path: text, quoted: c.words[i].quoted}
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
