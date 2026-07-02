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
	HasExpansion   bool
	// Pipeline groups commands joined by `|`/`|&`: all stages of one pipeline
	// share the same non-zero id, in left-to-right order. Standalone commands
	// have Pipeline == 0. Rules use this to reason about dataflow across a pipe
	// (e.g. a protected file read upstream and exfiltrated downstream).
	Pipeline int
}

// safeReadVerbs are commands that only read or inspect their arguments — they
// never delete, copy, execute a payload, or spawn a subshell. Self-protection
// consumers (rules, evasion) use IsSafeReadVerb to decide whether a parsed
// command word can clear a substring-triggered deny: anything outside this set
// (a wrapper like sudo/env/sh/xargs/find, or an unknown binary) is opaque and
// must fail closed. sed/awk/eval are deliberately absent — they can mutate in
// place or execute.
var safeReadVerbs = map[string]bool{
	"cat": true, "grep": true, "egrep": true, "fgrep": true, "rg": true,
	"ls": true, "head": true, "tail": true, "wc": true, "echo": true,
	"printf": true, "test": true, "[": true, "true": true, "false": true,
	"pwd": true, "stat": true, "file": true, "diff": true, "cmp": true,
	"sort": true, "uniq": true, "cut": true, "jq": true, "git": true,
	"go": true, "tac": true, "nl": true,
}

// IsSafeReadVerb reports whether name is a read-only/inspection command that can
// clear a substring-triggered self-protection deny. See safeReadVerbs.
func IsSafeReadVerb(name string) bool {
	return safeReadVerbs[name]
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
		if call, ok := stmt.Cmd.(*syntax.CallExpr); ok && len(call.Args) > 0 {
			hasWord = true
			name, exp := wordText(call.Args[0])
			c.Name = name
			c.HasExpansion = exp
			for _, w := range call.Args[1:] {
				t, e := wordText(w)
				c.Args = append(c.Args, t)
				c.HasExpansion = c.HasExpansion || e
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
		}

		// Emit the command when it has a command word, or when it carries
		// redirects that must be attributed even without one (compound-command
		// redirects, bare `VAR=val > f`). Skip pure structural statements.
		if hasWord || len(c.WriteRedirects) > 0 || len(c.ReadRedirects) > 0 {
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

// wordText renders a shell word to its literal text (quotes removed) and reports
// whether any part of it was a shell expansion. Expansions contribute no literal
// text but flip the expansion flag, so `"$X"` yields ("", true) while the
// single-quoted literal `'eval "$("'` yields (`eval "$("`, false).
func wordText(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	var b strings.Builder
	hasExpansion := false
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(p.Value)
		case *syntax.SglQuoted:
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, dp := range p.Parts {
				if lit, ok := dp.(*syntax.Lit); ok {
					b.WriteString(lit.Value)
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
