package cmdscan

import (
	"slices"
	"strings"
)

// This file finds where a command line may run a program, judged on the raw
// text rather than a parse (see InvokedSpecs): the text is split into simple
// commands at shell operators and into words at blanks, with quotes removed,
// and only a word in command position, or one that may be a program another
// program runs, counts; a word of a quoted string of several words counts
// only when the text may run that string as code.

// shellWord is one word of the raw text.
type shellWord struct {
	// path is the word with its quotes removed and its backslashes kept,
	// as a Windows path is written.
	path string
	// inString is set when the word is part of a quoted string that holds
	// several words ("qsdev teardown"): the shell passes that string to its
	// program as one argument, so it is text, which only a program that runs
	// text as code reads as a command (see runsQuotedText). A word quoted on
	// its own ("qsdev") is the argument it spells, which a program may run.
	inString bool
}

// simpleCommand is the words of one simple command of the raw text, from a
// command separator to the next.
type simpleCommand struct {
	words []shellWord
	// texts holds each word with its quotes and backslashes removed, as the
	// shell's escapes read it (q\sdev is qsdev).
	texts []string
	// programs holds the indexes of the words in command position: the
	// program word of the command and of each command line it runs.
	programs []int
	// runsString is set when a wrapper runs a string as a command line
	// (`env -S '...'`).
	runsString bool
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
// Quotes nest as the shell reads them (a " inside '...' does not open a
// string), and every quote character is dropped from the words, so a quote
// cannot hide a program name. A word that shares a quoted string with
// another word is marked inString.
func splitCommands(text string) []*simpleCommand {
	var cmds []*simpleCommand
	cur := &simpleCommand{}
	var b strings.Builder
	// quote is the quote character of the open string, or 0; multiWord is
	// set once the open string holds a break, so every word in it from then
	// on, and the one the break ends, is part of a string of several words.
	var quote byte
	multiWord, inString := false, false
	endWord := func() {
		if b.Len() > 0 {
			cur.words = append(cur.words, shellWord{path: b.String(), inString: inString})
			cur.texts = append(cur.texts, strings.ReplaceAll(b.String(), `\`, ""))
		}
		b.Reset()
		inString = false
	}
	for i := 0; i < len(text); i++ {
		r := text[i]
		if r == '\'' || r == '"' {
			switch quote {
			case 0:
				quote, multiWord = r, false
			case r:
				quote = 0
			}
			continue
		}
		isBreak := strings.IndexByte(commandBreaks+wordBreaks, r) >= 0
		multiWord = multiWord || quote != 0 && isBreak
		inString = inString || quote != 0 && multiWord
		switch {
		case strings.IndexByte(commandBreaks, r) >= 0:
			endWord()
			if len(cur.words) > 0 {
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
// program app, each with its invocations, judged in the text as written, in
// the text with every expansion made opaque (opaqueExpansions), which keeps a
// substitution's output from splitting into words of its own, and in the
// text with its quoted escapes printed (printedEscapes).
func commandsInvoking(command, app string) []*simpleCommand {
	var out []*simpleCommand
	for _, text := range []string{command, opaqueExpansions(command), printedEscapes(command)} {
		cmds := splitCommands(text)
		names := programNames(app, cmds)
		for _, c := range cmds {
			c.findPrograms(names)
		}
		textRuns := runsQuotedText(cmds)
		for _, c := range cmds {
			c.findMentions(names, textRuns)
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

// findPrograms records the words of c in command position, and the
// invocations of the program names names among them: the program word of c,
// or of a command line it runs (`sh -c`, eval, `script -c`), followed past
// reserved words, assignments and wrappers (see Program).
func (c *simpleCommand) findPrograms(names func(shellWord, string) bool) {
	for start := 0; start < len(c.texts); {
		for start < len(c.texts) && (commandPrefixWords[c.texts[start]] || isAssignment(c.texts[start])) {
			start++
		}
		run := Program(c.texts[start:])
		var p int
		switch {
		case run.CommandString:
			// The wrapper's command line starts with its option's argument.
			c.runsString = true
			p = start + run.StringRest - 1
			c.setWord(p, run.StringHead)
			start = p
			continue
		case run.Index < 0:
			start = len(c.texts)
			continue
		}
		p = start + run.Index
		c.programs = append(c.programs, p)
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
}

// findMentions records, as mentions, the arguments of c that name the
// program names names: a program may run its arguments (`find -exec`,
// `devenv shell`). A word in a quoted string of several words counts only
// when textRuns is set: the text may run such a string as code.
func (c *simpleCommand) findMentions(names func(shellWord, string) bool, textRuns bool) {
	for i, w := range c.words {
		if (!w.inString || textRuns) && names(w, c.texts[i]) {
			c.invs = append(c.invs, invocation{at: i, literalProgram: true, mention: true})
		}
	}
}

// runsQuotedText reports whether cmds may run a quoted string as code, so
// that "qsdev teardown" is a command rather than text: a word in command
// position, or an unquoted argument (which find -exec, xargs or devenv shell
// may run), names a program that runs code it is given (see RunsCode); a
// wrapper runs a string as a command line (`env -S '...'`); or a program run
// by its path is a file another word names, as one the text writes and then
// runs (`echo ... > x.sh && ./x.sh`).
func runsQuotedText(cmds []*simpleCommand) bool {
	scripts := map[string]bool{}
	for _, c := range cmds {
		if c.runsString {
			return true
		}
		for _, p := range c.programs {
			if RunsCode(c.texts[p]) {
				return true
			}
			if strings.ContainsAny(c.texts[p], `/\`) {
				scripts[wrapperName(c.texts[p])] = true
			}
		}
	}
	for _, c := range cmds {
		programs := make(map[int]bool, len(c.programs))
		for _, p := range c.programs {
			programs[p] = true
		}
		for i, w := range c.words {
			if programs[i] {
				continue
			}
			if !w.inString && RunsCode(c.texts[i]) || scripts[wrapperName(c.texts[i])] {
				return true
			}
		}
	}
	return false
}

// setWord makes the word at i read as text: the first word of a command line
// whose option holds it attached (`--command=qsdev`).
func (c *simpleCommand) setWord(i int, text string) {
	if c.texts[i] != text {
		c.words[i] = shellWord{path: text, inString: c.words[i].inString}
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
