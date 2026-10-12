package rules

import (
	"regexp/syntax"
	"slices"
	"unicode"
)

// A command-line pattern (pkill/pgrep -f) is searched for anywhere in a
// process's command line. Matching it against the bare guarded names misses
// a pattern that also needs the rest of a guarded process's command line
// (`q[s]dev mcp`, `[c]laude --resume`), and no fixed list of those command
// lines can be complete. patternSpansGuard asks instead whether some string
// the pattern matches runs through the end of a guarded name that stands as
// a word, the shape of a guarded process's command line (`[path/]qsdev
// args`). The match may begin inside the name, or before it in a path
// prefix whose word runes the pattern spells out (`/usr/bin/q[s]dev`) and
// whose separators anything may read (`.q[s]dev`, `/opt/.*/q[s]dev`): a
// word read by a wildcard or a class (`node.*vite`) is any word at all, not
// the guarded program's path. The name is followed by a non-word byte or the
// end of the match. The walk is over the product of the compiled program and
// a small automaton for the word, so it costs O(len(prog) * len(guard)).

// Word-automaton states besides 0..len(guard), how much of the guard has
// been read since a word boundary.
const (
	spanState     = -1 // past the guard and a boundary: the pattern spans it
	prefixBreak   = -2 // in a spelled-out prefix, after a non-word rune
	prefixInWord  = -3 // in a spelled-out prefix, inside a word
	deadWordState = -4 // the match can no longer run through the guard
)

// patternSpansGuard reports whether prog, a compiled case-insensitive
// pattern, matches a string that runs through the end of guard (a
// lower-case name) as a word (see above).
func patternSpansGuard(prog *syntax.Prog, guard string) bool {
	type node struct{ pc, state int }
	seen := make(map[node]bool, len(prog.Inst)*(len(guard)+3))
	var stack []node
	push := func(pc, state int) {
		n := node{pc, state}
		if state != deadWordState && !seen[n] {
			seen[n] = true
			stack = append(stack, n)
		}
	}
	push(prog.Start, prefixBreak)
	for k := 0; k < len(guard); k++ {
		push(prog.Start, k)
	}
	probes := guardProbes(guard)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		inst := &prog.Inst[n.pc]
		switch inst.Op {
		case syntax.InstAlt, syntax.InstAltMatch:
			push(int(inst.Out), n.state)
			push(int(inst.Arg), n.state)
		case syntax.InstCapture, syntax.InstNop, syntax.InstEmptyWidth:
			push(int(inst.Out), n.state)
		case syntax.InstMatch:
			if n.state == len(guard) {
				return true
			}
		case syntax.InstFail:
		default: // the rune instructions
			literal := isLiteralInst(inst)
			for _, r := range runeProbes(inst, probes) {
				for _, next := range guardSteps(guard, n.state, r, literal) {
					if next == spanState {
						return true
					}
					push(int(inst.Out), next)
				}
			}
		}
	}
	return false
}

// guardSteps returns the states the word automaton for guard reaches from
// state over rune r, read by a literal instruction or not.
func guardSteps(guard string, state int, r rune, literal bool) []int {
	word := isWordRune(r)
	switch {
	case state == len(guard):
		if !word {
			return []int{spanState}
		}
		return []int{deadWordState}
	case state >= 0:
		if r == rune(guard[state]) {
			return []int{state + 1}
		}
		return []int{prefixState(word, literal)}
	case literal && state == prefixBreak && r == rune(guard[0]):
		return []int{1, prefixInWord}
	}
	return []int{prefixState(word, literal)}
}

// prefixState is the state after a rune that leaves the guard, or extends
// a prefix: a prefix's word runes are spelled out, and its separators may
// be read by anything.
func prefixState(word, literal bool) int {
	switch {
	case !word:
		return prefixBreak
	case literal:
		return prefixInWord
	default:
		return deadWordState
	}
}

// isWordRune reports whether r is a word byte (isLowerAlnum, after
// lower-casing).
func isWordRune(r rune) bool {
	r = unicode.ToLower(r)
	return r < 0x80 && isLowerAlnum(byte(r))
}

// isLiteralInst reports whether inst matches one rune only, up to case: a
// pattern spells that rune out (`q`, `[s]`).
func isLiteralInst(inst *syntax.Inst) bool {
	if inst.Op == syntax.InstRune1 {
		return true
	}
	if inst.Op != syntax.InstRune || len(inst.Rune) == 0 {
		return false
	}
	if len(inst.Rune) == 1 {
		return true
	}
	first := unicode.ToLower(inst.Rune[0])
	for i := 0; i+1 < len(inst.Rune); i += 2 {
		lo, hi := inst.Rune[i], inst.Rune[i+1]
		if lo != hi || unicode.ToLower(lo) != first {
			return false
		}
	}
	return true
}

// wordRunes are the word bytes (isLowerAlnum).
const wordRunes = "abcdefghijklmnopqrstuvwxyz0123456789"

// guardProbes returns the bytes of guard, each once: with one other word
// rune and one non-word rune (runeProbes) they tell every transition of
// the word automaton apart.
func guardProbes(guard string) []rune {
	var probes []rune
	for i := 0; i < len(guard); i++ {
		if r := rune(guard[i]); !slices.Contains(probes, r) {
			probes = append(probes, r)
		}
	}
	return probes
}

// runeProbes returns the probes inst matches, plus one other word rune and
// one non-word rune it matches, if any: every transition of the word
// automaton inst can take.
func runeProbes(inst *syntax.Inst, probes []rune) []rune {
	anyRune := inst.Op == syntax.InstRuneAny || inst.Op == syntax.InstRuneAnyNotNL
	matches := func(r rune) bool { return anyRune || inst.MatchRune(r) }
	var out []rune
	for _, r := range probes {
		if matches(r) {
			out = append(out, r)
		}
	}
	for _, r := range wordRunes {
		if !slices.Contains(probes, r) && matches(r) {
			out = append(out, r)
			break
		}
	}
	if anyRune {
		return append(out, ' ')
	}
	if r, ok := nonWordRune(inst); ok {
		out = append(out, r)
	}
	return out
}

// nonWordRune returns a rune inst matches that is not a word rune, if
// there is one. A range holds one when an end of it is one or it crosses
// an edge of a run of word runes.
func nonWordRune(inst *syntax.Inst) (rune, bool) {
	edges := []rune{'0' - 1, '9' + 1, 'A' - 1, 'Z' + 1, 'a' - 1, 'z' + 1}
	for i := 0; i+1 < len(inst.Rune); i += 2 {
		lo, hi := inst.Rune[i], inst.Rune[i+1]
		for _, r := range append([]rune{lo, hi}, edges...) {
			if r >= lo && r <= hi && !isWordRune(r) && inst.MatchRune(r) {
				return r, true
			}
		}
	}
	if len(inst.Rune) == 1 && !isWordRune(inst.Rune[0]) && inst.MatchRune(inst.Rune[0]) {
		return inst.Rune[0], true
	}
	return 0, false
}

// compileKillPattern compiles a process pattern case-insensitively, as
// patternTargets matches it, into the program patternSpansGuard walks.
func compileKillPattern(pattern string) (*syntax.Prog, error) {
	re, err := syntax.Parse(pattern, syntax.Perl|syntax.FoldCase)
	if err != nil {
		return nil, err
	}
	return syntax.Compile(re.Simplify())
}
