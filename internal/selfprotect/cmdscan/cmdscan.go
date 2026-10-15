// Package cmdscan parses shell command lines into their constituent simple
// commands using a POSIX parser, so self-protection rules can evaluate the
// actual command word and its real argument/redirect targets instead of
// substring-matching the raw command string.
//
// Callers MUST fail closed on a parse error: fall back to the conservative
// substring behavior rather than allowing a command that could not be parsed.
package cmdscan

import (
	"path"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ShellTools are the Claude Code tools whose tool_input.command runs in a
// shell: Bash, PowerShell (the primary shell on Windows, where Bash may not
// exist) and Monitor (a background command run in the same shell environment
// as Bash). Every hook and rule that inspects shell commands must cover all of
// them, or the others bypass it.
var ShellTools = []string{"Bash", "PowerShell", "Monitor"}

// IsShellTool reports whether toolName runs its tool_input.command in a shell.
func IsShellTool(toolName string) bool {
	return slices.Contains(ShellTools, toolName)
}

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
	// NameHasExpansion is set when the command word itself is built from an
	// expansion, so Name is not the program the shell runs.
	NameHasExpansion bool
	// ExpandedArgs holds the text of each argument word built from an
	// expansion, as rendered in Args: an expansion of unknown value renders
	// as nothing, so `"$(echo ~)/.config"` is "/.config" here.
	ExpandedArgs []string
	// TildeWords holds the indexes of the command's words (0 is Name, i is
	// Args[i-1]) that start with an unquoted, unescaped ~, which the shell
	// expands to a home directory. A quoted or escaped ~ stays literal.
	TildeWords []int
	// Assigns names the variables this statement sets or clears for the
	// command or the rest of the shell line: prefix assignments
	// (`GIT_EXTERNAL_DIFF=x git diff`) and bare assignment statements
	// (`PATH=/tmp/x`, emitted as a nameless Command), the names a declaration
	// builtin is given (export, declare, local, readonly, typeset; for a
	// nameref also the variable it refers to), a for or select loop variable,
	// the variables read, mapfile/readarray, printf -v, getopts and wait -p
	// store into, unset's operands, arithmetic assignments (`((X=1))`, let,
	// `$((X=1))`), ${X:=value} defaults, a coprocess name and a {X}> redirect.
	// Any of them can change what a later command word runs, so a command
	// with assignments is never proven read-only. Declaration builtins are
	// still reported as ordinary commands named by the builtin.
	Assigns []string
	// AssignsDynamic is set when the statement sets or clears a variable
	// whose name comes from an expansion (`export $X=v`, `read "$V"`,
	// `(( $X = 1 ))`), which Assigns cannot name: it may be any variable.
	AssignsDynamic bool
	// AssignTexts holds the assignments of the statement as rendered
	// (`NAME=value`, see wordText): the prefix or bare assignments of a
	// simple command and the operands of a declaration builtin, so a value
	// whose text a later `$NAME` carries can be judged:
	// `d=$(git rev-parse --git-dir)` and `export d=$(git rev-parse
	// --git-dir)` are "d=.git".
	AssignTexts []string
	// OutputRendered is set when the statement is the whole of a command
	// substitution whose output the scan renders into the word holding it
	// (see substOutput): `git rev-parse --git-dir` in
	// `rm "$(git rev-parse --git-dir)/x"`, whose word reads ".git/x". A
	// rule keying on what a command prints can read it there instead.
	OutputRendered bool
	// Pipeline groups commands joined by `|`/`|&`: all stages of one pipeline
	// share the same non-zero id, in left-to-right order. Standalone commands
	// have Pipeline == 0. Rules use this to reason about dataflow across a pipe
	// (e.g. a protected file read upstream and exfiltrated downstream).
	Pipeline int
	// Guard says whether the statement runs whenever the shell reaches it
	// or only depending on how other commands exit.
	Guard Guard
	// Tested is set when whether a guarded statement runs depends on the
	// exit status of a command that tests something (see StatusTests): the
	// left operand of an && or || list, or an if, while or until condition,
	// holding a program, whose status is unknown, a status test such as
	// `test` or `command -v`, or a test clause. A statement guarded only by
	// builtins taken to succeed, such as cd, is not Tested, so in
	// `cd "$dir" && prog` prog runs whenever the shell reaches it.
	Tested bool
	// Subshell is set when the statement runs in a subshell of the line's
	// shell: inside ( ), a command or process substitution, a stage of a
	// pipeline of more than one stage, or a background or coprocess
	// statement. An exit there, or a variable it sets, does not reach the
	// rest of the line.
	Subshell bool
	// FailureHandled is set when the line tests the statement's exit
	// status: it decides the status of the left operand of ||, of an if,
	// while or until condition, or of a negated pipeline. Its failing,
	// a program the shell cannot find (status 127) included, then only
	// picks what runs next (`prog --version || exit 0`, `prog || true`)
	// rather than becoming the status the line exits with.
	FailureHandled bool
	// StdinComputed is set when the statement's standard input carries
	// text computed at run time rather than written out on the line: a
	// here-string, a `<` word or an unquoted-delimiter here-document with an
	// expansion (process substitution included), a duplicated descriptor,
	// or, for a later pipeline stage, the output of a previous stage that is
	// not a literal producer (echo or printf of literal words without
	// escapes or directives, cat of literal files). The statement's own last
	// stdin redirect decides when it has one. A statement with no stdin of
	// its own inherits that of its enclosing one, so `... | { sh; }`
	// counts. ReadsProgramFromStdin says whether the command runs that text.
	StdinComputed bool
	// ScriptComputed is set when the command runs a shell, an interpreter
	// or `.`/`source`, directly or through wrappers, whose script operand
	// is a process substitution `<(...)` running anything but a single
	// literal producer (see StdinComputed): `bash <(curl …)`, `. <(curl
	// …)`, `source -- <(curl …)`, or a script shell's startup file given
	// so (`bash --init-file <(curl …) -i`). The program is then computed
	// at run time, as on a computed standard input; `. <(cat ./env.sh)`
	// stays literal.
	ScriptComputed bool
	// Defines names the shell function the statement defines
	// (`f() { ...; }`), emitted as a nameless Command before its body. A
	// later call of that name runs the function, not a program.
	Defines string
}

// Guard says whether a statement runs whenever the shell reaches it, or only
// depending on how other commands exit. Guards are ordered: a statement
// nested in a guarded one is guarded at least as much.
type Guard uint8

const (
	// Unguarded statements run whenever the shell reaches them.
	Unguarded Guard = iota
	// GuardedByAnd statements run only when the simple commands or pipelines
	// before them in && lists succeed (`cd "$dir" && prog`).
	GuardedByAnd
	// Guarded statements run only depending on anything else: a command
	// failing (`a || b`, `! a && b`), a test or arithmetic clause, a branch,
	// a loop or a function call.
	Guarded
)

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
	"cut": true, "jq": true, "tac": true, "nl": true, "printenv": true,
	"du": true,
}

// argReadOnly holds, for commands whose effect depends on their arguments, a
// predicate reporting whether a given argument list only reads.
var argReadOnly = map[string]func(args []string) bool{
	"git":    gitArgsReadOnly,
	"sort":   sortArgsReadOnly,
	"rg":     rgArgsReadOnly,
	"direnv": direnvArgsReadOnly,
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
// Command.Assigns and Command.AssignsDynamic).
func IsSafeReadCommand(c Command) bool {
	if len(c.Assigns) > 0 || c.AssignsDynamic {
		// An assignment can make a read-only command run code
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

// gitReadOnlySubcommands are git subcommands that only inspect the repository,
// whatever their arguments (GitReadOnly still vetoes an option that writes a
// file or runs a program). branch, tag, remote, stash, reflog, worktree,
// notes and symbolic-ref read only in some forms (gitListForms, GitReadOnly);
// everything else (rm, mv, checkout, restore, reset, clean, apply, am,
// filter-branch, config, ...) can mutate the repository or the working tree.
// ls-remote and `remote show` are left out: they contact a remote, which may
// run a transport program (--upload-pack, an ext:: URL).
var gitReadOnlySubcommands = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true, "grep": true,
	"ls-files": true, "blame": true, "rev-parse": true, "ls-tree": true,
	"cat-file": true, "describe": true, "shortlog": true, "rev-list": true,
	"for-each-ref": true, "merge-base": true, "name-rev": true, "show-ref": true,
	"count-objects": true, "var": true, "check-ignore": true, "whatchanged": true,
	"range-diff": true,
}

// gitReadSubcommands are, for a git subcommand that dispatches to
// subcommands of its own, those that only read. reflog and notes run their
// listing subcommand when given no argument at all (`git reflog`, `git
// notes`); any other first word (an option, a revision, a writing
// subcommand such as `reflog expire` or `worktree add`) is not proven a read.
var gitReadSubcommands = map[string]map[string]bool{
	"reflog":   {"show": true, "list": true, "exists": true},
	"worktree": {"list": true},
	"notes":    {"list": true, "show": true},
}

// gitListForm describes the list form of a git subcommand that otherwise
// creates, deletes or changes refs (branch, tag): the arguments it accepts
// while it only lists.
type gitListForm struct {
	// flags are the listing options that take no value, written whole.
	flags map[string]bool
	// shortFlags are the letters of the value-less short listing options a
	// cluster (-av) may combine.
	shortFlags string
	// valued are the listing options that take a value, attached with '=' or
	// as the next word.
	valued map[string]bool
	// lines reports that -n<num> (tag: print annotation lines) is accepted.
	lines bool
}

// gitListOptions are the filter and format options branch and tag share in
// their list form; each takes a value.
var gitListOptions = map[string]bool{
	"--sort": true, "--format": true, "--contains": true, "--no-contains": true,
	"--merged": true, "--no-merged": true, "--points-at": true,
}

// gitListForms are the list forms of branch and tag. -v is a listing option
// of branch but tag's --verify, which runs gpg, so tag does not accept it.
var gitListForms = map[string]gitListForm{
	"branch": {
		flags: map[string]bool{
			"--all": true, "--remotes": true, "--list": true, "--verbose": true,
			"--show-current": true, "--column": true, "--no-column": true,
			"--color": true, "--no-color": true, "--abbrev": true, "--ignore-case": true,
		},
		shortFlags: "arlvi",
		valued:     gitListOptions,
	},
	"tag": {
		flags:      map[string]bool{"--list": true, "--column": true, "--no-column": true, "--ignore-case": true},
		shortFlags: "li",
		valued:     gitListOptions,
		lines:      true,
	},
}

// GitReadOnly reports whether `git <sub> <args>` only reads the repository,
// judged by subcommand and form: an always-read-only subcommand
// (gitReadOnlySubcommands), branch or tag in their list form
// (gitListForm.readOnly), remote listing or printing a URL (`remote`,
// `remote -v`, `remote get-url o`), `stash list|show`, a read subcommand of
// reflog, worktree or notes (gitReadSubcommands), or symbolic-ref printing
// one ref (symbolicRefReads). Whatever the form,
// an option that writes a file (--output) or runs a program (grep -O,
// --open-files-in-pager) makes it a write (gitWritingOption). Anything not
// proven read-only, including an unknown option, is a write.
func GitReadOnly(sub string, args []string) bool {
	if slices.ContainsFunc(argsBeforeDashDash(args), gitWritingOption) {
		return false
	}
	if form, ok := gitListForms[sub]; ok {
		return form.readOnly(args)
	}
	switch sub {
	case "remote":
		if len(args) == 0 {
			return true
		}
		return args[0] == "get-url" || len(args) == 1 && (args[0] == "-v" || args[0] == "--verbose")
	case "stash":
		return len(args) > 0 && (args[0] == "list" || args[0] == "show")
	case "symbolic-ref":
		return symbolicRefReads(args)
	}
	if reads, ok := gitReadSubcommands[sub]; ok {
		if len(args) == 0 {
			return sub != "worktree"
		}
		return reads[args[0]]
	}
	return gitReadOnlySubcommands[sub]
}

// symbolicRefReads reports whether `git symbolic-ref <args>` only prints
// what a symbolic ref points to: one ref name and only the printing options
// (-q, --short, --[no-]recurse). A second operand sets the ref, and -d
// deletes it.
func symbolicRefReads(args []string) bool {
	operands := 0
	for _, a := range args {
		switch a {
		case "-q", "--quiet", "--short", "--recurse", "--no-recurse":
		default:
			if strings.HasPrefix(a, "-") {
				return false
			}
			operands++
		}
	}
	return operands == 1
}

// readOnly reports whether args keep the subcommand in its list form: every
// argument is a listing option of form (or the value of one), and a
// positional (a pattern) appears only when -l/--list selects the list mode.
// A positional without it names a ref to create, and `--` ends the options,
// so whatever follows it is a positional.
func (form gitListForm) readOnly(args []string) bool {
	list, positional := false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			positional = positional || i+1 < len(args)
			i = len(args)
		case a == "-l" || a == "--list":
			list = true
		case !strings.HasPrefix(a, "-") || a == "-":
			positional = true
		case form.valued[a]:
			i++ // the value is the next word, if any (--contains defaults to HEAD)
		case strings.HasPrefix(a, "--"):
			name, _, _ := strings.Cut(a, "=")
			if !form.flags[name] && !form.valued[name] {
				return false
			}
		case form.lines && strings.HasPrefix(a, "-n") && strings.Trim(a[2:], "0123456789") == "":
		default:
			cluster := a[1:]
			if strings.Trim(cluster, form.shortFlags) != "" {
				return false
			}
			list = list || strings.ContainsRune(cluster, 'l')
		}
	}
	return list || !positional
}

// argsBeforeDashDash returns the arguments before a `--`, which are the only
// ones git reads as options.
func argsBeforeDashDash(args []string) []string {
	if i := slices.Index(args, "--"); i >= 0 {
		return args[:i]
	}
	return args
}

// gitWritingOption reports whether a git argument writes a file or runs a
// program: --output writes the diff/log to a file, and grep's
// -O/--open-files-in-pager runs a program. Long options match any
// abbreviation git accepts, and a short-option cluster containing 'O' (-nO)
// is treated as -O.
func gitWritingOption(a string) bool {
	if name, ok := strings.CutPrefix(a, "--"); ok {
		name, _, _ = strings.Cut(name, "=")
		return isLongOptionPrefix(name, "output") || isLongOptionPrefix(name, "open-files-in-pager")
	}
	return len(a) > 1 && a[0] == '-' && strings.ContainsRune(a[1:], 'O')
}

// gitArgsReadOnly reports whether a git invocation is a read-only subcommand
// form (GitReadOnly). The subcommand must be the first argument, after any
// --no-pager or -P, which only stop git paging its output: any other global
// option before it (`-c alias.x=!sh`, `-C dir`, `--exec-path`) could redefine
// what runs, so it fails closed.
func gitArgsReadOnly(args []string) bool {
	for len(args) > 0 && (args[0] == "--no-pager" || args[0] == "-P") {
		args = args[1:]
	}
	return len(args) > 0 && GitReadOnly(args[0], args[1:])
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

// direnvTrustSubcommands are the direnv subcommands that leave the .envrc
// they name untouched: allow and deny (with their aliases) only record
// whether direnv may load it, in direnv's own data directory, and status,
// version and help only print. edit opens an editor on the file, exec and
// the hook/export family run it, and fetchurl downloads: not read-only.
var direnvTrustSubcommands = map[string]bool{
	"allow": true, "permit": true, "grant": true,
	"deny": true, "block": true, "revoke": true,
	"status": true, "version": true, "help": true,
}

// direnvArgsReadOnly reports whether a direnv invocation is a subcommand that
// does not change the file it names (see direnvTrustSubcommands). The
// subcommand must be the first argument.
func direnvArgsReadOnly(args []string) bool {
	return len(args) > 0 && direnvTrustSubcommands[args[0]]
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
	return ParseWithVars(command, nil)
}

// ParseWithVars is Parse with the values of known variables: a plain `$NAME`
// or `${NAME}` whose name is in vars renders as its value, so `$HOME/.x`
// yields the path it names. The word still counts as an expansion
// (Command.HasExpansion), since the line may reassign the variable.
func ParseWithVars(command string, vars map[string]string) ([]Command, error) {
	return parse(command, vars, false)
}

// ParseScript is Parse for a script that a command of an enclosing line
// runs (`sh -c 'S'`, `eval 'S'`, see Script), given whether that command's
// standard input is computed at run time (Command.StdinComputed). The
// script's statements that have no standard input of their own inherit it,
// so the `sh` of `curl … | sh -c sh` reads computed input.
func ParseScript(script string, stdinComputed bool) ([]Command, error) {
	return parse(script, nil, stdinComputed)
}

// parse is ParseWithVars given whether the line's own standard input is
// computed at run time (see ParseScript).
func parse(command string, vars map[string]string, stdinComputed bool) ([]Command, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	pipelineIDs, prevStages := assignPipelines(file)
	stdin := assignStdin(file, prevStages, stdinComputed)
	guards := assignGuards(file, vars)
	subshells := assignSubshells(file, pipelineIDs)
	handled := assignFailureHandled(file)
	rendered := renderedStmts(file)

	var cmds []Command
	syntax.Walk(file, func(node syntax.Node) bool {
		stmt, ok := node.(*syntax.Stmt)
		if !ok {
			return true
		}

		var c Command
		// A simple command contributes its command word and arguments. Compound
		// commands (blocks, subshells, loops) have no CallExpr here — Walk still
		// descends into their inner statements — but redirects on the compound
		// statement itself are attributed below to a nameless Command.
		hasWord := false
		switch cmd := stmt.Cmd.(type) {
		case *syntax.CallExpr:
			c = callCommand(cmd, vars)
			hasWord = len(cmd.Args) > 0
			c.ScriptComputed = hasWord && stdin.scriptComputed(cmd, append([]string{c.Name}, c.Args...))
		case *syntax.DeclClause:
			// export/declare/local/readonly/typeset/nameref: a builtin that
			// sets (and may export) variables for later commands.
			hasWord = true
			c.Name = cmd.Variant.Value
			for _, a := range cmd.Args {
				name, text, exp := assignText(a, vars)
				c.Args = append(c.Args, text)
				c.HasExpansion = c.HasExpansion || exp
				if name != "" {
					c.AssignTexts = append(c.AssignTexts, text)
				}
			}
		case *syntax.FuncDecl:
			hasWord = true
			c.Defines = cmd.Name.Value
		}
		c.Pipeline = pipelineIDs[stmt]
		c.Guard = guards[stmt].guard
		c.Tested = guards[stmt].tested
		c.Subshell = subshells[stmt]
		c.FailureHandled = handled[stmt]
		c.StdinComputed = stdin.computed(stmt)
		c.OutputRendered = rendered[stmt]

		other := stmtAssigns(stmt, vars)
		c.Assigns = append(c.Assigns, other.names...)
		c.AssignsDynamic = other.dynamic

		for _, r := range stmt.Redirs {
			if r.Word == nil {
				continue
			}
			t, e := wordText(r.Word, vars)
			c.HasExpansion = c.HasExpansion || e
			if isWriteOp(r.Op) {
				c.WriteRedirects = append(c.WriteRedirects, t)
			} else {
				c.ReadRedirects = append(c.ReadRedirects, t)
			}
			if r.Hdoc != nil {
				body, _ := wordText(r.Hdoc, vars)
				c.Heredocs = append(c.Heredocs, body)
			}
		}

		// Emit the command when it has a command word, or when it carries
		// assignments or redirects that must be attributed even without one
		// (bare `VAR=val`, compound-command redirects, `VAR=val > f`). Skip pure
		// structural statements.
		if hasWord || len(c.Assigns) > 0 || c.AssignsDynamic || len(c.WriteRedirects) > 0 || len(c.ReadRedirects) > 0 {
			cmds = append(cmds, c)
		}
		return true
	})
	return cmds, nil
}

// renderedStmts returns the statements whose output a Command field holds:
// the single statement of each command substitution substOutput knows the
// output of that wordText writes into a word Parse renders, a command word
// or argument, an assignment value or a redirect word (see
// Command.OutputRendered). A substitution anywhere else, in the operand of
// a parameter expansion (`${D:-$(…)}`), a for-loop word list or an array
// element, reaches no rendered word, so its output stays unrendered.
func renderedStmts(root syntax.Node) map[*syntax.Stmt]bool {
	rendered := make(map[*syntax.Stmt]bool)
	mark := func(w *syntax.Word) {
		if w == nil {
			return
		}
		for _, part := range w.Parts {
			parts := []syntax.WordPart{part}
			if dq, ok := part.(*syntax.DblQuoted); ok {
				parts = dq.Parts
			}
			for _, p := range parts {
				if cs, ok := p.(*syntax.CmdSubst); ok && substOutput(cs) != "" {
					rendered[cs.Stmts[0]] = true
				}
			}
		}
	}
	markAssigns := func(assigns []*syntax.Assign) {
		for _, a := range assigns {
			if a.Index == nil && a.Array == nil {
				mark(a.Value)
			}
		}
	}
	syntax.Walk(root, func(n syntax.Node) bool {
		stmt, ok := n.(*syntax.Stmt)
		if !ok {
			return true
		}
		switch cmd := stmt.Cmd.(type) {
		case *syntax.CallExpr:
			markAssigns(cmd.Assigns)
			for _, w := range cmd.Args {
				mark(w)
			}
		case *syntax.DeclClause:
			markAssigns(cmd.Args)
		}
		for _, r := range stmt.Redirs {
			mark(r.Word)
			mark(r.Hdoc)
		}
		return true
	})
	return rendered
}

// callCommand renders the simple command call: its prefix assignments,
// command word and arguments (see Command).
func callCommand(call *syntax.CallExpr, vars map[string]string) Command {
	var c Command
	for _, a := range call.Assigns {
		name, text, exp := assignText(a, vars)
		c.Assigns = append(c.Assigns, name)
		c.AssignTexts = append(c.AssignTexts, text)
		c.HasExpansion = c.HasExpansion || exp
	}
	if len(call.Args) == 0 {
		return c
	}
	name, exp := wordText(call.Args[0], vars)
	c.Name = name
	c.NameHasExpansion = exp
	c.HasExpansion = c.HasExpansion || exp
	for i, w := range call.Args {
		if leadingTilde(w) {
			c.TildeWords = append(c.TildeWords, i)
		}
	}
	for _, w := range call.Args[1:] {
		t, e := wordText(w, vars)
		c.Args = append(c.Args, t)
		c.HasExpansion = c.HasExpansion || e
		if e {
			c.ExpandedArgs = append(c.ExpandedArgs, t)
		}
	}
	return c
}

// assignPipelines maps each simple-command statement that is a stage of a pipe
// chain to a shared, non-zero pipeline id (in left-to-right order), and each
// stage after the first to the stage before it, whose output it reads.
// Statements not in any pipeline are absent from the maps, so an id lookup
// yields 0. Walk visits a pipe's outer BinaryCmd before its nested ones, so
// the outermost chain claims its stages first and nested BinaryCmd nodes are
// skipped once already assigned.
func assignPipelines(root syntax.Node) (map[*syntax.Stmt]int, map[*syntax.Stmt]*syntax.Stmt) {
	ids := make(map[*syntax.Stmt]int)
	prev := make(map[*syntax.Stmt]*syntax.Stmt)
	// nested holds the pipe BinaryCmds inside a chain already flattened, so
	// a long chain is flattened once rather than once per operator.
	nested := make(map[*syntax.BinaryCmd]bool)
	next := 1
	syntax.Walk(root, func(n syntax.Node) bool {
		bin, ok := n.(*syntax.BinaryCmd)
		if !ok || (bin.Op != syntax.Pipe && bin.Op != syntax.PipeAll) || nested[bin] {
			return true
		}
		var stmts []*syntax.Stmt
		collectPipeStmts(bin, &stmts, nested)
		if len(stmts) == 0 {
			return true
		}
		for i, s := range stmts {
			ids[s] = next
			if i > 0 {
				prev[s] = stmts[i-1]
			}
		}
		next++
		return true
	})
	return ids, prev
}

// stmtGuard is how a statement's running depends on other commands: its
// Guard, and whether a command that tests something decides it (see
// Command.Tested).
type stmtGuard struct {
	guard  Guard
	tested bool
}

// assignGuards maps each guarded statement to its stmtGuard; a lookup of any
// other yields an Unguarded, untested one. Walk visits a statement before the
// statements nested in it, so each passes its own guard down, raised by &&
// or || for the right operand and to Guarded inside any compound command
// other than a block, a subshell or time. The right operand of && or ||, and
// the bodies of if, while and until, are also Tested when their left operand
// or condition tests something (see statusTests).
func assignGuards(root syntax.Node, vars map[string]string) map[*syntax.Stmt]stmtGuard {
	guards := make(map[*syntax.Stmt]stmtGuard)
	simple := simpleSuccess{}
	tests := statusTests{vars: vars, memo: map[*syntax.Stmt]bool{}}
	raise := func(s *syntax.Stmt, g Guard, tested bool) {
		if s == nil {
			return
		}
		cur := guards[s]
		guards[s] = stmtGuard{guard: max(cur.guard, g), tested: cur.tested || tested}
	}
	// raiseChildren raises the statements nearest under node, nested in
	// stmt, to g; each passes it on when the walk below reaches it, so every
	// node is visited a bounded number of times however deep the nesting.
	raiseChildren := func(stmt *syntax.Stmt, node syntax.Node, g Guard, tested bool) {
		syntax.Walk(node, func(n syntax.Node) bool {
			if s, ok := n.(*syntax.Stmt); ok && s != stmt {
				raise(s, g, tested)
				return false
			}
			return true
		})
	}
	raiseAll := func(stmts []*syntax.Stmt, g Guard, tested bool) {
		for _, s := range stmts {
			raise(s, g, tested)
		}
	}
	syntax.Walk(root, func(n syntax.Node) bool {
		stmt, ok := n.(*syntax.Stmt)
		if !ok {
			return true
		}
		g := guards[stmt]
		raiseChildren(stmt, stmt, g.guard, g.tested) // command substitutions run with the statement
		switch cmd := stmt.Cmd.(type) {
		case nil, *syntax.CallExpr, *syntax.DeclClause, *syntax.Block, *syntax.Subshell, *syntax.TimeClause:
		case *syntax.BinaryCmd:
			tested := g.tested || tests.of(cmd.X)
			switch {
			case cmd.Op == syntax.OrStmt, cmd.Op == syntax.AndStmt && !simple.of(cmd.X):
				raise(cmd.Y, Guarded, tested)
			case cmd.Op == syntax.AndStmt:
				raise(cmd.Y, GuardedByAnd, tested)
			}
		case *syntax.IfClause:
			// Each branch is decided by its condition and those before it.
			tested := g.tested
			for c := cmd; c != nil; c = c.Else {
				raiseAll(c.Cond, Guarded, tested)
				tested = tested || tests.any(c.Cond)
				raiseAll(c.Then, Guarded, tested)
			}
		case *syntax.WhileClause:
			raiseAll(cmd.Cond, Guarded, g.tested)
			raiseAll(cmd.Do, Guarded, g.tested || tests.any(cmd.Cond))
		default:
			raiseChildren(stmt, cmd, Guarded, g.tested)
		}
		return true
	})
	return guards
}

// simpleSuccess memoizes whether a statement, the left operand of &&,
// succeeds exactly when its simple commands do: it is a simple command, a
// pipeline or an && list of those, and not negated. The memo keeps a long
// && chain, which nests to the left, linear.
type simpleSuccess map[*syntax.Stmt]bool

func (m simpleSuccess) of(s *syntax.Stmt) bool {
	if v, ok := m[s]; ok {
		return v
	}
	v := false
	switch cmd := s.Cmd.(type) {
	case *syntax.CallExpr, *syntax.DeclClause:
		v = true
	case *syntax.BinaryCmd:
		v = cmd.Op != syntax.OrStmt && m.of(cmd.X) && m.of(cmd.Y)
	}
	v = v && !s.Negated
	m[s] = v
	return v
}

// collectPipeStmts flattens a (possibly nested) pipe chain into its ordered leaf
// statements, descending through inner pipe BinaryCmds so `a | b | c` yields
// [a, b, c] regardless of how the parser associated the operators. Each
// inner pipe BinaryCmd it descends through is recorded in nested, when it
// is not nil.
func collectPipeStmts(bin *syntax.BinaryCmd, out *[]*syntax.Stmt, nested map[*syntax.BinaryCmd]bool) {
	appendPipeStmt(bin.X, out, nested)
	appendPipeStmt(bin.Y, out, nested)
}

func appendPipeStmt(s *syntax.Stmt, out *[]*syntax.Stmt, nested map[*syntax.BinaryCmd]bool) {
	if s == nil {
		return
	}
	if inner, ok := s.Cmd.(*syntax.BinaryCmd); ok && (inner.Op == syntax.Pipe || inner.Op == syntax.PipeAll) {
		if nested != nil {
			nested[inner] = true
		}
		collectPipeStmts(inner, out, nested)
		return
	}
	*out = append(*out, s)
}

// assignText renders an assignment: the variable name (empty for a bare
// option word such as the -x in `declare -x`), its text as written
// (`NAME=value`, or just the word), and whether any part used an expansion.
// Array values count as an expansion: their elements are not rendered.
func assignText(a *syntax.Assign, vars map[string]string) (name, text string, hasExpansion bool) {
	if a.Name != nil {
		name = a.Name.Value
	}
	value, exp := wordText(a.Value, vars)
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
// text for callers to interpret. A plain parameter named in vars renders as
// its value (see ParseWithVars).
func wordText(w *syntax.Word, vars map[string]string) (string, bool) {
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
				switch dp := dp.(type) {
				case *syntax.Lit:
					b.WriteString(unescapeDoubleQuoted(dp.Value))
				case *syntax.ParamExp:
					b.WriteString(knownParam(dp, vars))
					hasExpansion = true
				case *syntax.CmdSubst:
					b.WriteString(substOutput(dp))
					hasExpansion = true
				default:
					hasExpansion = true
				}
			}
		case *syntax.ParamExp:
			b.WriteString(knownParam(p, vars))
			hasExpansion = true
		case *syntax.CmdSubst:
			b.WriteString(substOutput(p))
			hasExpansion = true
		case *syntax.ArithmExp:
			hasExpansion = true
		default:
			hasExpansion = true
		}
	}
	return b.String(), hasExpansion
}

// leadingTilde reports whether w starts with an unquoted, unescaped ~, which
// the shell expands.
func leadingTilde(w *syntax.Word) bool {
	if len(w.Parts) == 0 {
		return false
	}
	lit, ok := w.Parts[0].(*syntax.Lit)
	return ok && strings.HasPrefix(lit.Value, "~")
}

// substOutput returns what the command substitution cs prints when its
// output is fixed by its words, all unquoted literals in a single statement
// with no assignment and no redirect but one discarding another stream than
// the output (`2>/dev/null`), see knownOutput. It returns "" for any other
// substitution, whose output is unknown.
func substOutput(cs *syntax.CmdSubst) string {
	if len(cs.Stmts) != 1 || slices.ContainsFunc(cs.Stmts[0].Redirs, func(r *syntax.Redirect) bool { return !discardsOtherStream(r) }) {
		return ""
	}
	call, ok := cs.Stmts[0].Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) == 0 {
		return ""
	}
	words := make([]string, 0, len(call.Args))
	for _, w := range call.Args {
		if len(w.Parts) != 1 {
			return ""
		}
		lit, ok := w.Parts[0].(*syntax.Lit)
		if !ok {
			return ""
		}
		words = append(words, unescapeUnquoted(lit.Value))
	}
	return knownOutput(words)
}

// discardsOtherStream reports whether the redirect r sends a numbered stream
// other than standard output to /dev/null (`2>/dev/null`), which leaves what
// the command prints unchanged.
func discardsOtherStream(r *syntax.Redirect) bool {
	if r.N == nil || r.N.Value == "1" || r.Word == nil || len(r.Word.Parts) != 1 {
		return false
	}
	if r.Op != syntax.RdrOut && r.Op != syntax.AppOut && r.Op != syntax.RdrClob {
		return false
	}
	lit, ok := r.Word.Parts[0].(*syntax.Lit)
	return ok && lit.Value == "/dev/null"
}

// knownOutput returns what the command words print when that is fixed by
// the words alone, with any path as written (a leading ~ is left for the
// caller to expand, as the shell would have):
//
//   - echo of words without an option prints them;
//   - `git [-C dir]... rev-parse` with exactly one of --git-dir,
//     --absolute-git-dir or --git-common-dir prints the git directory,
//     rendered [dir/].git, and with `--git-path X` prints [dir/].git/X;
//     -q, --quiet and --path-format= change neither which path it prints
//     nor, for these checks, how (a relative path is resolved where the
//     command runs);
//   - `command -v NAME`, `which NAME` and `type -P|-p NAME`, with or without
//     a "--" before NAME, print the program NAME runs, rendered NAME.
//
// It returns "" for anything else.
func knownOutput(words []string) string {
	switch words[0] {
	case "echo":
		if len(words) == 1 || strings.HasPrefix(words[1], "-") {
			return "" // an option changes the output
		}
		return strings.Join(words[1:], " ")
	case "git":
		return gitDirOutput(words[1:])
	}
	return lookupOutput(words)
}

// lookupOutput renders the program `command -v`, `which` or `type -P|-p`
// prints for NAME (see knownOutput), or "".
func lookupOutput(words []string) string {
	if len(words) < 2 {
		return ""
	}
	name := words[len(words)-1]
	opts := words[1 : len(words)-1]
	if n := len(opts); n > 0 && opts[n-1] == "--" {
		opts = opts[:n-1]
	} else if strings.HasPrefix(name, "-") {
		return ""
	}
	switch {
	case words[0] == "command" && slices.Equal(opts, []string{"-v"}),
		words[0] == "which" && len(opts) == 0,
		words[0] == "type" && (slices.Equal(opts, []string{"-P"}) || slices.Equal(opts, []string{"-p"})):
		return name
	}
	return ""
}

// gitDirOutput renders the git directory `git <args>` prints (see
// knownOutput), or "" when args are anything else.
func gitDirOutput(args []string) string {
	dir := ""
	for len(args) >= 2 && args[0] == "-C" {
		if path.IsAbs(args[1]) {
			dir = args[1]
		} else {
			dir = path.Join(dir, args[1])
		}
		args = args[2:]
	}
	args = slices.DeleteFunc(slices.Clone(args), func(a string) bool {
		return a == "-q" || a == "--quiet" || strings.HasPrefix(a, "--path-format=")
	})
	switch {
	case len(args) == 2 && args[0] == "rev-parse" &&
		(args[1] == "--git-dir" || args[1] == "--absolute-git-dir" || args[1] == "--git-common-dir"):
		return path.Join(dir, ".git")
	case len(args) == 3 && args[0] == "rev-parse" && args[1] == "--git-path" && !strings.HasPrefix(args[2], "-"):
		return path.Join(dir, ".git", args[2])
	}
	return ""
}

// knownParam returns the value vars gives a plain parameter expansion ($NAME or
// ${NAME}), and "" for any other expansion or an unknown name.
func knownParam(p *syntax.ParamExp, vars map[string]string) string {
	if p.Param == nil || p.Excl || p.Length || p.Width || p.IsSet || p.Flags != nil ||
		p.NestedParam != nil || p.Index != nil || len(p.Modifiers) > 0 ||
		p.Slice != nil || p.Repl != nil || p.Names != 0 || p.Exp != nil {
		return ""
	}
	return vars[p.Param.Value]
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
