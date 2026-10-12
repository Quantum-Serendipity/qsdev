package cmdscan

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// interpreterOptions describes how an interpreter is told where its program
// comes from. Short options may be clustered (`python3 -Bc code`, `perl
// -ne code`); a long option may carry its value after "=".
type interpreterOptions struct {
	// code are the options whose argument is the program text (`-c`, `-e`).
	code []string
	// script are the options whose argument names the program to run
	// (`python3 -m mod`).
	script []string
	// values are the other options that take an argument: the rest of a
	// short-option cluster, the text after "=", or else the next word.
	values []string
}

// interpreters are the script interpreters other than the shells (see
// scriptShells), keyed by program name without a version suffix (python3
// and python3.12 are python). Each runs a script file given as its first
// operand, or else reads its program from standard input.
var interpreters = map[string]interpreterOptions{
	"python": {code: []string{"-c"}, script: []string{"-m"}, values: []string{"-W", "-X", "--check-hash-based-pycs"}},
	"node": {
		code:   []string{"-e", "--eval", "-p", "--print"},
		values: []string{"-r", "--require", "--import", "--loader", "--experimental-loader", "--input-type", "-C", "--conditions", "--title"},
	},
	"perl": {code: []string{"-e", "-E"}, values: []string{"-I", "-M", "-m"}},
	"ruby": {code: []string{"-e"}, values: []string{"-I", "-r", "-C", "-E", "--encoding"}},
}

// interpreterAliases are other names the interpreters are installed under:
// Debian's nodejs, and PyPy, which takes CPython's options.
var interpreterAliases = map[string]string{"nodejs": "node", "pypy": "python"}

// interpreterName returns the name a program word is looked up by in
// interpreters: its case-folded program name (see wrapperName) without a
// version suffix, an alias (interpreterAliases) resolved.
func interpreterName(word string) string {
	name := strings.TrimRight(wrapperName(word), "0123456789.")
	if alias, ok := interpreterAliases[name]; ok {
		return alias
	}
	return name
}

// IsInterpreter reports whether the program named name is a script
// interpreter other than a shell: python, node, perl or ruby, named by path,
// in any case, with any version suffix and by an alias (/usr/bin/python3,
// PYTHON3, python3.12, nodejs, pypy3). IsScriptShell reports the shells.
func IsInterpreter(name string) bool {
	_, ok := interpreters[interpreterName(name)]
	return ok
}

// stdinPaths are the file names that open the process's own standard input:
// a script operand or `<` word naming one reads the same stream as no
// operand or redirect at all.
var stdinPaths = []string{"/dev/stdin", "/dev/fd/0", "/proc/self/fd/0"}

// isStdinPath reports whether the word w names the process's standard input.
func isStdinPath(w string) bool { return slices.Contains(stdinPaths, w) }

// isSocketPath reports whether the word w names one of bash's network
// pseudo-files (/dev/tcp/HOST/PORT, /dev/udp/HOST/PORT), whose content is
// whatever the remote end sends.
func isSocketPath(w string) bool {
	return strings.HasPrefix(w, "/dev/tcp/") || strings.HasPrefix(w, "/dev/udp/")
}

// sourceKind says where a shell or interpreter takes its program from.
type sourceKind int

const (
	fromStdin  sourceKind = iota // standard input (no operand, -s, -, /dev/stdin)
	fromCode                     // an inline program given to a code option (-c, -e)
	fromScript                   // a script file or module operand
)

// programSource is where a shell or interpreter takes its program from and,
// for an inline program, the index of its word in the arguments, or -1 when
// the code option is the last word (the program is left to xargs to append).
type programSource struct {
	kind sourceKind
	word int
}

// ReadsProgramFromStdin reports whether c runs a shell or an interpreter,
// directly or through wrappers, that reads its program from standard input:
// it is given no inline program (-c, -e, --eval, ...) and no script operand,
// is told to read standard input (`bash -s`, `python3 -`), or is given a
// script operand naming it (`sh /dev/stdin`). A busybox applet counts as the
// shell or interpreter it names; `sudo -s`, `sudo -i`, `doas -s` and su
// without -c start a shell that reads it, and PowerShell reads it given no
// arguments, or only switches (`pwsh -NoProfile`), or `-Command -`/`-File
// -`. A `.` or `source` of a standard-input path runs it in the shell
// itself, and `env -S 'sh -s'` runs the words its string splits into.
// Under xargs, or parallel without --pipe, the program's operands come from
// input and its standard input is not the line's, so it never counts (see
// ProgramFromInput).
func ReadsProgramFromStdin(c Command) bool {
	return readsProgramFromStdin(append([]string{c.Name}, c.Args...))
}

// readsProgramFromStdin is ReadsProgramFromStdin for the words of a
// command.
func readsProgramFromStdin(words []string) bool {
	run := Program(words)
	switch {
	case run.CommandString && !run.StringShell:
		// `env -S 'sh -s'` splits its string into the words it runs. A
		// string run through a shell is a script (see Script).
		return readsProgramFromStdin(append(strings.Fields(run.StringHead), words[run.StringRest:]...))
	case run.Index < 0:
		return !run.CommandString && !run.LookupOnly && startsLoginShell(words)
	}
	if argInputWrapper(words[:run.Index]) >= 0 {
		return false
	}
	src, ok := sourceOf(words[run.Index], words[run.Index+1:])
	return ok && src.kind == fromStdin
}

// ProgramFromInput reports whether c runs a shell or interpreter whose
// program comes from its input: it reads its program from standard input
// (ReadsProgramFromStdin); or xargs or parallel, which build the program's
// arguments from their own input, give it a code option and no code word
// (`xargs sh -c`, `xargs -0 python3 -c`) or a code word holding their
// replace string (`xargs -I{} sh -c {}`, `parallel sh -c {}`), so the input
// becomes the program text; or parallel is given no command, so it runs each
// input line as one (`… | parallel -j4`).
func ProgramFromInput(c Command) bool {
	if ReadsProgramFromStdin(c) {
		return true
	}
	words := append([]string{c.Name}, c.Args...)
	run := Program(words)
	if run.Index < 0 {
		return !run.CommandString && !run.LookupOnly && runsInputLines(words)
	}
	x := argInputWrapper(words[:run.Index])
	if x < 0 {
		return false
	}
	args := words[run.Index+1:]
	src, ok := sourceOf(words[run.Index], args)
	if !ok || src.kind != fromCode {
		return false
	}
	if src.word < 0 {
		return true
	}
	replace := replaceString(words[x], words[x+1:run.Index])
	return replace != "" && strings.Contains(args[src.word], replace)
}

// ArgsFromInput reports whether the program the words run, behind wrappers,
// is given arguments that a wrapper builds from its own standard input
// (`xargs sh`, `parallel python3`, but not `parallel --pipe sh`), so its
// argv is not all on the line and its standard input is not the line's.
func ArgsFromInput(words []string) bool {
	run := Program(words)
	return run.Index > 0 && argInputWrapper(words[:run.Index]) >= 0
}

// InterpreterCode returns the inline programs the interpreter program is
// given in args, the words after it: the value of each of its code options
// (`python3 -c CODE`, `perl -e A -e B`, `perl -necode`, `node --eval=CODE`),
// wherever it appears, which over-reports rather than misses one. It is nil
// when program is not an interpreter (see IsInterpreter); a shell's -c
// script is reported by Script.
func InterpreterCode(program string, args []string) []string {
	opts, ok := interpreters[interpreterName(program)]
	if !ok {
		return nil
	}
	var code []string
	next := func(j int) {
		if j+1 < len(args) {
			code = append(code, args[j+1])
		}
	}
	for j, a := range args {
		switch {
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a, "=")
			if slices.Contains(opts.code, name) {
				if hasValue {
					code = append(code, value)
				} else {
					next(j)
				}
			}
		case len(a) > 1 && a[0] == '-':
			// Both the attached text and the next word: `node -pe CODE`
			// takes no attached value, perl's `-necode` does.
			if rest, ok := opts.clusterCode(a); ok {
				if rest != "" {
					code = append(code, rest)
				}
				next(j)
			}
		}
	}
	return code
}

// clusterCode reports whether the short-option cluster a holds a code
// option before any option that takes the rest of it as its value, and
// returns the text attached after that option (`-necode` gives "code").
func (o interpreterOptions) clusterCode(a string) (string, bool) {
	for k := 1; k < len(a); k++ {
		switch opt := "-" + a[k:k+1]; {
		case slices.Contains(o.code, opt):
			return a[k+1:], true
		case slices.Contains(o.values, opt):
			return "", false
		}
	}
	return "", false
}

// argInputWrapper returns the index among wrappers, the words before a
// program, of the last one that builds the program's arguments from its own
// input (wrapperArgInputs, unless given a wrapperStdinPipes option), or -1.
func argInputWrapper(wrappers []string) int {
	for i := len(wrappers) - 1; i >= 0; i-- {
		name := wrapperName(wrappers[i])
		if !wrapperArgInputs[name] {
			continue
		}
		pipes := slices.ContainsFunc(WrapperOptions(name, wrappers[i+1:]), func(o WrapperOption) bool {
			return o.Is(wrapperStdinPipes[name]...)
		})
		if !pipes {
			return i
		}
	}
	return -1
}

// runsInputLines reports whether the words, which name no program, run
// parallel given no command and no argument source other than its standard
// input, so it runs each input line as a command.
func runsInputLines(words []string) bool {
	for i, w := range words {
		if wrapperName(w) != "parallel" {
			continue
		}
		return !slices.ContainsFunc(WrapperOptions("parallel", words[i+1:]), func(o WrapperOption) bool {
			return o.Is("-a", "--arg-file", "--pipe", "--pipepart")
		})
	}
	return false
}

// replaceString returns the string the argument-building wrapper (see
// argInputWrapper) replaces with its input given args, the words after it:
// parallel's -I or --replace argument, else "{}"; xargs' as
// xargsReplaceString returns it.
func replaceString(wrapper string, args []string) string {
	if wrapperName(wrapper) != "parallel" {
		return xargsReplaceString(args)
	}
	replace := "{}"
	for _, o := range WrapperOptions("parallel", args) {
		if o.Is("-I", "--replace") {
			replace = o.Arg
		}
	}
	return replace
}

// xargsReplaceString returns the string xargs replaces with each input line
// given args, the words after it: -I's argument, or "{}" for -i and
// --replace without one; "" when it appends its input instead.
func xargsReplaceString(args []string) string {
	replace := ""
	for _, o := range WrapperOptions("xargs", args) {
		switch {
		case o.Is("-I"):
			replace = o.Arg
		case o.Name == "-i", o.Is("--replace"):
			replace = "{}"
			if o.Arg != "" {
				replace = o.Arg
			}
		}
	}
	return replace
}

// sourceOf returns where the program name, given args, takes its program
// from, unwrapping busybox to the applet it runs (a word index stays one
// into args); false when it is not a shell or interpreter.
func sourceOf(name string, args []string) (programSource, bool) {
	if wrapperName(name) == "busybox" && len(args) > 0 {
		src, ok := sourceOf(args[0], args[1:])
		if src.kind != fromStdin && src.word >= 0 {
			src.word++
		}
		return src, ok
	}
	switch fold := wrapperName(name); {
	case name != "eval" && SourcesCode(name):
		// `. FILE`, `source FILE`: the shell itself runs FILE, which reads
		// standard input when it names it (`. /dev/stdin`).
		if len(args) > 0 && isStdinPath(args[0]) {
			return programSource{kind: fromStdin}, true
		}
		return programSource{kind: fromScript}, true
	case IsScriptShell(name):
		return shellSource(args), true
	case fold == "su":
		if _, rest, ok := CommandLine(append([]string{name}, args...)); ok {
			return programSource{kind: fromCode, word: rest - 2}, true // the word holding the command
		}
		return programSource{kind: fromStdin}, true
	case fold == "pwsh" || fold == "powershell":
		return powerShellSource(args), true
	}
	opts, ok := interpreters[interpreterName(name)]
	if !ok {
		return programSource{}, false
	}
	return opts.source(args), true
}

// loginShellOptions are the options with which a wrapper given no command
// starts a shell (`sudo -s`, `sudo -i`, `doas -s`), which reads its program
// from standard input.
var loginShellOptions = map[string][]string{
	"sudo": {"-s", "--shell", "-i", "--login"},
	"doas": {"-s"},
}

// startsLoginShell reports whether the words, which run no named program,
// hold a wrapper that then starts a shell (loginShellOptions).
func startsLoginShell(words []string) bool {
	for i, w := range words {
		name := wrapperName(w)
		shellOpts, ok := loginShellOptions[name]
		if ok && slices.ContainsFunc(WrapperOptions(name, words[i+1:]), func(o WrapperOption) bool { return o.Is(shellOpts...) }) {
			return true
		}
	}
	return false
}

// shellSource is sourceOf for the arguments of a script shell: -c runs the
// script operand, -s reads standard input whatever operands follow, and
// otherwise the first operand is the script file, unless it names standard
// input.
func shellSource(args []string) programSource {
	if slices.ContainsFunc(args, isShellCodeOption) {
		k, ok := shellScriptIndex(args)
		if !ok {
			k = -1
		}
		return programSource{kind: fromCode, word: k}
	}
	skip, fromStdinOpt := 0, false
	for i, a := range args {
		switch {
		case skip > 0:
			skip--
		case a == "--" || a == "-":
			if fromStdinOpt || i+1 >= len(args) || isStdinPath(args[i+1]) {
				return programSource{kind: fromStdin}
			}
			return programSource{kind: fromScript, word: i + 1}
		case len(a) > 1 && (a[0] == '-' || a[0] == '+'):
			if a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 's') {
				fromStdinOpt = true
			}
			skip = shellOptionArgs(a)
		case fromStdinOpt || isStdinPath(a):
			return programSource{kind: fromStdin}
		default:
			return programSource{kind: fromScript, word: i}
		}
	}
	return programSource{kind: fromStdin}
}

// isShellCodeOption reports whether a is a short-option cluster holding a
// script shell's -c (see shellScriptIndex).
func isShellCodeOption(a string) bool {
	return len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 'c')
}

// powerShellValueOptions are the PowerShell parameters, other than those
// that give the program, that take the next word as their value.
var powerShellValueOptions = []string{
	"-executionpolicy", "-ep", "-ex", "-workingdirectory", "-wd", "-configurationname", "-config",
	"-outputformat", "-o", "-of", "-inputformat", "-inp", "-if", "-windowstyle", "-w",
	"-settingsfile", "-settings", "-custompipename", "-psconsolefile", "-version", "-v",
	"-configurationfile",
}

// powerShellCodeOptions are the PowerShell parameters that take an encoded
// program as their value.
var powerShellCodeOptions = []string{"-encodedcommand", "-e", "-ec", "-encodedarguments", "-ea"}

// powerShellSource is sourceOf for the arguments of PowerShell: it runs
// standard input given only switches and valued parameters (none, or
// `-NoProfile -NonInteractive`), or "-" as the -Command or -File value or
// as an operand. A -Command, -File or -EncodedCommand value, or an operand
// (pwsh's script, Windows PowerShell's command), gives it its program.
func powerShellSource(args []string) programSource {
	is := func(a string, opts []string) bool {
		return slices.ContainsFunc(opts, func(s string) bool { return strings.EqualFold(s, a) })
	}
	commandOpts := append(slices.Clone(commandLineOptions["pwsh"].words), "-file", "-f")
	for j := 0; j < len(args); j++ {
		a := args[j]
		switch {
		case a == "-":
			return programSource{kind: fromStdin}
		case is(a, commandOpts):
			if j+1 < len(args) && args[j+1] == "-" {
				return programSource{kind: fromStdin}
			}
			return powerShellValue(args, j)
		case is(a, powerShellCodeOptions):
			return powerShellValue(args, j)
		case is(a, powerShellValueOptions):
			j++
		case strings.HasPrefix(a, "-"):
			// A switch (-NoProfile, -NonInteractive, -NoLogo, ...).
		default:
			return programSource{kind: fromScript, word: j}
		}
	}
	return programSource{kind: fromStdin}
}

// powerShellValue is the program given as the value of the PowerShell
// parameter args[j]: the next word, or -1 when there is none.
func powerShellValue(args []string, j int) programSource {
	if j+1 < len(args) {
		return programSource{kind: fromCode, word: j + 1}
	}
	return programSource{kind: fromCode, word: -1}
}

// source is sourceOf for the arguments of an interpreter.
func (o interpreterOptions) source(args []string) programSource {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-":
			return programSource{kind: fromStdin}
		case a == "--":
			if i+1 >= len(args) || isStdinPath(args[i+1]) {
				return programSource{kind: fromStdin}
			}
			return programSource{kind: fromScript, word: i + 1}
		case strings.HasPrefix(a, "--"):
			name, _, hasValue := strings.Cut(a, "=")
			switch {
			case slices.Contains(o.code, name):
				return o.codeAt(args, i+1, hasValue)
			case slices.Contains(o.script, name):
				return programSource{kind: fromScript, word: i}
			case slices.Contains(o.values, name) && !hasValue:
				i++
			}
		case len(a) > 1 && a[0] == '-':
			n, src, decided := o.cluster(args, i)
			if decided {
				return src
			}
			i += n
		case isStdinPath(a):
			return programSource{kind: fromStdin}
		default:
			return programSource{kind: fromScript, word: i}
		}
	}
	return programSource{kind: fromStdin}
}

// codeAt is the inline program of a code option whose value is attached
// (inline) or else the word at index next of args, if there is one.
func (o interpreterOptions) codeAt(args []string, next int, inline bool) programSource {
	switch {
	case inline:
		return programSource{kind: fromCode, word: next - 1}
	case next < len(args):
		return programSource{kind: fromCode, word: next}
	}
	return programSource{kind: fromCode, word: -1}
}

// cluster reads the short-option cluster args[i]. It reports whether an
// option in it gives the program (inline code or a script) and where, and
// otherwise how many following words its last option takes as its value.
func (o interpreterOptions) cluster(args []string, i int) (n int, src programSource, decided bool) {
	a := args[i]
	for k := 1; k < len(a); k++ {
		opt := "-" + a[k:k+1]
		switch {
		case slices.Contains(o.code, opt):
			return 0, o.codeAt(args, i+1, k+1 < len(a)), true
		case slices.Contains(o.script, opt):
			return 0, programSource{kind: fromScript, word: i}, true
		case slices.Contains(o.values, opt) && k+1 < len(a):
			return 0, programSource{}, false // the value is the rest of the cluster
		case slices.Contains(o.values, opt):
			return 1, programSource{}, false
		}
	}
	return 0, programSource{}, false
}

// stdinRedirect reports whether the redirect r replaces standard input, and
// whether what it supplies is computed at run time: a here-string or `<`
// word with an expansion (process substitution included) or naming a
// network pseudo-file (/dev/tcp/...), a here-document whose unquoted
// delimiter lets its body expand, or a duplicated descriptor. A `<` word
// naming standard input itself (`< /dev/stdin`, `<&0`) replaces nothing.
func stdinRedirect(r *syntax.Redirect) (replaces, computed bool) {
	if r.N != nil && r.N.Value != "0" {
		return false, false
	}
	switch r.Op {
	case syntax.RdrIn, syntax.RdrInOut:
		text, exp := wordText(r.Word, nil)
		if !exp && isStdinPath(text) {
			return false, false
		}
		return true, exp || isSocketPath(text)
	case syntax.WordHdoc:
		_, exp := wordText(r.Word, nil)
		return true, exp
	case syntax.Hdoc, syntax.DashHdoc:
		_, exp := wordText(r.Hdoc, nil)
		return true, exp
	case syntax.DplIn:
		if text, exp := wordText(r.Word, nil); !exp && text == "0" {
			return false, false
		}
		return true, true
	}
	return false, false
}

// execRedirect is an `exec` given no command whose redirect replaces the
// standard input of the shell running it, and so of every later statement
// in its scope.
type execRedirect struct {
	stmt *syntax.Stmt
	// scope is the subshell, command or process substitution, or pipeline
	// stage the exec runs in, which its redirect does not outlive; nil for
	// the line's own shell.
	scope    syntax.Node
	computed bool
}

// stdinState computes, for each statement, whether its standard input is
// computed at run time (see Command.StdinComputed).
type stdinState struct {
	prev   map[*syntax.Stmt]*syntax.Stmt // a later pipeline stage's previous stage
	parent map[*syntax.Stmt]*syntax.Stmt // the nearest enclosing statement
	execs  []execRedirect                // in source order
	calls  map[string][]*syntax.Stmt     // the statements calling each function name
	memo   map[*syntax.Stmt]bool
	// root is whether the line's own standard input is computed: what a
	// top-level statement with none of its own inherits (see ParseScript).
	root bool
}

// assignStdin returns the stdinState of the statements under root, given
// the previous stage of each later pipeline stage (see assignPipelines) and
// whether the line's own standard input is computed. It walks the tree once,
// recording each statement's nearest enclosing one, each `exec` that
// redirects standard input and the calls of each name.
func assignStdin(root syntax.Node, prev map[*syntax.Stmt]*syntax.Stmt, rootComputed bool) *stdinState {
	s := &stdinState{prev: prev, parent: map[*syntax.Stmt]*syntax.Stmt{}, calls: map[string][]*syntax.Stmt{}, memo: map[*syntax.Stmt]bool{}, root: rootComputed}
	// enclosing and scopes hold, per node on the path being walked, the
	// nearest statement and the nearest scope (see execRedirect) at or above
	// it; Walk calls f(nil) after a node's children. stages are the
	// statements that are pipeline stages, which run in a subshell.
	var enclosing []*syntax.Stmt
	var scopes []syntax.Node
	stages := map[*syntax.Stmt]bool{}
	syntax.Walk(root, func(n syntax.Node) bool {
		if n == nil {
			enclosing, scopes = enclosing[:len(enclosing)-1], scopes[:len(scopes)-1]
			return true
		}
		var top *syntax.Stmt
		var scope syntax.Node
		if len(enclosing) > 0 {
			top, scope = enclosing[len(enclosing)-1], scopes[len(scopes)-1]
		}
		switch n := n.(type) {
		case *syntax.Stmt:
			if top != nil {
				s.parent[n] = top
			}
			top = n
			if stages[n] {
				scope = n
			}
			s.record(n, scope)
		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe || n.Op == syntax.PipeAll {
				stages[n.X], stages[n.Y] = true, true
			}
		case *syntax.Subshell, *syntax.CmdSubst, *syntax.ProcSubst:
			scope = n
		}
		enclosing, scopes = append(enclosing, top), append(scopes, scope)
		return true
	})
	return s
}

// record notes stmt, run in scope, if it is a function call or an `exec`
// that redirects standard input.
func (s *stdinState) record(stmt *syntax.Stmt, scope syntax.Node) {
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Args) == 0 {
		return
	}
	name, exp := wordText(call.Args[0], nil)
	if exp {
		return
	}
	s.calls[name] = append(s.calls[name], stmt)
	if name != "exec" || len(call.Args) > 1 {
		return
	}
	for _, r := range stmt.Redirs {
		if replaces, c := stdinRedirect(r); replaces {
			s.execs = append(s.execs, execRedirect{stmt: stmt, scope: scope, computed: c})
		}
	}
}

// lastExec returns the last `exec` redirect that ends before stmt and whose
// scope holds it, or nil.
func (s *stdinState) lastExec(stmt *syntax.Stmt) *execRedirect {
	var last *execRedirect
	for i := range s.execs {
		e := &s.execs[i]
		if e.stmt.End().Offset() > stmt.Pos().Offset() {
			continue
		}
		if e.scope == nil || e.scope.Pos().Offset() <= stmt.Pos().Offset() && stmt.End().Offset() <= e.scope.End().Offset() {
			last = e
		}
	}
	return last
}

// computed reports whether the standard input of stmt is computed at run
// time: its own last stdin redirect decides; otherwise a later pipeline stage
// reads what the previous stage writes, which is computed unless that stage is
// a literal producer; otherwise a function body reads what any call of the
// function is given; otherwise an earlier `exec` redirect in the same
// enclosing statement decides; otherwise the statement inherits the
// enclosing statement's standard input.
func (s *stdinState) computed(stmt *syntax.Stmt) bool {
	if v, ok := s.memo[stmt]; ok {
		return v
	}
	// A recursive function's body reaches itself through its own call; that
	// cycle counts as computed, so a recursive function fails closed.
	s.memo[stmt] = true
	v, own := false, false
	for _, r := range stmt.Redirs {
		if replaces, c := stdinRedirect(r); replaces {
			own, v = true, c
		}
	}
	if !own {
		v = s.inherited(stmt)
	}
	s.memo[stmt] = v
	return v
}

// inherited is computed for a statement with no stdin redirect of its own.
func (s *stdinState) inherited(stmt *syntax.Stmt) bool {
	if p, ok := s.prev[stmt]; ok {
		return !s.literalProducer(p)
	}
	if fn, ok := stmt.Cmd.(*syntax.FuncDecl); ok {
		return slices.ContainsFunc(s.calls[fn.Name.Value], s.computed)
	}
	p, hasParent := s.parent[stmt]
	if e := s.lastExec(stmt); e != nil && (!hasParent || e.stmt.Pos().Offset() >= p.Pos().Offset()) {
		return e.computed
	}
	if !hasParent {
		return s.root
	}
	return s.computed(p)
}

// literalProducer reports whether the pipeline stage stmt writes only text
// written out on the line: echo of literal words with no escape sequence,
// printf of literal words with no format directive and no escape but `\n`,
// or cat of literal files (or of a literal standard input).
func (s *stdinState) literalProducer(stmt *syntax.Stmt) bool {
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) == 0 {
		return false
	}
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		text, exp := wordText(w, nil)
		if exp {
			return false
		}
		words[i] = text
	}
	for _, r := range stmt.Redirs {
		if r.Word != nil {
			if text, exp := wordText(r.Word, nil); exp || isSocketPath(text) {
				return false
			}
		}
	}
	switch args := words[1:]; words[0] {
	case "echo":
		return !slices.ContainsFunc(args, func(a string) bool { return strings.Contains(a, `\`) })
	case "printf":
		return !slices.ContainsFunc(args, func(a string) bool { return strings.Contains(a, "%") || hasEscapeOtherThanNewline(a) })
	case "cat":
		operands := slices.DeleteFunc(slices.Clone(args), func(a string) bool { return len(a) > 1 && a[0] == '-' })
		if len(operands) > 0 && !slices.ContainsFunc(operands, func(a string) bool { return a == "-" || isStdinPath(a) || isSocketPath(a) }) {
			return true
		}
		if slices.ContainsFunc(operands, isSocketPath) {
			return false
		}
		return !s.computed(stmt)
	}
	return false
}

// hasEscapeOtherThanNewline reports whether the printf argument a holds a
// backslash escape other than `\n`, which can spell any byte.
func hasEscapeOtherThanNewline(a string) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != '\\' {
			continue
		}
		if i+1 >= len(a) || a[i+1] != 'n' {
			return true
		}
		i++
	}
	return false
}
