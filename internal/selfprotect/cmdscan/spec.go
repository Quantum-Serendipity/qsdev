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
	// ReadOnly holds the spellings of the flags (e.g. "--dry-run", "--help")
	// that make an invocation read-only, or print instead of running it;
	// such an invocation never matches.
	ReadOnly []string
	// ValueFlags holds the spellings of the flags that take the next word as
	// their value, so a ReadOnly spelling there (`--reason --help`) is a
	// value, not the flag.
	ValueFlags []string
	// Flags match an invocation that sets any of them to its Value.
	Flags []FlagCond
	// Args returns the positional arguments that match an invocation; it is
	// called only once an invocation may match. Nil matches none.
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
// are skipped, and so is a word after a flag written without "=", which may
// be its value. A flag value that does not parse as a boolean counts as
// matching, so a spelling the CLI would reject never hides a match. A word
// whose value the shell computes (IsDynamicWord) may stand for the rest of
// the path and any flag or argument, so it matches.
func (s CommandSpec) Matches(argv []string) bool {
	c := &simpleCommand{texts: append([]string{""}, argv...)}
	c.invs = []invocation{{literalProgram: true}}
	return s.invokedIn(c) != noMatch
}

// matchKind is how a command invokes a spec's subcommand.
type matchKind uint8

const (
	noMatch matchKind = iota
	// literalMatch: the program and the words that decide the match are
	// written out.
	literalMatch
	// computedMatch: the match rests on a word the shell computes or on
	// input xargs appends, which may stand for what the spec names.
	computedMatch
)

// invokedIn returns how c invokes the spec's subcommand at any of its
// invocations: a literal match over a computed one.
func (s CommandSpec) invokedIn(c *simpleCommand) matchKind {
	var rest *restIndex
	best := noMatch
	for _, inv := range c.invs {
		if k := s.invokedAt(c, inv, &rest); k != noMatch && (best == noMatch || k == literalMatch) {
			best = k
		}
	}
	return best
}

// invokedAt reports how inv invokes the spec's subcommand (see Matches). A
// dynamic word, or the end of an open invocation, ends the scan as a
// computed match once something literal anchors it; a literal read-only
// flag before any "--" (`--dry-run`, `--help`) makes it no match at all.
// rest indexes c's words for the spec; it is built the first time an
// invocation matches and shared by the others.
func (s CommandSpec) invokedAt(c *simpleCommand, inv invocation, rest **restIndex) matchKind {
	k, end := s.pathMatch(c.texts, inv)
	if k == noMatch {
		return noMatch
	}
	if *rest == nil {
		*rest = s.indexRest(c.texts)
	}
	if (*rest).readOnly[inv.at+1] {
		return noMatch
	}
	if end >= 0 {
		return (*rest).match(s, inv, end)
	}
	return k
}

// pathMatch matches the spec's path against the words after inv's program
// word. When the path is written out it returns literalMatch and the index
// just past it, for the words from there on to decide (see restIndex.match);
// otherwise end is -1 and the match is the one a dynamic word or an open end
// makes, or none.
func (s CommandSpec) pathMatch(words []string, inv invocation) (k matchKind, end int) {
	anchored := inv.literalProgram
	level := 0
	// valueSlot is set after a flag written without "=": cobra takes the
	// next word as its value when the flag is not a known boolean, so a word
	// there that is not the path may be skipped (`--version x self-update`).
	valueSlot := false
	i := inv.at + 1
	for ; i < len(words) && level < len(s.Path); i++ {
		w := words[i]
		switch {
		case IsDynamicWord(w):
			// Word splitting may turn it into the rest of the path.
			if anchored && !inv.mention {
				return computedMatch, -1
			}
			return noMatch, -1
		case strings.HasPrefix(w, "-"):
			valueSlot = !strings.Contains(w, "=")
			continue
		case !slices.Contains(s.Path[level], w):
			if valueSlot {
				valueSlot = false
				continue
			}
			return noMatch, -1
		}
		valueSlot = false
		level++
		anchored = true
	}
	switch {
	case level == len(s.Path):
		return literalMatch, i
	case inv.open && anchored && !inv.mention:
		return computedMatch, -1
	}
	return noMatch, -1
}

// restIndex records, for each index of a command's words, what the words
// from there on hold for one spec, so that every invocation in a command is
// judged in time linear in its length.
type restIndex struct {
	// readOnly[i] is set when a literal read-only flag follows before any
	// "--", other than as the value of a flag that takes one (ValueFlags).
	readOnly []bool
	// cond[i] is set when a literal flag or argument the spec's conditions
	// name follows; dynamic[i] when a word the shell computes does.
	cond, dynamic []bool
}

// indexRest indexes words for the spec.
func (s CommandSpec) indexRest(words []string) *restIndex {
	n := len(words)
	r := &restIndex{readOnly: make([]bool, n+1), cond: make([]bool, n+1), dynamic: make([]bool, n+1)}
	var args []string
	if s.Args != nil {
		args = s.Args()
	}
	for i := n - 1; i >= 0; i-- {
		w := words[i]
		isValue := i > 0 && slices.Contains(s.ValueFlags, words[i-1])
		r.readOnly[i] = w != "--" && (r.readOnly[i+1] || !isValue && flagValue(w, s.ReadOnly, true))
		r.dynamic[i] = r.dynamic[i+1] || IsDynamicWord(w)
		r.cond[i] = r.cond[i+1] || slices.ContainsFunc(s.Flags, func(f FlagCond) bool { return flagValue(w, f.Spellings, f.Value) }) ||
			!strings.HasPrefix(w, "-") && slices.Contains(args, w)
	}
	return r
}

// match judges the words from index end on, after the spec's path written
// out at inv: an unconditional spec, or a literal flag or argument its
// conditions name, matches literally; otherwise, unless inv is a mention, an
// open invocation and a dynamic word (which may be such a flag or argument)
// match as computed.
func (r *restIndex) match(s CommandSpec, inv invocation, end int) matchKind {
	switch {
	case s.Always() || r.cond[end]:
		if inv.literalProgram {
			return literalMatch
		}
		return computedMatch
	case inv.mention:
		return noMatch
	case inv.open || r.dynamic[end]:
		return computedMatch
	}
	return noMatch
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

// Invoked is a spec a command invokes, and how it was found.
type Invoked struct {
	Spec CommandSpec
	// Computed is set when the match rests on a word the shell computes (a
	// variable, a substitution, a glob, an alias for a computed value) or on
	// input xargs appends, rather than on the subcommand written out.
	Computed bool
}

// InvokedSpecs returns the specs that command invokes the program app with,
// judged on the raw text: the text is split into simple commands at shell
// operators and into words at blanks, with quotes dropped, so quoting,
// wrappers (env, timeout, xargs, ...), command lines run by another program
// (sh -c, eval, script -c, PowerShell -Command, cmd /c) and compound
// commands do not hide an invocation. A backslash is read both as a shell
// escape (q\sdev) and as a Windows path separator.
//
// The program counts where it may run: as the program word of a simple
// command or of such a command line, past reserved words, assignments and
// wrappers. An unquoted argument of another program that names it counts as
// well, for a program may run its arguments (`find -exec`, `devenv shell`),
// but only when the subcommand is written out: `grep qsdev *.go` is no
// invocation. A quoted argument (`grep "qsdev teardown" docs/*`) is data. A
// quoted script string is split into commands like the rest of the text, so
// a separator inside data (`echo "a; qsdev teardown"`) errs towards the
// human gate.
//
// Words the shell computes fail closed rather than being resolved: a program
// word built from an expansion (`$Q defaults pin`, `$(printf qs)dev ...`,
// `{qsdev,} ...`) counts as the program when a literal path word follows it,
// a subcommand word built from one (`qsdev defaults $P`, `qsdev defaults
// $(echo pin)`) matches whatever it stands for, and an invocation under xargs
// may take its remaining words from its input (`echo pin | xargs qsdev
// defaults`, `xargs -I X X defaults pin`). Only a command whose program and
// subcommand words are all computed (`$A $B`) is not matched. A literal
// read-only flag (ReadOnly) before any "--" clears an invocation whatever
// else it holds (`qsdev $CMD --help`).
func InvokedSpecs(command, app string, specs []CommandSpec) []Invoked {
	var hit []Invoked
	for _, c := range commandsInvoking(command, app) {
		for _, s := range specs {
			if k := s.invokedIn(c); k != noMatch {
				hit = append(hit, Invoked{Spec: s, Computed: k == computedMatch})
			}
		}
	}
	return hit
}

// InvokesProgram reports whether command may run the program app, judged on
// the raw text the way InvokedSpecs judges it, counting only a word that
// names app literally: in command position, or as an unquoted argument of
// another program.
func InvokesProgram(command, app string) bool {
	return slices.ContainsFunc(commandsInvoking(command, app), func(c *simpleCommand) bool {
		return slices.ContainsFunc(c.invs, func(inv invocation) bool { return inv.literalProgram })
	})
}

// IsDynamicWord reports whether the shell computes word rather than taking it
// as written: it holds a parameter expansion or a command substitution ($,
// including the placeholder opaqueExpansions leaves) or a glob character.
func IsDynamicWord(word string) bool {
	return strings.ContainsAny(word, "$*?[")
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
