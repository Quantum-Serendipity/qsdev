package devenv

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// This file groups the attribute definitions of the generated devenv.nix.
//
// devenv.nix is assembled from independent pieces (the template, ecosystem
// module fragments, the LSP section, tool sections), each of which writes
// attribute paths such as `languages.go = {...};` or `env.GOFLAGS = "...";`.
// Nix merges those, but the result repeats top-level keys (`languages`,
// `services`, `scripts`, ...), which the always-on statix hook rejects
// (W20 repeated_keys), and two pieces that set the same leaf path produce a
// file that does not even parse. normalizeNixModule parses the bindings and
// renders each attribute set with every key defined once, turning a key that
// several bindings share into one nested attribute set, and reports a leaf
// path that is defined twice as an error.

// errNixSyntax reports generated Nix the attribute normalizer cannot scan.
var errNixSyntax = errors.New("unsupported Nix syntax")

// nixSource is Nix source text plus lexical facts gathered while scanning it.
type nixSource struct {
	src string
	// noShift holds the offsets of newlines inside double-quoted strings,
	// where leading whitespace on the next line is part of the value, so
	// reindenting must leave that line alone.
	noShift map[int]bool
	// idents holds the identifiers referenced in code (not attribute
	// selections such as the `devenv` in `config.devenv`).
	idents map[string]bool
}

func newNixSource(src string) *nixSource {
	return &nixSource{src: src, noShift: make(map[int]bool), idents: make(map[string]bool)}
}

// nixBinding is one `attr.path = value;` definition.
type nixBinding struct {
	trivia  string   // whitespace and comments before the binding
	path    []string // attribute path segments as written
	value   string   // value expression text
	valueAt int      // offset of value in the source
	start   int      // offset of the first path segment
	end     int      // offset just past the terminating ';'
	indent  int      // column of the first path segment
}

func (s *nixSource) errorf(p int, format string, args ...any) error {
	line := strings.Count(s.src[:min(p, len(s.src))], "\n") + 1
	return fmt.Errorf("%w: line %d: %s", errNixSyntax, line, fmt.Sprintf(format, args...))
}

func isNixIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNixIdentChar(c byte) bool {
	return isNixIdentStart(c) || (c >= '0' && c <= '9') || c == '\'' || c == '-'
}

// describeNixPath renders an attribute path for an error message. Quoted and
// interpolated segments can hold arbitrary text, and the module embeds
// service credentials in string literals, so only plain identifier segments
// are echoed; any other segment is shown as a placeholder.
func describeNixPath(path []string) string {
	parts := make([]string, len(path))
	for i, seg := range path {
		parts[i] = `"..."`
		if isPlainNixIdent(seg) {
			parts[i] = seg
		}
	}
	return strings.Join(parts, ".")
}

func isPlainNixIdent(seg string) bool {
	if seg == "" || !isNixIdentStart(seg[0]) {
		return false
	}
	for i := 1; i < len(seg); i++ {
		if !isNixIdentChar(seg[i]) {
			return false
		}
	}
	return true
}

func (s *nixSource) identEnd(p int) int {
	for p < len(s.src) && isNixIdentChar(s.src[p]) {
		p++
	}
	return p
}

// skipTrivia skips whitespace and comments starting at p.
func (s *nixSource) skipTrivia(p int) (int, error) {
	for p < len(s.src) {
		switch {
		case strings.IndexByte(" \t\r\n", s.src[p]) >= 0:
			p++
		case s.src[p] == '#':
			p = s.skipLineComment(p)
		case strings.HasPrefix(s.src[p:], "/*"):
			q := strings.Index(s.src[p+2:], "*/")
			if q < 0 {
				return 0, s.errorf(p, "unterminated comment")
			}
			p += q + 4
		default:
			return p, nil
		}
	}
	return p, nil
}

func (s *nixSource) skipLineComment(p int) int {
	if q := strings.IndexByte(s.src[p:], '\n'); q >= 0 {
		return p + q
	}
	return len(s.src)
}

// lexString returns the offset just past the double-quoted string at p.
func (s *nixSource) lexString(p int) (int, error) {
	src := s.src
	for q := p + 1; q < len(src); {
		switch {
		case src[q] == '"':
			return q + 1, nil
		case src[q] == '\\' && q+1 < len(src):
			if src[q+1] == '\n' {
				s.noShift[q+1] = true
			}
			q += 2
		case src[q] == '\n':
			s.noShift[q] = true
			q++
		case strings.HasPrefix(src[q:], "${"):
			end, err := s.lexInterpolation(q + 2)
			if err != nil {
				return 0, err
			}
			q = end
		case strings.HasPrefix(src[q:], "$$"): // "$${" is a literal "${"
			q += 2
		default:
			q++
		}
	}
	return 0, s.errorf(p, "unterminated string")
}

// lexIndentedString returns the offset just past the indented string at p.
func (s *nixSource) lexIndentedString(p int) (int, error) {
	src := s.src
	for q := p + 2; q < len(src); {
		switch {
		case strings.HasPrefix(src[q:], "'''"), strings.HasPrefix(src[q:], "''$"):
			q += 3
		case strings.HasPrefix(src[q:], "''\\"):
			q += 4
		case strings.HasPrefix(src[q:], "''"):
			return q + 2, nil
		case strings.HasPrefix(src[q:], "${"):
			end, err := s.lexInterpolation(q + 2)
			if err != nil {
				return 0, err
			}
			q = end
		case strings.HasPrefix(src[q:], "$$"):
			q += 2
		default:
			q++
		}
	}
	return 0, s.errorf(p, "unterminated indented string")
}

// lexInterpolation scans the expression of a `${...}` whose body starts at p
// and returns the offset just past its closing brace.
func (s *nixSource) lexInterpolation(p int) (int, error) {
	q, err := s.scanExpr(p, false)
	if err != nil {
		return 0, err
	}
	if q >= len(s.src) || s.src[q] != '}' {
		return 0, s.errorf(p, "unterminated interpolation")
	}
	return q + 1, nil
}

// scanExpr scans an expression starting at p and returns the offset of the
// character that ends it: an unmatched closing bracket, or (with stopOnSemi)
// the ';' ending a binding. Semicolons that belong to let bindings or to
// `with`/`assert` do not end the expression.
func (s *nixSource) scanExpr(p int, stopOnSemi bool) (int, error) {
	src := s.src
	depth, lets, withs := 0, 0, 0
	for p < len(src) {
		c := src[p]
		var err error
		switch {
		case c == '#':
			p = s.skipLineComment(p)
		case strings.HasPrefix(src[p:], "/*"):
			p, err = s.skipTrivia(p)
		case c == '"':
			p, err = s.lexString(p)
		case strings.HasPrefix(src[p:], "''"):
			p, err = s.lexIndentedString(p)
		case strings.HasPrefix(src[p:], "${"):
			depth++
			p += 2
		case c == '{' || c == '[' || c == '(':
			depth++
			p++
		case c == '}' || c == ']' || c == ')':
			if depth == 0 {
				return p, nil
			}
			depth--
			p++
		case c == ';':
			if depth == 0 && stopOnSemi {
				switch {
				case lets > 0:
				case withs > 0:
					withs--
				default:
					return p, nil
				}
			}
			p++
		case isNixIdentStart(c):
			q := s.identEnd(p)
			word := src[p:q]
			if p == 0 || src[p-1] != '.' {
				s.idents[word] = true
			}
			if depth == 0 {
				switch word {
				case "let":
					lets++
				case "in":
					lets = max(lets-1, 0)
				case "with", "assert":
					withs++
				}
			}
			p = q
		default:
			p++
		}
		if err != nil {
			return 0, err
		}
	}
	return p, nil
}

// parseBindings parses the bindings between p and end (exclusive) and returns
// them with the trailing trivia after the last one.
func (s *nixSource) parseBindings(p, end int) ([]nixBinding, string, error) {
	var out []nixBinding
	for {
		triviaAt := p
		q, err := s.skipTrivia(p)
		if err != nil {
			return nil, "", err
		}
		if q >= end {
			return out, s.src[triviaAt:end], nil
		}
		b := nixBinding{trivia: s.src[triviaAt:q], start: q, indent: q - (strings.LastIndexByte(s.src[:q], '\n') + 1)}
		if p, err = s.parseBinding(&b, q); err != nil {
			return nil, "", err
		}
		if p > end {
			return nil, "", s.errorf(b.start, "binding runs past its attribute set")
		}
		out = append(out, b)
	}
}

// parseBinding parses the attribute path, value and ';' of the binding at p
// into b and returns the offset after the ';'.
func (s *nixSource) parseBinding(b *nixBinding, p int) (int, error) {
	src := s.src
	for {
		var q int
		var err error
		switch {
		case p < len(src) && src[p] == '"':
			q, err = s.lexString(p)
		case strings.HasPrefix(src[p:], "${"):
			q, err = s.lexInterpolation(p + 2)
		case p < len(src) && isNixIdentStart(src[p]):
			q = s.identEnd(p)
		default:
			return 0, s.errorf(p, "expected an attribute name")
		}
		if err != nil {
			return 0, err
		}
		b.path = append(b.path, src[p:q])
		if p, err = s.skipTrivia(q); err != nil {
			return 0, err
		}
		if p < len(src) && src[p] == '.' {
			if p, err = s.skipTrivia(p + 1); err != nil {
				return 0, err
			}
			continue
		}
		break
	}
	if p >= len(src) || src[p] != '=' || strings.HasPrefix(src[p:], "==") {
		return 0, s.errorf(p, "expected '=' after %s", describeNixPath(b.path))
	}
	p++
	for p < len(src) && strings.IndexByte(" \t\r\n", src[p]) >= 0 {
		p++
	}
	b.valueAt = p
	q, err := s.scanExpr(p, true)
	if err != nil {
		return 0, err
	}
	if q >= len(src) || src[q] != ';' {
		return 0, s.errorf(b.start, "expected ';' after the value of %s", describeNixPath(b.path))
	}
	b.value = strings.TrimRight(src[p:q], " \t\r\n")
	b.end = q + 1
	return b.end, nil
}

// attrsBody reports whether b's value is an attribute set literal and, if so,
// the offsets just inside its braces.
func (s *nixSource) attrsBody(b nixBinding) (open, closing int, ok bool) {
	if !strings.HasPrefix(b.value, "{") {
		return 0, 0, false
	}
	q, err := s.scanExpr(b.valueAt+1, false)
	if err != nil || q != b.valueAt+len(b.value)-1 || s.src[q] != '}' {
		return 0, 0, false
	}
	return b.valueAt + 1, q, true
}

// nixAttrKey returns the attribute name a path segment denotes, so that
// `"qsdev-build"` and `qsdev-build` group together. Interpolated names are
// returned as written.
func nixAttrKey(seg string) string {
	if len(seg) >= 2 && seg[0] == '"' && !strings.ContainsAny(seg[1:len(seg)-1], `\$"`) {
		return seg[1 : len(seg)-1]
	}
	return seg
}

// nixEntry is a binding placed at path (a suffix of its written path).
type nixEntry struct {
	b    nixBinding
	path []string
}

type nixEntryGroup struct {
	key     string
	entries []nixEntry
}

// groupEntries groups entries by their first path segment. Each group is
// placed where its shortest-path member was (the `env = {...}` block rather
// than the first `env.X`), ties going to the earliest.
func groupEntries(entries []nixEntry) []nixEntryGroup {
	index := make(map[string]int)
	var groups []nixEntryGroup
	for _, e := range entries {
		k := nixAttrKey(e.path[0])
		i, ok := index[k]
		if !ok {
			i = len(groups)
			index[k] = i
			groups = append(groups, nixEntryGroup{key: k})
		}
		groups[i].entries = append(groups[i].entries, e)
	}
	slices.SortStableFunc(groups, func(a, b nixEntryGroup) int {
		return a.anchor().b.start - b.anchor().b.start
	})
	return groups
}

// anchor returns the member whose position the group takes: the one with
// the shortest path, ties going to the earliest.
func (g nixEntryGroup) anchor() nixEntry {
	best := g.entries[0]
	for _, e := range g.entries[1:] {
		if len(e.path) < len(best.path) {
			best = e
		}
	}
	return best
}

// renderInPlace renders bindings that keep their original attribute set,
// shifted by delta columns. Bindings whose key is unique keep their text;
// keys shared by several bindings become one nested attribute set.
func (s *nixSource) renderInPlace(bs []nixBinding, delta int) (string, error) {
	entries := make([]nixEntry, len(bs))
	for i, b := range bs {
		entries[i] = nixEntry{b: b, path: b.path}
	}
	var out strings.Builder
	for i, g := range groupEntries(entries) {
		var text string
		var err error
		if len(g.entries) == 1 {
			text, err = s.renderOneInPlace(g.entries[0].b, delta)
		} else {
			first := g.entries[0].b
			text, err = s.renderGroup(g, nil, first.indent+delta, i == 0)
		}
		if err != nil {
			return "", err
		}
		out.WriteString(text)
	}
	return out.String(), nil
}

func (s *nixSource) renderOneInPlace(b nixBinding, delta int) (string, error) {
	value, err := s.renderValue(b, delta)
	if err != nil {
		return "", err
	}
	head := s.reindent(s.src[b.start-len(b.trivia):b.valueAt], b.start-len(b.trivia), delta)
	return head + value + ";", nil
}

// renderValue renders b's value shifted by delta columns, normalizing the
// bindings of an attribute set literal.
func (s *nixSource) renderValue(b nixBinding, delta int) (string, error) {
	open, closing, ok := s.attrsBody(b)
	if !ok {
		return s.reindent(b.value, b.valueAt, delta), nil
	}
	inner, trailing, err := s.parseBindings(open, closing)
	if err != nil {
		return "", err
	}
	body, err := s.renderInPlace(inner, delta)
	if err != nil {
		return "", err
	}
	// The closing brace shifts with its line.
	at := closing - len(trailing)
	return "{" + body + s.reindent(s.src[at:closing+1], at, delta), nil
}

// renderGroup renders the entries sharing one key as a single binding at
// indent, prefixed by the collapsed attribute path prefix.
func (s *nixSource) renderGroup(g nixEntryGroup, prefix []string, indent int, first bool) (string, error) {
	// Two pieces may set the same leaf to the same value (java and scala
	// both enable languages.java); one definition says it all.
	if e, ok := s.identicalLeaves(g); ok {
		return s.renderMember(e, prefix, indent, first)
	}
	var children []nixEntry
	var header nixBinding
	hasHeader := false
	for _, e := range g.entries {
		if len(e.path) > 1 {
			children = append(children, nixEntry{b: e.b, path: e.path[1:]})
			continue
		}
		open, closing, ok := s.attrsBody(e.b)
		if !ok {
			return "", fmt.Errorf("devenv.nix defines %s more than once", describeNixPath(append(slices.Clone(prefix), e.path...)))
		}
		inner, _, err := s.parseBindings(open, closing)
		if err != nil {
			return "", err
		}
		for _, ib := range inner {
			children = append(children, nixEntry{b: ib, path: ib.path})
		}
		if !hasHeader {
			header, hasHeader = e.b, true
		}
	}
	path := append(slices.Clone(prefix), g.entries[0].path[0])

	var trivia string
	if hasHeader {
		trivia = renderTrivia(header.trivia, indent, first)
	} else {
		trivia = renderTrivia(blankOnly(g.anchor().b.trivia), indent, first)
	}

	sub := groupEntries(children)
	if len(sub) == 1 && len(sub[0].entries) > 1 {
		text, err := s.renderGroup(sub[0], path, indent, first)
		if err != nil || !hasHeader {
			return text, err
		}
		// The collapsed group also carries this level's comments.
		return trivia + strings.TrimLeft(text, "\n \t"), nil
	}

	pad := strings.Repeat(" ", indent)
	var out strings.Builder
	out.WriteString(trivia + strings.Join(path, ".") + " = {")
	for i, cg := range sub {
		var text string
		var err error
		if len(cg.entries) == 1 {
			text, err = s.renderMember(cg.entries[0], nil, indent+2, i == 0)
		} else {
			text, err = s.renderGroup(cg, nil, indent+2, i == 0)
		}
		if err != nil {
			return "", err
		}
		out.WriteString(text)
	}
	out.WriteString("\n" + pad + "};")
	return out.String(), nil
}

// identicalLeaves reports whether every entry of g sets the group's key
// itself to the same non-attribute-set value, returning the first.
func (s *nixSource) identicalLeaves(g nixEntryGroup) (nixEntry, bool) {
	first := g.entries[0]
	for _, e := range g.entries {
		if _, _, attrs := s.attrsBody(e.b); len(e.path) != 1 || attrs || e.b.value != first.b.value {
			return nixEntry{}, false
		}
	}
	return first, true
}

// renderMember renders one entry moved into a regrouped attribute set,
// its path extended by the collapsed prefix.
func (s *nixSource) renderMember(e nixEntry, prefix []string, indent int, first bool) (string, error) {
	value, err := s.renderValue(e.b, indent-e.b.indent)
	if err != nil {
		return "", err
	}
	path := append(slices.Clone(prefix), e.path...)
	return renderTrivia(e.b.trivia, indent, first) + strings.Join(path, ".") + " = " + value + ";", nil
}

// renderTrivia re-renders the comments of trivia at indent, keeping one blank
// line before them when the original had one (except for a first member).
func renderTrivia(trivia string, indent int, first bool) string {
	pad := strings.Repeat(" ", indent)
	lines := strings.Split(trivia, "\n")
	var out strings.Builder
	blank := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			blank = blank || (i > 0 && i < len(lines)-1 && out.Len() == 0)
			continue
		}
		if out.Len() == 0 && blank && !first {
			out.WriteString("\n")
		}
		out.WriteString("\n" + pad + line)
	}
	if out.Len() == 0 && blank && !first {
		out.WriteString("\n")
	}
	out.WriteString("\n" + pad)
	return out.String()
}

// blankOnly keeps only the line structure of trivia, dropping comments that
// describe the binding rather than the group it now belongs to.
func blankOnly(trivia string) string {
	if strings.Contains(trivia, "\n\n") || strings.Count(trivia, "\n") > 1 {
		return "\n\n"
	}
	return "\n"
}

// reindent shifts every line of text after the first by delta columns.
// Lines that start inside a double-quoted string (whose leading whitespace
// is part of the value) and empty lines are left alone. Lines of an indented
// string, whitespace-only ones included, all move together, which keeps the
// string's value because Nix strips their common indentation.
// Negative shifts are not applied: they could change an indented string.
func (s *nixSource) reindent(text string, at, delta int) string {
	if delta <= 0 || !strings.Contains(text, "\n") {
		return text
	}
	pad := strings.Repeat(" ", delta)
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		out.WriteByte(text[i])
		if text[i] != '\n' || s.noShift[at+i] {
			continue
		}
		rest := text[i+1:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[:nl]
		}
		if rest != "" {
			out.WriteString(pad)
		}
	}
	return out.String()
}

// normalizeNixModule parses a devenv.nix module (`{ formals }: { bindings }`)
// and renders it with every attribute defined once per attribute set. It
// also drops module arguments the body never references, which deadnix
// would report.
func normalizeNixModule(src string) (string, error) {
	s := newNixSource(src)
	m, err := s.moduleBody()
	if err != nil {
		return "", err
	}
	// Identifiers seen while scanning the formals are not references.
	s.idents = make(map[string]bool)
	bindings, trailing, err := s.parseBindings(m.open+1, m.closing)
	if err != nil {
		return "", err
	}
	body, err := s.renderInPlace(bindings, 0)
	if err != nil {
		return "", err
	}
	header := src[:m.formalsOpen+1] + usedFormals(src[m.formalsOpen+1:m.formalsEnd], s.idents) + src[m.formalsEnd:m.open+1]
	return header + body + trailing + src[m.closing:], nil
}

// nixModule holds the brace offsets of a `{ formals }: { ... }` module.
type nixModule struct {
	formalsOpen, formalsEnd int
	open, closing           int
}

// moduleBody locates the attribute set of a `{ formals }: { ... }` module
// and returns the offsets of its braces.
func (s *nixSource) moduleBody() (nixModule, error) {
	var m nixModule
	src := s.src
	p, err := s.skipTrivia(0)
	m.formalsOpen = p
	if err != nil {
		return m, err
	}
	if p >= len(src) || src[p] != '{' {
		return m, s.errorf(p, "expected the module argument set")
	}
	m.formalsEnd, err = s.scanExpr(p+1, false)
	if err != nil {
		return m, err
	}
	if m.formalsEnd >= len(src) || src[m.formalsEnd] != '}' || strings.Contains(src[p+1:m.formalsEnd], "{") {
		return m, s.errorf(p, "expected a plain module argument set")
	}
	q, err := s.skipTrivia(m.formalsEnd + 1)
	if err != nil {
		return m, err
	}
	if q >= len(src) || src[q] != ':' {
		return m, s.errorf(q, "expected ':' after the module argument set")
	}
	if m.open, err = s.skipTrivia(q + 1); err != nil {
		return m, err
	}
	if m.open >= len(src) || src[m.open] != '{' {
		return m, s.errorf(m.open, "expected the module attribute set")
	}
	if m.closing, err = s.scanExpr(m.open+1, false); err != nil {
		return m, err
	}
	if m.closing >= len(src) || src[m.closing] != '}' {
		return m, s.errorf(m.open, "unterminated module attribute set")
	}
	return m, nil
}

// NixModuleAttrs returns the attribute paths a devenv.nix module defines,
// flattened through nested attribute sets (`languages.go.enable`), mapped
// to their value text. It lets callers check what the file configures
// without depending on how the definitions are nested.
func NixModuleAttrs(src string) (map[string]string, error) {
	s := newNixSource(src)
	m, err := s.moduleBody()
	if err != nil {
		return nil, err
	}
	bindings, _, err := s.parseBindings(m.open+1, m.closing)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	if err := s.flattenAttrs(bindings, "", out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *nixSource) flattenAttrs(bs []nixBinding, prefix string, out map[string]string) error {
	for _, b := range bs {
		keys := make([]string, len(b.path))
		for i, seg := range b.path {
			keys[i] = nixAttrKey(seg)
		}
		path := prefix + strings.Join(keys, ".")
		open, closing, ok := s.attrsBody(b)
		if !ok {
			out[path] = b.value
			continue
		}
		inner, _, err := s.parseBindings(open, closing)
		if err != nil {
			return err
		}
		if len(inner) == 0 {
			out[path] = b.value
		}
		if err := s.flattenAttrs(inner, path+".", out); err != nil {
			return err
		}
	}
	return nil
}

// usedFormals rewrites a `pkgs, lib, config, ...` argument list keeping only
// the names in used and the ellipsis.
func usedFormals(formals string, used map[string]bool) string {
	var keep []string
	for _, f := range strings.Split(formals, ",") {
		name := strings.TrimSpace(f)
		if name == "..." || used[name] || !isPlainFormal(name) {
			keep = append(keep, name)
		}
	}
	return " " + strings.Join(keep, ", ") + " "
}

// isPlainFormal reports whether f is a bare argument name (no default).
func isPlainFormal(f string) bool {
	if f == "" || !isNixIdentStart(f[0]) {
		return false
	}
	for i := 1; i < len(f); i++ {
		if !isNixIdentChar(f[i]) {
			return false
		}
	}
	return true
}

// dropNixBindings removes from a devenv.nix fragment (a sequence of bindings)
// every binding whose attribute path drop matches, together with the
// comments that precede it.
func dropNixBindings(fragment string, drop func(path []string) bool) (string, error) {
	s := newNixSource(fragment)
	bindings, trailing, err := s.parseBindings(0, len(fragment))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, b := range bindings {
		path := make([]string, len(b.path))
		for i, seg := range b.path {
			path[i] = nixAttrKey(seg)
		}
		if drop(path) {
			continue
		}
		out.WriteString(fragment[b.start-len(b.trivia) : b.end])
	}
	out.WriteString(trailing)
	return out.String(), nil
}
