package cmdscan

import (
	"slices"

	"mvdan.cc/sh/v3/syntax"
)

// This file models how builtins exit and where statements run: which
// builtins test something with their exit status, which end the shell,
// which run code in the shell itself, and which statements run in a
// subshell. Callers that follow what a command line runs (such as the hook
// program check) build on it instead of keeping their own builtin tables.

// statusBuiltins are the builtins whose exit status tests something
// (`test -x prog && prog`), so a command after them in an && list may never
// run. Other builtins, such as cd, are taken to succeed.
var statusBuiltins = map[string]bool{"test": true, "[": true, "false": true, "type": true, "hash": true}

// shellEnders are the builtins that end the shell running them. return is
// not one: it ends only a function or a sourced file, and at the top level
// bash reports an error and carries on. logout ends only a login shell,
// which a command line is not run in.
var shellEnders = map[string]bool{"exit": true}

// shellSourcers are the builtins that run code in the shell itself: a
// sourced file or an eval string can set any variable, PATH included, for
// the rest of the line.
var shellSourcers = map[string]bool{".": true, "source": true, "eval": true}

// EndsShell reports whether the builtin name ends the shell that runs it
// (see shellEnders). It does not cover exec, which Program reports.
func EndsShell(name string) bool { return shellEnders[name] }

// SourcesCode reports whether the builtin name runs code in the shell
// itself, which can set any variable for later commands (see shellSourcers).
func SourcesCode(name string) bool { return shellSourcers[name] }

// ShellBuiltin returns the builtin the shell runs for c, when the program it
// runs, directly or through the builtins command and builtin, is a literal
// shell builtin (see IsShellBuiltin and ProgramRun.ShellRuns). A word built
// from an expansion is not known to name one.
func (c Command) ShellBuiltin() (string, bool) {
	if c.Name == "" {
		return "", false
	}
	words := append([]string{c.Name}, c.Args...)
	run := Program(words)
	if run.Index < 0 || !run.ShellRuns {
		return "", false
	}
	word := words[run.Index]
	expanded := c.NameHasExpansion
	if run.Index > 0 {
		expanded = slices.Contains(c.ExpandedArgs, word)
	}
	if expanded || !IsShellBuiltin(word) {
		return "", false
	}
	return word, true
}

// statusTest reports whether the exit status of the simple command c tests
// something: it runs a program, whose status is unknown, a status-test
// builtin (statusBuiltins), `command -v`, or a command string; or it is a
// bare assignment whose value comes from an expansion, which takes the
// status of its last command substitution. A builtin such as cd, and a
// wrapper given no command (`exec 2>/dev/null`), are taken to succeed.
func statusTest(c Command) bool {
	if c.Name == "" {
		return c.HasExpansion
	}
	if name, ok := c.ShellBuiltin(); ok {
		return statusBuiltins[name]
	}
	run := Program(append([]string{c.Name}, c.Args...))
	return run.Index >= 0 || run.LookupOnly || run.CommandString
}

// statusTests memoizes whether a statement's exit status tests something
// (see statusTest), so a long && chain, which nests to the left, stays
// linear.
type statusTests struct {
	vars map[string]string
	memo map[*syntax.Stmt]bool
}

// of reports whether the exit status of s tests something. A list, block or
// subshell tests something when any statement in it does; any other
// compound command (a test clause, a loop, ...) does.
func (t statusTests) of(s *syntax.Stmt) bool {
	if s == nil {
		return false
	}
	if v, ok := t.memo[s]; ok {
		return v
	}
	v := true
	switch cmd := s.Cmd.(type) {
	case nil, *syntax.DeclClause:
		v = false
	case *syntax.CallExpr:
		v = statusTest(callCommand(cmd, t.vars))
	case *syntax.BinaryCmd:
		v = t.of(cmd.X) || t.of(cmd.Y)
	case *syntax.Block:
		v = t.any(cmd.Stmts)
	case *syntax.Subshell:
		v = t.any(cmd.Stmts)
	}
	t.memo[s] = v
	return v
}

// any reports whether any of stmts tests something.
func (t statusTests) any(stmts []*syntax.Stmt) bool {
	return slices.ContainsFunc(stmts, t.of)
}

// assignSubshells returns the statements that run in a subshell of the
// line's shell (see Command.Subshell), given the pipeline stages pipelines
// maps. Every statement nested in one runs in it too.
func assignSubshells(root syntax.Node, pipelines map[*syntax.Stmt]int) map[*syntax.Stmt]bool {
	subs := make(map[*syntax.Stmt]bool)
	var mark func(node syntax.Node, sub bool)
	markAll := func(stmts []*syntax.Stmt) {
		for _, s := range stmts {
			mark(s, true)
		}
	}
	mark = func(node syntax.Node, sub bool) {
		syntax.Walk(node, func(n syntax.Node) bool {
			switch n := n.(type) {
			case *syntax.Stmt:
				in := sub || n.Background || n.Coprocess || pipelines[n] != 0
				if !in {
					return true
				}
				subs[n] = true
				if sub {
					return true
				}
				for _, r := range n.Redirs {
					mark(r, true)
				}
				if n.Cmd != nil {
					mark(n.Cmd, true)
				}
				return false
			case *syntax.Subshell:
				if sub {
					return true
				}
				markAll(n.Stmts)
				return false
			case *syntax.CmdSubst:
				if sub {
					return true
				}
				markAll(n.Stmts)
				return false
			case *syntax.ProcSubst:
				if sub {
					return true
				}
				markAll(n.Stmts)
				return false
			}
			return true
		})
	}
	mark(root, false)
	return subs
}
