package cmdscan

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// maxScriptDepth bounds how many nested shell scripts (`sh -c "sh -c '...'"`)
// ScriptStatements descends into.
const maxScriptDepth = 4

// ScriptStatements returns the commands a shell runs for script, one per
// simple command or pipeline, each printed on its own: the statements of a
// list (`a; b`, `a && b`, `a || b`, `a &`, one per line), of blocks,
// subshells, branches, loops and command substitutions, and, recursively,
// of the script a statement runs through a shell's -c option or eval, or
// feeds a script shell as a here-document or here-string (`sh <<EOF`). A
// pipeline stays whole (`curl x | sh`), since that is what a rule about it
// names. A deny rule anchored at the start of a command can then match a
// statement that does not start the script.
//
// A statement that does not start with the program it runs is also
// returned from that program on (see programForm): without its assignments
// (`X=1 curl x`) and wrappers (`nohup curl x`, `timeout 5 curl x`), and with
// a quoted or escaped command word unquoted (`'curl' x`, `\curl x`), each
// stage of a pipeline alike, so a rule anchored at the program's name
// matches it however it is spelled.
//
// A script that does not parse is taken line by line, as a shell reads it,
// and a line that does not parse is returned as it stands, so text the
// parser rejects still reaches the rules.
func ScriptStatements(script string) []string {
	return scriptStatements(script, maxScriptDepth)
}

func scriptStatements(script string, depth int) []string {
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		if !strings.ContainsAny(script, "\n\r") {
			if line := strings.TrimSpace(script); line != "" {
				return []string{line}
			}
			return nil
		}
		var out []string
		for _, line := range strings.FieldsFunc(script, func(r rune) bool { return r == '\n' || r == '\r' }) {
			out = append(out, scriptStatements(line, depth)...)
		}
		return out
	}
	var out []string
	pipeStage := make(map[*syntax.Stmt]bool)
	printer := syntax.NewPrinter(syntax.SingleLine(true))
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Stmt:
			if text, ok := statementText(printer, n, pipeStage); ok {
				out = append(out, text)
				if form, ok := programForm(printer, n); ok {
					out = append(out, form)
				}
			}
			if depth > 0 {
				out = append(out, hereDocStatements(n, depth-1)...)
			}
		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe || n.Op == syntax.PipeAll {
				pipeStage[n.X], pipeStage[n.Y] = true, true
			}
		case *syntax.CallExpr:
			if depth > 0 {
				out = append(out, nestedStatements(n, depth-1)...)
			}
		}
		return true
	})
	return out
}

// statementText prints stmt when it is a simple command or a whole pipeline
// (not a stage of one, see pipeStage), without the !, & or comments around
// it, which a deny rule does not name.
func statementText(printer *syntax.Printer, stmt *syntax.Stmt, pipeStage map[*syntax.Stmt]bool) (string, bool) {
	if pipeStage[stmt] {
		return "", false
	}
	switch c := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		if len(c.Args) == 0 {
			return "", false
		}
	case *syntax.BinaryCmd:
		if c.Op != syntax.Pipe && c.Op != syntax.PipeAll {
			return "", false
		}
	default:
		return "", false
	}
	bare := *stmt
	bare.Negated, bare.Background, bare.Coprocess, bare.Comments = false, false, false, nil
	var b strings.Builder
	if err := printer.Print(&b, &bare); err != nil {
		return "", false
	}
	return strings.TrimSpace(b.String()), true
}

// nestedStatements returns the statements of the script call runs through a
// shell's -c option or eval (see Script), with its words as the shell
// passes them.
func nestedStatements(call *syntax.CallExpr, depth int) []string {
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		words[i], _ = wordText(w, nil)
	}
	script, ok := ShellScript(words)
	if !ok {
		return nil
	}
	return append([]string{script}, scriptStatements(script, depth)...)
}

// programForm renders stmt, a simple command or a whole pipeline, from the
// program each of its commands runs (see Program): assignments and
// wrappers are left out, and each word is unquoted and quoted again only
// where it needs it (see QuoteWord), except a word with an expansion, which
// is printed as written. Redirections are left out, and pipeline stages are
// joined with "|". It returns false when no command of stmt has an
// assignment, a wrapper or a quoted or escaped command word to leave out,
// so the statement as printed already starts with its program, or when a
// command runs no single named program.
func programForm(printer *syntax.Printer, stmt *syntax.Stmt) (string, bool) {
	stages := []*syntax.Stmt{stmt}
	if bin, ok := stmt.Cmd.(*syntax.BinaryCmd); ok {
		stages = stages[:0]
		collectPipeStmts(bin, &stages)
	}
	parts := make([]string, 0, len(stages))
	changed := false
	for _, st := range stages {
		call, ok := st.Cmd.(*syntax.CallExpr)
		if !ok {
			var b strings.Builder
			if err := printer.Print(&b, st.Cmd); err != nil {
				return "", false
			}
			parts = append(parts, strings.TrimSpace(b.String()))
			continue
		}
		text, stripped, ok := callProgramForm(printer, call)
		if !ok {
			return "", false
		}
		parts = append(parts, text)
		changed = changed || stripped
	}
	return strings.Join(parts, " | "), changed
}

// callProgramForm renders call from the word naming the program it runs
// (see programForm), and whether anything was left out or unquoted before
// that word: an assignment, a wrapper, or quotes or escapes in it.
func callProgramForm(printer *syntax.Printer, call *syntax.CallExpr) (text string, stripped, ok bool) {
	words := make([]string, len(call.Args))
	expanded := make([]bool, len(call.Args))
	for i, w := range call.Args {
		words[i], expanded[i] = wordText(w, nil)
	}
	idx := Program(words).Index
	if idx < 0 {
		return "", false, false
	}
	var name strings.Builder
	if err := printer.Print(&name, call.Args[idx]); err != nil {
		return "", false, false
	}
	stripped = len(call.Assigns) > 0 || idx > 0 || name.String() != words[idx]
	out := make([]string, 0, len(words)-idx)
	for i := idx; i < len(words); i++ {
		if !expanded[i] {
			out = append(out, QuoteWord(words[i]))
			continue
		}
		var b strings.Builder
		if err := printer.Print(&b, call.Args[i]); err != nil {
			return "", false, false
		}
		out = append(out, b.String())
	}
	return strings.Join(out, " "), stripped, true
}

// hereDocStatements returns the body of each here-document or here-string
// stmt feeds a script shell, which runs its standard input as a script, and
// the statements of that body.
func hereDocStatements(stmt *syntax.Stmt, depth int) []string {
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil
	}
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		words[i], _ = wordText(w, nil)
	}
	if idx := Program(words).Index; idx < 0 || !IsScriptShell(words[idx]) {
		return nil
	}
	var out []string
	for _, r := range stmt.Redirs {
		body := r.Hdoc
		if r.Op == syntax.WordHdoc {
			body = r.Word
		} else if r.Op != syntax.Hdoc && r.Op != syntax.DashHdoc {
			continue
		}
		text, _ := wordText(body, nil)
		if text = strings.TrimSpace(text); text != "" {
			out = append(out, text)
			out = append(out, scriptStatements(text, depth)...)
		}
	}
	return out
}
