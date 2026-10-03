package cmdscan

import (
	"path"
	"slices"
	"strconv"
	"strings"
)

// CommandSpec describes the invocations of one CLI subcommand that a rule
// matches, as words after the program name: the subcommand path, then flags
// and positional arguments in any order.
type CommandSpec struct {
	// Path holds, per subcommand level below the program, the words that
	// name it (its name and aliases).
	Path [][]string
	// ReadOnly holds the spellings of the flag (e.g. "--dry-run") that make
	// an invocation read-only; such an invocation never matches.
	ReadOnly []string
	// Flags match an invocation that sets any of them to its Value.
	Flags []FlagCond
	// Args returns the positional arguments that match an invocation; it is
	// called only once the path matched. Nil matches none.
	Args func() []string
}

// FlagCond is a boolean flag set to Value, written as any of Spellings
// ("--force", "-f"). A bare spelling sets it to true; "--force=false" sets
// it to false.
type FlagCond struct {
	Spellings []string
	Value     bool
}

// Always reports whether every invocation of the path matches: no flag or
// argument condition narrows it.
func (s CommandSpec) Always() bool {
	return len(s.Flags) == 0 && s.Args == nil
}

// Matches reports whether argv, the words after the program name, invoke the
// spec's subcommand in a matching way. Flags before and between path words
// are skipped. A flag value that does not parse as a boolean counts as
// matching, so a spelling the CLI would reject never hides a match. A word
// whose value the shell computes (IsDynamicWord) may stand for the rest of
// the path and any flag or argument, so it matches.
func (s CommandSpec) Matches(argv []string) bool {
	return s.invokedBy(invocation{args: argv, literalProgram: true})
}

// invocation is one place a command may run the program: the words after
// the program word, and what is known about them.
type invocation struct {
	args []string
	// literalProgram is set when the program word names the program itself;
	// otherwise it is a word whose value is unknown (a variable, a command
	// substitution, a word xargs replaces), and at least one path word must
	// be literal for a match, so that every `cp $a $b` is not one.
	literalProgram bool
	// open is set when words the text does not show may follow args: xargs
	// appends what it reads from its input.
	open bool
}

// invokedBy reports whether inv invokes the spec's subcommand in a matching
// way (see Matches). A dynamic word, or the end of an open invocation, ends
// the scan as a match once something literal anchors it.
func (s CommandSpec) invokedBy(inv invocation) bool {
	anchored := inv.literalProgram
	level := 0
	for i, w := range inv.args {
		if level == len(s.Path) {
			return s.restMatches(inv.args[i:], inv.open)
		}
		switch {
		case IsDynamicWord(w):
			// Word splitting may turn it into the rest of the path.
			return anchored
		case strings.HasPrefix(w, "-"):
			continue
		case !slices.Contains(s.Path[level], w):
			return false
		}
		level++
		anchored = true
	}
	if level == len(s.Path) {
		return s.restMatches(nil, inv.open)
	}
	return inv.open && anchored
}

// restMatches reports whether rest, the words after the spec's path, make
// the invocation match: a literal read-only flag never does; otherwise an
// unconditional spec, an open invocation and a dynamic word (which may be
// the flag or argument a condition names) do, and else a flag or argument
// the spec's conditions name.
func (s CommandSpec) restMatches(rest []string, open bool) bool {
	if slices.ContainsFunc(rest, func(w string) bool { return flagValue(w, s.ReadOnly, true) }) {
		return false
	}
	if s.Always() || open || slices.ContainsFunc(rest, IsDynamicWord) {
		return true
	}
	for _, w := range rest {
		for _, f := range s.Flags {
			if flagValue(w, f.Spellings, f.Value) {
				return true
			}
		}
	}
	if s.Args == nil {
		return false
	}
	args := s.Args()
	return slices.ContainsFunc(rest, func(w string) bool {
		return !strings.HasPrefix(w, "-") && slices.Contains(args, w)
	})
}

// flagValue reports whether word sets the flag spelled as one of spellings to
// want.
func flagValue(word string, spellings []string, want bool) bool {
	name, value, hasValue := strings.Cut(word, "=")
	if !slices.Contains(spellings, name) {
		return false
	}
	if !hasValue {
		return want
	}
	got, err := strconv.ParseBool(value)
	return err != nil || got == want
}

// InvokedSpecs returns the specs that command invokes the program app with,
// judged on the raw text: quotes are dropped and the text is split at
// whitespace and shell operators, so quoting, wrappers (env, sh -c, eval) and
// compound commands do not hide an invocation. A backslash is read both as a
// shell escape (q\sdev) and as a Windows path separator. An invocation
// mentioned in an argument (`echo "qsdev teardown"`) matches too; the
// caller's rule errs towards the human gate.
//
// Words the shell computes fail closed rather than being resolved: a program
// word built from an expansion (`$Q defaults pin`, `$(printf qs)dev ...`,
// `{qsdev,} ...`) counts as the program when a literal path word follows it,
// a subcommand word built from one (`qsdev defaults $P`, `qsdev defaults
// $(echo pin)`) matches whatever it stands for, and an invocation under xargs
// may take its remaining words from its input (`echo pin | xargs qsdev
// defaults`, `xargs -I X X defaults pin`). Only a command whose program and
// subcommand words are all computed (`$A $B`) is not matched.
func InvokedSpecs(command, app string, specs []CommandSpec) []CommandSpec {
	var hit []CommandSpec
	for _, inv := range invocations(command, app) {
		for _, s := range specs {
			if s.invokedBy(inv) {
				hit = append(hit, s)
			}
		}
	}
	return hit
}

// InvokesProgram reports whether command may run the program app, judged on
// the raw text the way InvokedSpecs judges it, counting only a word that
// names app literally.
func InvokesProgram(command, app string) bool {
	return slices.ContainsFunc(invocations(command, app), func(inv invocation) bool { return inv.literalProgram })
}

// IsDynamicWord reports whether the shell computes word rather than taking it
// as written: it holds a parameter expansion or a command substitution ($,
// including the placeholder opaqueExpansions leaves) or a glob character.
func IsDynamicWord(word string) bool {
	return strings.ContainsAny(word, "$*?[")
}

// invocations returns each place command may run the program app (see
// InvokedSpecs), in the text as written and in the text with every
// expansion made opaque (opaqueExpansions), which keeps a substitution's
// output from splitting into words of its own.
func invocations(command, app string) []invocation {
	var out []invocation
	for _, text := range []string{command, opaqueExpansions(command)} {
		out = append(out, textInvocations(text, app)...)
	}
	return out
}

// textInvocations returns the invocations of app in text, split into words
// as InvokedSpecs describes.
func textInvocations(text, app string) []invocation {
	raw := strings.FieldsFunc(strings.NewReplacer(`'`, "", `"`, "").Replace(text), isWordBreak)
	words := make([]string, len(raw))
	for i, w := range raw {
		words[i] = strings.ReplaceAll(w, `\`, "")
	}
	aliases := appAliases(words, app)
	xargsAt := slices.IndexFunc(words, func(w string) bool { return ProgramName(w) == "xargs" })
	var out []invocation
	for i, w := range raw {
		inv := invocation{args: words[i+1:], open: xargsAt >= 0 && xargsAt < i}
		switch {
		// Case-folded: Windows and macOS resolve QSDEV to qsdev.
		case strings.EqualFold(ProgramName(w), app) || strings.EqualFold(ProgramName(words[i]), app) ||
			slices.Contains(aliases, words[i]):
			inv.literalProgram = true
		case IsDynamicWord(ProgramName(words[i])):
		case inv.open && !strings.HasPrefix(words[i], "-"):
			// xargs may replace this word with the program (-I, -J).
			inv.open = false
		default:
			continue
		}
		out = append(out, inv)
	}
	return out
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

// opaqueExpansions returns command with every command substitution ($(...)
// and backquotes), braced parameter expansion (${...}), brace expansion
// ({a,b}, {1..3}) and find/xargs placeholder ({}) replaced by the dynamic
// word "$_", so a program or subcommand word the shell builds from one is
// seen as one dynamic word. An unterminated substitution runs to the end of
// the text. It runs in time linear in the text, whatever its nesting.
func opaqueExpansions(command string) string {
	sc := newBracketScan(command)
	var b strings.Builder
	for i := 0; i < len(command); {
		if end := sc.expansionEnd(i); end > i {
			b.WriteString("$_")
			i = end
			continue
		}
		b.WriteByte(command[i])
		i++
	}
	return b.String()
}

// bracketScan indexes a text for opaqueExpansions: where each ( and { closes,
// and running counts of the characters that tell a brace expansion from a
// brace group.
type bracketScan struct {
	s string
	// closer[i] is the index of the bracket closing the ( or { at i, or -1.
	closer []int
	// breaks, commas and dots count, before each index, the word breaks
	// that rule out a brace expansion, the commas, and the ".." pairs.
	breaks, commas, dots []int
}

// newBracketScan indexes s in one pass.
func newBracketScan(s string) bracketScan {
	sc := bracketScan{
		s:      s,
		closer: make([]int, len(s)),
		breaks: make([]int, len(s)+1),
		commas: make([]int, len(s)+1),
		dots:   make([]int, len(s)+1),
	}
	var parens, braces []int
	for i := range len(s) {
		sc.closer[i] = -1
		sc.breaks[i+1], sc.commas[i+1], sc.dots[i+1] = sc.breaks[i], sc.commas[i], sc.dots[i]
		switch c := s[i]; c {
		case '(':
			parens = append(parens, i)
		case '{':
			braces = append(braces, i)
		case ')':
			parens = sc.close(parens, i)
		case '}':
			braces = sc.close(braces, i)
		case ',':
			sc.commas[i+1]++
		case '.':
			if i > 0 && s[i-1] == '.' {
				sc.dots[i+1]++
			}
		case ' ', '\t', '\n', ';', '|', '&':
			sc.breaks[i+1]++
		}
	}
	return sc
}

// close records that the bracket at i closes the innermost open one on
// stack, and returns the stack without it.
func (sc bracketScan) close(stack []int, i int) []int {
	if n := len(stack); n > 0 {
		sc.closer[stack[n-1]] = i
		return stack[:n-1]
	}
	return stack
}

// expansionEnd returns the index just past the expansion opaqueExpansions
// replaces at index i, or i when none starts there.
func (sc bracketScan) expansionEnd(i int) int {
	s := sc.s
	switch {
	case strings.HasPrefix(s[i:], "$("), strings.HasPrefix(s[i:], "${"):
		if j := sc.closer[i+1]; j >= 0 {
			return j + 1
		}
		return len(s)
	case s[i] == '`':
		if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
			return i + j + 2
		}
		return len(s)
	case s[i] == '{':
		j := sc.closer[i]
		if j < 0 || sc.breaks[j]-sc.breaks[i+1] > 0 {
			return i
		}
		// {} (find/xargs), {a,b} or {1..3}: a ".." pair inside the braces
		// ends at index i+2 or later.
		if j == i+1 || sc.commas[j]-sc.commas[i+1] > 0 || sc.dots[j]-sc.dots[min(i+2, j)] > 0 {
			return j + 1
		}
	}
	return i
}

// isWordBreak reports whether r separates words for InvokedSpecs.
func isWordBreak(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', ';', '|', '&', '(', ')', '<', '>', '`', '{', '}':
		return true
	}
	return false
}

// ProgramName returns the program a command word names: its base name (after
// a / or \ separator) without a Windows executable suffix, so
// `C:\bin\qsdev.exe` and `/usr/bin/qsdev` both name qsdev. Compare the result
// case-insensitively: Windows and macOS file systems resolve QSDEV to qsdev.
func ProgramName(word string) string {
	base := path.Base(strings.ReplaceAll(word, `\`, "/"))
	if ext := path.Ext(base); strings.EqualFold(ext, ".exe") {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}
