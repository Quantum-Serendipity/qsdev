package rules

import (
	"cmp"
	"math"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
)

// adversarialPayloads are the U18-04 shapes that ran for seconds to minutes
// before the brace scan and the missing-tail canonicalization were fixed.
// build(n) returns the payload at size n; at n itself each stays under
// hookio.MaxCommandBytes and MaxSimpleCommands. maxRatio bounds how much
// doubling n may multiply the evaluation's CPU time, zero meaning
// linearRatio; cpuBudget bounds the best CPU time at n, zero meaning
// defaultCPUBudget (scaled by raceScale). tool is the shell tool that runs
// the payload, zero meaning Bash.
var adversarialPayloads = []struct {
	name      string
	build     func(n int) string
	n         int
	maxRatio  float64
	cpuBudget time.Duration
	want      Verdict
	cd        bool
	tool      string
}{
	{name: "unclosed braces", build: repeatAfter("echo ", "{", ""), n: 60000, want: Allow},
	{name: "unclosed braces then protected write", build: repeatAfter("echo ", "{", "; rm -rf .claude/settings.json"), n: 60000, want: Deny},
	{name: "unclosed brace words", build: repeatAfter("echo ", "{x", ""), n: 30000, want: Allow},
	{name: "many comma groups in one word", build: repeatAfter("echo ", "{a,b}", ""), n: 12000, want: Allow},
	{name: "many comma groups in a removed word", build: repeatAfter("rm ", "{a,b}", ""), n: 12000, want: Deny},
	{name: "many sequence groups in one word", build: repeatAfter("echo ", "{1..2}", ""), n: 10000, want: Allow},
	{name: "deeply nested non-groups", build: func(n int) string {
		return "echo " + strings.Repeat("{", n) + strings.Repeat("}", n)
	}, n: 30000, want: Allow},
	// 20,000 path variants, each canonicalized against the filesystem: linear
	// but syscall-bound, so it gets a wider CPU budget.
	{name: "many comma group words", build: repeatAfter("echo ", "{a,b} ", ""), n: 10000, cpuBudget: 3 * time.Second, want: Allow},
	// The same words as operands of commands that also stat, resolve and
	// symlink-resolve each one (hook targets, ancestors, guarded files).
	{name: "many removed comma group words", build: repeatAfter("rm ", "{a,b} ", ""), n: 10000, cpuBudget: 3 * time.Second, want: Allow},
	{name: "many moved comma group words", build: repeatAfter("mv ", "{a,b} ", "d"), n: 10000, cpuBudget: 3 * time.Second, want: Allow},
	// A cd chain is quadratic: the working directory grows one component per
	// cd and every later command handles it. Doubling it measures about 3
	// (4 asymptotically); the cubic canonicalization before the fix measured
	// about 6.
	{name: "long cd chain then protected write", build: repeatAfter("", "cd a && ", "rm .claude/settings.json"), n: 1999, maxRatio: quadraticRatio, want: Deny, cd: true},
	{name: "long cd chain", build: repeatAfter("", "cd a && ", "true"), n: 1999, maxRatio: quadraticRatio, want: Allow, cd: true},
	// PowerShell lines go through the PowerShell tokenizer (and the POSIX
	// parse the rules that only add denies still use): many quoted words,
	// many commands, and runs of here-string openers and of here-string
	// lines that never close.
	{name: "powershell many quoted words", build: repeatAfter("Get-Content ", `'.cl''aude\x' `, ""), n: 4000, want: Allow, tool: "PowerShell"},
	{name: "powershell many commands then delete", build: repeatAfter("", "gc a; ", `ri .claude\settings.json`), n: 1999, want: Deny, tool: "PowerShell"},
	{name: "powershell here-string openers", build: repeatAfter("Write-Output ", "@' ", ""), n: 15000, want: Allow, tool: "PowerShell"},
	{name: "powershell unclosed here-string", build: repeatAfter("Write-Output @'\n", "'x\n", ""), n: 15000, want: Allow, tool: "PowerShell"},
}

// repeatAfter returns a payload builder that repeats unit n times between
// prefix and suffix.
func repeatAfter(prefix, unit, suffix string) func(n int) string {
	return func(n int) string { return prefix + strings.Repeat(unit, n) + suffix }
}

// The doubling ratio compares CPU time, not wall time: a preempted run on a
// loaded machine takes longer by the clock but not in CPU, so the ratio
// measures the algorithm rather than the load. Each size still takes the
// best of several samples, at least minTimingRuns and, while the ratio is
// over its limit, up to maxTimingRuns; a worse complexity class never gets
// under the limit however many samples it is given.
const (
	minTimingRuns = 3
	maxTimingRuns = 10
)

// minCPUSample is the least CPU time one sample spans, repeating the
// evaluation as needed, so a coarse process clock still resolves it.
const minCPUSample = 10 * cpuClockResolution

// minRatioBaseline floors the smaller timing so a sub-millisecond run cannot
// inflate the doubling ratio.
const minRatioBaseline = time.Millisecond

// Doubling a linear scan's input doubles its time and a quadratic one's
// quadruples it, so linearRatio passes the first and fails the second.
// quadraticRatio sits likewise between the cd chain's quadratic and its
// former cubic cost.
const (
	linearRatio    = 3.0
	quadraticRatio = 5.0
)

// defaultCPUBudget is the CPU time one adversarial evaluation may take at
// full size. CPU time does not grow with machine load, so this backstop to
// the doubling ratio stays tight without flaking.
const defaultCPUBudget = time.Second

// evalTiming is the wall-clock and CPU time of one evaluation.
type evalTiming struct{ wall, cpu time.Duration }

// min keeps the best wall and CPU times independently.
func (a evalTiming) min(b evalTiming) evalTiming {
	return evalTiming{wall: min(a.wall, b.wall), cpu: min(a.cpu, b.cpu)}
}

// timeEval evaluates command in a fresh project directory, repeating until
// the sample spans minCPUSample, fails on a verdict other than want, and
// returns the mean time per evaluation. It collects garbage first so an
// earlier sample's allocations are not charged to this one.
func timeEval(t *testing.T, tool, command string, want Verdict) evalTiming {
	t.Helper()
	cwd := filepath.Join(t.TempDir(), "project")
	runtime.GC()
	startWall, startCPU := time.Now(), processCPUTime(t)
	var runs time.Duration
	for {
		ctx := EvalContext{ToolName: tool, Command: command, CWD: cwd}
		if v, _ := Tier1Rules.EvaluateAll(&ctx); v != want {
			t.Fatalf("verdict = %v, want %v (payload of %d bytes)", v, want, len(command))
		}
		runs++
		if cpu := processCPUTime(t) - startCPU; cpu >= minCPUSample {
			return evalTiming{wall: time.Since(startWall) / runs, cpu: cpu / runs}
		}
	}
}

// doublingTimes samples half and full alternately, so a load spike does not
// land on one size only, and returns the best timing of each once their CPU
// ratio is under limit or maxTimingRuns pairs have run.
func doublingTimes(t *testing.T, tool, half, full string, want Verdict, limit float64) (bestHalf, bestFull evalTiming) {
	t.Helper()
	bestHalf = evalTiming{wall: math.MaxInt64, cpu: math.MaxInt64}
	bestFull = bestHalf
	for run := 1; run <= maxTimingRuns; run++ {
		bestHalf = bestHalf.min(timeEval(t, tool, half, want))
		bestFull = bestFull.min(timeEval(t, tool, full, want))
		if run >= minTimingRuns && doublingRatio(bestHalf.cpu, bestFull.cpu) < limit {
			break
		}
	}
	return bestHalf, bestFull
}

// doublingRatio is full/half with half floored at minRatioBaseline.
func doublingRatio(half, full time.Duration) float64 {
	return float64(full) / float64(max(half, minRatioBaseline))
}

// runAdversarial evaluates the payloads of tool whose cd field equals cd at
// n and at 2n. The best wall time at n must be inside hookio.EvalDeadline (scaled by
// raceScale), the absolute property the hook relies on; the best CPU time at
// n must be inside the payload's cpuBudget; and the CPU doubling ratio must
// stay under the payload's maxRatio, which pins the complexity. Neither CPU
// bound depends on how loaded the machine is. The ratio doubles n rather than
// halving it: below n a sample lasts a few milliseconds and the garbage
// collector's pacing, not the algorithm, decides it, so a linear scan could
// measure over linearRatio. Callers are not parallel
// so sibling tests do not add to the process CPU time.
func runAdversarial(t *testing.T, tool string, cd bool) {
	t.Helper()
	deadline := hookio.EvalDeadline * raceScale
	for _, p := range adversarialPayloads {
		if p.cd != cd || cmp.Or(p.tool, "Bash") != tool {
			continue
		}
		t.Run(p.name, func(t *testing.T) {
			limit := p.maxRatio
			if limit == 0 {
				limit = linearRatio
			}
			atN, doubled := doublingTimes(t, tool, p.build(p.n), p.build(2*p.n), p.want, limit)
			if atN.wall > deadline {
				t.Errorf("EvaluateAll took %v at best, want under the %v hook deadline", atN.wall, deadline)
			}
			if budget := cmp.Or(p.cpuBudget, defaultCPUBudget) * raceScale; atN.cpu > budget {
				t.Errorf("EvaluateAll took %v of CPU at best, want under %v", atN.cpu, budget)
			}
			if ratio := doublingRatio(atN.cpu, doubled.cpu); ratio >= limit {
				t.Errorf("doubling the payload took %.2fx the CPU time (%v -> %v), want under %.1fx",
					ratio, atN.cpu, doubled.cpu, limit)
			}
		})
	}
}

// TestFindBraceGroup_Linear pins the U18-04 brace fix: an unclosed `{` no
// longer rescans the rest of the word, and a word with many groups neither
// recurses once per group nor builds a copy of itself per level.
func TestFindBraceGroup_Linear(t *testing.T) { runAdversarial(t, "Bash", false) }

// TestScannedCommands_LongCdChainUnderBudget pins the U18-04 cd-chain fix: a
// chain of cd into missing directories no longer re-cleans the whole missing
// tail per component for every later command.
func TestScannedCommands_LongCdChainUnderBudget(t *testing.T) { runAdversarial(t, "Bash", true) }

// TestPowerShellNative_LinearCPU pins the PowerShell tokenizer as one linear
// pass: doubling a PowerShell payload at most doubles the CPU time of its
// evaluation, here-string openers and unclosed here-strings included.
func TestPowerShellNative_LinearCPU(t *testing.T) { runAdversarial(t, "PowerShell", false) }

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
		command := p.build(p.n)
		b.Run(p.name, func(b *testing.B) {
			for b.Loop() {
				ctx := EvalContext{ToolName: cmp.Or(p.tool, "Bash"), Command: command, CWD: cwd}
				Tier1Rules.EvaluateAll(&ctx)
			}
		})
	}
}
