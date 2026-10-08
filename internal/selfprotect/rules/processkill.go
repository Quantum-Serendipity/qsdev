package rules

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// SP-009 judges the program a command runs and the argv it runs it with,
// not the line's text: `grep -rn 'pkill qsdev' docs/` and a data
// here-document that mentions killing claude run no kill, while `pkill -f
// 'q.dev'` names no guarded process literally and still ends one. Code the
// parser cannot see (a here-document, here-string or pipe a shell or
// interpreter reads its program from, an interpreter's -c/-e code, a line
// that does not parse) gets a statement-local text check instead, and so
// does the text of any other program not proven read-only, which may run
// its words as a command line (`watch pkill qsdev`, `trap '...' EXIT`).

// maxKillPattern bounds a pattern operand SP-009 compiles; a longer one
// fails closed.
const maxKillPattern = 1 << 10

// killGuards returns the lower-cased names of the processes SP-009 guards:
// the CLI (branding) and Claude Code.
func killGuards() []string {
	return []string{strings.ToLower(branding.Get().AppName), "claude"}
}

// killsGuardedProcess reports whether ctx's shell line ends, or may end, a
// guarded process (see killScan). A PowerShell line is also judged in its
// own dialect (psKillsProtected).
func killsGuardedProcess(ctx *EvalContext) bool {
	k := newKillScan(ctx.Command)
	if isPowerShell(ctx) {
		return psKillsProtected(ctx) || k.list(psAsCommands(ctx.powerShell().cmds))
	}
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		return k.text(ctx.Command)
	}
	return anyCommandListIn(cmds, 0, k.list, k.text)
}

// killScan judges the commands of one shell line for SP-009.
type killScan struct {
	line    string
	guards  []string
	targets []string // what a pattern operand is matched against
	images  []string // what a taskkill image wildcard is matched against

	named, namedKnown   bool // memo of line naming a guard
	verb, verbKnown     bool // memo of assignsKillVerb
	lineKill, lineKnown bool // memo of lineKills
	finds               bool // a process finder on the line may find a guard
}

func newKillScan(line string) *killScan {
	k := &killScan{line: line, guards: killGuards()}
	for _, g := range k.guards {
		k.images = append(k.images, g, g+".exe")
	}
	k.targets = append(slices.Clone(k.images), k.guards[0]+" selfprotect")
	return k
}

// namesGuard reports whether the line's text names a guarded process as a
// word (read once per line), or a process finder on it has an operand that
// may find one (findsGuard), as the source of a kill's process IDs.
func (k *killScan) namesGuard() bool {
	if !k.namedKnown {
		k.named, k.namedKnown = hasGuardWord(k.line, k.guards), true
	}
	return k.named || k.finds
}

// text reports whether code text the parser cannot see has a statement
// that kills a guarded process (see textKillsGuard).
func (k *killScan) text(s string) bool { return textKillsGuard(s, k.guards) }

// list reports whether a command of cmds ends a guarded process, or a shell
// or interpreter that reads its program from a pipe is fed a statement
// that does by an earlier stage of its pipeline (`echo 'pkill qsdev' |
// sh`). Each stage's text is read at most once. The list's process finders
// are judged first, as a kill may read their output from a substitution or
// a pipe in any order.
func (k *killScan) list(cmds []cmdscan.Command) bool {
	k.finds = k.finds || slices.ContainsFunc(cmds, k.findsGuard)
	var fed map[int]bool // pipeline id: an earlier stage's text kills a guard
	for _, c := range cmds {
		if k.command(c) {
			return true
		}
		if c.Pipeline == 0 {
			continue
		}
		if fed[c.Pipeline] {
			if stdinCodeReader(c) {
				return true
			}
			continue
		}
		if k.text(commandText(c)) {
			if fed == nil {
				fed = map[int]bool{}
			}
			fed[c.Pipeline] = true
		}
	}
	return false
}

// killArgv is the argv a kill verb runs with.
type killArgv struct {
	words   []string
	dynamic func(int) bool // whether words[j] is computed
	fed     bool           // xargs appends operands read from stdin
}

// multiCallBinaries run the applet their first operand names.
var multiCallBinaries = map[string]bool{"busybox": true, "toybox": true}

// command reports whether c ends a guarded process: the program it runs,
// behind wrappers (cmdscan.Program) and a multi-call binary, is a kill verb
// whose argv targets one, or a shell or interpreter given code that does.
func (k *killScan) command(c cmdscan.Command) bool {
	words := append([]string{c.Name}, c.Args...)
	run := cmdscan.Program(words)
	if run.CommandString {
		return k.text(run.StringHead + " " + strings.Join(words[run.StringRest:], " "))
	}
	i, a := programArgv(c, words, run)
	if i < 0 {
		return false
	}
	switch programVerb(words[i]) {
	case "kill":
		return k.kill(a)
	case "pkill":
		return k.patternKill(a, pkillOptions)
	case "killall":
		return k.patternKill(a, killallOptions)
	case "taskkill":
		return k.taskkill(a)
	case "systemctl":
		return k.serviceStop(a, systemctlStops, systemctlValueOptions)
	case "launchctl":
		return k.serviceStop(a, launchctlStops, nil)
	}
	return k.code(c, words, i) || k.opaque(c, words, i)
}

// opaque reports whether c, whose program words[i] is no kill verb SP-009
// models, may still end a guarded process. A program proven read-only
// (cmdscan.IsSafeReadCommand: echo, grep, cat) only prints or reads its
// words. Any other may run them as a command line (watch, flock, su, ssh,
// tmux, at, trap, git rebase -x, awk's system()), so its text keeps the
// statement-local text check base applied to the whole line, and a
// computed word in it may hold a guarded process's ID when the line names
// one (`parallel kill ::: $(pgrep qsdev)`). A computed program word may be a
// kill verb a line assignment names (`K=pkill; $K qsdev`), and a shell,
// interpreter or sourcer given a computed word (`sh < <(...)`, `source
// <(...)`) may read its code from anywhere on the line, which is then text.
func (k *killScan) opaque(c cmdscan.Command, words []string, i int) bool {
	if cmdscan.IsSafeReadCommand(c) {
		return false
	}
	text := opaqueText(c, words[i])
	switch {
	case k.text(text):
		return true
	case c.HasExpansion && k.namesGuard() && statementKills(text):
		return true
	case c.WordHasExpansion(i) && k.namesGuard() && k.assignsKillVerb():
		return true
	}
	return c.HasExpansion && runsCode(programVerb(words[i])) && k.lineKills()
}

// lineKills reports whether the line's whole text has a statement that
// kills a guarded process (textKillsGuard); it is read once per line.
func (k *killScan) lineKills() bool {
	if !k.lineKnown {
		k.lineKill, k.lineKnown = k.text(k.line), true
	}
	return k.lineKill
}

// opaqueText is the text of c a program it runs may take as a command line:
// its words, and its here-documents and here-strings unless it is a shell or
// interpreter given a script operand, for which they are data on stdin
// (`python3 script.py <<< '...'`).
func opaqueText(c cmdscan.Command, program string) string {
	name := programVerb(program)
	if (interpreterVerbs[name] || cmdscan.IsScriptShell(name)) && !stdinCodeReader(c) {
		return strings.Join(append([]string{c.Name}, c.Args...), " ")
	}
	return commandText(c)
}

// runsCode reports whether the program name runs code it is given as a
// file or a stream: a shell, an interpreter, or a sourcing builtin.
func runsCode(name string) bool {
	return interpreterVerbs[name] || cmdscan.IsScriptShell(name) || cmdscan.SourcesCode(name)
}

// assignsKillVerb reports whether the line assigns a variable a value that
// holds a kill verb (`K=pkill`, `x="kill -9"`), which a computed program
// word may expand to; it is read once per line.
func (k *killScan) assignsKillVerb() bool {
	if !k.verbKnown {
		k.verb = slices.ContainsFunc(cmdscan.AssignedValues(k.line), statementKills)
		k.verbKnown = true
	}
	return k.verb
}

// programArgv returns the index in words (c's) of the program run runs,
// behind a multi-call binary, and the argv it runs with; the index is -1
// when c runs no program.
func programArgv(c cmdscan.Command, words []string, run cmdscan.ProgramRun) (int, killArgv) {
	i := run.Index
	if i < 0 || run.CommandString {
		return -1, killArgv{}
	}
	if multiCallBinaries[programVerb(words[i])] && i+1 < len(words) && !isFlag(words[i+1]) {
		i++
	}
	return i, killArgv{
		words:   words[i+1:],
		dynamic: func(j int) bool { return c.WordHasExpansion(i + 1 + j) },
		fed:     slices.ContainsFunc(words[:i], func(w string) bool { return programVerb(w) == "xargs" }),
	}
}

// finderOptions are the options of the process finders whose output a
// kill reads (`kill $(pgrep -f 'q[s]dev')`, `pgrep q.dev | xargs kill`):
// pgrep shares pkill's, and pidof takes names.
var finderOptions = map[string]killOptions{
	"pgrep": pkillOptions,
	"pidof": {valueShort: "oS", valueLong: []string{"--omit-pid", "--separator"}},
}

// findsGuard reports whether c runs a process finder with an operand that
// may name a guarded process, judged as patternKill judges its operands.
func (k *killScan) findsGuard(c cmdscan.Command) bool {
	words := append([]string{c.Name}, c.Args...)
	i, a := programArgv(c, words, cmdscan.Program(words))
	if i < 0 {
		return false
	}
	o, ok := finderOptions[programVerb(words[i])]
	return ok && k.operandsTarget(a, o)
}

// programVerb returns the lower-cased program a command word names.
func programVerb(word string) string { return strings.ToLower(cmdscan.ProgramName(word)) }

// kill judges kill, whose operands are process IDs and job specs: a
// computed operand counts when the line names a guarded process
// (`kill $(pgrep qsdev)`), so `kill $!` stays allowed, and so does a job
// spec unless it names one (`kill %qsdev`).
func (k *killScan) kill(a killArgv) bool {
	for j, w := range a.words {
		if a.dynamic(j) && k.namesGuard() {
			return true
		}
		if strings.HasPrefix(w, "%") && hasGuardWord(w, k.guards) {
			return true
		}
	}
	return a.fed && k.namesGuard()
}

// killOptions describes the options of a kill verb whose operands are name
// patterns.
type killOptions struct {
	valueShort  string   // short options that take a value
	valueLong   []string // long options that take a value
	regexAlways bool     // every operand is a regular expression
	regexShort  byte     // the option that makes operands regular expressions
	regexLong   string
	fullShort   byte // the option that matches operands against command lines
	fullLong    string
}

// patternMode is how a kill verb's options make it match its operands.
type patternMode struct {
	regex bool // operands are regular expressions
	full  bool // operands are matched against whole command lines
}

// pkillOptions are procps pkill's: every operand is an extended regular
// expression, matched against the name or, with -f, the command line.
var pkillOptions = killOptions{
	valueShort: "dgGPstuUFrO",
	valueLong: []string{
		"--delimiter", "--pgroup", "--group", "--parent", "--session", "--terminal", "--euid",
		"--uid", "--pidfile", "--signal", "--ns", "--nslist", "--runstates", "--older", "--cgroup",
	},
	regexAlways: true,
	fullShort:   'f',
	fullLong:    "--full",
}

// killallOptions are psmisc killall's: an operand is a name unless -r is
// given.
var killallOptions = killOptions{
	valueShort: "suoynZ",
	valueLong:  []string{"--signal", "--user", "--older-than", "--younger-than", "--ns", "--context"},
	regexShort: 'r',
	regexLong:  "--regexp",
}

// patternKill judges pkill and killall: an operand that is computed, or
// fed by xargs, names a guard as a word, or (as a pattern) matches one,
// targets a guarded process. Option values are skipped.
func (k *killScan) patternKill(a killArgv, o killOptions) bool {
	return a.fed || k.operandsTarget(a, o)
}

// operandsTarget reports whether a pattern operand of a may name a guarded
// process (patternTargets).
func (k *killScan) operandsTarget(a killArgv, o killOptions) bool {
	ops, mode := o.operands(a)
	return slices.ContainsFunc(ops, func(j int) bool {
		return k.patternTargets(a.words[j], a.dynamic(j), mode)
	})
}

// operands returns the indexes of a's operands and how they are matched.
func (o killOptions) operands(a killArgv) (ops []int, mode patternMode) {
	mode.regex = o.regexAlways
	for j := 0; j < len(a.words); j++ {
		w := a.words[j]
		switch {
		case a.dynamic(j) || w == "-" || !isFlag(w):
			ops = append(ops, j)
		case w == "--":
			for r := j + 1; r < len(a.words); r++ {
				ops = append(ops, r)
			}
			return ops, mode
		case strings.HasPrefix(w, "--"):
			name, _, inline := strings.Cut(w, "=")
			mode.regex = mode.regex || isLongPrefix(name, o.regexLong)
			mode.full = mode.full || isLongPrefix(name, o.fullLong)
			if !inline && slices.Contains(o.valueLong, name) {
				j++
			}
		case !isSignalSpec(w[1:]):
			m, skip := o.shortCluster(w[1:])
			mode.regex = mode.regex || m.regex
			mode.full = mode.full || m.full
			if skip {
				j++
			}
		}
	}
	return ops, mode
}

// isLongPrefix reports whether name, a long option word, is an
// abbreviation of long (when long is set).
func isLongPrefix(name, long string) bool {
	return len(name) > 2 && strings.HasPrefix(long, name)
}

// shortCluster reads a short-option cluster: how it makes operands match,
// and whether its value is the next word.
func (o killOptions) shortCluster(cluster string) (mode patternMode, nextIsValue bool) {
	for p := 0; p < len(cluster); p++ {
		c := cluster[p]
		mode.regex = mode.regex || o.regexShort != 0 && c == o.regexShort
		mode.full = mode.full || o.fullShort != 0 && c == o.fullShort
		if strings.IndexByte(o.valueShort, c) >= 0 {
			return mode, p == len(cluster)-1
		}
	}
	return mode, false
}

// isSignalSpec reports whether s, a word after its dash, is a signal
// (`-9`, `-KILL`, `-SIGHUP`, `-RTMIN+1`) rather than options: digits, or two
// or more upper-case letters and digits.
func isSignalSpec(s string) bool {
	if s == "" {
		return false
	}
	digits, upper := true, len(s) >= 2
	for i := 0; i < len(s); i++ {
		c := s[i]
		digits = digits && c >= '0' && c <= '9'
		upper = upper && (c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '+' || c == '-')
	}
	return digits || upper
}

// patternTargets reports whether a pattern operand may name a guarded
// process: it is computed, names one as a word, or, as a regular
// expression, matches one (case-insensitively) or, matched against command
// lines, matches one that runs through one (patternSpansGuard: `q[s]dev
// mcp`); one that will not compile or is over maxKillPattern fails closed.
// A name operand counts when it holds a glob character.
func (k *killScan) patternTargets(w string, dynamic bool, mode patternMode) bool {
	if dynamic || hasGuardWord(w, k.guards) {
		return true
	}
	if !mode.regex {
		return strings.ContainsAny(w, "*?[")
	}
	if len(w) > maxKillPattern {
		return true
	}
	re, err := regexp.Compile("(?i)" + w)
	if err != nil {
		return true
	}
	if slices.ContainsFunc(k.targets, re.MatchString) {
		return true
	}
	if !mode.full {
		return false
	}
	prog, err := compileKillPattern(w)
	if err != nil {
		return true
	}
	return slices.ContainsFunc(k.guards, func(g string) bool { return patternSpansGuard(prog, g) })
}

// taskkillValueParams are the taskkill parameters that take a value.
var taskkillValueParams = map[string]bool{"im": true, "fi": true, "pid": true, "s": true, "u": true, "p": true}

// taskkill judges taskkill (/ or - parameters, any case): an /IM image that
// is computed, names a guard, or is a wildcard matching one, and an /FI
// filter that is computed or names one, target a guarded process.
func (k *killScan) taskkill(a killArgv) bool {
	for j := 0; j < len(a.words); j++ {
		w := a.words[j]
		if len(w) < 2 || (w[0] != '/' && w[0] != '-') || !taskkillValueParams[strings.ToLower(w[1:])] {
			continue
		}
		if j+1 >= len(a.words) {
			break
		}
		j++
		v, dynamic := a.words[j], a.dynamic(j)
		switch strings.ToLower(w[1:]) {
		case "im":
			if dynamic || k.imageTargets(v) {
				return true
			}
		case "fi":
			if dynamic || hasGuardWord(v, k.guards) {
				return true
			}
		}
	}
	return a.fed
}

// imageTargets reports whether a taskkill image name may be a guarded
// process: it names one, or is a wildcard matching one; a malformed
// wildcard fails closed.
func (k *killScan) imageTargets(image string) bool {
	if hasGuardWord(image, k.guards) {
		return true
	}
	if !strings.Contains(image, "*") && !strings.Contains(image, "?") {
		return false
	}
	pattern := strings.ToLower(image)
	return slices.ContainsFunc(k.images, func(name string) bool {
		ok, err := path.Match(pattern, name)
		return ok || err != nil
	})
}

// systemctlStops and launchctlStops are the subcommands that stop, kill,
// restart or unregister a unit or job.
var (
	systemctlStops = map[string]bool{
		"stop": true, "kill": true, "restart": true, "try-restart": true, "reload-or-restart": true,
		"try-reload-or-restart": true, "disable": true, "mask": true, "freeze": true,
	}
	launchctlStops = map[string]bool{
		"kill": true, "stop": true, "bootout": true, "remove": true, "unload": true, "disable": true,
	}
)

// systemctlValueOptions are the systemctl options that take the next word.
var systemctlValueOptions = []string{
	"-t", "--type", "-p", "--property", "-s", "--signal", "-H", "--host", "-M", "--machine",
	"-n", "--lines", "-o", "--output", "--kill-whom", "--kill-value", "--root", "--state",
	"--job-mode", "--what", "--timestamp", "--image", "--drop-in", "--when", "--message",
}

// serviceStop judges a service manager: when its subcommand (the first
// operand) is a stop verb, or computed, a unit or label word that is
// computed or holds a guard as a token (`qsdev-mcp`, `com.qsdev.mcp`)
// targets a guarded process.
func (k *killScan) serviceStop(a killArgv, stops map[string]bool, valueOptions []string) bool {
	verb := false
	for j := 0; j < len(a.words); j++ {
		w, dynamic := a.words[j], a.dynamic(j)
		switch {
		case isFlag(w) && !dynamic:
			if !verb && !strings.Contains(w, "=") && slices.Contains(valueOptions, w) {
				j++
			}
		case !verb:
			if !dynamic && !stops[strings.ToLower(w)] {
				return false
			}
			verb = true
		case dynamic || hasGuardWord(w, k.guards):
			return true
		}
	}
	return false
}

// codeFlags are the interpreter options whose value is program text.
var codeFlags = []string{"-c", "-e", "-E", "--eval", "-p", "--print"}

// code reports whether the program words[i] runs is a shell or interpreter
// given code that kills a guarded process: an interpreter's -c/-e text, or
// the here-documents and here-strings one reading its program from stdin
// is given. A shell's -c script and eval are parsed (anyCommandListIn).
func (k *killScan) code(c cmdscan.Command, words []string, i int) bool {
	name := programVerb(words[i])
	if interpreterVerbs[name] && !cmdscan.IsScriptShell(name) {
		for j := i + 1; j+1 < len(words); j++ {
			if isCodeFlag(words[j]) && k.text(words[j+1]) {
				return true
			}
		}
	}
	if !stdinCodeReader(c) {
		return false
	}
	return slices.ContainsFunc(c.Heredocs, k.text) || slices.ContainsFunc(c.ReadRedirects, k.text)
}

// isCodeFlag reports whether an interpreter option word takes program text
// as the next word: one of codeFlags, or a short cluster ending in c or e
// (`perl -ne`, `python3 -Ic`).
func isCodeFlag(w string) bool {
	if slices.Contains(codeFlags, w) {
		return true
	}
	return len(w) > 2 && w[0] == '-' && w[1] != '-' && strings.ContainsRune("ceE", rune(w[len(w)-1]))
}

// stdinCodeReader reports whether c runs, behind wrappers, a shell or
// interpreter that reads its program from stdin: one given no operand,
// only options (`bash -s`, `python3 -`).
func stdinCodeReader(c cmdscan.Command) bool {
	words := append([]string{c.Name}, c.Args...)
	i := cmdscan.Program(words).Index
	if i < 0 {
		return false
	}
	name := programVerb(words[i])
	if !interpreterVerbs[name] && !cmdscan.IsScriptShell(name) {
		return false
	}
	return !slices.ContainsFunc(words[i+1:], func(w string) bool { return !isFlag(w) })
}

// commandText is the text a command can feed a pipe: its words, its
// here-documents and here-strings.
func commandText(c cmdscan.Command) string {
	return strings.Join(slices.Concat([]string{c.Name}, c.Args, c.Heredocs, c.ReadRedirects), " ")
}

// killTextVerbs are the kill verbs the text check reads as tokens, and
// serviceManagers the programs that stop a service with one of their
// stop subcommands.
var (
	killTextVerbs   = map[string]bool{"kill": true, "pkill": true, "killall": true, "taskkill": true}
	serviceManagers = map[string]map[string]bool{"systemctl": systemctlStops, "launchctl": launchctlStops}
)

// textKillsGuard reports whether code text has a statement (split at
// newlines and ;|&) that holds a kill-verb token and names a guarded
// process as a word. Tokens are bounded by characters other than
// [A-Za-z0-9_.-], so `kill-after`, `TestKill` and `os.kill` are not the
// verb; a service manager counts with one of its stop subcommands.
func textKillsGuard(text string, guards []string) bool {
	for _, stmt := range strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune("\n;|&", r) }) {
		if statementKills(stmt) && hasGuardWord(stmt, guards) {
			return true
		}
	}
	return false
}

// statementKills reports whether one statement holds a kill-verb token.
func statementKills(stmt string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(stmt), func(r rune) bool {
		inToken := r < 0x80 && isLowerAlnum(byte(r)) || r == '_' || r == '.' || r == '-'
		return !inToken
	})
	var stops map[string]bool
	for _, t := range tokens {
		if killTextVerbs[t] {
			return true
		}
		if m, ok := serviceManagers[t]; ok {
			stops = m
		}
	}
	return stops != nil && slices.ContainsFunc(tokens, func(t string) bool { return stops[t] })
}

// hasGuardWord reports whether text names a guarded process as a word,
// case-insensitively: an occurrence bounded by characters other than
// [a-z0-9], so `qsdev.exe`, `qsdev-mcp` and `com.qsdev.mcp` name it and
// `qsdevtools` does not.
func hasGuardWord(text string, guards []string) bool {
	lower := strings.ToLower(text)
	for _, g := range guards {
		for from := 0; from < len(lower); {
			j := strings.Index(lower[from:], g)
			if j < 0 {
				break
			}
			start, end := from+j, from+j+len(g)
			if (start == 0 || !isLowerAlnum(lower[start-1])) && (end == len(lower) || !isLowerAlnum(lower[end])) {
				return true
			}
			from = start + 1
		}
	}
	return false
}

func isLowerAlnum(c byte) bool { return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' }

// psAsCommands renders PowerShell commands as the commands the argv checks
// judge: a computed word (a variable, subexpression or splat) or a glob is
// an expanded argument.
func psAsCommands(cmds []cmdscan.PSCommand) []cmdscan.Command {
	out := make([]cmdscan.Command, len(cmds))
	for i, c := range cmds {
		out[i] = cmdscan.Command{Name: c.Name, Args: c.Args}
		for _, a := range c.Args {
			if cmdscan.IsDynamicWord(a) || strings.HasPrefix(a, "@") {
				out[i].ExpandedArgs = append(out[i].ExpandedArgs, a)
			}
		}
	}
	return out
}
