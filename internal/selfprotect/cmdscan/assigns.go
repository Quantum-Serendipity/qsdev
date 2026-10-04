package cmdscan

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// This file finds the variables a statement sets or clears other than
// through its prefix assignments. Bash has many such forms besides NAME=value:
// the declaration builtins, a for or select loop variable, the builtins that
// store into a named variable (read, mapfile, printf -v, getopts, wait -p),
// unset, arithmetic assignments, ${NAME:=value} defaults, a coprocess name and
// a {NAME}> redirect. Any of them can set PATH or HOME for a later command.

// assignSet collects the variables a statement sets.
type assignSet struct {
	names []string
	// dynamic records a variable whose name comes from an expansion
	// (`export $X=v`, `read "$V"`, `(( $X = 1 ))`).
	dynamic bool
}

func (a *assignSet) add(name string) {
	if name, _, _ = strings.Cut(name, "["); name != "" {
		a.names = append(a.names, name)
	}
}

// addWord adds the variable word w names, or records it as dynamic when w is
// built from an expansion.
func (a *assignSet) addWord(w *syntax.Word, vars map[string]string) {
	text, exp := wordText(w, vars)
	if exp {
		a.dynamic = true
		return
	}
	a.add(text)
}

// stmtAssigns returns the variables stmt sets other than through the prefix
// assignments of a simple command (see Command.Assigns).
func stmtAssigns(stmt *syntax.Stmt, vars map[string]string) assignSet {
	var a assignSet
	switch cmd := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		a.builtin(cmd.Args, vars)
	case *syntax.DeclClause:
		a.decl(cmd.Variant.Value, declWords(cmd.Args, vars))
	case *syntax.ForClause:
		if wi, ok := cmd.Loop.(*syntax.WordIter); ok && wi.Name != nil {
			a.add(wi.Name.Value)
		}
	case *syntax.CoprocClause:
		if cmd.Name == nil {
			a.add("COPROC")
		} else {
			a.addWord(cmd.Name, vars)
		}
	}
	for _, r := range stmt.Redirs {
		if r.N != nil && strings.HasPrefix(r.N.Value, "{") {
			a.add(strings.Trim(r.N.Value, "{}")) // {fd}>file stores the fd
		}
	}
	a.expansions(stmt, vars)
	return a
}

// expansions adds the arithmetic assignments (`(( X = 1 ))`, `let X++`,
// `$((X=1))`) and ${X:=value} defaults inside stmt, but not inside the
// statements it holds, which are scanned on their own.
func (a *assignSet) expansions(stmt *syntax.Stmt, vars map[string]string) {
	syntax.Walk(stmt, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Stmt:
			return n == stmt
		case *syntax.BinaryArithm:
			if isArithAssign(n.Op) {
				a.arithTarget(n.X, vars)
			}
		case *syntax.UnaryArithm:
			if n.Op == syntax.Inc || n.Op == syntax.Dec {
				a.arithTarget(n.X, vars)
			}
		case *syntax.ParamExp:
			if n.Exp != nil && (n.Exp.Op == syntax.AssignUnset || n.Exp.Op == syntax.AssignUnsetOrNull) {
				if n.Param == nil || n.Excl {
					a.dynamic = true
				} else {
					a.add(n.Param.Value)
				}
			}
		case *syntax.LetClause:
			for _, e := range n.Exprs {
				if w, ok := e.(*syntax.Word); ok {
					a.arithText(w, vars) // a quoted expression: `let 'X+=1'`
				}
			}
		}
		return true
	})
}

// isArithAssign reports whether op assigns to its left operand.
func isArithAssign(op syntax.BinAritOperator) bool {
	switch op {
	case syntax.Assgn, syntax.AddAssgn, syntax.SubAssgn, syntax.MulAssgn,
		syntax.QuoAssgn, syntax.RemAssgn, syntax.AndAssgn, syntax.OrAssgn,
		syntax.XorAssgn, syntax.ShlAssgn, syntax.ShrAssgn:
		return true
	}
	return false
}

// arithTarget adds the variable an arithmetic assignment's left operand x
// names. An operand that is not a plain word, or is built from an expansion
// (`(( $X = 1 ))` assigns the variable X names), is dynamic.
func (a *assignSet) arithTarget(x syntax.ArithmExpr, vars map[string]string) {
	w, ok := x.(*syntax.Word)
	if !ok {
		a.dynamic = true
		return
	}
	a.addWord(w, vars)
}

// arithText adds the assignments of an arithmetic expression given as the
// text of word w (`let 'X+=1'`), parsing it as `(( text ))`.
func (a *assignSet) arithText(w *syntax.Word, vars map[string]string) {
	text, exp := wordText(w, vars)
	if exp {
		a.dynamic = true
		return
	}
	file, err := syntax.NewParser().Parse(strings.NewReader("(("+text+"))"), "")
	if err != nil || len(file.Stmts) != 1 {
		a.dynamic = true
		return
	}
	a.expansions(file.Stmts[0], vars)
}

// varBuiltin describes a builtin that stores into the variables its options
// or operands name.
type varBuiltin struct {
	// argOpts are the short options that take an argument, and targetOpts
	// those of them whose argument names a variable.
	argOpts, targetOpts string
	// operands returns which operands (by position) name variables.
	operands func(n int) []int
}

func allOperands(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func operandAt(i int) func(int) []int {
	return func(n int) []int {
		if i < n {
			return []int{i}
		}
		return nil
	}
}

func noOperands(int) []int { return nil }

// varBuiltins are the builtins, other than the declaration builtins, that set
// or clear the variables they name.
var varBuiltins = map[string]varBuiltin{
	"read":      {argOpts: "adinNptu", targetOpts: "a", operands: allOperands},
	"mapfile":   {argOpts: "dnOsuCc", operands: operandAt(0)},
	"readarray": {argOpts: "dnOsuCc", operands: operandAt(0)},
	"printf":    {argOpts: "v", targetOpts: "v", operands: noOperands},
	"getopts":   {operands: operandAt(1)},
	"wait":      {argOpts: "p", targetOpts: "p", operands: noOperands},
	"unset":     {operands: allOperands},
}

// declBuiltins are the declaration builtins. Called through a wrapper
// (`builtin export X=1`) they parse as a simple command, not a DeclClause.
var declBuiltins = map[string]bool{
	"export": true, "declare": true, "typeset": true, "local": true, "readonly": true, "nameref": true,
}

// builtin adds the variables a simple command with words sets when its
// program (behind any wrapper such as builtin or command) is a builtin that
// stores into named variables.
func (a *assignSet) builtin(words []*syntax.Word, vars map[string]string) {
	texts := make([]string, len(words))
	for i, w := range words {
		texts[i], _ = wordText(w, vars)
	}
	i := ProgramWordIndex(texts)
	if i < 0 || hasExpansion(words[i], vars) {
		return // an expanded command word is NameHasExpansion's concern
	}
	name, args := ProgramName(texts[i]), words[i+1:]
	switch {
	case declBuiltins[name]:
		dw := make([]declWord, len(args))
		for j, w := range args {
			dw[j] = splitDeclWord(w, vars)
		}
		a.decl(name, dw)
	case name == "let":
		for _, w := range args {
			a.arithText(w, vars)
		}
	default:
		if b, ok := varBuiltins[name]; ok {
			a.varBuiltin(b, args, vars)
		}
	}
}

func hasExpansion(w *syntax.Word, vars map[string]string) bool {
	_, exp := wordText(w, vars)
	return exp
}

// varBuiltin adds the variables builtin b, given args, stores into. Options
// end at "--" or the first word that is not one; an option word built from
// an expansion may be any option, so it is dynamic.
func (a *assignSet) varBuiltin(b varBuiltin, args []*syntax.Word, vars map[string]string) {
	j := 0
options:
	for ; j < len(args); j++ {
		text, exp := wordText(args[j], vars)
		switch {
		case text == "--":
			j++
			break options
		case len(text) < 2 || text[0] != '-':
			if exp {
				a.dynamic = true // the word may expand to an option
			}
			break options
		case exp:
			a.dynamic = true
			continue
		}
		for k := 1; k < len(text); k++ {
			opt := text[k]
			if !strings.ContainsRune(b.argOpts, rune(opt)) {
				continue
			}
			target := strings.ContainsRune(b.targetOpts, rune(opt))
			if k+1 < len(text) {
				if target {
					a.add(text[k+1:])
				}
			} else if j+1 < len(args) {
				j++
				if target {
					a.addWord(args[j], vars)
				}
			}
			break
		}
	}
	operands := args[j:]
	for _, k := range b.operands(len(operands)) {
		a.addWord(operands[k], vars)
	}
}

// declWord is one argument of a declaration builtin: an option, or a
// variable name with an optional value.
type declWord struct {
	option string // the option word (`-n`, `+x`), when it is one
	name   string
	value  string
	// valueSet reports NAME=value (not a bare NAME); valueExp that the value
	// came from an expansion; dynamic that the name or option did.
	valueSet, valueExp, dynamic bool
}

// declWords converts the arguments of a DeclClause, which the parser splits
// into a name and a value when the name is literal.
func declWords(args []*syntax.Assign, vars map[string]string) []declWord {
	out := make([]declWord, 0, len(args))
	for _, as := range args {
		if as.Name == nil {
			out = append(out, splitDeclWord(as.Value, vars))
			continue
		}
		d := declWord{name: as.Name.Value, valueSet: !as.Naked}
		if as.Value != nil {
			d.value, d.valueExp = wordText(as.Value, vars)
		}
		if as.Array != nil {
			d.valueExp = true
		}
		out = append(out, d)
	}
	return out
}

// splitDeclWord splits a declaration argument the parser left whole (an
// option, a quoted `"NAME=value"`, or a word built from an expansion) into a
// declWord. The name part is what precedes the first "=" of the literal text
// before any expansion: an expansion before it makes the name dynamic.
func splitDeclWord(w *syntax.Word, vars map[string]string) declWord {
	text, exp := wordText(w, vars)
	prefix, complete := literalPrefix(w)
	if strings.HasPrefix(prefix, "-") || strings.HasPrefix(prefix, "+") {
		return declWord{option: text, dynamic: !complete}
	}
	if name, _, ok := strings.Cut(prefix, "="); ok {
		_, rest, _ := strings.Cut(text, "=")
		return declWord{name: strings.TrimSuffix(name, "+"), value: rest, valueSet: true, valueExp: exp}
	}
	if !complete || prefix == "" {
		return declWord{dynamic: true}
	}
	return declWord{name: text}
}

// literalPrefix returns the text of w up to its first expansion, with quotes
// removed, and whether w has no expansion at all.
func literalPrefix(w *syntax.Word) (string, bool) {
	var b strings.Builder
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(unescapeUnquoted(p.Value))
		case *syntax.SglQuoted:
			if p.Dollar {
				text, ok := decodeANSIC(p.Value)
				b.WriteString(text)
				if !ok {
					return b.String(), false
				}
			} else {
				b.WriteString(p.Value)
			}
		case *syntax.DblQuoted:
			for _, dp := range p.Parts {
				lit, ok := dp.(*syntax.Lit)
				if !ok {
					return b.String(), false
				}
				b.WriteString(unescapeDoubleQuoted(lit.Value))
			}
		default:
			return b.String(), false
		}
	}
	return b.String(), true
}

// decl adds the variables a declaration builtin named variant sets, exports,
// un-exports or unsets: every name it is given (`export -n HOME` and `local
// HOME` hide HOME from the programs run later). With -n (declare, typeset,
// local, nameref), NAME=target makes NAME a reference, so a later assignment
// to NAME sets target, which is added too.
func (a *assignSet) decl(variant string, words []declWord) {
	nameref := variant == "nameref"
	for _, d := range words {
		switch {
		case d.dynamic:
			a.dynamic = true
		case d.option != "":
			if variant != "export" && variant != "readonly" &&
				strings.HasPrefix(d.option, "-") && strings.ContainsRune(d.option, 'n') {
				nameref = true
			}
		default:
			a.add(d.name)
			if nameref && d.valueSet {
				if d.valueExp {
					a.dynamic = true
				} else {
					a.add(d.value)
				}
			}
		}
	}
}
