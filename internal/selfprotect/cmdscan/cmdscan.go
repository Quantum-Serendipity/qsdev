// Package cmdscan parses shell command lines into their constituent simple
// commands using a POSIX parser, so self-protection rules can evaluate the
// actual command word and its real argument/redirect targets instead of
// substring-matching the raw command string.
//
// Callers MUST fail closed on a parse error: fall back to the conservative
// substring behavior rather than allowing a command that could not be parsed.
package cmdscan

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Command is one simple command extracted from a shell line: its command word
// (after any leading VAR=val assignments), its literal argument words, the
// targets of its redirects split by direction, and whether any word used a
// shell expansion ($VAR, $(...), or $((...))).
//
// Redirects are attributed to the statement that carries them, so a redirect on
// a compound command (e.g. `{ echo x; } > f` or `( echo x ) > f`) is reported
// as a nameless Command (Name == "") whose WriteRedirects holds the target,
// alongside the inner commands. WriteRedirects lists targets that a command can
// create or clobber (`>`, `>>`, `<>`, `>|`, `&>`, ...); ReadRedirects lists
// input sources (`<`, `<&`, here-docs) which are reads, not mutations.
type Command struct {
	Name           string
	Args           []string
	WriteRedirects []string
	ReadRedirects  []string
	// Heredocs holds the literal bodies of the command's here-documents
	// (`<<EOF ... EOF`), which a shell such as `sh <<EOF` executes as a script.
	Heredocs     []string
	HasExpansion bool
	// Assigns names the variables this statement sets for the command or the
	// rest of the shell line: prefix assignments (`GIT_EXTERNAL_DIFF=x git
	// diff`) and bare assignment statements (`PATH=/tmp/x`, emitted as a
	// nameless Command). Either can change what a later command word runs, so
	// a command with assignments is never proven read-only. Declaration
	// builtins (export, declare, local, ...) are reported as ordinary commands
	// named by the builtin.
	Assigns []string
	// Pipeline groups commands joined by `|`/`|&`: all stages of one pipeline
	// share the same non-zero id, in left-to-right order. Standalone commands
	// have Pipeline == 0. Rules use this to reason about dataflow across a pipe
	// (e.g. a protected file read upstream and exfiltrated downstream).
	Pipeline int
}

// safeReadVerbs are commands that only read or inspect their arguments whatever
// those arguments are — they never write a named operand, delete, copy,
// execute a payload, or spawn a subshell. Self-protection consumers (rules,
// evasion) use IsSafeReadCommand to decide whether a parsed command can clear a
// substring-triggered deny: anything it does not prove read-only (a wrapper
// like sudo/env/sh/xargs/find, or an unknown binary) is opaque and must fail
// closed. sed/awk/eval are deliberately absent — they can mutate in place or
// execute — as are go (run/generate execute code) and uniq (its second operand
// is an output file). Commands that are read-only only for some arguments
// (git, sort, rg) are modelled by argReadOnly instead.
var safeReadVerbs = map[string]bool{
	"cat": true, "grep": true, "egrep": true, "fgrep": true,
	"ls": true, "head": true, "tail": true, "wc": true, "echo": true,
	"printf": true, "test": true, "[": true, "true": true, "false": true,
	"pwd": true, "stat": true, "file": true, "diff": true, "cmp": true,
	"cut": true, "jq": true, "tac": true, "nl": true,
}

// argReadOnly holds, for commands whose effect depends on their arguments, a
// predicate reporting whether a given argument list only reads.
var argReadOnly = map[string]func(args []string) bool{
	"git":  gitArgsReadOnly,
	"sort": sortArgsReadOnly,
	"rg":   rgArgsReadOnly,
}

// IsSafeReadVerb reports whether name is a command that is read-only for ANY
// arguments, so the command word alone proves it cannot mutate. Prefer
// IsSafeReadCommand, which also accepts the read-only uses of commands such as
// `git diff` that can write or execute with other arguments.
func IsSafeReadVerb(name string) bool {
	return safeReadVerbs[name]
}

// IsSafeReadCommand reports whether the parsed command c only reads or inspects
// its arguments and can therefore clear a substring-triggered self-protection
// deny: either its command word is read-only for any arguments (see
// safeReadVerbs), or it is an argument-dependent command (git, sort, rg) whose
// actual arguments are read-only — and it sets no variables (see
// Command.Assigns).
func IsSafeReadCommand(c Command) bool {
	if len(c.Assigns) > 0 {
		// A prefix assignment can make a read-only command run code
		// (GIT_EXTERNAL_DIFF, LD_PRELOAD, PATH).
		return false
	}
	if safeReadVerbs[c.Name] {
		return true
	}
	if readOnly, ok := argReadOnly[c.Name]; ok {
		return readOnly(c.Args)
	}
	return false
}

// gitReadOnlySubcommands are git subcommands that only inspect the repository.
// Everything else (rm, mv, checkout, restore, reset, clean, apply, am, stash,
// worktree, filter-branch, config, ...) can mutate the working tree.
var gitReadOnlySubcommands = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true, "grep": true,
	"ls-files": true, "blame": true, "rev-parse": true,
}

// gitArgsReadOnly reports whether a git invocation is a read-only subcommand
// with no option that writes a file or runs a program. The subcommand must be
// the first argument: a global option before it (`-c alias.x=!sh`, `-C dir`,
// `--exec-path`) could redefine what runs, so it fails closed.
func gitArgsReadOnly(args []string) bool {
	if len(args) == 0 || !gitReadOnlySubcommands[args[0]] {
		return false
	}
	for _, a := range args[1:] {
		// --output writes the diff/log to a file; grep -O/--open-files-in-pager
		// runs a program. Long options match any abbreviation git accepts, and
		// a short-option cluster containing 'O' (-nO) is treated as -O.
		if a == "--" {
			break
		}
		if name, ok := strings.CutPrefix(a, "--"); ok {
			name, _, _ = strings.Cut(name, "=")
			if isLongOptionPrefix(name, "output") || isLongOptionPrefix(name, "open-files-in-pager") {
				return false
			}
			continue
		}
		if len(a) > 1 && a[0] == '-' && strings.ContainsRune(a[1:], 'O') {
			return false
		}
	}
	return true
}

// sortArgsReadOnly reports whether a sort invocation writes only to stdout:
// no -o/--output file and no --compress-program to execute. A short-option
// cluster containing 'o' is treated as -o (conservatively), and a long option
// matches any unambiguous prefix, as getopt_long allows.
func sortArgsReadOnly(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
		if name, ok := strings.CutPrefix(a, "--"); ok {
			name, _, _ = strings.Cut(name, "=")
			if isLongOptionPrefix(name, "output") || isLongOptionPrefix(name, "compress-program") {
				return false
			}
			continue
		}
		if len(a) > 1 && a[0] == '-' && strings.ContainsRune(a[1:], 'o') {
			return false
		}
	}
	return true
}

// rgArgsReadOnly reports whether a ripgrep invocation runs no helper program:
// --pre and --hostname-bin execute a command.
func rgArgsReadOnly(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
		name, _, _ := strings.Cut(a, "=")
		if name == "--pre" || name == "--hostname-bin" {
			return false
		}
	}
	return true
}

// isLongOptionPrefix reports whether name (a long option without its leading
// "--") abbreviates option, as getopt_long accepts. The empty name is "--",
// the end-of-options marker, which is not an abbreviation.
func isLongOptionPrefix(name, option string) bool {
	return name != "" && strings.HasPrefix(option, name)
}

// isWriteOp reports whether a redirect operator creates or clobbers its target
// (an output/mutation), as opposed to reading from it. Callers treat only write
// targets as mutation candidates; read redirects (`<`, `<&`, here-docs) are not.
func isWriteOp(op syntax.RedirOperator) bool {
	switch op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrInOut, syntax.DplOut,
		syntax.RdrClob, syntax.AppClob, syntax.RdrAll, syntax.RdrAllClob,
		syntax.AppAll, syntax.AppAllClob:
		return true
	default: // RdrIn, DplIn, Hdoc, DashHdoc, WordHdoc are reads
		return false
	}
}

// Parse parses a shell command line into its simple commands. On a parse error
// it returns (nil, err); callers should treat that as "unparseable" and fall
// back to conservative substring checks (fail closed), never fail open.
func Parse(command string) ([]Command, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	pipelineIDs := assignPipelines(file)

	var cmds []Command
	syntax.Walk(file, func(node syntax.Node) bool {
		stmt, ok := node.(*syntax.Stmt)
		if !ok {
			return true
		}

		var c Command
		c.Pipeline = pipelineIDs[stmt]
		// A simple command contributes its command word and arguments. Compound
		// commands (blocks, subshells, loops) have no CallExpr here — Walk still
		// descends into their inner statements — but redirects on the compound
		// statement itself are attributed below to a nameless Command.
		hasWord := false
		switch cmd := stmt.Cmd.(type) {
		case *syntax.CallExpr:
			for _, a := range cmd.Assigns {
				name, _, exp := assignText(a)
				c.Assigns = append(c.Assigns, name)
				c.HasExpansion = c.HasExpansion || exp
			}
			if len(cmd.Args) > 0 {
				hasWord = true
				name, exp := wordText(cmd.Args[0])
				c.Name = name
				c.HasExpansion = c.HasExpansion || exp
				for _, w := range cmd.Args[1:] {
					t, e := wordText(w)
					c.Args = append(c.Args, t)
					c.HasExpansion = c.HasExpansion || e
				}
			}
		case *syntax.DeclClause:
			// export/declare/local/readonly/typeset/nameref: a builtin that
			// sets (and may export) variables for later commands.
			hasWord = true
			c.Name = cmd.Variant.Value
			for _, a := range cmd.Args {
				_, text, exp := assignText(a)
				c.Args = append(c.Args, text)
				c.HasExpansion = c.HasExpansion || exp
			}
		}

		for _, r := range stmt.Redirs {
			if r.Word == nil {
				continue
			}
			t, e := wordText(r.Word)
			c.HasExpansion = c.HasExpansion || e
			if isWriteOp(r.Op) {
				c.WriteRedirects = append(c.WriteRedirects, t)
			} else {
				c.ReadRedirects = append(c.ReadRedirects, t)
			}
			if r.Hdoc != nil {
				body, _ := wordText(r.Hdoc)
				c.Heredocs = append(c.Heredocs, body)
			}
		}

		// Emit the command when it has a command word, or when it carries
		// assignments or redirects that must be attributed even without one
		// (bare `VAR=val`, compound-command redirects, `VAR=val > f`). Skip pure
		// structural statements.
		if hasWord || len(c.Assigns) > 0 || len(c.WriteRedirects) > 0 || len(c.ReadRedirects) > 0 {
			cmds = append(cmds, c)
		}
		return true
	})
	return cmds, nil
}

// assignPipelines maps each simple-command statement that is a stage of a pipe
// chain to a shared, non-zero pipeline id (in left-to-right order). Statements
// not in any pipeline are absent from the map, so a lookup yields 0. Walk visits
// a pipe's outer BinaryCmd before its nested ones, so the outermost chain claims
// its stages first and nested BinaryCmd nodes are skipped once already assigned.
func assignPipelines(root syntax.Node) map[*syntax.Stmt]int {
	ids := make(map[*syntax.Stmt]int)
	next := 1
	syntax.Walk(root, func(n syntax.Node) bool {
		bin, ok := n.(*syntax.BinaryCmd)
		if !ok || (bin.Op != syntax.Pipe && bin.Op != syntax.PipeAll) {
			return true
		}
		var stmts []*syntax.Stmt
		collectPipeStmts(bin, &stmts)
		if len(stmts) == 0 {
			return true
		}
		if _, done := ids[stmts[0]]; done {
			return true // nested chain already covered by an outer pipeline
		}
		for _, s := range stmts {
			ids[s] = next
		}
		next++
		return true
	})
	return ids
}

// collectPipeStmts flattens a (possibly nested) pipe chain into its ordered leaf
// statements, descending through inner pipe BinaryCmds so `a | b | c` yields
// [a, b, c] regardless of how the parser associated the operators.
func collectPipeStmts(bin *syntax.BinaryCmd, out *[]*syntax.Stmt) {
	appendPipeStmt(bin.X, out)
	appendPipeStmt(bin.Y, out)
}

func appendPipeStmt(s *syntax.Stmt, out *[]*syntax.Stmt) {
	if s == nil {
		return
	}
	if inner, ok := s.Cmd.(*syntax.BinaryCmd); ok && (inner.Op == syntax.Pipe || inner.Op == syntax.PipeAll) {
		collectPipeStmts(inner, out)
		return
	}
	*out = append(*out, s)
}

// assignText renders an assignment: the variable name (empty for a bare
// option word such as the -x in `declare -x`), its text as written
// (`NAME=value`, or just the word), and whether any part used an expansion.
// Array values count as an expansion: their elements are not rendered.
func assignText(a *syntax.Assign) (name, text string, hasExpansion bool) {
	if a.Name != nil {
		name = a.Name.Value
	}
	value, exp := wordText(a.Value)
	switch {
	case a.Index != nil || a.Array != nil:
		exp = true
	case name == "":
		return "", value, exp
	case a.Naked:
		return name, name, exp
	}
	return name, name + "=" + value, exp
}

// wordText renders a shell word to the literal text the shell would pass to the
// command (quotes and escapes removed) and reports whether any part of it was a
// shell expansion. Expansions contribute no literal text but flip the expansion
// flag, so `"$X"` yields ("", true) while the single-quoted literal
// `'eval "$("'` yields (`eval "$("`, false). Backslash escapes are removed the
// way the shell removes them (`.cl\aude` is `.claude`) and ANSI-C `$'..'`
// strings are decoded (`$'\x2e'claude` is `.claude`), so a protected path cannot
// hide behind an escape spelling. Glob and brace characters are left in the
// text for callers to interpret.
func wordText(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	var b strings.Builder
	hasExpansion := false
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(unescapeUnquoted(p.Value))
		case *syntax.SglQuoted:
			if p.Dollar {
				text, ok := decodeANSIC(p.Value)
				b.WriteString(text)
				hasExpansion = hasExpansion || !ok
			} else {
				b.WriteString(p.Value)
			}
		case *syntax.DblQuoted:
			for _, dp := range p.Parts {
				if lit, ok := dp.(*syntax.Lit); ok {
					b.WriteString(unescapeDoubleQuoted(lit.Value))
				} else {
					hasExpansion = true
				}
			}
		case *syntax.ParamExp, *syntax.CmdSubst, *syntax.ArithmExp:
			hasExpansion = true
		default:
			hasExpansion = true
		}
	}
	return b.String(), hasExpansion
}

// unescapeUnquoted performs the shell's quote removal on an unquoted literal: a
// backslash escapes the next character (a backslash-newline is a line
// continuation and disappears).
func unescapeUnquoted(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		if s[i] != '\n' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// unescapeDoubleQuoted performs quote removal inside double quotes, where a
// backslash only escapes `$`, a backtick, `"`, `\` and newline; before any other
// character it is kept literally.
func unescapeDoubleQuoted(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) || !strings.ContainsRune("$`\"\\\n", rune(s[i+1])) {
			b.WriteByte(s[i])
			continue
		}
		i++
		if s[i] != '\n' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// ansiCSimple maps the single-character ANSI-C escapes to their values.
var ansiCSimple = map[byte]string{
	'a': "\a", 'b': "\b", 'e': "\x1b", 'E': "\x1b", 'f': "\f", 'n': "\n",
	'r': "\r", 't': "\t", 'v': "\v", '\\': `\`, '\'': "'", '"': `"`, '?': "?",
}

// decodeANSIC decodes the body of a bash `$'...'` string. It reports ok=false
// when it meets an escape it does not model (such as `\cX`), so the caller can
// treat the word as opaque (fail closed) instead of trusting a wrong rendering.
func decodeANSIC(s string) (string, bool) {
	var b strings.Builder
	ok := true
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		c := s[i]
		if v, simple := ansiCSimple[c]; simple {
			b.WriteString(v)
			continue
		}
		switch {
		case c >= '0' && c <= '7':
			n, width := parseDigits(s[i:], 8, 3)
			b.WriteByte(byte(n))
			i += width - 1
		case c == 'x' || c == 'u' || c == 'U':
			maxDigits := map[byte]int{'x': 2, 'u': 4, 'U': 8}[c]
			n, width := parseDigits(s[i+1:], 16, maxDigits)
			if width == 0 {
				b.WriteByte('\\')
				b.WriteByte(c)
				continue
			}
			if c == 'x' {
				b.WriteByte(byte(n))
			} else {
				b.WriteRune(rune(n))
			}
			i += width
		default:
			// Unknown escapes (and \c control escapes) are not modelled.
			b.WriteByte('\\')
			b.WriteByte(c)
			ok = false
		}
	}
	return b.String(), ok
}

// parseDigits parses up to maxDigits leading digits of s in the given base
// (8 or 16) and returns the value and how many bytes it consumed.
func parseDigits(s string, base, maxDigits int) (int, int) {
	n, width := 0, 0
	for width < maxDigits && width < len(s) {
		d := digitValue(s[width])
		if d < 0 || d >= base {
			break
		}
		n = n*base + d
		width++
	}
	return n, width
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
