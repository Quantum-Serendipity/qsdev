package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
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
	glob glob.Glob
}

func (c *globCondition) Evaluate(ctx *EvalContext) (bool, error) {
	path := ctx.FilePath
	if path == "" {
		path = extractPathFromInput(ctx.ToolInput)
	}
	return c.glob.Match(path), nil
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

type commandCondition struct {
	re *regexp.Regexp
}

func (c *commandCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return c.re.MatchString(ctx.Command), nil
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
	glob glob.Glob
}

func (c *deniedPathCheckCondition) Evaluate(ctx *EvalContext) (bool, error) {
	path := ctx.FilePath
	if path == "" {
		path = extractPathFromInput(ctx.ToolInput)
	}
	return c.glob.Match(path), nil
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

// semanticCondition performs a deterministic, offline heuristic match for the
// `semantic` condition type. Full LLM-backed evaluation is deferred, but the
// condition MUST NOT be a silent no-op (a rule that can never fire is a silent
// enforcement gap). It matches when any indicator phrase — built-in or quoted
// in the rule's prompt — appears in the tool input, command, or file path.
type semanticCondition struct {
	prompt     string
	indicators []string
}

func newSemanticCondition(prompt string) *semanticCondition {
	seen := make(map[string]struct{})
	indicators := make([]string, 0, len(semanticIndicators))

	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		indicators = append(indicators, s)
	}

	for _, m := range semanticQuotedRE.FindAllStringSubmatch(prompt, -1) {
		add(m[1])
	}
	for _, ind := range semanticIndicators {
		add(ind)
	}

	return &semanticCondition{prompt: prompt, indicators: indicators}
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

func CompileCondition(cond Condition) (CompiledCondition, error) {
	switch cond.Type {
	case ToolMatch:
		return &toolMatchCondition{toolName: cond.ToolName}, nil

	case PathGlob:
		g, err := glob.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &globCondition{glob: g}, nil

	case RegexMatch:
		re, err := regexp.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: invalid regex %q: %w", cond.Type, cond.Pattern, err)
		}
		return &regexCondition{re: re}, nil

	case CommandMatch:
		pattern := `(?:^|\s)` + regexp.QuoteMeta(cond.Pattern) + `(?:\s|$)`
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &commandCondition{re: re}, nil

	case FileExistence:
		return &fileExistenceCondition{path: cond.Path}, nil

	case FileType:
		return &fileTypeCondition{path: cond.Path, fileType: cond.FileType}, nil

	case DeniedPathCheck:
		g, err := glob.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &deniedPathCheckCondition{glob: g}, nil

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

func extractPathFromInput(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	for _, key := range []string{"file_path", "path", "file"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
