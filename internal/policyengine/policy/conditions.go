package policy

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

type CompiledCondition interface {
	Evaluate(ctx *EvalContext) (bool, error)
}

type toolMatchCondition struct {
	toolName string
}

func (c *toolMatchCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return ctx.ToolName == c.toolName, nil
}

type globCondition struct {
	matcher *PathMatcher
}

func (c *globCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return matchContextPaths(c.matcher, ctx), nil
}

type regexCondition struct {
	re *regexp.Regexp
}

func (c *regexCondition) Evaluate(ctx *EvalContext) (bool, error) {
	target := ctx.Command
	if target == "" {
		target = string(ctx.ToolInput)
	}
	return c.re.MatchString(target), nil
}

// commandCondition matches a command word, or a whitespace-separated word
// sequence such as "npm install", inside a shell command line. The line is
// parsed so the pattern is found in every simple command — after `;`, `&&`,
// `|`, inside `$(...)`, backticks and subshells — and a command word invoked by
// path (`/usr/bin/curl`) or with a backslash escape (`\curl`) still matches.
// The whitespace-bounded regex is kept as an additional matcher so quoted
// mentions continue to match, and a line that cannot be parsed falls back to a
// regex that also treats shell metacharacters and a leading path as word
// boundaries (fail closed).
type commandCondition struct {
	words    []string
	re       *regexp.Regexp
	fallback *regexp.Regexp
}

// shellWordBoundary is the regex character class body of the characters that
// delimit a command word in shell syntax: whitespace, command separators and
// grouping, redirections, substitution and escape characters, and quotes.
const shellWordBoundary = `\s;&|()<>` + "`" + `$\\'"`

func newCommandCondition(pattern string) (*commandCondition, error) {
	quoted := regexp.QuoteMeta(pattern)
	re, err := regexp.Compile(`(?:^|\s)` + quoted + `(?:\s|$)`)
	if err != nil {
		return nil, fmt.Errorf("compiling command pattern %q: %w", pattern, err)
	}
	fallback, err := regexp.Compile(`(?:^|[` + shellWordBoundary + `/])` + quoted + `(?:[` + shellWordBoundary + `]|$)`)
	if err != nil {
		return nil, fmt.Errorf("compiling command pattern %q: %w", pattern, err)
	}
	return &commandCondition{words: strings.Fields(pattern), re: re, fallback: fallback}, nil
}

// maxNestedScriptDepth bounds how deeply commandCondition re-parses argument
// words as nested shell scripts (`sh -c '...'`, `eval "..."`).
const maxNestedScriptDepth = 3

func (c *commandCondition) Evaluate(ctx *EvalContext) (bool, error) {
	if ctx.Command == "" {
		return false, nil
	}
	return c.matchLine(ctx.Command, 0), nil
}

// matchLine reports whether the pattern occurs in the shell line. An argument,
// here-string or here-document word that itself contains shell syntax is
// re-parsed as a nested script, so a command smuggled through `sh -c`,
// `bash -c`, `eval`, `xargs sh -c` or `sh <<EOF` is found without a list of
// wrapper commands.
func (c *commandCondition) matchLine(line string, depth int) bool {
	if c.re.MatchString(line) {
		return true
	}
	cmds, err := cmdscan.Parse(line)
	if err != nil {
		return c.fallback.MatchString(line)
	}
	for _, cmd := range cmds {
		if cmd.Name == "" {
			continue
		}
		argv := append([]string{cmd.Name}, cmd.Args...)
		if containsWordSequence(argv, c.words) {
			return true
		}
		if depth >= maxNestedScriptDepth {
			continue
		}
		// Arguments (`sh -c '...'`), here-strings (`sh <<< '...'`) and
		// here-document bodies (`sh <<EOF`) can all carry a nested script.
		for _, word := range slices.Concat(cmd.Args, cmd.ReadRedirects, cmd.Heredocs) {
			if strings.ContainsAny(word, shellScriptChars) && c.matchLine(word, depth+1) {
				return true
			}
		}
	}
	return false
}

// shellScriptChars are the characters whose presence in an argument word means
// it may be a nested shell script rather than a plain operand.
const shellScriptChars = " \t\n;&|()<>`$"

// containsWordSequence reports whether want appears as a contiguous run of
// words in argv.
func containsWordSequence(argv, want []string) bool {
	if len(want) == 0 {
		return false
	}
	for i := 0; i+len(want) <= len(argv); i++ {
		match := true
		for j, w := range want {
			if !commandWordMatches(argv[i+j], w) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// commandWordMatches compares a parsed shell word to a pattern word after
// removing backslash escapes. A path-like word (`/usr/bin/curl`, `./curl`,
// `bin/curl`) also matches by its basename; URLs are not treated as paths. On a
// case-insensitive filesystem (`CURL` runs curl) the comparison folds case.
func commandWordMatches(word, want string) bool {
	word = strings.ReplaceAll(word, `\`, "")
	equal := func(a, b string) bool {
		if caseInsensitiveFS() {
			return strings.EqualFold(a, b)
		}
		return a == b
	}
	if equal(word, want) {
		return true
	}
	return strings.Contains(word, "/") && !strings.Contains(word, "://") && equal(path.Base(word), want)
}

type fileExistenceCondition struct {
	path string
}

func (c *fileExistenceCondition) Evaluate(ctx *EvalContext) (bool, error) {
	_, err := os.Stat(filepath.Join(ctx.CWD, c.path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking file existence for %q: %w", c.path, err)
}

type fileTypeCondition struct {
	path     string
	fileType string
}

func (c *fileTypeCondition) Evaluate(ctx *EvalContext) (bool, error) {
	info, err := os.Lstat(filepath.Join(ctx.CWD, c.path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking file type for %q: %w", c.path, err)
	}

	switch c.fileType {
	case "file":
		return info.Mode().IsRegular(), nil
	case "directory":
		return info.IsDir(), nil
	case "symlink":
		return info.Mode()&os.ModeSymlink != 0, nil
	default:
		return false, fmt.Errorf("unknown file type %q", c.fileType)
	}
}

type deniedPathCheckCondition struct {
	matcher *PathMatcher
}

func (c *deniedPathCheckCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return matchContextPaths(c.matcher, ctx), nil
}

// semanticIndicators are built-in, high-signal phrases that indicate prompt
// injection, data exfiltration, or attempts to tamper with security controls.
// They are matched case-insensitively against the evaluated tool call. The set
// is grounded in the semantic rule prompts from the security-pattern design
// (MCP-001 tool-description injection, MCP-006 cross-server exfiltration,
// CG-008 anti-rationalization). Every compiled semantic condition includes
// these in addition to any phrase quoted in its own prompt.
var semanticIndicators = []string{
	"ignore previous instructions",
	"ignore all previous instructions",
	"disregard previous instructions",
	"disregard all previous instructions",
	"override system behavior",
	"override system prompt",
	"reveal your system prompt",
	"exfiltrate",
	"disable security",
	"disable the security",
}

// semanticQuotedRE extracts single-quoted phrases from a semantic prompt. The
// design's semantic prompts enumerate the concrete phrases to look for inside
// single quotes (e.g. 'pre-existing issue', 'out of scope'), so those become
// rule-specific indicators.
var semanticQuotedRE = regexp.MustCompile(`'([^']+)'`)

// minQuotedIndicatorLen is the shortest single-quoted phrase promoted to a
// rule-specific indicator. A shorter phrase (e.g. 'x' or 'go') is too generic
// and would make the rule fire on nearly every tool call — a self-inflicted
// over-block from a careless prompt. Built-in indicators are curated and exempt.
const minQuotedIndicatorLen = 5

// semanticCondition performs a deterministic, offline heuristic match for the
// `semantic` condition type. Full LLM-backed evaluation is deferred, but the
// condition MUST NOT be a silent no-op (a rule that can never fire is a silent
// enforcement gap). It matches when any indicator phrase — built-in or quoted
// in the rule's prompt — appears in the tool input, command, or file path.
type semanticCondition struct {
	indicators []string
}

func newSemanticCondition(prompt string) *semanticCondition {
	seen := make(map[string]struct{})
	indicators := make([]string, 0, len(semanticIndicators))

	add := func(s string, minLen int) {
		s = strings.ToLower(strings.TrimSpace(s))
		if len(s) < minLen {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		indicators = append(indicators, s)
	}

	// Quoted phrases come from an author-written prompt, so enforce a length
	// floor; built-in indicators are curated multi-word phrases.
	for _, m := range semanticQuotedRE.FindAllStringSubmatch(prompt, -1) {
		add(m[1], minQuotedIndicatorLen)
	}
	for _, ind := range semanticIndicators {
		add(ind, 1)
	}

	return &semanticCondition{indicators: indicators}
}

func (c *semanticCondition) Evaluate(ctx *EvalContext) (bool, error) {
	haystack := strings.ToLower(semanticContent(ctx))
	if haystack == "" {
		return false, nil
	}
	for _, ind := range c.indicators {
		if strings.Contains(haystack, ind) {
			return true, nil
		}
	}
	return false, nil
}

// semanticContent assembles the textual surface a semantic condition inspects:
// the raw tool input (arguments/description), the command, and the file path.
func semanticContent(ctx *EvalContext) string {
	var b strings.Builder
	if len(ctx.ToolInput) > 0 {
		b.Write(ctx.ToolInput)
		b.WriteByte(' ')
	}
	b.WriteString(ctx.Command)
	b.WriteByte(' ')
	b.WriteString(ctx.FilePath)
	return b.String()
}

type allCondition struct {
	children []CompiledCondition
}

func (c *allCondition) Evaluate(ctx *EvalContext) (bool, error) {
	for _, child := range c.children {
		result, err := child.Evaluate(ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

type anyCondition struct {
	children []CompiledCondition
}

func (c *anyCondition) Evaluate(ctx *EvalContext) (bool, error) {
	for _, child := range c.children {
		result, err := child.Evaluate(ctx)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

type notCondition struct {
	child CompiledCondition
}

func (c *notCondition) Evaluate(ctx *EvalContext) (bool, error) {
	result, err := c.child.Evaluate(ctx)
	if err != nil {
		return false, err
	}
	return !result, nil
}

// CompileCondition validates cond's parameters and compiles it. Parameter
// errors (a missing required field, an unknown file_type) are reported here, at
// policy load time, rather than surfacing on every evaluation as a fail-closed
// Block that would stop every tool call the rule is indexed under.
func CompileCondition(cond Condition) (CompiledCondition, error) {
	if err := validateConditionParams(cond); err != nil {
		return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
	}

	switch cond.Type {
	case ToolMatch:
		return &toolMatchCondition{toolName: cond.ToolName}, nil

	case PathGlob:
		m, err := CompilePathMatcher(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &globCondition{matcher: m}, nil

	case RegexMatch:
		re, err := regexp.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: invalid regex %q: %w", cond.Type, cond.Pattern, err)
		}
		return &regexCondition{re: re}, nil

	case CommandMatch:
		c, err := newCommandCondition(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return c, nil

	case FileExistence:
		return &fileExistenceCondition{path: cond.Path}, nil

	case FileType:
		return &fileTypeCondition{path: cond.Path, fileType: cond.FileType}, nil

	case DeniedPathCheck:
		m, err := CompilePathMatcher(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &deniedPathCheckCondition{matcher: m}, nil

	case Semantic:
		return newSemanticCondition(cond.Prompt), nil

	case All:
		children, err := compileChildren(cond.Conditions)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &allCondition{children: children}, nil

	case Any:
		children, err := compileChildren(cond.Conditions)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &anyCondition{children: children}, nil

	case Not:
		if cond.Condition == nil {
			return nil, fmt.Errorf("compiling %s condition: missing child condition", cond.Type)
		}
		child, err := CompileCondition(*cond.Condition)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &notCondition{child: child}, nil

	default:
		return nil, fmt.Errorf("compiling condition: unknown type %q", cond.Type)
	}
}

func compileChildren(conditions []Condition) ([]CompiledCondition, error) {
	children := make([]CompiledCondition, 0, len(conditions))
	for _, child := range conditions {
		compiled, err := CompileCondition(child)
		if err != nil {
			return nil, err
		}
		children = append(children, compiled)
	}
	return children, nil
}

// fileTypes are the values accepted by a file_type condition.
var fileTypes = []string{"file", "directory", "symlink"}

// validateConditionParams checks that cond carries the parameters its type
// requires. Composite children are validated when they are compiled.
func validateConditionParams(cond Condition) error {
	switch cond.Type {
	case ToolMatch:
		if cond.ToolName == "" {
			return fmt.Errorf("tool_name is required")
		}
	case PathGlob, RegexMatch, CommandMatch, DeniedPathCheck:
		if strings.TrimSpace(cond.Pattern) == "" {
			return fmt.Errorf("pattern is required")
		}
	case FileExistence:
		if cond.Path == "" {
			return fmt.Errorf("path is required")
		}
	case FileType:
		if cond.Path == "" {
			return fmt.Errorf("path is required")
		}
		if !slices.Contains(fileTypes, cond.FileType) {
			return fmt.Errorf("unknown file_type %q (expected one of %s)", cond.FileType, strings.Join(fileTypes, ", "))
		}
	case All, Any:
		// An empty `all` is vacuously true and would match every tool call.
		if len(cond.Conditions) == 0 {
			return fmt.Errorf("at least one child condition is required")
		}
	}
	return nil
}
