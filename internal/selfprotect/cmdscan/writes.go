package cmdscan

import (
	"path"
	"slices"
	"strings"
)

// WrittenOperands returns the arguments of c that name a file c creates or
// changes, for the commands whose file writes are modelled, and false for any
// other command, whose every operand a caller must treat as possibly written.
// Redirects are not operands: see Command.WriteRedirects. Modelled are:
//
//   - source and ., which read and run a script file and write none of their
//     operands, when the script is a literal file (not built from an
//     expansion, a device such as /dev/stdin, or fed by a redirect or
//     here-document);
//   - sed without an in-place option, which writes only to standard output,
//     when its script is literal and holds no command that writes a file or
//     runs one (w, W, e; any of those letters counts) and is not read from a
//     file (-f);
//   - ln and link, which write only the link they create, not its source:
//     link's second operand, and for ln the links lnLinks lists;
//   - gh pr and gh issue create, comment, edit, view and list, which write
//     no file, when those words are literal (see ghWritesNoFile).
//
// isDir reports whether a word names an existing directory; it should
// answer true when that is unknown.
//
// A command that sets variables, or whose command word is built from an
// expansion, is not modelled: either can change what runs. Neither is a
// command whose name the line redefines (see Redefinitions): the caller must
// check that, as c alone does not show it.
func WrittenOperands(c Command, isDir func(word string) bool) ([]string, bool) {
	if len(c.Assigns) > 0 || c.AssignsDynamic || c.NameHasExpansion {
		return nil, false
	}
	switch c.Name {
	case "source", ".":
		return nil, sourcesLiteralFile(c)
	case "sed":
		return nil, !c.HasExpansion && sedWritesStdoutOnly(c.Args)
	case "ln":
		return lnLinks(c.Args, isDir)
	case "link":
		if len(c.Args) != 2 {
			return nil, false
		}
		return c.Args[1:], true
	case "gh":
		return nil, ghWritesNoFile(c)
	}
	return nil, false
}

// Redefinitions returns whether cmds, the commands of one line, may make a
// command name run something other than the program it names: a shell
// function the line defines (Command.Defines), or an alias it defines
// (`alias gh='rm -f'`, also through a wrapper such as builtin). An alias
// command with a word built from an expansion may define any name, and so
// may a script the line sources (`. ./defs.sh`, `source x`), whose
// definitions are not seen; the one source command of a line still names
// itself, so `source .npmrc` alone keeps its modelling. Where on the line
// the definition stands is not judged: a loop may run it before the
// command.
func Redefinitions(cmds []Command) func(name string) bool {
	names := map[string]bool{}
	anyName := false
	sources := 0
	for _, c := range cmds {
		if c.Defines != "" {
			names[c.Defines] = true
		}
		words := append([]string{c.Name}, c.Args...)
		if runsScript(words) {
			sources++
			continue
		}
		if !slices.Contains(words, "alias") {
			continue
		}
		anyName = anyName || c.HasExpansion || c.NameHasExpansion
		for _, d := range aliasDefinitions(words) {
			names[d.name] = true
		}
	}
	return func(name string) bool {
		if anyName || names[name] || sources > 1 {
			return true
		}
		return sources == 1 && !runsScript([]string{name})
	}
}

// runsScript reports whether words, a command word and its arguments, source
// a script into the shell (`source x`, `. x`), also through the builtin or
// command wrapper, so that the script may define functions and aliases.
func runsScript(words []string) bool {
	if len(words) > 1 && (words[0] == "builtin" || words[0] == "command") {
		words = words[1:]
	}
	return words[0] == "source" || words[0] == "."
}

// ghTextSubcommands are the gh pull request and issue subcommands that write
// no file: their operands are numbers, URLs and text (--title, --body), and
// they print to standard output.
var ghTextSubcommands = map[string]bool{
	"create": true, "comment": true, "edit": true, "view": true, "list": true,
}

// ghWritesNoFile reports whether c, a gh command, creates, comments on,
// edits, views or lists a pull request or issue, named by literal words
// (`gh pr create`), so that it writes no file. Its words that come from an
// expansion (a --body of `$(cat <<'EOF' ...)`) are text, not paths; the
// commands of the substitution are judged on their own.
func ghWritesNoFile(c Command) bool {
	if len(c.Args) < 2 || slices.Contains(c.ExpandedArgs, c.Args[0]) || slices.Contains(c.ExpandedArgs, c.Args[1]) {
		return false
	}
	return (c.Args[0] == "pr" || c.Args[0] == "issue") && ghTextSubcommands[c.Args[1]]
}

// sourcesLiteralFile reports whether c, a source or . command, runs a script
// file named by a literal word: not an expansion, a device or a process's
// file descriptor (/dev/stdin, /proc/self/fd/0), and not with a redirect or
// here-document that could feed it the script.
func sourcesLiteralFile(c Command) bool {
	if len(c.Args) == 0 || c.HasExpansion || len(c.ReadRedirects) > 0 || len(c.Heredocs) > 0 {
		return false
	}
	script := c.Args[0]
	return script != "-" && !strings.HasPrefix(script, "-") &&
		!strings.HasPrefix(script, "/dev/") && !strings.HasPrefix(script, "/proc/")
}

// sedFlagOptions are the long options of GNU sed that take no argument and
// neither write a file nor run a program.
var sedFlagOptions = map[string]bool{
	"quiet": true, "silent": true, "regexp-extended": true, "separate": true,
	"unbuffered": true, "null-data": true, "posix": true, "debug": true,
	"sandbox": true, "follow-symlinks": true,
}

// sedWritesStdoutOnly reports whether sed given args writes only to standard
// output: no in-place option (-i, also clustered as in -ni, and any
// abbreviation of --in-place), no script file (-f, --file), and no script
// that can write or run (see sedScriptReadOnly). An option it does not know
// fails closed, as an abbreviated long option may be one of those.
func sedWritesStdoutOnly(args []string) bool {
	var scripts []string
	explicit := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			if !explicit && len(scripts) == 0 && i+1 < len(args) {
				scripts = append(scripts, args[i+1])
			}
			i = len(args)
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			switch {
			case name == "expression":
				if !hasValue {
					if i+1 >= len(args) {
						return false
					}
					i++
					value = args[i]
				}
				scripts, explicit = append(scripts, value), true
			case name == "line-length" && hasValue:
			case sedFlagOptions[name] && !hasValue:
			default:
				return false
			}
		case strings.HasPrefix(a, "-") && a != "-":
			script, consumed, ok := sedShortCluster(a[1:], args[i+1:])
			if !ok {
				return false
			}
			if script != nil {
				scripts, explicit = append(scripts, *script), true
			}
			i += consumed
		default:
			if !explicit && len(scripts) == 0 {
				scripts = append(scripts, a)
			}
		}
	}
	if len(scripts) == 0 {
		return false
	}
	for _, s := range scripts {
		if !sedScriptReadOnly(s) {
			return false
		}
	}
	return true
}

// sedShortCluster parses the short-option cluster opts (without its dash) of
// sed, given the words after it as next. It returns the script an -e in the
// cluster gives, how many of next the cluster consumes, and false when the
// cluster edits in place (-i), reads a script file (-f) or holds an option it
// does not know.
func sedShortCluster(opts string, next []string) (*string, int, bool) {
	for j, r := range opts {
		switch r {
		case 'n', 'E', 'r', 's', 'u', 'z':
		case 'e', 'l':
			value, consumed := opts[j+1:], 0
			if value == "" {
				if len(next) == 0 {
					return nil, 0, false
				}
				value, consumed = next[0], 1
			}
			if r == 'l' {
				return nil, consumed, true
			}
			return &value, consumed, true
		default: // i (in place), f (script file) and anything unknown
			return nil, 0, false
		}
	}
	return nil, 0, true
}

// sedScriptReadOnly reports whether the sed script holds none of the letters
// of the commands and flags that write a file or run a program (w, W, e).
// It errs toward false: a script such as '/export/p' that only names an e is
// not proven read-only.
func sedScriptReadOnly(script string) bool {
	return !strings.ContainsAny(script, "wWe")
}

// lnLinks returns the links ln given args may create: below its
// -t/--target-directory directory, or else its last operand or, when that is
// a directory (isDir, a trailing slash, or more than one source), below it;
// each under the base name of a source operand. Its only operand names a
// link created in the working directory under the same name. ln's other
// options take no argument, except -S/--suffix.
func lnLinks(args []string, isDir func(string) bool) ([]string, bool) {
	var operands []string
	target := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			operands = append(operands, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			takesValue := name != "" && (strings.HasPrefix("target-directory", name) || strings.HasPrefix("suffix", name))
			if !takesValue {
				continue
			}
			if !hasValue {
				if i+1 >= len(args) {
					return nil, false
				}
				i++
				value = args[i]
			}
			if strings.HasPrefix("target-directory", name) {
				target = value
			}
		case strings.HasPrefix(a, "-") && a != "-":
			for j, r := range a[1:] {
				if r != 't' && r != 'S' {
					continue
				}
				value := a[j+2:]
				if value == "" {
					if i+1 >= len(args) {
						return nil, false
					}
					i++
					value = args[i]
				}
				if r == 't' {
					target = value
				}
				break
			}
		default:
			operands = append(operands, a)
		}
	}
	if target == "" {
		if len(operands) < 2 {
			return operands, true
		}
		target, operands = operands[len(operands)-1], operands[:len(operands)-1]
		if len(operands) == 1 && !strings.HasSuffix(target, "/") && !isDir(target) {
			return []string{target}, true
		}
	}
	links := []string{target}
	for _, src := range operands {
		links = append(links, strings.TrimSuffix(target, "/")+"/"+path.Base(strings.ReplaceAll(src, `\`, "/")))
	}
	return links, true
}
