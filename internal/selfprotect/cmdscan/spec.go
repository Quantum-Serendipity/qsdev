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

// matchKind is how a command invokes a spec's subcommand, ordered from the
// least to the most certain.
type matchKind uint8

const (
	noMatch matchKind = iota
	// computedMatch: the match rests on a word the shell computes or on
	// input xargs appends, which may stand for what the spec names.
	computedMatch
	// mentionMatch: the subcommand is written out after the program named
	// outside command position (see invocation.mention), as text a program
	// may run or only hold.
	mentionMatch
	// literalMatch: the program, in command position, and the words that
	// decide the match are written out.
	literalMatch
)

// invokedIn returns how c invokes the spec's subcommand at any of its
// invocations: the most certain match.
func (s CommandSpec) invokedIn(c *simpleCommand) matchKind {
	idx := &commandIndex{}
	best := noMatch
	for _, inv := range c.invs {
		best = max(best, s.invokedAt(c, inv, idx))
	}
	return best
}

// commandIndex holds what judging one command's words for one spec has
// learned, shared by its invocations so that judging them all stays linear
// in the command's length: the path scans (built on the first invocation)
// and the rest index (built the first time an invocation's path matches).
type commandIndex struct {
	paths *pathScan
	rest  *restIndex
}

// invokedAt reports how inv invokes the spec's subcommand (see Matches). A
// dynamic word, or the end of an open invocation, ends the scan as a
// computed match once something literal anchors it; a literal read-only
// flag before any "--" (`--dry-run`, `--help`) makes it no match at all,
// except after a mention: where the text runs it is not known, so the flag
// may be another command's word (`parallel ::: "qsdev teardown" --help`).
func (s CommandSpec) invokedAt(c *simpleCommand, inv invocation, idx *commandIndex) matchKind {
	if idx.paths == nil {
		idx.paths = s.newPathScan(c.texts)
	}
	k, end := idx.paths.match(inv)
	if k == noMatch {
		return noMatch
	}
	if idx.rest == nil {
		idx.rest = s.indexRest(c.texts)
	}
	if !inv.mention && idx.rest.readOnly[inv.at+1] {
		return noMatch
	}
	if end >= 0 {
		k = idx.rest.match(s, inv, end)
	}
	if k == literalMatch && inv.mention {
		return mentionMatch
	}
	return k
}

// pathStop is why a scan of the spec's path stopped.
type pathStop uint8

const (
	// pathFailed: a literal word that is neither a flag, a flag's value nor
	// the next path word.
	pathFailed pathStop = iota
	// pathComplete: every path word is written out.
	pathComplete
	// pathDynamic: a word the shell computes, which word splitting may turn
	// into the rest of the path.
	pathDynamic
	// pathExhausted: the words ran out before the path did.
	pathExhausted
)

// pathOutcome is where a scan of the spec's path stopped: why, at which path
// level, and, for a complete path, the index just past it.
type pathOutcome struct {
	stop  pathStop
	level int
	end   int
}

// pathScan scans a command's words for the spec's path. A scan's state is
// the word index, the path level reached and whether the word may be a
// flag's value; each state leads to exactly one next one, so every state on
// a scan shares its outcome, which is recorded once and reused by every
// later scan that reaches the state (`echo qsdev -v qsdev -v ...` scans each
// word once, not once per mention).
type pathScan struct {
	spec  CommandSpec
	words []string
	// memo[(i*levels+level)*2+valueSlot] holds the outcome of the scan from
	// that state, once known.
	memo []*pathOutcome
}

// newPathScan returns an empty pathScan of words for the spec.
func (s CommandSpec) newPathScan(words []string) *pathScan {
	return &pathScan{spec: s, words: words, memo: make([]*pathOutcome, (len(words)+1)*(len(s.Path)+1)*2)}
}

// match matches the spec's path against the words after inv's program word.
// When the path is written out it returns literalMatch and the index just
// past it, for the words from there on to decide (see restIndex.match);
// otherwise end is -1 and the match is the one a dynamic word or an open end
// makes, or none.
func (ps *pathScan) match(inv invocation) (k matchKind, end int) {
	o := ps.scan(inv.at+1, 0, false)
	// A literal program, a computed one the text may have assigned the
	// program's name to, or a literal path word anchors a computed match.
	anchored := inv.literalProgram || inv.anchored || o.level > 0
	switch {
	case o.stop == pathComplete:
		return literalMatch, o.end
	case inv.mention || !anchored:
		return noMatch, -1
	case o.stop == pathDynamic, o.stop == pathExhausted && inv.open:
		return computedMatch, -1
	}
	return noMatch, -1
}

// scan returns the outcome of the scan from word i at path level level.
// valueSlot is set after a flag written without "=": cobra takes the next
// word as its value when the flag is not a known boolean, so a word there
// that is not the path may be skipped (`--version x self-update`).
func (ps *pathScan) scan(i, level int, valueSlot bool) pathOutcome {
	var chain []int
	var out pathOutcome
	for {
		key := ps.key(i, level, valueSlot)
		if o := ps.memo[key]; o != nil {
			out = *o
			break
		}
		chain = append(chain, key)
		next, done, o := ps.step(i, level, valueSlot)
		if done {
			out = o
			break
		}
		i, level, valueSlot = next.i, next.level, next.valueSlot
	}
	for _, key := range chain {
		ps.memo[key] = &out
	}
	return out
}

// scanState is one state of a path scan.
type scanState struct {
	i, level  int
	valueSlot bool
}

// step advances a path scan by one word: the next state, or the outcome
// when the scan stops at this one.
func (ps *pathScan) step(i, level int, valueSlot bool) (next scanState, done bool, o pathOutcome) {
	path := ps.spec.Path
	switch {
	case level == len(path):
		return next, true, pathOutcome{stop: pathComplete, level: level, end: i}
	case i == len(ps.words):
		return next, true, pathOutcome{stop: pathExhausted, level: level}
	}
	switch w := ps.words[i]; {
	case IsDynamicWord(w):
		return next, true, pathOutcome{stop: pathDynamic, level: level}
	case strings.HasPrefix(w, "-"):
		return scanState{i + 1, level, !strings.Contains(w, "=")}, false, o
	case slices.Contains(path[level], w):
		return scanState{i + 1, level + 1, false}, false, o
	case valueSlot:
		return scanState{i + 1, level, false}, false, o
	}
	return next, true, pathOutcome{stop: pathFailed, level: level}
}

// key returns the memo index of a scan state.
func (ps *pathScan) key(i, level int, valueSlot bool) int {
	k := (i*(len(ps.spec.Path)+1) + level) * 2
	if valueSlot {
		k++
	}
	return k
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
	// Mention is set when the command is written out but the program is
	// named outside command position (an argument, a quoted string, an
	// assignment's value): the text mentions the command, which a program
	// may run as code or may only hold as data.
	Mention bool
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
// wrappers. A word anywhere else that names it is a mention (Invoked.Mention)
// and counts only when the subcommand is written out after it: an argument
// of another program, which may run its arguments (`find -exec`, `devenv
// shell`); a word of a quoted string, whichever program takes it, for many
// run text they are given as code (`echo "..." | sh`, `trap "..." EXIT`,
// `git rebase -x "..."`, `vim -c '!...'`); and an assignment's value, which
// a later word may expand (`x="..."; $x`, `GIT_EDITOR="..." git commit`). A
// word ending in the program name after a character other than a letter,
// digit or underscore names it too (`x=qsdev`, `!qsdev`). So text that only
// mentions a command (`git commit -m "explain qsdev teardown"`, `grep -l
// "qsdev teardown" docs/*`) matches, while `grep -rn qsdev internal/*.go`
// does not: a glob or variable after a mention is data, not a computed
// subcommand. A quoted script string is split into commands like the rest
// of the text, so a separator inside data (`echo "a; qsdev teardown"`)
// errs towards the human gate.
//
// Words the shell computes fail closed rather than being resolved: a program
// word built from an expansion (`$Q defaults pin`, `$(printf qs)dev ...`,
// `{qsdev,} ...`) counts as the program when a literal path word follows it,
// a subcommand word built from one (`qsdev defaults $P`, `qsdev defaults
// $(echo pin)`) matches whatever it stands for, and an invocation under xargs
// may take its remaining words from its input (`echo pin | xargs qsdev
// defaults`, `xargs -I X X defaults pin`). A command whose program and
// subcommand words are all computed (`$A $B`) is matched only when the text
// assigns the program's name to a variable (`Q=qsdev; $Q $S`). A literal
// read-only flag (ReadOnly) before any "--" clears an invocation whatever
// else it holds (`qsdev $CMD --help`).
func InvokedSpecs(command, app string, specs []CommandSpec) []Invoked {
	var hit []Invoked
	for _, c := range commandsInvoking(command, app) {
		for _, s := range specs {
			if k := s.invokedIn(c); k != noMatch {
				hit = append(hit, Invoked{Spec: s, Computed: k == computedMatch, Mention: k == mentionMatch})
			}
		}
	}
	return hit
}

// InvokesProgram reports whether command may run the program app, judged on
// the raw text the way InvokedSpecs judges it, counting only a word that
// names app literally: in command position, or as a mention (an argument, a
// word of a quoted string, an assignment's value).
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
