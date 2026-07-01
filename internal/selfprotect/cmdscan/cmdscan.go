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
// targets of its output redirects, and whether any word used a shell expansion
// ($VAR, $(...), or $((...))).
type Command struct {
	Name         string
	Args         []string
	Redirects    []string
	HasExpansion bool
}

// Parse parses a shell command line into its simple commands. On a parse error
// it returns (nil, err); callers should treat that as "unparseable" and fall
// back to conservative substring checks (fail closed), never fail open.
func Parse(command string) ([]Command, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	var cmds []Command
	syntax.Walk(file, func(node syntax.Node) bool {
		stmt, ok := node.(*syntax.Stmt)
		if !ok {
			return true
		}
		call, ok := stmt.Cmd.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}

		var c Command
		name, exp := wordText(call.Args[0])
		c.Name = name
		c.HasExpansion = exp
		for _, w := range call.Args[1:] {
			t, e := wordText(w)
			c.Args = append(c.Args, t)
			c.HasExpansion = c.HasExpansion || e
		}
		for _, r := range stmt.Redirs {
			if r.Word == nil {
				continue
			}
			t, _ := wordText(r.Word)
			c.Redirects = append(c.Redirects, t)
		}
		cmds = append(cmds, c)
		return true
	})
	return cmds, nil
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
