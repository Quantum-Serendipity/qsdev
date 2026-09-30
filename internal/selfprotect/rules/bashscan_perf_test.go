package rules

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// adversarialPayloads are the U18-04 shapes that ran for seconds to minutes
// before the brace scan and the missing-tail canonicalization were made
// linear. Each stays under hookio.MaxCommandBytes and MaxSimpleCommands, so
// the rules themselves must answer within budget (1s unless budget says
// otherwise, scaled by raceScale under the race detector).
var adversarialPayloads = []struct {
	name    string
	command string
	want    Verdict
	cd      bool
	budget  time.Duration
}{
	{name: "unclosed braces", command: "echo " + strings.Repeat("{", 60000), want: Allow},
	{name: "unclosed braces then protected write", command: "echo " + strings.Repeat("{", 60000) + "; rm -rf .claude/settings.json", want: Deny},
	{name: "unclosed brace words", command: "echo " + strings.Repeat("{x", 30000), want: Allow},
	{name: "many comma groups in one word", command: "echo " + strings.Repeat("{a,b}", 12000), want: Allow},
	{name: "many comma groups in a removed word", command: "rm " + strings.Repeat("{a,b}", 12000), want: Deny},
	{name: "many sequence groups in one word", command: "echo " + strings.Repeat("{1..2}", 10000), want: Allow},
	{name: "deeply nested non-groups", command: "echo " + strings.Repeat("{", 30000) + strings.Repeat("}", 30000), want: Allow},
	// 20,000 path variants, each canonicalized against the filesystem: linear
	// but syscall-bound, so it gets a wider budget that is still well inside
	// the hook's evaluation deadline.
	{name: "many comma group words", command: "echo " + strings.Repeat("{a,b} ", 10000), want: Allow, budget: 3 * time.Second},
	{name: "long cd chain then protected write", command: strings.Repeat("cd a && ", 1999) + "rm .claude/settings.json", want: Deny, cd: true},
	{name: "long cd chain", command: strings.Repeat("cd a && ", 1999) + "true", want: Allow, cd: true},
}

// evalBashTimed evaluates command in a fresh project directory and returns the
// verdict and the wall-clock time EvaluateAll took.
func evalBashTimed(t *testing.T, command string) (Verdict, time.Duration) {
	t.Helper()
	ctx := EvalContext{ToolName: "Bash", Command: command, CWD: filepath.Join(t.TempDir(), "project")}
	start := time.Now()
	v, _ := Tier1Rules.EvaluateAll(&ctx)
	return v, time.Since(start)
}

// runAdversarial evaluates the payloads whose cd field equals cd and checks
// each verdict and its time budget. Callers are not parallel so the timings
// are not skewed by sibling tests.
func runAdversarial(t *testing.T, cd bool) {
	t.Helper()
	for _, p := range adversarialPayloads {
		if p.cd != cd {
			continue
		}
		t.Run(p.name, func(t *testing.T) {
			budget := p.budget
			if budget == 0 {
				budget = time.Second
			}
			budget *= raceScale
			v, took := evalBashTimed(t, p.command)
			if v != p.want {
				t.Errorf("verdict = %v, want %v", v, p.want)
			}
			if took > budget {
				t.Errorf("EvaluateAll took %v, want under %v", took, budget)
			}
		})
	}
}

// TestFindBraceGroup_Linear pins the U18-04 brace fix: an unclosed `{` no
// longer rescans the rest of the word, and a word with many groups neither
// recurses once per group nor builds a copy of itself per level.
func TestFindBraceGroup_Linear(t *testing.T) { runAdversarial(t, false) }

// TestScannedCommands_LongCdChainUnderBudget pins the U18-04 cd-chain fix: a
// chain of cd into missing directories no longer re-cleans the whole missing
// tail per component for every later command.
func TestScannedCommands_LongCdChainUnderBudget(t *testing.T) { runAdversarial(t, true) }

// findBraceGroup reports the first group braceGroups finds, in the shape of
// the original findBraceGroup, so the two can be compared.
func findBraceGroup(s string) (open, closing int, alts []string, seq bool) {
	groups := braceGroups(s)
	if len(groups) == 0 {
		return -1, -1, nil, false
	}
	g := groups[0]
	if g.seq {
		return g.open, g.closing, nil, true
	}
	return g.open, g.closing, g.alternatives(s), false
}

// findBraceGroupReference is the original quadratic findBraceGroup, which
// rescanned to the end of the word for every unclosed `{`. It is the oracle
// the linear scan must agree with.
func findBraceGroupReference(s string) (open, closing int, alts []string, seq bool) {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		depth, start := 0, i+1
		var parts []string
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '{':
				depth++
			case ',':
				if depth == 1 {
					parts = append(parts, s[start:j])
					start = j + 1
				}
			case '}':
				depth--
				if depth != 0 {
					continue
				}
				if len(parts) > 0 {
					return i, j, append(parts, s[start:j]), false
				}
				if strings.Contains(s[i+1:j], "..") {
					return i, j, nil, true
				}
				j = len(s)
			}
		}
	}
	return -1, -1, nil, false
}

// expandBracesReference is the original one-group-per-call brace expansion
// over findBraceGroupReference.
func expandBracesReference(s string) ([]string, bool) {
	open, closing, alts, seq := findBraceGroupReference(s)
	if open < 0 {
		return []string{s}, true
	}
	prefix, suffix := s[:open], s[closing+1:]
	if seq {
		return expandBracesReference(prefix + "*" + suffix)
	}
	var out []string
	for _, alt := range alts {
		vs, ok := expandBracesReference(prefix + alt + suffix)
		if !ok {
			return nil, false
		}
		out = append(out, vs...)
		if len(out) > maxBraceVariants {
			return nil, false
		}
	}
	return out, true
}

var braceWords = []struct {
	name string
	word string
}{
	{"no braces", "plain.txt"},
	{"comma group", "{a,b}.txt"},
	{"protected comma group", "{.claude,x}/settings.json"},
	{"empty alternative", "x{,.bak}"},
	{"nested groups", "{a,{b,c}}d"},
	{"group inside a non-group", "{x{a,b}}"},
	{"non-group only", "{x}"},
	{"empty braces", "{}"},
	{"sequence", "{a..z}"},
	{"sequence then group", "{1..3}{a,b}"},
	{"group containing a sequence", "{a,{1..3}}"},
	{"sequence containing a group", "{x..{a,b}}"},
	{"sequence formed by expansion", "{{x,.}.y}"},
	{"comma group containing dots", "{a..b,c}"},
	{"unmatched open", "{a,b"},
	{"unmatched open before group", "{ {a,b}"},
	{"unmatched open around group", "{x{a,b}"},
	{"unmatched open with comma", "{a,{b}"},
	{"stray close", "a}b{c,d}"},
	{"quoted literal braces", "'{a,b}'"},
	{"double-quoted literal braces", `"{.claude,x}"/settings.json`},
	{"escaped brace", `\{a,b}`},
	{"sibling groups", "{a,b}{c,d}{e,f}"},
	{"too many variants", strings.Repeat("{a,b}", 7)},
	{"exactly 64 variants", strings.Repeat("{a,b}", 6)},
	{"many sequences", strings.Repeat("{1..2}", 100)},
	{"many unmatched opens", strings.Repeat("{", 100) + "{a,b}"},
}

// TestFindBraceGroup_Behaviour pins the linear scan to the original
// algorithm, group for group, and brace expansion to its variant list.
func TestFindBraceGroup_Behaviour(t *testing.T) {
	t.Parallel()
	for _, tt := range braceWords {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			open, closing, alts, seq := findBraceGroup(tt.word)
			wOpen, wClosing, wAlts, wSeq := findBraceGroupReference(tt.word)
			if open != wOpen || closing != wClosing || seq != wSeq || !slices.Equal(alts, wAlts) {
				t.Errorf("findBraceGroup(%q) = (%d, %d, %q, %v), want (%d, %d, %q, %v)",
					tt.word, open, closing, alts, seq, wOpen, wClosing, wAlts, wSeq)
			}
			got, ok := expandBraces(tt.word)
			want, wantOK := expandBracesReference(tt.word)
			if ok != wantOK || !slices.Equal(got, want) {
				t.Errorf("expandBraces(%q) = (%q, %v), want (%q, %v)", tt.word, got, ok, want, wantOK)
			}
		})
	}
}

// TestFindBraceGroup_ExhaustiveAgreement compares the linear scan and brace
// expansion with the reference on every word of up to seven characters over
// the characters that matter to them.
func TestFindBraceGroup_ExhaustiveAgreement(t *testing.T) {
	t.Parallel()
	const alphabet = "{},.a"
	words := []string{""}
	for range 7 {
		var next []string
		for _, w := range words {
			for i := range len(alphabet) {
				next = append(next, w+alphabet[i:i+1])
			}
		}
		for _, w := range next {
			open, closing, alts, seq := findBraceGroup(w)
			wOpen, wClosing, wAlts, wSeq := findBraceGroupReference(w)
			if open != wOpen || closing != wClosing || seq != wSeq || !slices.Equal(alts, wAlts) {
				t.Fatalf("findBraceGroup(%q) = (%d, %d, %q, %v), want (%d, %d, %q, %v)",
					w, open, closing, alts, seq, wOpen, wClosing, wAlts, wSeq)
			}
			got, ok := expandBraces(w)
			want, wantOK := expandBracesReference(w)
			if ok != wantOK || !slices.Equal(got, want) {
				t.Fatalf("expandBraces(%q) = (%q, %v), want (%q, %v)", w, got, ok, want, wantOK)
			}
		}
		words = next
	}
}

// TestExpandBraces_Pinned keeps the documented behaviour explicit, not only
// relative to the reference implementation.
func TestExpandBraces_Pinned(t *testing.T) {
	t.Parallel()
	tests := []struct {
		word   string
		want   []string
		wantOK bool
	}{
		{"{.claude,x}/settings.json", []string{".claude/settings.json", "x/settings.json"}, true},
		{"{a..z}", []string{"*"}, true},
		{"f{a,b}{1..9}", []string{"fa*", "fb*"}, true},
		{strings.Repeat("{a,b}", 7), nil, false},
		{strings.Repeat("{a,b}", 65), nil, false},
	}
	for _, tt := range tests {
		got, ok := expandBraces(tt.word)
		if ok != tt.wantOK || !slices.Equal(got, tt.want) {
			t.Errorf("expandBraces(%.40q) = (%q, %v), want (%q, %v)", tt.word, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestBraceExpansionVerdicts checks the rule-level outcomes that depend on
// brace expansion.
func TestBraceExpansionVerdicts(t *testing.T) {
	t.Parallel()
	runBashCases(t, []bashCase{
		{name: "protected comma group", command: "rm {.claude,x}/settings.json"},
		{name: "variant overflow fails closed", command: "rm " + strings.Repeat("{a,b}", 7)},
		{name: "sequence becomes a glob", command: "rm .c{a..z}aude/settings.json"},
	}, Deny)
}

func BenchmarkSelfprotectAdversarial(b *testing.B) {
	cwd := filepath.Join(b.TempDir(), "project")
	for _, p := range adversarialPayloads {
		b.Run(p.name, func(b *testing.B) {
			for b.Loop() {
				ctx := EvalContext{ToolName: "Bash", Command: p.command, CWD: cwd}
				Tier1Rules.EvaluateAll(&ctx)
			}
		})
	}
}
