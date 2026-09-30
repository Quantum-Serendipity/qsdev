package docs_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/archtest"
)

// claimsFile binds user-facing security sentences to the tests that prove
// them (security §7 TR-1). The repository root is this directory's parent.
const claimsFile = "security-claims.yaml"

// enforceCoverage turns the warn-mode coverage report (keyword sentences no
// claim covers, claims with no test) into failures. Metric M10 warns in W0
// and enforces from W3.
const enforceCoverage = false

type claimsDoc struct {
	Version   int          `yaml:"version"`
	Scan      []string     `yaml:"scan"`
	Keywords  []string     `yaml:"keywords"`
	Claims    []claim      `yaml:"claims"`
	Retracted []retraction `yaml:"retracted"`
}

type claim struct {
	ID    string   `yaml:"id"`
	Text  string   `yaml:"text"`
	Files []string `yaml:"files"`
	Tests []string `yaml:"tests"`
	Tiers []string `yaml:"tiers"`
}

// retraction is an overclaim that was withdrawn; it must not come back. It
// holds either a literal phrase or a regular expression pattern for its
// variants, both matched case-insensitively against folded text.
type retraction struct {
	Phrase  string `yaml:"phrase"`
	Pattern string `yaml:"pattern"`
	Reason  string `yaml:"reason"`
	Finding string `yaml:"finding"`
}

// testDecl is one top-level Test function declaration.
type testDecl struct {
	pkg string // package directory relative to the module root
	// skipVia names the function whose body calls Skip, Skipf or SkipNow:
	// the test itself or a same-package helper it reaches; "" if none.
	skipVia string
}

// testIndex maps each Test function name to its declarations.
type testIndex map[string][]testDecl

// claimsResult holds hard failures and warn-mode coverage gaps.
type claimsResult struct {
	failures []string
	warnings []string
}

func (r *claimsResult) fail(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *claimsResult) warn(format string, args ...any) {
	r.warnings = append(r.warnings, fmt.Sprintf(format, args...))
}

func TestSecurityClaims(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(claimsFile)
	if err != nil {
		t.Fatalf("reading %s: %v", claimsFile, err)
	}
	repo, err := archtest.Load("..")
	if err != nil {
		t.Fatalf("loading module: %v", err)
	}
	res, err := checkClaims(os.DirFS(".."), data, indexTests(repo))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.failures {
		t.Error(f)
	}
	report := t.Logf
	if enforceCoverage {
		report = t.Errorf
	}
	for _, w := range res.warnings {
		report("%s", w)
	}
	report("%d security-claim coverage warning(s)", len(res.warnings))
}

// indexTests records every Test function declared in a _test.go file.
func indexTests(repo *archtest.Repo) testIndex {
	funcs := map[string]map[string]*ast.FuncDecl{} // package key -> name -> decl
	for _, f := range repo.Files {
		key := f.Pkg + " " + f.AST.Name.Name
		if funcs[key] == nil {
			funcs[key] = map[string]*ast.FuncDecl{}
		}
		for _, decl := range f.AST.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Body != nil {
				funcs[key][fn.Name.Name] = fn
			}
		}
	}
	idx := testIndex{}
	for _, f := range repo.Files {
		if !f.IsTest {
			continue
		}
		pkgFuncs := funcs[f.Pkg+" "+f.AST.Name.Name]
		for _, decl := range f.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Body == nil {
				continue
			}
			via := skipVia(fn, pkgFuncs, map[string]bool{})
			idx[fn.Name.Name] = append(idx[fn.Name.Name], testDecl{pkg: f.Pkg, skipVia: via})
		}
	}
	return idx
}

// skipVia returns the name of fn, or of a same-package function it calls
// directly or transitively, whose body calls Skip, Skipf or SkipNow
// (subtests included), or "" when none does.
func skipVia(fn *ast.FuncDecl, pkgFuncs map[string]*ast.FuncDecl, seen map[string]bool) string {
	seen[fn.Name.Name] = true
	var helpers []*ast.FuncDecl
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			switch fun.Sel.Name {
			case "Skip", "Skipf", "SkipNow":
				found = true
			}
		case *ast.Ident:
			if h, ok := pkgFuncs[fun.Name]; ok && !seen[fun.Name] {
				helpers = append(helpers, h)
			}
		}
		return !found
	})
	if found {
		return fn.Name.Name
	}
	for _, h := range helpers {
		if seen[h.Name.Name] {
			continue
		}
		if via := skipVia(h, pkgFuncs, seen); via != "" {
			return via
		}
	}
	return ""
}

// lookup resolves a claim's test binding: a bare TestName, which must be
// unique in the module, or <package dir>.TestName.
func (idx testIndex) lookup(binding string) (testDecl, error) {
	pkg, name := "", binding
	if i := strings.LastIndex(binding, "."); i >= 0 {
		pkg, name = binding[:i], binding[i+1:]
	}
	var hits []testDecl
	for _, d := range idx[name] {
		if pkg == "" || d.pkg == pkg {
			hits = append(hits, d)
		}
	}
	switch {
	case len(hits) == 0:
		return testDecl{}, fmt.Errorf("test %s does not exist", binding)
	case len(hits) > 1:
		pkgs := make([]string, len(hits))
		for i, d := range hits {
			pkgs[i] = d.pkg
		}
		return testDecl{}, fmt.Errorf("test %s is ambiguous (packages %s); qualify it as <package dir>.%s",
			binding, strings.Join(pkgs, ", "), name)
	}
	return hits[0], nil
}

// checkClaims runs every TR-1 check of claimsYAML against the files in root.
// The error reports a claims file or scan glob that cannot be used at all.
func checkClaims(root fs.FS, claimsYAML []byte, tests testIndex) (claimsResult, error) {
	var res claimsResult
	doc, err := parseClaims(claimsYAML)
	if err != nil {
		return res, err
	}
	files, err := scanFiles(root, doc.Scan)
	if err != nil {
		return res, err
	}
	checkSchema(doc, &res)
	checkClaimText(root, doc, &res)
	checkClaimTests(doc, tests, &res)
	contents := make(map[string][]string, len(files))
	for _, name := range files {
		lines, err := readLines(root, name)
		if err != nil {
			res.fail("%v", err)
			continue
		}
		contents[name] = lines
	}
	checkCoverage(files, contents, doc, &res)
	// The claims file itself must not spell out what it retracts, so the
	// withdrawn wording appears nowhere in the repository.
	contents[claimsFile] = splitLines(claimsYAML)
	checkRetracted(append(files, claimsFile), contents, doc, &res)
	return res, nil
}

func parseClaims(data []byte) (*claimsDoc, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var doc claimsDoc
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", claimsFile, err)
	}
	return &doc, nil
}

var errEmptyGlob = errors.New("scan glob matches no file")

func scanFiles(root fs.FS, globs []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, g := range globs {
		matches, err := fs.Glob(root, g)
		if err != nil {
			return nil, fmt.Errorf("scan glob %q: %w", g, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("%q: %w", g, errEmptyGlob)
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out, nil
}

func checkSchema(doc *claimsDoc, res *claimsResult) {
	if doc.Version != 1 {
		res.fail("%s: version %d, want 1", claimsFile, doc.Version)
	}
	if len(doc.Keywords) == 0 {
		res.fail("%s: no keywords", claimsFile)
	}
	ids := map[string]bool{}
	for i, c := range doc.Claims {
		switch {
		case c.ID == "":
			res.fail("%s: claim #%d has no id", claimsFile, i+1)
		case ids[c.ID]:
			res.fail("%s: duplicate claim id %q", claimsFile, c.ID)
		}
		ids[c.ID] = true
		if normalize(c.Text) == "" || len(c.Files) == 0 {
			res.fail("%s: claim %q needs text and files", claimsFile, c.ID)
		}
	}
	for i, r := range doc.Retracted {
		if (normalize(r.Phrase) == "") == (r.Pattern == "") || r.Reason == "" || r.Finding == "" {
			res.fail("%s: retracted #%d needs one of phrase or pattern, a reason and a finding", claimsFile, i+1)
		}
	}
}

// checkClaimText requires each claim's text in every file it lists.
func checkClaimText(root fs.FS, doc *claimsDoc, res *claimsResult) {
	for _, c := range doc.Claims {
		want := normalize(c.Text)
		for _, name := range c.Files {
			lines, err := readLines(root, name)
			if err != nil {
				res.fail("claim %q: %v", c.ID, err)
				continue
			}
			if want != "" && !strings.Contains(normalize(strings.Join(lines, "\n")), want) {
				res.fail("claim %q: text not found in %s (drifted?): %q", c.ID, name, want)
			}
		}
	}
}

// testName strips an optional package qualifier from a test binding.
func testName(binding string) string {
	return binding[strings.LastIndex(binding, ".")+1:]
}

// checkClaimTests requires each listed test to exist and never skip. A
// static check: the tests themselves run in their own packages.
func checkClaimTests(doc *claimsDoc, tests testIndex, res *claimsResult) {
	for _, c := range doc.Claims {
		if len(c.Tests) == 0 {
			res.warn("claim %q: no test bound yet", c.ID)
		}
		for _, name := range c.Tests {
			d, err := tests.lookup(name)
			switch {
			case err != nil:
				res.fail("claim %q: %v", c.ID, err)
			case d.skipVia == testName(name):
				res.fail("claim %q: test %s calls t.Skip", c.ID, name)
			case d.skipVia != "":
				res.fail("claim %q: test %s calls t.Skip via %s", c.ID, name, d.skipVia)
			}
		}
	}
}

// checkRetracted rejects any withdrawn overclaim, fenced code included. It
// matches in any case, across line wraps and through markdown emphasis.
func checkRetracted(files []string, contents map[string][]string, doc *claimsDoc, res *claimsResult) {
	matchers := make([]*regexp.Regexp, len(doc.Retracted))
	for i, r := range doc.Retracted {
		m, err := r.matcher()
		if err != nil {
			res.fail("%s: retracted #%d: %v", claimsFile, i+1, err)
		}
		matchers[i] = m
	}
	for _, name := range files {
		text, starts := foldLines(contents[name])
		for i, r := range doc.Retracted {
			if matchers[i] == nil {
				continue
			}
			for _, loc := range matchers[i].FindAllStringIndex(text, -1) {
				line := sort.Search(len(starts), func(n int) bool { return starts[n] > loc[0] })
				kind, what := "phrase", r.Phrase
				if r.Pattern != "" {
					kind, what = "pattern", r.Pattern
				}
				res.fail("%s:%d: retracted %s %q matched %q (finding %s: %s)",
					name, line, kind, what, text[loc[0]:loc[1]], r.Finding, r.Reason)
			}
		}
	}
}

// matcher compiles the retraction to a case-insensitive expression over
// folded text; nil without error means there is nothing to match.
func (r retraction) matcher() (*regexp.Regexp, error) {
	expr := r.Pattern
	if expr == "" {
		phrase := fold(r.Phrase)
		if phrase == "" {
			return nil, nil
		}
		expr = regexp.QuoteMeta(phrase)
	}
	m, err := regexp.Compile("(?i)" + expr)
	if err != nil {
		return nil, fmt.Errorf("pattern %q: %w", r.Pattern, err)
	}
	return m, nil
}

// foldLines joins the folded lines into one text and returns the offset at
// which each line starts in it.
func foldLines(lines []string) (string, []int) {
	var b strings.Builder
	starts := make([]int, len(lines))
	for i, line := range lines {
		starts[i] = b.Len()
		norm := fold(line)
		if norm == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
			starts[i] = b.Len()
		}
		b.WriteString(norm)
	}
	return b.String(), starts
}

// inlineMarkup is markdown emphasis and code-span markup, which retraction
// matching ignores.
var inlineMarkup = strings.NewReplacer("*", "", "_", "", "`", "")

// markdownLink matches an inline markdown link; retraction matching keeps
// only its text, so a link target cannot split a withdrawn phrase.
var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// fold lowercases s, reduces links to their text, drops inline markup, folds
// apostrophes and collapses whitespace.
func fold(s string) string {
	s = markdownLink.ReplaceAllString(s, "$1")
	return strings.ToLower(normalize(apostrophes.Replace(inlineMarkup.Replace(s))))
}

// checkCoverage warns about prose lines with a security keyword that no
// claim for that file covers. Fenced code is not prose.
func checkCoverage(files []string, contents map[string][]string, doc *claimsDoc, res *claimsResult) {
	if len(doc.Keywords) == 0 {
		return
	}
	quoted := make([]string, len(doc.Keywords))
	for i, k := range doc.Keywords {
		quoted[i] = regexp.QuoteMeta(k)
	}
	keyword := regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoted, "|") + `)\b`)
	for _, name := range files {
		texts := claimTexts(doc, name)
		inFence := false
		for i, line := range contents[name] {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = !inFence
				continue
			}
			if inFence || !keyword.MatchString(apostrophes.Replace(line)) || covered(normalize(line), texts) {
				continue
			}
			res.warn("%s:%d: uncovered security sentence: %s", name, i+1, strings.TrimSpace(line))
		}
	}
}

// apostrophes folds typographic apostrophes so keywords such as "can't"
// match either spelling.
var apostrophes = strings.NewReplacer("\u2019", "'")

func claimTexts(doc *claimsDoc, file string) []string {
	var out []string
	for _, c := range doc.Claims {
		for _, f := range c.Files {
			if path.Clean(f) == file {
				out = append(out, normalize(c.Text))
			}
		}
	}
	return out
}

// covered reports whether line holds a claim text, or is one wrapped line of
// a longer claim.
func covered(line string, texts []string) bool {
	for _, t := range texts {
		if t != "" && (strings.Contains(line, t) || strings.Contains(t, line)) {
			return true
		}
	}
	return false
}

// readLines returns name's lines, split by splitLines.
func readLines(root fs.FS, name string) ([]string, error) {
	data, err := fs.ReadFile(root, name)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	return splitLines(data), nil
}

// splitLines splits data into lines with CRLF line endings normalised.
func splitLines(data []byte) []string {
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
}

// normalize collapses every whitespace run to one space.
func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
